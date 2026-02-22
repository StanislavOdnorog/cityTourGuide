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

### TASK-004: Настройка линтинга и форматирования для Go-бэкенда
- **Дата**: 2026-02-22
- **Статус**: done
- **Что сделано**:
  - `.golangci.yml` создан с линтерами: errcheck, govet, staticcheck, bodyclose, sqlclosecheck, nilerr, gofmt, goimports, revive, misspell, unconvert, prealloc, gocritic, gosec
  - Исключения для тест-файлов настроены (gosec, errcheck, gocritic, prealloc)
  - `Makefile` создан с целями: lint, fmt, test, build, clean
  - `make build` компилирует оба бинарника (api, worker) в `bin/`
  - `make test` запускает тесты с `-race -cover`
  - `make fmt` форматирует код через gofmt + goimports
  - golangci-lint v1.64.8 и goimports установлены
- **Тесты**:
  - `make lint` — 0 ошибок на чистом проекте (pass)
  - Unused var → `make lint` — ошибка найдена (pass)
  - `make fmt` — форматирование работает (pass)
  - `make build` — оба бинарника созданы в bin/ (pass)

### TASK-005: Инициализация React Native (Expo) проекта для мобильного приложения
- **Дата**: 2026-02-22
- **Статус**: done
- **Что сделано**:
  - Expo проект создан через `npx create-expo-app mobile --template blank-typescript` (SDK 54)
  - TypeScript strict mode включён в tsconfig.json
  - Expo Router v6 установлен и настроен (expo-router, expo-linking, expo-constants, react-native-screens, react-native-safe-area-context, react-native-gesture-handler, expo-system-ui)
  - app.json обновлён: name "City Stories Guide", slug "city-stories-guide", scheme "city-stories", plugin expo-router
  - package.json main entry point — `expo-router/entry`
  - Файловая маршрутизация создана: app/_layout.tsx (Root Stack), app/index.tsx (redirect → onboarding)
  - Route groups: app/(auth)/ (login), app/(main)/ (home), app/onboarding/ (index)
  - Каждая группа имеет свой _layout.tsx
  - src/ структура создана: api, services, store, components, hooks, constants, utils, types — каждая с index.ts
  - Path alias @/* → src/* настроен в tsconfig.json (baseUrl + paths)
  - Тестовый импорт `@/constants` работает в app/(main)/home.tsx
- **Тесты**:
  - `npx expo config` — конфиг валиден, SDK 54.0.0 (pass)
  - `npx expo-doctor` — 17/17 checks passed (pass)
  - `tsc --noEmit` — 0 ошибок типов (pass)
  - `ls -R app/ src/` — все директории и файлы на месте (pass)
  - Path alias @/ — TypeScript резолвит импорт из @/constants (pass)

### TASK-011: Настройка golang-migrate и миграция: PostGIS extension + таблица cities
- **Дата**: 2026-02-22
- **Статус**: done
- **Что сделано**:
  - golang-migrate v4.18.1 установлен (ARM64 binary)
  - Миграция 000001_create_extensions: `CREATE EXTENSION IF NOT EXISTS postgis` и `postgis_topology`
  - Миграция 000002_create_cities: таблица cities (id, name, name_ru, country, center_lat, center_lng, radius_km, is_active, download_size_mb, created_at, updated_at)
  - Makefile обновлён: цели `migrate-up`, `migrate-down`, `migrate-create`
  - DATABASE_URL по умолчанию настроен на localhost:5433 (локальный Docker PostgreSQL)
- **Тесты**:
  - `make migrate-up` — обе миграции выполнены успешно (pass)
  - `\dt` — таблица cities существует (pass)
  - `SELECT PostGIS_Version()` — PostGIS 3.6 активен (pass)
  - `make migrate-down` — таблица cities удалена (pass)
  - `make migrate-up` — таблица cities создана снова (pass)
  - `make lint` — 0 ошибок после изменений (pass)

### TASK-012: Миграция: таблица POI (Point of Interest) с PostGIS geography
- **Дата**: 2026-02-22
- **Статус**: done
- **Что сделано**:
  - Миграция `000003_create_pois.up.sql` создана
  - ENUM типы `poi_type` (building, street, park, monument, church, bridge, square, museum, district, other) и `poi_status` (active, disabled, pending_review) созданы
  - Таблица `poi` создана: id, city_id FK→cities, name, name_ru, location GEOGRAPHY(POINT,4326), type, tags JSONB, address, interest_score, status, created_at, updated_at
  - GIST-индекс `idx_poi_location` на location для пространственных запросов
  - Составной индекс `idx_poi_city_status` на (city_id, status)
  - FK constraint на city_id с ON DELETE CASCADE
  - Down-миграция корректно удаляет таблицу и оба ENUM типа
- **Тесты**:
  - `make migrate-up` — миграция выполнена успешно (pass)
  - INSERT 5 POI с координатами Тбилиси — успешно (pass)
  - ST_DWithin запрос (500m от Rike Park) — возвращает 3 ближайших POI с distance_m (pass)
  - INSERT с невалидным city_id=999 — FK ошибка (pass)
  - `make migrate-down` — таблица и ENUM типы удалены (pass)
  - `make migrate-up` — таблица создана снова (pass)
  - `make lint` — 0 ошибок (pass)

### TASK-013: Миграция: таблица stories
- **Дата**: 2026-02-22
- **Статус**: done
- **Что сделано**:
  - Миграция `000004_create_stories.up.sql` создана
  - ENUM типы `story_layer_type` (atmosphere, human_story, hidden_detail, time_shift, general) и `story_status` (active, disabled, reported, pending_review) созданы
  - Таблица `story` создана: id, poi_id FK→poi, language, text, audio_url, duration_sec, layer_type, order_index, is_inflation, confidence, sources JSONB, status, created_at, updated_at
  - Составной индекс `idx_story_poi_language_status` на (poi_id, language, status)
  - FK constraint на poi_id с ON DELETE CASCADE
  - Down-миграция корректно удаляет таблицу и оба ENUM типа
- **Тесты**:
  - `make migrate-up` — миграция выполнена успешно (pass)
  - INSERT 2 stories (EN + RU) с валидными данными — успешно (pass)
  - INSERT с невалидным poi_id=999 — FK ошибка (pass)
  - EXPLAIN SELECT по (poi_id, language, status) — использует idx_story_poi_language_status (pass)
  - `make migrate-down` — таблица и ENUM типы удалены (pass)
  - `make migrate-up` — таблица создана снова (pass)
  - `make lint` — 0 ошибок (pass)

### TASK-014: Миграции: users, user_listening, reports, purchases, inflation_jobs
- **Дата**: 2026-02-22
- **Статус**: done
- **Что сделано**:
  - Миграция `000005_create_users.up.sql`: таблица users (id UUID PK DEFAULT gen_random_uuid(), email UNIQUE, name, auth_provider ENUM, language_pref, is_anonymous, created_at, updated_at), частичный индекс на email
  - Миграция `000006_create_user_listening.up.sql`: таблица user_listening (id SERIAL, user_id FK→users, story_id FK→story, listened_at, completed, location GEOGRAPHY), индексы (user_id, story_id) и (user_id, listened_at DESC)
  - Миграция `000007_create_reports.up.sql`: таблица report (id SERIAL, story_id FK→story, user_id FK→users, type ENUM, comment, user_lat, user_lng, status ENUM DEFAULT 'new', resolved_at, created_at), частичный индекс WHERE status='new', индекс на story_id
  - Миграция `000008_create_purchases.up.sql`: таблица purchase (id SERIAL, user_id FK→users, type ENUM, city_id FK→cities NULLABLE, platform, transaction_id, price DECIMAL, is_ltd, expires_at, created_at), индексы на user_id и уникальный на transaction_id
  - Миграция `000009_create_inflation_jobs.up.sql`: таблица inflation_job (id SERIAL, poi_id FK→poi, status ENUM, trigger_type ENUM, segments_count, max_segments, started_at, completed_at, error_log, created_at), частичный индекс на status IN ('pending','running'), индекс на poi_id
  - ENUM типы созданы: auth_provider, report_type, report_status, purchase_type, inflation_job_status, inflation_trigger_type
  - Все down-миграции корректно удаляют таблицы и ENUM типы
- **Тесты**:
  - `make migrate-up` — все 5 миграций выполнены успешно (pass)
  - `\dt` — все 9 таблиц существуют (cities, poi, story, users, user_listening, report, purchase, inflation_job + schema_migrations) (pass)
  - INSERT в каждую из 5 новых таблиц с валидными данными — успешно (pass)
  - FK constraint: INSERT с невалидным user_id — ошибка FK (pass)
  - `make migrate-down` (5 раз) — таблицы удалены в обратном порядке (9→5) (pass)
  - `make migrate-up` — все таблицы созданы снова (pass)
  - 15 индексов на новых таблицах проверены через pg_indexes (pass)
  - `make lint` — 0 ошибок (pass)

### TASK-016: Domain-модели Go: структуры всех сущностей
- **Дата**: 2026-02-22
- **Статус**: done
- **Что сделано**:
  - Создано 8 файлов доменных моделей в `internal/domain/`:
    - `city.go` — City struct (11 полей), соответствует таблице cities
    - `poi.go` — POI struct (13 полей), ENUMs: POIType (10 значений), POIStatus (3 значения). Lat/Lng как float64 вместо PostGIS GEOGRAPHY для удобства Go-кода
    - `story.go` — Story struct (14 полей), ENUMs: StoryLayerType (5 значений), StoryStatus (4 значения). JSONB sources как json.RawMessage
    - `user.go` — User struct (8 полей), ENUM: AuthProvider (3 значения). ID как string (UUID)
    - `listening.go` — UserListening struct (7 полей). Nullable координаты как *float64
    - `report.go` — Report struct (10 полей), ENUMs: ReportType (3 значения), ReportStatus (4 значения)
    - `purchase.go` — Purchase struct (10 полей), ENUM: PurchaseType (3 значения). Price как float64
    - `inflation.go` — InflationJob struct (10 полей), ENUMs: InflationJobStatus (4 значения), InflationTriggerType (2 значения)
  - Все ENUM значения определены как Go типы с const block
  - JSON tags добавлены на все поля (83 поля total)
  - Nullable поля используют указатели (*string, *int, *float64, *time.Time, *int16)
  - JSONB поля используют json.RawMessage
- **Тесты**:
  - `go build ./internal/domain/...` — компиляция успешна (pass)
  - Все 8 структур имеют json tags на каждом поле (pass)
  - `make lint` — 0 ошибок линтинга (pass)

### TASK-015: Конфигурация бэкенда: загрузка ENV переменных и структура Config
- **Дата**: 2026-02-22
- **Статус**: done
- **Что сделано**:
  - Создан `internal/config/config.go` с полной структурой Config
  - Sub-structs: ServerConfig (Port, Mode), DatabaseConfig (URL), S3Config (Endpoint, AccessKey, SecretKey, Bucket), ClaudeConfig (APIKey), ElevenLabsConfig (APIKey), JWTConfig (Secret, AccessTTL, RefreshTTL)
  - godotenv v1.5.1 установлен для загрузки `.env` файлов
  - `Load()` — загружает ENV с defaults (Port=8080, GIN_MODE=debug, S3_BUCKET=city-stories, AccessTTL=15m, RefreshTTL=7d)
  - `validate()` — проверяет обязательные поля (DATABASE_URL, JWT_SECRET), падает с понятной ошибкой
  - `LogSafe()` — маскирует sensitive данные: Database URL = ***, API keys показывают только первые/последние 4 символа
  - `cmd/api/main.go` интегрирован с config.Load(), gin.SetMode()
  - `backend/.env.example` создан со всеми переменными и комментариями
  - Удалён дублирующий `doc.go` (package comment перенесён в config.go)
- **Тесты**:
  - Запуск без DATABASE_URL — `config: DATABASE_URL is required`, exit 1 (pass)
  - Запуск без JWT_SECRET — `config: JWT_SECRET is required`, exit 1 (pass)
  - Запуск с .env — конфиг загружен корректно, сервер стартует (pass)
  - API keys замаскированы в логах: `sk-a***cdef`, Database URL = `***` (pass)
  - `make lint` — 0 ошибок (pass)
  - `make build` — оба бинарника скомпилированы (pass)

### TASK-017: Repository: подключение к PostgreSQL (pgx) и город CRUD
- **Дата**: 2026-02-22
- **Статус**: done
- **Что сделано**:
  - Установлен `pgx/v5` v5.8.0 (pgxpool для connection pooling)
  - Создан `internal/repository/db.go` — `NewPool(ctx, databaseURL)` инициализирует pgxpool.Pool с ping-проверкой
  - Создан `internal/repository/errors.go` — `ErrNotFound` для несуществующих записей
  - Создан `internal/repository/city_repo.go` — полный CRUD: Create, GetByID, GetAll, Update, Delete
  - Создан `internal/repository/testing.go` — TestPool helper для интеграционных тестов
  - Создан `internal/repository/city_repo_test.go` — 9 интеграционных тестов (build tag `integration`)
  - `cmd/api/main.go` рефакторен: `run()` паттерн, graceful shutdown (SIGINT/SIGTERM), pool.Close() через defer, ReadHeaderTimeout для gosec
  - GetByID/Update/Delete возвращают `ErrNotFound` при отсутствии записи
  - Все SQL-запросы параметризованы ($1, $2...) для защиты от SQL injection
- **Тесты**:
  - Запуск приложения — `Database connection established` в логах (pass)
  - Create → GetByID → Update → Delete полный цикл (TestCityRepo_FullCRUDCycle) (pass)
  - GetByID несуществующего → ErrNotFound (TestCityRepo_GetByID_NotFound) (pass)
  - `go test -tags integration -race ./internal/repository/...` — 9/9 тестов (pass)
  - `make lint` — 0 ошибок (pass)
  - `make build` — оба бинарника скомпилированы (pass)

### TASK-018: Repository: POI CRUD с PostGIS пространственными запросами
- **Дата**: 2026-02-22
- **Статус**: done
- **Что сделано**:
  - Создан `internal/repository/poi_repo.go` — полный POI CRUD + FindNearby
  - Структура `NearbyPOI` расширяет domain.POI с DistanceM
  - Create: использует `ST_SetSRID(ST_MakePoint(lng, lat), 4326)::geography` для вставки координат
  - GetByID: извлекает lat/lng через `ST_Y(location::geometry)` / `ST_X(location::geometry)`
  - GetByCityID: поддержка фильтров по status и type через динамические параметры
  - Update: обновляет все поля включая location через ST_MakePoint
  - Delete: с проверкой RowsAffected для ErrNotFound
  - FindNearby: `ST_DWithin` запрос с INNER JOIN story (status='active', language filter), DISTINCT ON (p.id), обёрнутый подзапросом для ORDER BY interest_score DESC, distance_m ASC, LIMIT 20
  - Создан `internal/repository/poi_repo_test.go` — 13 интеграционных тестов (build tag `integration`)
- **Тесты**:
  - 13/13 интеграционных тестов POI repo проходят с `-race` (pass)
  - Create POI с ST_MakePoint — координаты сохраняются корректно (pass)
  - GetByCityID с фильтрами status/type — фильтрация работает (pass)
  - FindNearby: 500m радиус возвращает ближайшие POI с stories (pass)
  - FindNearby: 50m радиус возвращает подмножество (pass)
  - FindNearby: language filter — EN/RU фильтрация работает (pass)
  - EXPLAIN запроса — использует idx_poi_city_status и idx_story_poi_language_status (pass)
  - Full CRUD cycle: Create → GetByID → Update → Delete (pass)
  - `make lint` — 0 ошибок (pass)
  - `make build` — оба бинарника скомпилированы (pass)

### TASK-019: Repository: Story CRUD и UserListening tracking
- **Дата**: 2026-02-22
- **Статус**: done
- **Что сделано**:
  - Создан `internal/repository/story_repo.go` — полный Story CRUD + CountByPOI
  - Методы: Create, GetByID, GetByPOIID(poiID, language, status), Update, Delete, CountByPOI
  - GetByPOIID поддерживает фильтрацию по языку (обязательно) и статусу (опционально)
  - Результаты GetByPOIID сортируются по order_index, created_at
  - Создан `internal/repository/listening_repo.go` — UserListening tracking
  - Методы: Create(userID, storyID, completed, lat, lng), GetListenedStoryIDs(userID), HasListened(userID, storyID)
  - Create поддерживает nullable location через ST_MakePoint (два SQL-варианта: с координатами и без)
  - GetListenedStoryIDs использует DISTINCT для дедупликации
  - HasListened использует EXISTS subquery для эффективной проверки
  - Создан `internal/repository/story_repo_test.go` — 10 интеграционных тестов
  - Создан `internal/repository/listening_repo_test.go` — 6 интеграционных тестов
  - Helper-функции: createTestStory, createTestUserDirect, deleteTestUser
- **Тесты**:
  - Create POI → Create 3 stories (2 EN + 1 RU) — успешно (pass)
  - GetByPOIID language=en — возвращает только 2 EN истории (pass)
  - GetByPOIID с status filter — фильтрация работает (pass)
  - CountByPOI — 0 до вставки, 3 после (pass)
  - Create listening с координатами — lat/lng сохраняются через ST_MakePoint (pass)
  - Create listening без координат — lat/lng = nil (pass)
  - GetListenedStoryIDs — возвращает 2 прослушанных, не содержит непрослушанную (pass)
  - GetListenedStoryIDs дедупликация — 2 записи → 1 уникальный ID (pass)
  - HasListened — true для прослушанной, false для новой (pass)
  - Full CRUD cycle для Story: Create → GetByID → Update → Delete (pass)
  - `go test -tags integration -race` — 38/38 тестов (pass)
  - `make lint` — 0 ошибок (pass)
  - `make build` — оба бинарника скомпилированы (pass)

### TASK-020: Service: NearbyService — ядро выбора историй
- **Дата**: 2026-02-22
- **Статус**: done
- **Что сделано**:
  - Создан `internal/service/nearby_service.go` — NearbyService с scoring-алгоритмом из PRD
  - Метод `GetNearbyStories(ctx, lat, lng, radiusM, heading, speed, userID, language) → []StoryCandidate`
  - Структура `StoryCandidate`: poi_id, poi_name, story_id, story_text, audio_url, duration_sec, distance_m, score
  - **Scoring-алгоритм**: `score = base_interest_score + proximity_bonus + direction_bonus`
    - `ProximityBonus`: линейно растёт от 0 (на границе радиуса) до 30 (на расстоянии 0м)
    - `DirectionBonus`: +20% от base score если POI в пределах ±45° от heading пользователя
    - `Bearing`: вычисление начального азимута (формула haversine initial bearing)
    - `AngleDiff`: минимальная угловая разница с wraparound (0°/360°)
  - Прослушанные истории исключаются через `GetListenedStoryIDs`
  - Возвращается не более 5 кандидатов, отсортированных по score DESC
  - Dependency injection через интерфейсы: POIFinder, StoryGetter, ListeningGetter
  - Добавлен `FindNearbyAll` метод в POIRepo (FindNearby без cityID фильтра)
  - Создан `internal/service/nearby_service_test.go` — 29 unit-тестов:
    - ProximityBonus: 6 тестов (zero distance, half, at radius, beyond, zero radius, linearity)
    - Bearing: 4 теста (N/S/E/W)
    - AngleDiff: 4 теста (same, opposite, wraparound, symmetry)
    - DirectionBonus: 4 теста (ahead, behind, at limit, no heading)
    - CalculateScore: 2 теста (all components, no bonuses)
    - GetNearbyStories: 9 тестов (sorted, listened excluded, all listened, direction bonus, max 5, no POIs, empty user, multiple stories per POI, candidate fields)
- **Тесты**:
  - `go test -v -race ./internal/service/...` — 29/29 тестов PASS (pass)
  - Sorted by score: POI с высоким interest_score первым (pass)
  - Listened excluded: прослушанная история не возвращается (pass)
  - Direction bonus: POI впереди (heading=90°, POI на восток) имеет выше score чем POI позади (pass)
  - Proximity bonus: линейно от 0 до 30 (pass)
  - Max 5 candidates: из 8 возвращается только 5 (pass)
  - `make lint` — 0 ошибок (pass)
  - `make build` — оба бинарника скомпилированы (pass)

### TASK-021: Handler: GET /api/v1/nearby-stories — главный API endpoint
- **Дата**: 2026-02-22
- **Статус**: done
- **Что сделано**:
  - Создан `internal/handler/nearby_handler.go` — handler для GET /api/v1/nearby-stories
  - Интерфейс `NearbyStoriesGetter` для dependency injection (тестируемость)
  - Полная валидация query-параметров: lat [-90,90] (required), lng [-180,180] (required), radius [10,500] (default 150), heading (default -1), speed (default 0), language (default "en"), user_id (optional)
  - HTTP 400 с описательным `error` полем при невалидных данных
  - HTTP 200 с `{"data": [...]}` при успешном запросе (пустой массив если нет историй)
  - HTTP 500 при ошибке сервиса (без утечки деталей)
  - `cmd/api/main.go` рефакторен: инициализация POIRepo, StoryRepo, ListeningRepo, NearbyService, NearbyHandler
  - Маршрут зарегистрирован: GET /api/v1/nearby-stories
  - Удалён неиспользуемый `_ = repository.NewCityRepo(pool)`
  - Создан `internal/handler/nearby_handler_test.go` — 11 unit-тестов:
    - Success: 200 с данными, POI name и audio_url в ответе
    - EmptyResult: 200 с пустым массивом (не null)
    - MissingLat: 400 "lat is required"
    - MissingLng: 400 "lng is required"
    - InvalidLat: 3 sub-теста (too high, too low, not a number)
    - InvalidLng: 3 sub-теста (too high, too low, not a number)
    - InvalidRadius: 3 sub-теста (too small, too large, not a number)
    - DefaultValues: radius=150, heading=-1, speed=0, language="en"
    - AllParams: все параметры передаются в сервис корректно
    - ServiceError: 500 с generic error message
    - ResponseIncludesAudioURL: audio_url, duration_sec, distance_m, score в ответе
    - BoundaryLatLng: lat=-90/90 и lng=-180/180 — валидные
- **Тесты**:
  - `go test -race ./internal/handler/...` — 11/11 тестов PASS (pass)
  - GET /api/v1/nearby-stories?lat=41.7151&lng=44.8271&language=en — 200 с данными (pass)
  - GET без lat — 400 "lat is required" (pass)
  - GET с lat=999 — 400 "lat must be between -90 and 90" (pass)
  - audio_url присутствует в ответе (pass)
  - `make lint` — 0 ошибок (pass)
  - `make build` — оба бинарника скомпилированы (pass)

### TASK-023: S3-клиент: загрузка/скачивание аудиофайлов, presigned URLs
- **Дата**: 2026-02-22
- **Статус**: done
- **Что сделано**:
  - Установлен `aws-sdk-go-v2` v1.41.1 с модулями s3, config, credentials
  - Создан `internal/platform/s3/client.go` — S3-compatible storage client
  - Структура `Config`: Endpoint, AccessKey, SecretKey, Bucket, Region
  - `NewClient(ctx, *Config)` — инициализирует клиент с `BaseEndpoint` + `UsePathStyle` (совместимость с MinIO)
  - `ensureBucket()` — проверяет существование bucket через HeadBucket, создаёт если отсутствует
  - `Upload(ctx, key, reader, contentType) → url` — загружает файл через PutObject, возвращает публичный URL
  - `GetPresignedURL(ctx, key, expiry) → url` — генерирует presigned URL через s3.PresignClient
  - `Delete(ctx, key)` — удаляет объект через DeleteObject
  - `Exists(ctx, key) → bool` — проверяет наличие через HeadObject с обработкой NotFound
  - `AudioKey(cityID, poiID, storyID)` — helper для формирования ключа `audio/{cityID}/{poiID}/{storyID}.mp3`
  - `isNotFound()` — проверка ошибок NotFound/NoSuchKey через errors.As + smithy.APIError
  - Создан `internal/platform/s3/client_test.go` — 8 интеграционных тестов (build tag `integration`)
- **Тесты**:
  - NewClient_CreatesBucket — bucket создаётся при инициализации (pass)
  - Upload_And_Exists — загрузка файла + проверка существования (pass)
  - Upload_MP3_ContentType — загрузка с content-type audio/mpeg (pass)
  - GetPresignedURL — presigned URL генерируется, файл скачивается через HTTP GET (pass)
  - Delete — файл удаляется, Exists возвращает false (pass)
  - Exists_NotFound — несуществующий ключ возвращает false без ошибки (pass)
  - FullCycle_Upload_Presign_Delete — полный цикл: Upload → Exists → Presign → Download → Delete → Verify gone (pass)
  - AudioKey — формат ключа `audio/1/10/100.mp3` (pass)
  - `go test -tags integration -race ./internal/platform/s3/...` — 8/8 тестов (pass)
  - `make lint` — 0 ошибок (pass)
  - `make build` — оба бинарника скомпилированы (pass)

### TASK-033: Mobile: API-клиент и типы данных
- **Дата**: 2026-02-22
- **Статус**: done
- **Что сделано**:
  - Установлен `axios` HTTP клиент в mobile проекте
  - Создан `src/types/index.ts` — 8 доменных интерфейсов (City, POI, Story, User, UserListening, Report, Purchase, NearbyStoryCandidate), 8 union types для ENUM'ов (POIType, POIStatus, StoryLayerType, StoryStatus, AuthProvider, ReportType, ReportStatus, PurchaseType), API request/response типы (NearbyStoriesResponse, TrackListeningRequest, ReportStoryRequest, ApiError)
  - Создан `src/api/client.ts` — axios instance с baseURL из `@/constants`, JWT interceptor (добавляет Bearer token к запросам), 401 interceptor с refresh token queue (множественные запросы ждут обновления), 429 interceptor с retry (Retry-After header), `setTokens()` и `setRefreshHandler()` для внешнего управления токенами
  - Создан `src/api/endpoints.ts` — `fetchNearbyStories(params)` → GET /api/v1/nearby-stories, `trackListening(request)` → POST /api/v1/listenings, `reportStory(request)` → POST /api/v1/reports, `fetchCities()` → GET /api/v1/cities, `fetchCityById(id)` → GET /api/v1/cities/:id
  - Обновлён `src/api/index.ts` — re-export всех функций, типов и клиента
  - Все TypeScript типы соответствуют backend Go структурам (json tags → interface fields)
- **Тесты**:
  - `tsc --noEmit` — 0 ошибок типов (pass)
  - `--listFiles` — все 3 файла (client.ts, types/index.ts, endpoints.ts) резолвятся TypeScript (pass)
  - JWT token автоматически добавляется через request interceptor (pass — code review)
  - 401 → refresh token → retry, 429 → retry with backoff (pass — code review)
  - `@/api` и `@/types` path aliases работают (pass)
