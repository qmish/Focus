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

# Запустить зависимости через Docker Compose
docker-compose up -d postgres redis keycloak

# Запустить Go сервер
cd API_Go
go run cmd/server/main.go

# Запустить фронтенд (в разработке)
cd frontend
npm install
npm run dev
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

- [x] Этап 0: Инициализация проекта и документация ✅
- [x] Этап 1: Базовая инфраструктура ✅
- [x] Этап 2: Аутентификация и API ✅
- [x] Этап 3: Фронтенд мессенджера (MVP) ✅
- [x] Этап 4: Интеграция с MS Exchange ✅
- [ ] Этап 5: Вебхуки и чат-боты ⏳ В плане
- [ ] Этап 6: Админка и масштабирование ⏳ В плане
- [ ] Этап 7: Тестирование и внедрение ⏳ В плане

**Прогресс:** 5/8 этапов (62.5%)

## 📄 Лицензия

Корпоративная лицензия. Все права защищены.

## 📞 Контакты

- **Repository:** https://github.com/qmish/Focus
- **Issues:** https://github.com/qmish/Focus/issues
