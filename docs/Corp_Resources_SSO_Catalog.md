# Каталог корпоративных ресурсов под SSO

Дата: 25 марта 2026 г.

## Цель

Зафиксировать перечень корпоративных ресурсов, которые должны быть доступны через единую модель авторизации `AD -> Keycloak -> Focus`.

## Источник идентичности

- Корневой IdP: корпоративный AD (Azure AD / AD FS по целевой инфраструктуре).
- IAM-слой и brokering: Keycloak.
- Приложения-потребители: Focus API, Focus frontend, Focus admin, Jitsi Meet, Exchange/Graph интеграции.

## Перечень ресурсов

| Ресурс | Тип | Модель доступа | Роли | Скоупы | Примечание |
|---|---|---|---|---|---|
| Focus Web App (`frontend`) | Пользовательский портал | OIDC (browser) | `user`, `moderator`, `admin` | `focus.read` | Session JWT audience `focus-frontend` |
| Focus Admin App (`frontend-admin`) | Админ-панель | OIDC (browser) | `admin` | `focus.admin` | Доступ только через admin policies |
| Focus API | Backend API | Bearer JWT | `user`, `moderator`, `admin`, `service` | `focus.read`, `focus.write`, `focus.admin` | Централизованный `RequireAccess` + ABAC |
| Focus WebSocket (`/api/v1/ws`) | Realtime канал | Bearer JWT | `user`, `moderator`, `admin`, `service` | `focus.read`, `focus.write` | Room-level authorization |
| Exchange Calendar (Microsoft Graph) | Корпоративный календарь | Delegated/OBO (по security-модели) | `user`, `moderator`, `admin` | `focus.calendar`, `exchange.calendar`, `Calendars.ReadWrite` | Аудит операций create/update/delete включен |
| Jitsi Meet (fork) | Видеоконференции | JWT + embed | `user`, `moderator`, `admin` | `focus.read` | Брендинг/локализация управляются через `dynamicBrandingUrl` |
| Сервисные интеграции (bots/webhooks/service clients) | Machine-to-machine | Service JWT | `service` | `focus.service` | Ограничения по `AUTH_SERVICE_AUDIENCES/SCOPES` |

## Базовые требования доступа

- Все пользовательские потоки проходят через Keycloak без локальной регистрации.
- Роли и скоупы приходят из claims (включая group mapping) и применяются на backend middleware/ABAC.
- Для сервисных токенов обязательны проверки audience и required scopes.
- Сессионные JWT валидируются по целевому audience и поддерживают ротацию secrets.

## Границы ответственности

- AD/IT Security: жизненный цикл учетных записей и групп.
- Keycloak/IAM owner: federation, group mapping, client policies.
- Focus backend owner: enforcement RBAC/ABAC, аудит security-критичных операций.
- Platform/SRE: секреты, ротация, stage/prod rollout, observability.
