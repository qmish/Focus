# Roadmap v2: Execution Backlog

Версия документа: 2.0  
Дата обновления: 25 марта 2026  
Статус: В работе

Цель: довести `Focus` до production-ready с использованием форка `H:\jitsi-meet-master` и единой авторизации через Keycloak для AD, MS Exchange и корпоративных ресурсов.

## Правила ведения backlog

- [ ] Обновлять статус задач минимум 2 раза в неделю.
- [ ] Для каждой закрытой задачи фиксировать ссылку на PR/commit и краткий итог.
- [ ] Не закрывать этап без прохождения критериев готовности этапа.
- [ ] Любые блокеры фиксировать в конце документа в разделе "Блокеры и риски".

## Этап 0. Подготовка и выравнивание источников правды

### 0.1 Актуализация артефактов проекта
- [x] Провести сверку `README.md`, `ANALYSIS.md`, `docs/Roadmap.md` с фактическим кодом.
- [x] Зафиксировать текущий фактический статус (что действительно работает, что в TODO).
- [x] Удалить/пометить устаревшие утверждения о "100% готовности" там, где есть заглушки.
- [x] Создать changelog изменений для документации.

### 0.2 Нормализация UI-ассетов (`pics`)
- [x] Найти и зафиксировать фактический путь/содержимое `pics` (лого, иконки, фоны, бренд-ресурсы).
- [x] Согласовать структуру ассетов: `logo`, `favicon`, `backgrounds`, `icons`, `avatars`.
- [x] Подготовить единый `branding-manifest` (имена файлов, назначение, формат, размеры).
- [x] Определить, какие ассеты применяются в `Focus frontend/admin`, а какие в `jitsi-meet` форке.
- [x] Конвертировать `pics/hdphoto1.wdp` в web-совместимый формат (`png`/`webp`) и обновить manifest.

### Критерии готовности этапа 0
- [x] Документация синхронизирована с кодом.
- [x] `pics` и структура бренд-ассетов формально зафиксированы.
- [x] Есть утвержденный `branding-manifest`.

---

## Этап 1. Identity и доступы: Keycloak как единая авторизация (AD + Exchange + Corp Resources)

### 1.1 Интеграция Keycloak с AD (Identity Brokering)
- [ ] Настроить Keycloak как брокер идентичности к AD (Azure AD или on-prem AD FS, по целевой инфраструктуре).
- [ ] Подключить OIDC/SAML provider в Keycloak для AD.
- [x] Настроить маппинг групп и ролей AD в claims Keycloak.
- [x] Определить и внедрить role model: `user`, `moderator`, `admin`, `service`.
- [ ] Настроить SCIM/LDAP sync (если требуется по инфраструктуре компании).

### 1.2 Авторизация для корпоративных ресурсов
- [x] Определить перечень корп. ресурсов, требующих SSO (внутренние API, порталы, файловые ресурсы, сервисы).
- [x] Реализовать policy mapping в Keycloak: группы AD -> роли/скоупы в Focus API.
- [x] Внедрить audience validation для session JWT в backend middleware (конфигурируемый `AUTH_REQUIRED_AUDIENCE`).
- [x] Настроить audience/scope для сервисных клиентов.
- [x] Внедрить централизованную проверку ролей и скопов в backend middleware.
- [x] Внедрить ABAC policy engine (resource/action/context) поверх RBAC для критичных операций.

### 1.3 Интеграция с MS Exchange (через Microsoft Graph)
- [ ] Зарегистрировать/проверить приложение в Azure AD с нужными permissions (`Calendars.ReadWrite` и смежные).
- [ ] Настроить корректный flow доступа (delegated/OBO по согласованной security-модели).
- [ ] Привязать пользователя из Keycloak к Exchange identity без ручных костылей.
- [x] Включить аудит календарных операций (создание/обновление/удаление).

### 1.4 Безопасность токенов и сессий
- [x] Разделить секреты: session JWT и Jitsi JWT.
- [x] Настроить ротацию секретов и сроков жизни токенов.
- [x] Реализовать logout/invalidate flow (включая blacklist/revocation для API сессий).
- [x] Добавить аудит авторизаций/ошибок авторизации.

### Критерии готовности этапа 1
- [ ] Пользователь логинится через AD -> Keycloak -> Focus без ручной регистрации.
- [ ] Роли и доступы приходят из AD групп и корректно применяются в API.
- [ ] Доступ к Exchange работает в выбранной модели доступа.
- [ ] Токены и сессии соответствуют security-требованиям.

---

## Этап 2. Выравнивание backend auth-потоков и API

### 2.1 Согласование модели токенов frontend/admin/backend
- [x] Утвердить единую модель: `session_token` передается в API (`Authorization: Bearer`) и в WS (`Authorization` или `token/access_token` query для reconnect).
- [x] Привести `frontend` и `frontend-admin` к единому контракту авторизации (единый источник access token из auth-store c fallback на `localStorage`).
- [x] Обновить `/auth/me`, refresh/logout semantics при необходимости.

### 2.2 Защита WebSocket
- [x] Добавить обязательную аутентификацию на `/api/v1/ws`.
- [x] Внедрить room-level authorization на websocket события.
- [x] Добавить обработку reconnect с проверкой истекшего токена.

### 2.3 Завершение TODO-критики в backend
- [x] Реализовать TODO по `admin conferences` (получение активных конференций и завершение через backend room lifecycle).
- [x] Реализовать TODO по `calendar cancellation notification`.
- [x] Закрыть технические TODO по auth/logout flow.

### Критерии готовности этапа 2
- [x] API и WS используют согласованную auth-модель.
- [x] Нет открытых критических TODO в auth/admin/calendar.
- [x] Пройдены smoke-тесты защищенных API и WS.

---

## Этап 3. Webhooks и Bots (production-реализация)

### 3.1 Webhooks inbound/outbound
- [x] Реализовать проверку подписи входящих webhook.
- [x] Сохранять входящие события в БД с трассировкой и idempotency key.
- [x] Реализовать outbound dispatcher с retry/backoff и dead-letter логикой.
- [x] Добавить админ-видимость доставок webhook и ошибок.

### 3.2 Bots engine
- [x] Реализовать реальную отправку сообщений ботом в комнаты.
- [x] Реализовать команды `/create`, `/schedule`, `/status` c backend-интеграцией.
- [x] Связать ботов с календарем Exchange и комнатами Jitsi.
- [x] Добавить rate-limit и permission-check на bot-команды.

### Критерии готовности этапа 3
- [x] Webhooks проходят полный цикл в обе стороны.
- [x] Bot-команды выполняют реальные действия, а не заглушки.
- [x] Ошибки webhook/bot наблюдаемы в админке и логах.

---

## Этап 4. Форк `jitsi-meet-master` и корпоративная кастомизация UI

### 4.1 Форк и структура сопровождения
- [x] Создать/подключить форк `jitsi-meet-master` как отдельный управляемый компонент проекта.
- [x] Зафиксировать стратегию обновления upstream (частота, merge-policy, owner).
- [x] Описать "границу кастомизаций": config-only vs code-level изменения.

### 4.2 Брендинг через `pics`
- [x] Подключить logo/favicon/backgrounds/icons из `pics` в Jitsi UI.
- [x] Реализовать/подключить `dynamicBrandingUrl` endpoint.
- [x] Настроить `customTheme` и `customIcons` на базе корпоративного брендбука.
- [x] Отключить нежелательные публичные функции (по security/policy требованиям).

### 4.3 Интеграция с Focus frontend
- [x] Обновить точку встраивания Jitsi в `Focus` с учетом форка.
- [x] Согласовать события iframe API и действия в приложении (join/leave/moderation).
- [x] Проверить совместимость интерфейсных настроек и локализации.

### Критерии готовности этапа 4
- [ ] Jitsi UI визуально соответствует бренду компании.
- [x] Кастомизация воспроизводима из `pics` + manifest без ручных правок.
- [ ] Focus стабильно работает с форкнутым Jitsi.

---

## Этап 5. Frontend и Admin интеграция с backend

### 5.1 Frontend
- [x] Завершить интеграцию страниц комнат/сообщений с реальными API-ответами.
- [x] Добавить единый API client, обработку ошибок, loading/retry states.
- [x] Реализовать realtime UX на websocket (новые сообщения, системные события, reconnect).

### 5.2 Admin frontend
- [x] Довести users/stats/ban-unban до реального backend состояния.
- [x] Добавить управление конференциями (просмотр/завершение) через backend интеграцию.
- [x] Добавить раздел наблюдаемости webhook/bot ошибок.
- [x] Расширить раздел наблюдаемости audit-ошибками auth/calendar операций.

### Критерии готовности этапа 5
- [ ] Все ключевые пользовательские сценарии работают без моков.
- [x] Админка отображает реальное состояние системы.
- [x] Ошибки корректно обрабатываются и визуализируются.

---

## Этап 6. DevOps и среды (dev/stage/prod)

### 6.1 Docker Compose и локальный контур
- [x] Расширить `docker-compose` сервисами `frontend`, `frontend-admin`, (опционально) локальный jitsi stack.
- [x] Настроить единые env-конвенции и секреты для локальной разработки.
- [x] Обеспечить команду "one command up" для dev-окружения.

### 6.2 Kubernetes/stage/prod
- [x] Актуализировать манифесты для Focus API/frontends/Jitsi fork.
- [x] Проверить autoscaling-политики для API и JVB.
- [x] Настроить ingress/TLS, сетевые политики, секреты и ротацию.

### 6.3 CI/CD
- [x] Включить e2e/load/security проверки в pipeline gates.
- [x] Разделить pipelines для Focus и Jitsi fork (с согласованными релизными окнами).
- [x] Реализовать стратегию rollout/rollback (stage -> prod).

### Критерии готовности этапа 6
- [x] Локальная и stage среды поднимаются и воспроизводимы.
- [x] Pipeline блокирует релизы при провале quality/security gates.
- [x] Есть отработанный rollback на production.

---

## Этап 7. Тестирование, безопасность, приемка

### 7.1 Тестирование
- [x] Добавить API-level e2e smoke для auth/room/chat/webhook/bot/admin базовых негативных сценариев.
- [x] Добавить API-level happy-path e2e для авторизованных `auth/me`, `rooms`, `admin/stats` через валидный session JWT.
- [x] Добавить API-level user journey e2e: `create room -> join call -> send/list message -> admin conferences`.
- [x] Подготовить отдельный stage-harness для e2e с реальной OIDC аутентификацией (manual workflow + browser flow).
- [x] Добавить e2e сценарии с реальной аутентификацией и пользовательскими happy-path потоками (auth -> room -> chat -> call -> admin).
- [x] Покрыть e2e: auth, room, chat, call, webhook, bot, admin flows.
- [x] Подготовить целевые load-профили и manual pipeline для stage (`API + Jitsi/JVB`) с фиксированными thresholds.
- [x] Подготовить stage regression harness для бренд-кастомизаций Jitsi (`dynamicBrandingUrl`/assets/config).
- [ ] Прогнать load-тесты API + Jitsi/JVB в целевых профилях нагрузки.
- [ ] Прогнать регрессию после бренд-кастомизаций Jitsi.

### 7.2 Security hardening
- [x] Провести security-review auth/WS/webhook/token flows.
- [x] Проверить RBAC/ABAC политики по ролям AD.
- [x] Провести сканы зависимостей и контейнеров.
- [x] Настроить SARIF-репортинг и triage-процесс для high/critical findings в CI.
- [ ] Закрыть критические и высокие findings до go-live.

### 7.3 Приемка и go-live
- [ ] Провести UAT с пилотной группой.
- [x] Подготовить UAT protocol/checklist и шаблон фиксации результатов для пилотной группы.
- [x] Подготовить runbook on-call и инцидентные инструкции.
- [x] Зафиксировать release checklist и критерии go-live.

### Критерии готовности этапа 7
- [ ] Все тестовые контуры пройдены.
- [ ] Security findings в допустимом уровне риска.
- [ ] Получено формальное согласование на запуск.

---

## Блокеры и риски

- [ ] Риск: рассинхрон токен-моделей между frontend и backend.
- [ ] Риск: сложность сопровождения форка Jitsi при частых upstream-обновлениях.
- [ ] Риск: неполный или несогласованный набор ассетов `pics`.
- [ ] Риск: зависимость от корпоративных политик AD/Exchange и сроков доступа.

## Быстрый трек (критический минимум на ближайшие 2 спринта)

- [x] Закрыть auth-модель и WS auth.
- [x] Закрыть TODO в webhooks/bots/admin/calendar.
- [x] Внедрить базовый брендинг Jitsi из `pics`.
- [x] Стабилизировать stage + e2e smoke.

