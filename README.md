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

## 📈 Статус реализации

**Последний релиз:** v0.5.0 (24 марта 2026 г.)  
**Общий прогресс:** ~70%

### Реализовано (✅)

- [x] **Бэкенд API** — 100% готово
  - REST API (rooms, messages, calendar, admin)
  - Keycloak OIDC аутентификация
  - Jitsi JWT генерация
  - MS Exchange интеграция
  - WebSocket Hub
  - Webhooks и чат-боты

- [x] **Фронтенд (React)** — компоненты готовы
  - Страницы: Login, Rooms, Room, Profile
  - Jitsi Meeting компонент
  - Zustand store

- [x] **Admin Frontend (React)** — компоненты готовы
  - Dashboard, Users, Conferences, Settings
  - Admin store

- [x] **Kubernetes** — 100% готов
  - HPA конфигурации
  - Prometheus мониторинг
  - Production манифесты

- [x] **Тесты** — 86 тестов проходят
  - Unit тесты (models, auth, jitsi, exchange)
  - Integration тесты (handlers, webhooks, bots)

- [x] **Документация** — 11 документов
  - Architecture, HLD, Infrastructure
  - API, Database, Integration
  - Roadmap, Security

### Требуется доработка (⚠️)

- [ ] **Frontend интеграция** — подключить к API
  - Добавить axios client
  - Настроить interceptor для токенов
  - Интегрировать REST API вызовы
  - Обработать ошибки и loading states

- [ ] **Admin интеграция** — подключить к API
  - Admin API calls
  - Users CRUD интеграция
  - Stats и ban/unban

- [ ] **WebSocket** — интегрировать во frontend
  - Подключение к WebSocket
  - Real-time сообщения
  - Typing indicators

- [ ] **Docker Compose** — настроить окружение
  - Сервисы: postgres, redis, keycloak
  - Сервисы: api, frontend, admin
  - Network configuration

- [ ] **CI/CD** — запустить pipeline
  - GitHub Actions тестирование
  - Автоматический деплой

- [ ] **E2E и Load тесты** — запустить
  - Playwright сценарии
  - k6 нагрузочные тесты
  - OWASP ZAP security scan

### Прогресс по этапам

| Этап | Статус | Готовность |
|------|--------|------------|
| Этап 0: Инициализация | ✅ Завершён | 100% |
| Этап 1: Инфраструктура | ✅ Завершён | 100% |
| Этап 2: Аутентификация и API | ✅ Завершён | 100% |
| Этап 3: Фронтенд | ⚠️ Интеграция | 60% |
| Этап 4: MS Exchange | ✅ Завершён | 100% |
| Этап 5: Вебхуки и боты | ⏳ В плане | 0% |
| Этап 6: Админка | ⚠️ Интеграция | 60% |
| Этап 7: Тестирование | ⏳ В плане | 0% |

**5 из 8 этапов завершены (62.5%)**  
**До production-ready:** ~4 недели

## 📄 Лицензия

Корпоративная лицензия. Все права защищены.

## 📞 Контакты

- **Repository:** https://github.com/qmish/Focus
- **Issues:** https://github.com/qmish/Focus/issues
