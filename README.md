# City Stories Guide

Ambient audio guide that tells stories about nearby places while you walk. A "parallel audio layer" over Google Maps — navigation stays in Maps, and City Stories Guide turns any walk into a living documentary.

## Monorepo Structure

```
cityStoriesGuide/
├── backend/       # Go API server (Gin, PostgreSQL + PostGIS, pgx)
│   ├── cmd/       # Entry points (api, worker)
│   ├── internal/  # Business logic (handler, service, repository, domain)
│   ├── migrations/# SQL migrations (golang-migrate)
│   └── scripts/   # Helper scripts
├── mobile/        # React Native (Expo) mobile app
│   ├── app/       # Expo Router file-based routes
│   └── src/       # App source (api, services, store, components, hooks)
├── admin/         # React admin panel (Vite + Ant Design)
│   └── src/       # Admin source (pages, components, api, hooks)
├── infra/         # Infrastructure configs
│   ├── docker-compose.yml
│   ├── Caddyfile
│   └── backup/
├── .github/       # GitHub Actions CI/CD workflows
├── tasks.json     # Structured task list
├── progress.md    # Agent progress log
└── PRD.md         # Product Requirements Document
```

## Tech Stack

| Layer          | Technology                                      |
|----------------|------------------------------------------------|
| Backend        | Go, Gin, PostgreSQL 16 + PostGIS 3.4, pgx     |
| Mobile         | React Native (Expo), Expo Router, Zustand      |
| Admin Panel    | React, TypeScript, Vite, Ant Design, Leaflet   |
| Infrastructure | Docker Compose, Caddy, GitHub Actions          |
| AI / TTS       | Claude API (story generation), ElevenLabs (audio) |
| Storage        | S3-compatible (MinIO dev, Backblaze B2 prod)   |

## Getting Started

See individual README files in each subdirectory for setup instructions.

## License

Private — All rights reserved.
