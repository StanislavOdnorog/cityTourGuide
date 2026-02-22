#!/bin/bash
set -e

TASKS_FILE="tasks.json"

log() {
    local level="$1"
    shift
    printf '[%s] [%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$level" "$*"
}

# Agent selection:
# - Set RALPH_AGENT=claude or RALPH_AGENT=codex to force.
# - Otherwise always prefers Claude first.
resolve_agent() {
    if [[ -n "${RALPH_AGENT:-}" ]]; then
        echo "$RALPH_AGENT"
        return 0
    fi
    if command -v claude >/dev/null 2>&1; then
        echo "claude"
        return 0
    fi
    if command -v codex >/dev/null 2>&1; then
        echo "codex"
        return 0
    fi
    return 1
}

run_agent() {
    local agent="$1"
    local prompt="$2"

    case "$agent" in
        claude)
            claude --permission-mode acceptEdits -p "$prompt"
            ;;
        codex)
            local output_file
            output_file="$(mktemp -t ralph_codex.XXXXXX)"
            # Use non-interactive Codex exec and capture only the last message.
            codex exec --dangerously-bypass-approvals-and-sandbox --color never -C "$PWD" --output-last-message "$output_file" "$prompt" >/dev/null
            cat "$output_file"
            rm -f "$output_file"
            ;;
        *)
            echo "Unsupported agent: $agent" >&2
            return 1
            ;;
    esac
}

# Функция проверки наличия pending задач
has_pending_tasks() {
    pending_count=$(grep -c '"status": "pending"' "$TASKS_FILE" 2>/dev/null || echo "0")
    [ "$pending_count" -gt 0 ]
}

iteration=1

is_rate_limited() {
  local s="$1"
  [[ "$s" == *"You've hit your limit"* ]] || [[ "$s" == *"hit your limit"* ]] || [[ "$s" == *"You've hit your usage limit"* ]]
}

other_agent() {
  local current="$1"
  if [[ "$current" == "claude" ]]; then
    echo "codex"
  else
    echo "claude"
  fi
}

is_claude_unavailable() {
  local s="$1"
  [[ "$s" == *"Claude is unavailable"* ]] \
    || [[ "$s" == *"Claude is currently unavailable"* ]] \
    || [[ "$s" == *"currently unavailable"* ]] \
    || [[ "$s" == *"overloaded"* ]] \
    || [[ "$s" == *"temporarily unavailable"* ]] \
    || [[ "$s" == *"service unavailable"* ]]
}

while has_pending_tasks; do
    iteration_attempt=1
    echo "Итерация $iteration"
    echo "-----------------------------------"

    # Показываем текущий статус задач
    pending=$(grep -c '"status": "pending"' "$TASKS_FILE" 2>/dev/null || echo "0")
    done_count=$(grep -c '"status": "done"' "$TASKS_FILE" 2>/dev/null || echo "0")
    echo "Задач pending: $pending, done: $done_count"
    echo "-----------------------------------"

    # Always start with claude
    if command -v claude >/dev/null 2>&1; then
        agent="claude"
    elif command -v codex >/dev/null 2>&1; then
        agent="codex"
    else
        echo "Не найден поддерживаемый агент. Установите 'claude' или 'codex'." >&2
        exit 1
    fi
    log INFO "Selected agent: $agent"


    prompt=$(cat <<'EOF'
@tasks.json @progress.txt

## Инструкция

1. Прочти tasks.json и progress.txt. Выбери ОДНУ задачу с наивысшим приоритетом
   (critical > high > medium > low) и статусом "pending".
   Убедись, что ВСЕ задачи из dependencies имеют статус "done".

2. Реализуй задачу. Работай ТОЛЬКО над выбранной задачей.
   Не трогай код, не связанный с ней.

3. Запусти TypeScript проверку: выполни `npx tsc --noEmit`.
   Исправь все ошибки перед продолжением.

4. Запусти тесты для выбранной задачи:
   Выполни `./scripts/verify-task.sh TASK-XXX` (замени XXX на номер задачи).
   - Если тесты падают — исправь код и запусти снова.
   - Только после прохождения всех тестов переходи к следующему шагу.
   - Если тест содержит только `expect(true).toBe(true)` — это placeholder,
     замени его реальной проверкой перед финализацией.

5. Меняй status задачи на "done" ТОЛЬКО после прохождения всех тестов из test_steps.

6. Добавь запись в progress.txt: дата, ID задачи, что сделано, результат тестов.

7. Сделай git commit с осмысленным сообщением.

РАБОТАЙ ТОЛЬКО НАД ОДНОЙ ФИЧЕЙ.
Если задача полностью выполнена и все тесты прошли, выведи <promise>COMPLETE</promise>.
EOF
)
  while :; do
    log INFO "Iteration $iteration, attempt $iteration_attempt: running agent '$agent'"
    set +e
    result=$(run_agent "$agent" "$prompt" 2>&1)
    exit_code=$?
    set -e

    log INFO "Agent '$agent' finished with exit code $exit_code"
    echo "$result"

    if [[ "$agent" == "claude" ]] && is_claude_unavailable "$result"; then
      if command -v codex >/dev/null 2>&1; then
        echo "⚠️ Claude unavailable. Switching to Codex..."
        log WARN "Claude unavailable detected, switching to codex for this iteration"
        agent="codex"
        iteration_attempt=$((iteration_attempt + 1))
        continue
      fi
      echo "❌ Claude unavailable and codex is not installed."
      log ERROR "Claude unavailable and codex not installed"
      exit 1
    fi

    if is_rate_limited "$result"; then
      # 1. Try claude first (already done, we start with claude)
      # 2. If claude is rate limited, try codex
      if [[ "$agent" == "claude" ]] && command -v codex >/dev/null 2>&1; then
        echo "⚠️ Rate limit hit for 'claude'. Switching to 'codex'..."
        log WARN "Rate limit detected for 'claude'. Switching to 'codex'"
        agent="codex"
        iteration_attempt=$((iteration_attempt + 1))
        continue
      fi

      # 3. If codex is rate limited, wait 900 seconds by default
      if [[ "$agent" == "codex" ]]; then
        rate_limit_sleep="${RALPH_RATE_LIMIT_SLEEP_SECONDS:-900}"
        wake_at=$(date -v+${rate_limit_sleep}S '+%Y-%m-%d %H:%M:%S' 2>/dev/null || date -d "+${rate_limit_sleep} seconds" '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo "unknown")
        echo "⏳ Rate limit hit for 'codex'. Sleeping ${rate_limit_sleep} seconds..."
        log WARN "Rate limit detected for 'codex'. Sleeping ${rate_limit_sleep}s, wake-up at $wake_at"
        sleep "$rate_limit_sleep"
        log INFO "Retrying after rate-limit sleep"
        # Reset to claude after sleep
        if command -v claude >/dev/null 2>&1; then
          agent="claude"
        fi
        iteration_attempt=$((iteration_attempt + 1))
        continue
      fi

      # If claude is rate limited but codex is not available, wait
      if [[ "$agent" == "claude" ]] && ! command -v codex >/dev/null 2>&1; then
        rate_limit_sleep="${RALPH_RATE_LIMIT_SLEEP_SECONDS:-900}"
        wake_at=$(date -v+${rate_limit_sleep}S '+%Y-%m-%d %H:%M:%S' 2>/dev/null || date -d "+${rate_limit_sleep} seconds" '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo "unknown")
        echo "⏳ Rate limit hit for 'claude'. Codex not available. Sleeping ${rate_limit_sleep} seconds..."
        log WARN "Rate limit detected for 'claude'. Codex not available. Sleeping ${rate_limit_sleep}s, wake-up at $wake_at"
        sleep "$rate_limit_sleep"
        log INFO "Retrying after rate-limit sleep"
        iteration_attempt=$((iteration_attempt + 1))
        continue
      fi

      echo "❌ Rate limit hit for '$agent' and unable to handle."
      log ERROR "Rate limit for '$agent' and unable to handle; exiting"
      exit 1
    fi

    # real failure
    if [ $exit_code -ne 0 ]; then
      echo "❌ Agent failed with exit code $exit_code"
      log ERROR "Agent '$agent' failed without matching retry conditions (exit code $exit_code)"
      exit $exit_code
    fi

    log INFO "Iteration $iteration succeeded on attempt $iteration_attempt with agent '$agent'"
    break
  done


    echo "$result"

    if [[ "$result" == *"<promise>COMPLETE</promise>"* ]]; then
        echo "✓ TASK выполнен!"
        # Проверяем, остались ли ещё pending задачи
        remaining=$(grep -c '"status": "pending"' "$TASKS_FILE" 2>/dev/null || echo "0")
        if [ "$remaining" -eq 0 ]; then
            echo "🎉 Все задачи выполнены!"
            exit 0
        fi
        echo "Осталось задач: $remaining. Продолжаю..."
    fi

    ((iteration++))
done

echo "Все задачи выполнены! Итераций: $((iteration-1))"
