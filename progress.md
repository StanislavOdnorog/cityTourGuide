# Progress Log — City Stories Guide

Этот файл используется агентами для логирования прогресса по задачам из tasks.json.

## Формат записи

```
### TASK-XXX: Краткое описание
- **Дата**: YYYY-MM-DD
- **Статус**: done / in_progress / blocked
- **Что сделано**: описание
- **Проблемы**: если были
- **Следующий шаг**: если задача не завершена
```

---

## Записи

### TASK-001: Инициализация Git-репозитория и корневых конфигурационных файлов
- **Дата**: 2026-02-22
- **Статус**: done
- **Что сделано**:
  - Git-репозиторий инициализирован (`git init`)
  - .gitignore создан (Go, Node.js, React Native/Expo, IDE, .env, build-артефакты)
  - .editorconfig создан (Go — tabs, JS/TS — 2 spaces, SQL — 2 spaces, UTF-8, LF)
  - README.md создан (описание проекта, структура монорепо, таблица tech stack)
  - Первый коммит создан с root config файлами и проектной документацией
- **Тесты**:
  - git status — чистое состояние (pass)
  - .gitignore содержит node_modules, *.exe, .env, vendor/ (pass)
  - .editorconfig — Go tabs, JS/TS 2 spaces (pass)

### TASK-002: Инициализация Go-бэкенда: модуль, структура директорий, точка входа
- **Дата**: 2026-02-22
- **Статус**: done
- **Что сделано**:
  - `go mod init github.com/saas/city-stories-guide/backend` выполнен
  - Gin v1.11.0 установлен как зависимость
  - cmd/api/main.go создан: HTTP-сервер на порту из ENV (default 8080)
  - cmd/worker/main.go создан: placeholder для фоновых задач
  - Структура internal/ создана: config, domain, handler, middleware, repository, service, platform, worker
  - doc.go файлы созданы во всех internal пакетах
  - GET /healthz возвращает 200 `{"status": "ok"}`
  - Директории migrations/ и scripts/ созданы
- **Тесты**:
  - `go build ./cmd/api` — успешная компиляция (pass)
  - `go build ./cmd/worker` — успешная компиляция (pass)
  - `curl localhost:8080/healthz` — 200 `{"status":"ok"}` (pass)
  - `ls -R internal/` — все 8 пакетов с doc.go (pass)

### TASK-003: Docker Compose для локальной разработки: PostgreSQL+PostGIS и MinIO
- **Дата**: 2026-02-22
- **Статус**: done
- **Что сделано**:
  - Создана директория `infra/` с `docker-compose.yml`
  - PostgreSQL: собран из `postgres:16` + PostGIS 3 через `infra/postgres/Dockerfile` (ARM64-совместимый)
  - Init-скрипт `initdb-postgis.sh` создаёт расширения `postgis` и `postgis_topology`
  - MinIO: `minio/minio:latest` с API на 9000 и Console на 9001
  - Volumes `pgdata` и `miniodata` для персистенции данных
  - `.env.example` с переменными окружения для обоих сервисов
  - Healthcheck настроен для PostgreSQL (`pg_isready`)
- **Проблемы**:
  - `postgis/postgis:16-3.4` не поддерживает ARM64 — решено через кастомный Dockerfile (postgres:16 + apt install postgis)
  - Порт 5432 занят другим контейнером — используется 5433 в `.env`
- **Тесты**:
  - `docker-compose up -d` — оба сервиса запускаются (pass)
  - `SELECT PostGIS_Version()` — возвращает 3.6 (pass)
  - MinIO Console на localhost:9001 — HTTP 200 (pass)
  - `docker-compose down && docker-compose up -d` — данные сохранились (pass)
