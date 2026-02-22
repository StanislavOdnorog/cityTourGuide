# PRD — City Stories Guide

## 1. App Overview & Goals

### 1.1 Product Vision

Ambient audio-гид, который рассказывает истории о местах вокруг пользователя, пока тот просто гуляет по городу. Приложение работает параллельно с Google Maps — навигация остаётся в Maps, а City Stories Guide создаёт "параллельный аудио-слой реальности", превращая любую прогулку в живой документальный фильм.

### 1.2 One-Sentence Pitch

Гуляй по городу — а он сам расскажет тебе свои истории.

### 1.3 Core Principles

- **Silence is default** — история звучит только когда есть что рассказать
- **Never interrupt** — никаких кнопок, остановок, маршрутов. Только ходьба + слушание
- **No behavior change** — пользователь делает то, что и так делает (наушники, телефон в кармане, прогулка)
- **Storyteller, not Wikipedia** — эмоциональный рассказ, а не сухие факты
- **Pacing matters** — темп рассказа зависит от скорости ходьбы

### 1.4 Success Metrics (MVP)

| Metric | Target |
|---|---|
| First walk completion rate | > 70% пользователей дослушали первую прогулку |
| Daily active listeners | 500+ в первый месяц после запуска |
| Average session duration | > 15 минут |
| Story report rate | < 5% историй получают репорт |
| App Store rating | > 4.5 |
| LTD conversions (первые 3 месяца) | 200+ покупок |

---

## 2. Target Audience

### 2.1 Primary: Solo-путешественник

- **Возраст**: 25–45 лет
- **Поведение**: часто путешествует один, предпочитает самостоятельное исследование города
- **Боль**: гуляет по красивым улицам, но город остаётся декорацией. 90% истории проходит мимо
- **Мотивация**: хочет возвращаться из поездки не с фотографиями, а с историями. Хочет чувствовать единение с городом
- **Платёжеспособность**: готов платить $5–30 за качественный travel-опыт

### 2.2 Secondary

- **Экспаты и цифровые номады** — живут в городе, но не знают его. Daily companion
- **Локальные жители** — хотят заново открыть свой город
- **Пары-путешественники** — один включил, оба слушают

### 2.3 Anti-personas (кому продукт НЕ подходит)

- Групповые туристы (у них уже есть гид)
- Люди, которые не носят наушники при ходьбе
- Пользователи без смартфона с GPS

---

## 3. Core Features & Functionality

### 3.1 MVP Features (v1)

#### F1: Background Location Tracking

Приложение отслеживает GPS-позицию пользователя в фоновом режиме.

- **Адаптивный режим**: при ходьбе — опрос GPS каждые 5–10 секунд; при остановке — переход на geofencing (энергосбережение)
- **Данные**: координаты, скорость, направление движения (compass heading)
- **Автопауза**: если пользователь стоит на месте > 2 минут — приложение переходит в спящий режим
- **Автовозобновление**: при начале движения — возобновляет мониторинг

**Acceptance criteria:**
- Приложение корректно отслеживает позицию в background на Android и iOS
- Батарея не расходуется более чем на 5–8% в час при активной прогулке
- Переключение между режимами (активный/спящий) происходит без задержки > 3 секунд

---

#### F2: Story Engine (Proximity-Triggered Narration)

Ядро продукта — система, которая решает, когда и какую историю рассказать.

**Логика работы:**

1. Каждые N секунд (зависит от скорости) система проверяет: есть ли поблизости (radius 50–150м) точка с историей
2. Из кандидатов выбирается лучшая точка по scoring-алгоритму
3. Воспроизводится аудио-история

**Scoring-алгоритм (приоритизация):**

```
score = base_interest_score
      + proximity_bonus (ближе = выше)
      + direction_bonus (впереди пользователя = выше)
      - recently_played_penalty (не повторять)
      - overload_penalty (не чаще 1 истории в 2 минуты)
```

**Pacing Rules:**
- Silence is default — не более 1 истории за 2 минуты
- При быстрой ходьбе — короткие истории (15–20 сек)
- При медленной прогулке — развёрнутые истории (30–45 сек)
- Не прерывать текущую историю ради новой
- Не повторять историю, которую пользователь уже слышал (в рамках сессии и глобально)

**Acceptance criteria:**
- Истории воспроизводятся только при приближении к точке интереса
- Пользователь никогда не слышит одну историю дважды за прогулку
- Между историями всегда есть пауза минимум 60 секунд
- Направление движения учитывается (приоритет точкам впереди)

---

#### F3: Story Content System (Caching + Inflation)

Двухуровневая система контента.

**Уровень 1: Cached Stories (pre-generated)**

- Для каждой точки интереса заранее сгенерирована и озвучена базовая история
- Хранится: текст + аудиофайл (MP3/AAC)
- Типичная длительность: 15–45 секунд
- Генерируется через Claude API → ElevenLabs TTS

**Уровень 2: Inflation (AI-extended content)**

- Если пользователь долго находится рядом с точкой, а кэшированная история закончилась
- AI (Claude) генерирует дополнительный контент о локации
- Новый контент озвучивается через ElevenLabs и кэшируется для будущих пользователей
- Ограничения: максимум 3 inflation-сегмента на точку (чтобы не генерировать бесконечно)

**Формат истории (Story Template):**

```
1. Anchor (1 предложение): где мы и что это
2. Hook (1 предложение): неожиданный факт или угол
3. Facts (2–4 коротких факта): только подтверждённые источниками
4. Meaning (1 предложение): почему это важно / что это раскрывает
```

**Типы историй (Story Layers):**
- Atmosphere — как это место ощущалось (звуки, запахи, настроение)
- Human stories — кто жил, любил, умирал здесь
- Hidden details — следы пуль, старые вывески, архитектурные детали
- Time shifts — "встань здесь 100 лет назад..."

**Acceptance criteria:**
- Каждая точка интереса имеет минимум 1 кэшированную историю
- Inflation запускается только если пользователь рядом > 60 секунд после окончания базовой истории
- Inflation-контент кэшируется и доступен следующим пользователям
- Максимум 3 inflation-сегмента на точку

---

#### F4: Audio Playback

- Фоновое воспроизведение аудио (экран заблокирован, другие приложения активны)
- Совместимость с Google Maps navigation audio (не перебивать)
- Плавное затухание/нарастание звука (fade in/out) между историями
- Поддержка Bluetooth-наушников и встроенного динамика
- Пауза при снятии наушников (если поддерживается устройством)

**Acceptance criteria:**
- Аудио играет при заблокированном экране
- Приложение не конфликтует с Google Maps voice navigation
- Fade in/out длится 0.5–1 секунду
- Автопауза при отключении наушников

---

#### F5: Offline Mode (Smart Cache + Download)

**Умный кэш:**
- При наличии интернета приложение предзагружает истории для ближайших зон (radius 500м вперёд по направлению движения)
- Уже прослушанные истории остаются в кэше до ручной очистки

**Скачивание города:**
- Кнопка "Download City" в настройках
- Скачиваются все кэшированные истории (текст + аудио) для выбранного города
- Индикатор прогресса скачивания
- Отображение размера пакета перед скачиванием (ожидаемо 100–300 МБ)
- Inflation в офлайне НЕ работает (требуется AI + TTS)

**Acceptance criteria:**
- При потере сети воспроизводятся все предзагруженные истории без задержки
- Download City скачивает 100% доступных кэшированных историй
- Пользователь видит размер пакета до начала скачивания
- Кэш можно очистить вручную из настроек

---

#### F6: User Reporting System

Пользователь может пожаловаться на историю прямо во время прослушивания (минимальное взаимодействие).

**Типы репортов:**
- "I can't see this place" — локация не соответствует реальности
- "Wrong fact" — фактическая ошибка в истории
- "Unpleasant / political / unnecessary" — неуместный контент

**Механика:**
- Кнопка репорта в notification bar (или shake-to-report)
- Выбор типа репорта — 1 тап
- Опциональный текстовый комментарий
- Репорт отправляется на сервер, привязан к story_id + user_location

**Acceptance criteria:**
- Репорт можно отправить за < 3 тапа
- Все репорты попадают в админ-панель в течение 1 минуты
- Пользователь получает подтверждение отправки репорта

---

#### F7: Push Notifications

**Геолокационные пуши:**
- "You're near an interesting area — start listening" (при входе в зону с плотностью историй)
- Не чаще 2 раз в день
- Только если приложение не активно

**Контентные пуши:**
- "New stories added in [district]" — при обновлении контента
- Не чаще 1 раза в неделю

**Acceptance criteria:**
- Геопуши срабатывают корректно по geofencing
- Пользователь может отключить каждый тип пушей отдельно
- Пуши не приходят во время активной сессии прослушивания

---

#### F8: Onboarding & First Walk

Критически важный момент — первая прогулка должна быть магической.

**Flow:**
1. Краткий экран: "Put on headphones. Start walking. The city will talk to you."
2. Выбор языка (EN / RU)
3. Разрешения: location (always), notifications, audio
4. Первая история начинается в течение 30 секунд ходьбы (специально подобранная "wow" история рядом с текущей позицией)

**Acceptance criteria:**
- Onboarding < 60 секунд до первой истории
- Первая история воспроизводится в течение 30 секунд начала ходьбы
- Если рядом нет точек — воспроизвести общую историю о районе/городе

---

#### F9: Authentication (Optional)

- Приложение работает без аккаунта
- Опциональная регистрация: Email + Password или OAuth (Google, Apple)
- Аккаунт даёт: синхронизацию покупок между устройствами, историю прослушиваний
- JWT token-based session management

**Acceptance criteria:**
- Приложение полностью функционально без регистрации
- При регистрации покупки синхронизируются между устройствами
- Sign in with Apple обязателен для iOS (требование App Store)

---

#### F10: Monetization (IAP)

**Первые 3 месяца — Lifetime Deal:**
- Разовый платёж (рекомендуемая цена $19–29) за доступ ко всем городам навсегда
- Продвигается как early adopter offer

**После LTD периода:**
- **City Pack**: $5 — разовая покупка, открывает все истории одного города навсегда
- **Monthly Subscription**: $7/мес — доступ ко всем городам

**Freemium:**
- Бесплатно: N историй в день (рекомендуемо 5–10 историй)
- После лимита: предложение купить город или подписку

**Acceptance criteria:**
- IAP корректно работает на iOS (StoreKit) и Android (Google Play Billing)
- LTD покупка даёт доступ ко всем текущим и будущим городам
- Переход с LTD на платную модель не затрагивает LTD-пользователей
- Freemium счётчик сбрасывается ежедневно

---

### 3.2 Features for v2 (Post-MVP)

- Редактирование историй в админке
- Аналитика (heatmaps, популярные истории, retention)
- Управление пользователями в админке
- Мультиязычность через AI-перевод (DE, FR, ES, ZH, JA...)
- Запись топовых историй профессиональным диктором
- Социальные функции (поделиться историей, "мне понравилось")
- Тематические прогулки (предложенные маршруты по теме: "Советский Тбилиси", "Средневековый город")
- Интеграция с Wear OS / Apple Watch

---

## 4. Tech Stack

### 4.1 Mobile App

| Component | Technology | Rationale |
|---|---|---|
| Framework | React Native (Expo) | Кроссплатформенность, быстрый запуск, JS-экосистема |
| Navigation | React Navigation | Стандарт для RN |
| Location | expo-location + react-native-background-geolocation | Фоновый GPS-трекинг с адаптивным режимом |
| Audio | expo-av или react-native-track-player | Background audio playback, поддержка плейлиста |
| Maps (внутри приложения) | react-native-maps (MapView) | Для отображения точек (минимальное использование) |
| Storage | AsyncStorage + SQLite (expo-sqlite) | Кэш историй и пользовательских данных |
| Push Notifications | expo-notifications + FCM/APNs | Гео- и контентные пуши |
| IAP | react-native-iap | In-App Purchases для iOS и Android |
| State Management | Zustand или Redux Toolkit | Управление состоянием приложения |

### 4.2 Backend

| Component | Technology | Rationale |
|---|---|---|
| Language | Go | Производительность, простота, низкое потребление ресурсов |
| HTTP Framework | Gin или Echo | Быстрый REST API |
| Database | PostgreSQL 16 + PostGIS | Пространственные запросы, зрелая экосистема |
| DB Driver | pgx | Лучший Go-драйвер для PostgreSQL |
| Migrations | golang-migrate | Версионирование схемы БД |
| AI Integration | Claude API (Anthropic) | Генерация историй и inflation |
| TTS | ElevenLabs API | Генерация аудио |
| File Storage | S3-compatible (MinIO на VPS или Backblaze B2) | Хранение аудиофайлов |
| Auth | JWT (golang-jwt) | Токены для опциональной аутентификации |
| Push | Firebase Cloud Messaging (FCM) + APNs | Push-уведомления |

### 4.3 Admin Panel

| Component | Technology | Rationale |
|---|---|---|
| Framework | React + TypeScript | Переиспользование JS-экспертизы |
| UI Library | Ant Design или MUI | Быстрая сборка интерфейса |
| Map Component | Mapbox GL JS или Leaflet | Интерактивная карта с точками |
| State | React Query (TanStack Query) | Кэширование и синхронизация с API |
| Build | Vite | Быстрая сборка |

### 4.4 Infrastructure

| Component | Technology | Rationale |
|---|---|---|
| Hosting | VPS (Hetzner) | $10–20/мес, отличная цена/производительность |
| OS | Ubuntu 24.04 LTS | Стабильность |
| Reverse Proxy | Caddy или Nginx | Автоматический HTTPS |
| Containerization | Docker + Docker Compose | Простой деплой и воспроизводимость |
| CI/CD | GitHub Actions | Автоматические билды и деплой |
| Monitoring | Prometheus + Grafana (или UptimeRobot для старта) | Мониторинг доступности |

### 4.5 Data Sources for Story Generation

| Source | Usage |
|---|---|
| OpenStreetMap (Overpass API) | POI, здания, улицы — структурированные данные |
| Wikipedia / Wikidata API | Исторические факты, описания |
| Local history sources | Блоги, архивы, тревел-гиды — через web search в Claude |
| Google Places API (опционально) | Дополнительные POI-данные |

---

## 5. Conceptual Data Model

### 5.1 Core Entities

```
┌──────────────┐     ┌──────────────────┐     ┌──────────────┐
│    City       │────<│      POI         │────<│    Story     │
│              │     │ (Point of        │     │              │
│ id           │     │  Interest)       │     │ id           │
│ name         │     │                  │     │ poi_id (FK)  │
│ name_ru      │     │ id               │     │ language     │
│ country      │     │ city_id (FK)     │     │ text         │
│ center_lat   │     │ name             │     │ audio_url    │
│ center_lng   │     │ name_ru          │     │ duration_sec │
│ radius_km    │     │ location (POINT) │     │ layer_type   │
│ is_active    │     │ type             │     │ order_index  │
│ download_size│     │ tags (JSONB)     │     │ is_inflation │
│ created_at   │     │ address          │     │ confidence   │
│ updated_at   │     │ interest_score   │     │ sources      │
└──────────────┘     │ status           │     │ status       │
                     │ created_at       │     │ created_at   │
                     │ updated_at       │     │ updated_at   │
                     └──────────────────┘     └──────────────┘

┌──────────────┐     ┌──────────────────┐     ┌──────────────┐
│    User       │     │  UserListening   │     │   Report     │
│              │     │                  │     │              │
│ id (UUID)    │     │ id               │     │ id           │
│ email        │     │ user_id (FK)     │     │ story_id(FK) │
│ name         │     │ story_id (FK)    │     │ user_id (FK) │
│ auth_provider│     │ listened_at      │     │ type (ENUM)  │
│ language_pref│     │ completed        │     │ comment      │
│ is_anonymous │     │ location (POINT) │     │ user_lat     │
│ created_at   │     │                  │     │ user_lng     │
│ updated_at   │     └──────────────────┘     │ status       │
└──────────────┘                               │ resolved_at  │
                                               │ created_at   │
┌──────────────┐     ┌──────────────────┐     └──────────────┘
│  Purchase     │     │  InflationJob    │
│              │     │                  │
│ id           │     │ id               │
│ user_id (FK) │     │ poi_id (FK)      │
│ type (ENUM)  │     │ status (ENUM)    │
│ city_id (FK) │     │ trigger_type     │
│ platform     │     │ segments_count   │
│ transaction  │     │ max_segments     │
│ price        │     │ started_at       │
│ is_ltd       │     │ completed_at     │
│ expires_at   │     │ error_log        │
│ created_at   │     │ created_at       │
└──────────────┘     └──────────────────┘
```

### 5.2 Key Fields Detail

**POI.type** (ENUM):
`building`, `street`, `park`, `monument`, `church`, `bridge`, `square`, `museum`, `district`, `other`

**POI.status** (ENUM):
`active`, `disabled`, `pending_review`

**Story.layer_type** (ENUM):
`atmosphere`, `human_story`, `hidden_detail`, `time_shift`, `general`

**Story.status** (ENUM):
`active`, `disabled`, `reported`, `pending_review`

**Report.type** (ENUM):
`wrong_location`, `wrong_fact`, `inappropriate_content`

**Report.status** (ENUM):
`new`, `reviewed`, `resolved`, `dismissed`

**Purchase.type** (ENUM):
`city_pack`, `subscription`, `lifetime`

**InflationJob.trigger_type** (ENUM):
`user_proximity`, `admin_manual`

### 5.3 Key Database Indexes

```sql
-- Главный пространственный запрос: "что рядом?"
CREATE INDEX idx_poi_location ON poi USING GIST (location);

-- Фильтрация активных точек по городу
CREATE INDEX idx_poi_city_status ON poi (city_id, status);

-- Поиск историй для точки
CREATE INDEX idx_story_poi_lang ON story (poi_id, language, status);

-- История прослушиваний пользователя (для дедупликации)
CREATE INDEX idx_listening_user_story ON user_listening (user_id, story_id);

-- Непросмотренные репорты
CREATE INDEX idx_report_status ON report (status) WHERE status = 'new';
```

### 5.4 Key API Queries

**Главный запрос — "Что рассказать?":**
```sql
SELECT p.id, p.name, p.interest_score,
       ST_Distance(p.location, ST_MakePoint($lng, $lat)::geography) as distance_m
FROM poi p
JOIN story s ON s.poi_id = p.id AND s.language = $lang AND s.status = 'active'
WHERE p.city_id = $city_id
  AND p.status = 'active'
  AND ST_DWithin(p.location, ST_MakePoint($lng, $lat)::geography, $radius_m)
  AND p.id NOT IN (SELECT poi_id FROM user_listening WHERE user_id = $user_id)
ORDER BY p.interest_score DESC, distance_m ASC
LIMIT 5;
```

---

## 6. UI Design Principles

### 6.1 Core Philosophy

Минимальный UI. Приложение должно требовать 0 взаимодействия во время прогулки. Экран — только для настройки до/после ходьбы.

### 6.2 Key Screens

**1. Home Screen**
- Большая кнопка "Start Walking" / "Stop"
- Текущий город и количество доступных историй
- Мини-карта с ближайшими точками (опционально)
- Индикатор: "Listening..." когда GPS активен

**2. Now Playing (Notification Bar)**
- Название текущей истории
- Кнопка Report
- Кнопка Pause/Resume
- Прогресс-бар

**3. City Screen**
- Карта города с точками историй
- Счётчик: "42 stories available / 7 listened"
- Кнопка "Download for Offline"
- Кнопка "Buy City" (если freemium лимит)

**4. Settings**
- Язык (EN / RU)
- Аккаунт (опционально)
- Уведомления (вкл/выкл по типам)
- Кэш (размер, очистка)
- Подписка / покупки

**5. Onboarding (3 экрана)**
- Экран 1: "The city has stories. You just need headphones."
- Экран 2: Выбор языка
- Экран 3: Permissions (location, notifications)

### 6.3 Design Guidelines

- **Dark theme by default** — приложение используется на ходу, часто на солнце
- **Large tap targets** — минимум 48dp, пользователь не смотрит на экран
- **Minimal text** — иконки и визуал важнее слов
- **Map-centric** — когда UI всё же нужен, карта с точками — главный элемент
- **Accessibility** — VoiceOver/TalkBack support (потенциально важная аудитория — visually impaired)

---

## 7. Security Considerations

### 7.1 Authentication & Authorization

- JWT access tokens (15 min TTL) + refresh tokens (30 days)
- Anonymous users получают device-based UUID для трекинга прослушиваний
- OAuth 2.0: Google, Apple Sign-In
- Пароли: bcrypt hashing, минимум 8 символов
- Rate limiting на auth endpoints: 5 попыток / минуту

### 7.2 API Security

- HTTPS only (TLS 1.3)
- API rate limiting: 60 requests/min для обычных endpoints, 10/min для AI-зависимых
- Input validation на все GPS-координаты (диапазоны lat/lng)
- API keys для админ-панели — отдельные от пользовательских токенов
- CORS: ограничение origins для админ-панели

### 7.3 Data Privacy

- GPS-данные не хранятся долгосрочно — только текущая сессия
- UserListening записи не содержат точных координат пользователя (только story_id)
- GDPR/privacy compliance: возможность удаления аккаунта и всех данных
- Анонимные пользователи: данные привязаны к device UUID, не к личности

### 7.4 Content Security

- Репорт-система для user-generated feedback
- AI-сгенерированный контент проходит базовую фильтрацию (Claude content policy)
- Админ может отключить любую историю мгновенно

### 7.5 Infrastructure Security

- SSH key-only access на VPS
- Firewall: открыты только 80, 443, 22
- Database не доступна извне (только через localhost)
- Docker containers с минимальными привилегиями
- Регулярные бэкапы БД (daily)

---

## 8. Development Phases

### Phase 1: Foundation (Weeks 1–3)

**Backend:**
- [ ] Инициализация Go-проекта, структура, Docker-setup
- [ ] PostgreSQL + PostGIS: миграции, базовая схема (City, POI, Story)
- [ ] REST API: CRUD для POI и Stories
- [ ] Пространственный запрос "nearby POIs" с PostGIS
- [ ] Хранилище аудиофайлов (S3-compatible)
- [ ] Health check и базовое логирование

**Content Pipeline:**
- [ ] Скрипт импорта POI из OpenStreetMap (Overpass API) для Тбилиси
- [ ] Интеграция Claude API для генерации историй
- [ ] Интеграция ElevenLabs API для генерации аудио
- [ ] Pipeline: координаты → Claude story → ElevenLabs audio → S3 → DB

**Результат:** Бэкенд с API и база, наполненная POI Тбилиси + первые 50 историй

---

### Phase 2: Mobile App Core (Weeks 4–6)

**Mobile:**
- [ ] React Native проект (Expo)
- [ ] Background location tracking (адаптивный режим)
- [ ] Story Engine: proximity detection → выбор истории → воспроизведение
- [ ] Audio playback (background, lock screen controls)
- [ ] Scoring-алгоритм для выбора лучшей истории
- [ ] Pacing logic (silence management, cooldown между историями)
- [ ] Now Playing notification

**Backend:**
- [ ] API endpoint: GET /nearby-stories (lat, lng, radius, language, user_id)
- [ ] Tracking listened stories per user/device

**Результат:** Работающий прототип — ходишь по Тбилиси, слушаешь истории

---

### Phase 3: Content & Polish (Weeks 7–8)

**Content:**
- [ ] Массовая генерация историй для Тбилиси (target: 200–500 POI)
- [ ] Качество-контроль: проверка первых 50 историй вручную
- [ ] Два языка: EN + RU для всех историй
- [ ] Настройка tone of voice в Claude промпте

**Mobile:**
- [ ] Onboarding flow (3 экрана)
- [ ] Home screen с кнопкой Start/Stop
- [ ] City screen с картой точек
- [ ] Settings screen
- [ ] Smart cache (предзагрузка ближайших историй)
- [ ] Download City функция
- [ ] Report system (UI + API)

**Результат:** Полноценное приложение с контентом для Тбилиси

---

### Phase 4: Monetization & Admin (Weeks 9–10)

**Mobile:**
- [ ] In-App Purchases (react-native-iap)
- [ ] Freemium logic (N бесплатных историй в день)
- [ ] LTD purchase flow
- [ ] Optional authentication (Email, Google, Apple)
- [ ] Push notifications (geo + content)

**Admin Panel:**
- [ ] React-приложение с авторизацией
- [ ] Карта с POI (Mapbox/Leaflet)
- [ ] Детальный вид POI: истории, статус, репорты
- [ ] Управление inflation: запуск/остановка для точки
- [ ] Модерация репортов: список, фильтры, действия (dismiss/disable story)

**Backend:**
- [ ] IAP verification (App Store + Google Play receipts)
- [ ] Purchase management API
- [ ] Admin API (protected endpoints)
- [ ] Inflation job runner (background worker)
- [ ] Push notification service (FCM + APNs)

**Результат:** Полный MVP готов к запуску

---

### Phase 5: Launch & Iterate (Weeks 11–12)

- [ ] Beta-тест с 20–30 пользователями в Тбилиси
- [ ] Исправление багов и UX-проблем
- [ ] Оптимизация батареи на реальных устройствах
- [ ] App Store + Google Play submission
- [ ] Landing page
- [ ] Запуск LTD кампании

**Результат:** Приложение в сторах, первые пользователи

---

## 9. Potential Challenges & Mitigations

| Challenge | Impact | Mitigation |
|---|---|---|
| **GPS-точность в плотной застройке** | Истории привязываются не к тем зданиям | Увеличить radius matching до 50–80м; использовать Wi-Fi positioning; тестировать на реальных улицах Тбилиси |
| **Расход батареи** | Пользователи удаляют приложение | Адаптивный GPS-режим; agressive sleep mode; тестировать и показывать battery usage < 8%/час |
| **Качество AI-историй** | Галлюцинации, скучный контент | Детальный system prompt (уже проработан); confidence scoring; репорт-система; ручная проверка первых 50 историй |
| **ElevenLabs стоимость** | Рост расходов при масштабировании | Кэширование аудио (генерируем один раз); мониторинг расходов; fallback на OpenAI TTS если нужно |
| **Тишина — нет историй рядом** | Пользователь думает приложение сломалось | Fallback: общая история о районе/улице; индикатор "Listening..." на экране; onboarding объясняет паузы |
| **App Store rejection** | Задержка запуска | Background location usage — подготовить детальное описание для review; следовать guidelines |
| **Конфликт аудио с Google Maps** | Перебивают друг друга | Audio ducking: понижать громкость истории при навигационной команде Maps |
| **Офлайн без inflation** | Ограниченный контент в офлайне | Достаточно плотное начальное покрытие (200–500 POI для Тбилиси) |
| **Локализация контента (RU)** | Двойная работа по контенту | Claude генерирует на обоих языках в одном запросе |

---

## 10. Future Expansion

### 10.1 New Cities
- Батуми, Кутаиси (Грузия — естественное расширение)
- Стамбул, Баку, Ереван (региональные хабы)
- Европейские города с богатой историей (Прага, Рим, Лиссабон)

### 10.2 New Features (v2+)
- **Мультиязычность**: AI-перевод на 10+ языков
- **Тематические прогулки**: "Советский Тбилиси", "Винный маршрут", "Архитектура модерна"
- **Social sharing**: поделиться историей в Instagram/Telegram
- **Favorites**: сохранить понравившиеся истории
- **Профессиональная озвучка**: топовые истории записаны диктором
- **AR-элементы**: показать, как здание выглядело раньше (фото-оверлей)
- **Партнёрства**: отели, Airbnb, туристические агентства — предустановка на устройства

### 10.3 Business Model Evolution
- B2B: лицензирование технологии для музеев и городов
- Спонсированные истории: рестораны и бизнесы платят за упоминание (с пометкой "sponsored")
- Premium-голоса: выбор между разными рассказчиками (актёры, историки)

---

## 11. Cost Estimation (Monthly, MVP)

| Item | Cost |
|---|---|
| VPS (Hetzner CX31) | $15 |
| Domain + CDN | $5 |
| Claude API (генерация историй) | $20–50 (разово при наполнении, потом ~$10/мес на inflation) |
| ElevenLabs (TTS) | $22/мес (Creator plan) |
| Backblaze B2 (audio storage) | $1–5 |
| Apple Developer Account | $99/год |
| Google Play Developer | $25 (разово) |
| **Total (monthly run)** | **~$50–70/мес** |
| **Initial setup** | **~$150–200 (генерация контента + accounts)** |

---

## 12. Technical References

- [React Native Background Geolocation](https://github.com/transistorsoft/react-native-background-geolocation)
- [react-native-track-player](https://github.com/doublesymmetry/react-native-track-player)
- [PostGIS Documentation](https://postgis.net/documentation/)
- [Claude API Documentation](https://docs.anthropic.com/en/api)
- [ElevenLabs API Documentation](https://docs.elevenlabs.io/api-reference)
- [Overpass API (OSM)](https://overpass-api.de/)
- [react-native-iap](https://github.com/dooboolab-community/react-native-iap)
- [Expo Location](https://docs.expo.dev/versions/latest/sdk/location/)
