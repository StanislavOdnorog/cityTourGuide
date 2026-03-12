#!/usr/bin/env python3

import json
import sys
from pathlib import Path

ALLOWED_STATUSES = {"todo", "in_progress", "blocked", "done"}
ALLOWED_PRIORITIES = {"low", "medium", "high"}
REQUIRED_TASK_FIELDS = {"id", "title", "status", "priority"}
OPTIONAL_TASK_FIELDS = {"notes", "depends_on"}
TASKS_PATH = Path(__file__).resolve().parent.parent / "tasks.json"


def fail(message: str) -> int:
    print(f"ERROR: {message}", file=sys.stderr)
    return 1


def main() -> int:
    if not TASKS_PATH.exists():
        return fail(f"missing required file: {TASKS_PATH}")

    try:
        data = json.loads(TASKS_PATH.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        return fail(f"{TASKS_PATH} is not valid JSON: {exc}")

    if not isinstance(data, dict):
        return fail("tasks.json must contain a top-level JSON object")

    if data.get("schema_version") != 1:
        return fail("tasks.json schema_version must be 1")

    tasks = data.get("tasks")
    if not isinstance(tasks, list) or not tasks:
        return fail("tasks.json must contain a non-empty tasks array")

    seen_ids: set[str] = set()
    task_ids: set[str] = set()
    for index, task in enumerate(tasks, start=1):
        if not isinstance(task, dict):
            return fail(f"task #{index} must be a JSON object")

        allowed_fields = REQUIRED_TASK_FIELDS | OPTIONAL_TASK_FIELDS
        unknown_fields = sorted(set(task) - allowed_fields)
        missing_fields = sorted(field for field in REQUIRED_TASK_FIELDS if field not in task)
        if missing_fields:
            return fail(f"task #{index} is missing required fields: {', '.join(missing_fields)}")
        if unknown_fields:
            return fail(f"task #{index} has unknown fields: {', '.join(unknown_fields)}")

        task_id = task.get("id")
        title = task.get("title")
        status = task.get("status")
        priority = task.get("priority")

        if not isinstance(task_id, str) or not task_id:
            return fail(f"task #{index} is missing a non-empty string id")
        if task_id in seen_ids:
            return fail(f"duplicate task id: {task_id}")
        seen_ids.add(task_id)
        task_ids.add(task_id)

        if not isinstance(title, str) or not title.strip():
            return fail(f"{task_id} is missing a non-empty string title")

        if status not in ALLOWED_STATUSES:
            allowed = ", ".join(sorted(ALLOWED_STATUSES))
            return fail(f"{task_id} has invalid status {status!r}; allowed: {allowed}")

        if priority not in ALLOWED_PRIORITIES:
            allowed = ", ".join(sorted(ALLOWED_PRIORITIES))
            return fail(f"{task_id} has invalid priority {priority!r}; allowed: {allowed}")

        notes = task.get("notes")
        if notes is not None and not isinstance(notes, str):
            return fail(f"{task_id} notes must be a string when present")

        depends_on = task.get("depends_on")
        if depends_on is not None:
            if not isinstance(depends_on, list) or not all(isinstance(item, str) and item for item in depends_on):
                return fail(f"{task_id} depends_on must be an array of non-empty task ids when present")
            if len(depends_on) != len(set(depends_on)):
                return fail(f"{task_id} depends_on contains duplicate task ids")

    for task in tasks:
        task_id = task["id"]
        depends_on = task.get("depends_on") or []
        missing_dependencies = sorted(dep_id for dep_id in depends_on if dep_id not in task_ids)
        if missing_dependencies:
            return fail(f"{task_id} depends_on references unknown task ids: {', '.join(missing_dependencies)}")
        if task_id in depends_on:
            return fail(f"{task_id} cannot depend on itself")

    print(f"tasks.json is valid: {len(tasks)} tasks checked.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
