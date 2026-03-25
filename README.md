# Focus — Корпоративный мессенджер

Корпоративный мессенджер с видеоконференциями на базе Jitsi Meet, интегрированный с Keycloak (SSO) и MS Exchange.

## 📋 Возможности

- 💬 Текстовый чат (one-to-one и групповой)
- 🎥 Видеоконференции через Jitsi Meet
- 🔐 Единый вход через Keycloak (OIDC)
- 📅 Интеграция с календарями MS Exchange
- 🤖 Чат-боты для автоматизации
- 🔗 Вебхуки для интеграции с внешними системами

## 🏗️ Архитектура

```
┌─────────────────┐      ┌─────────────────┐      ┌─────────────────┐
│   Frontend      │      │   Go Backend    │      │  Jitsi Stack    │
│  (React/Vue)    │◄────►│  (REST/WS)      │      │  (Prosody,      │
│  - Мессенджер   │      │   API_GO        │      │   JVB, Jicofo)  │
│  - Админка      │      │  - Keycloak     │      │                 │
└─────────────────┘      │  - Exchange     │      └────────┬────────┘
                         │  - Webhooks     │               │
                         │  - Bots         │               │
                         └─────────────────┘               │
                                │                          │
                                ▼                          ▼
                         ┌─────────────────┐      ┌─────────────────┐
                         │   Keycloak      │      │   MS Exchange   │
                         │   (SSO)         │      │   Calendar      │
                         └─────────────────┘      └─────────────────┘
```

## 📚 Документация

Полная документация доступна в папке [`docs/`](./docs/):

| Документ | Описание |
|----------|----------|
| [Roadmap](./docs/Roadmap.md) | План реализации по этапам |
| [Architecture](./docs/Architecture.md) | Общая архитектура системы |
| [HLD](./docs/HLD.md) | High-Level Design |
| [Infrastructure](./docs/Infrastructure.md) | Развёртывание в Kubernetes |
| [API](./docs/API.md) | Спецификация REST API |
| [Database](./docs/Database.md) | Проектирование БД |
| [Integration](./docs/Integration.md) | Интеграции с внешними системами |
| [Rollout/Rollback Runbook](./docs/Runbook_RolloutRollback.md) | Порядок релиза и отката stage -> prod |

## 🚀 Быстрый старт

### Требования

- Go 1.21+
- Docker & Docker Compose
- Kubernetes (для production)
- Keycloak (для SSO)
- PostgreSQL 15+
- Redis 7+

### Локальная разработка

```bash
# Клонировать репозиторий
git clone https://github.com/qmish/Focus.git
cd Focus

# Подготовить env (один раз)
cp .env.example .env

# One command up (весь dev-контур)
docker compose --env-file .env up -d --build

# Локальные URL:
# API            http://localhost:8080
# Keycloak       http://localhost:8180
# Frontend       http://localhost:5173
# Frontend Admin http://localhost:5174
```

Для PowerShell можно использовать helper-скрипт:
```powershell
.\scripts\dev-up.ps1
```

## 🛠️ Технологический стек

### Бэкенд
- **Go 1.21+** — основной язык
- **Chi/Gin** — HTTP роутинг
- **GORM** — ORM для PostgreSQL
- **PostgreSQL** — основная БД
- **Redis** — кэш и сессии

### Фронтенд
- **React 18** — UI фреймворк
- **TypeScript 5** — типобезопасность
- **Tailwind CSS** — стилизация

### Инфраструктура
- **Kubernetes** — оркестрация
- **Keycloak** — SSO/OIDC
- **Jitsi Meet** — видеоконференции
- **Prometheus + Grafana** — мониторинг

## 📦 Структура проекта

```
Focus/
├── docs/                    # Документация
│   ├── Roadmap.md
│   ├── Architecture.md
│   ├── HLD.md
│   ├── Infrastructure.md
│   ├── API.md
│   ├── Database.md
│   ├── Integration.md
│   └── ...
├── API_Go/                  # Go бэкенд (в разработке)
│   ├── cmd/
│   ├── internal/
│   ├── pkg/
│   └── ...
├── frontend/                # React фронтенд (в разработке)
└── docker-compose.yml       # Локальное окружение
```

## 📈 Актуальный статус

**Последний релиз:** v0.5.19 (25 марта 2026 г.)  
**Источник правды по плану:** `docs/Roadmap_v2.md`

### Что уже реализовано

- Backend: auth/API/webhooks/bots/admin + websocket auth/room-level access.
- Frontend: реальные room/message API-интеграции, единый API client, realtime websocket UX в `RoomPage`.
- Frontend-admin: реальные users/stats/ban-unban, конференции, раздел наблюдаемости webhook/bot ошибок.
- Dev/local: расширенный `docker-compose` (`api`, `frontend`, `frontend-admin`, `postgres`, `redis`, `keycloak`) + one-command startup.
- CI/CD: quality/security gates, API e2e smoke, k6 load smoke, разделение pipelines для Focus и jitsi-fork.

### Что остается до go-live

- Этап 1: корпоративные интеграции AD/Exchange/role mapping (частично подготовлено, не завершено).
- Этап 4: форк `jitsi-meet-master` и корпоративная кастомизация UI.
- Этап 6.2: актуализация k8s stage/prod манифестов (ingress/TLS/policies/secrets/rotation).
- Этап 7: полный e2e/load/security hardening и UAT/go-live.

### Примечание

Старый `docs/Roadmap.md` сохранен как исторический документ. Для текущего execution-backlog и фактического прогресса используется `docs/Roadmap_v2.md`.

## 📄 Лицензия

Корпоративная лицензия. Все права защищены.

## 📞 Контакты

- **Repository:** https://github.com/qmish/Focus
- **Issues:** https://github.com/qmish/Focus/issues
