# Exchange On-Prem EWS Integration

**Версия:** 1.0  
**Дата:** 27 марта 2026 г.

---

## 1. Область

Интеграция реализована **только для on-prem Exchange/OWA через EWS (SOAP)**.  
Microsoft Graph/Exchange Online для текущего контура не используется.

---

## 2. Backend компоненты

- `internal/exchange/ews.go`
  - CRUD событий (`GetEvents`, `CreateEvent`, `GetEvent`, `UpdateEvent`, `DeleteEvent`)
  - auth modes: `basic`, `ntlm`, `kerberos`
- `internal/exchange/sync_worker.go`
  - периодический polling Exchange и синхронизация в Focus
- `internal/models/meeting_link.go`
  - связь Focus room <-> Exchange event
- `internal/models/calendar_idempotency_key.go`
  - идемпотентность `POST /calendar/events`

---

## 3. Data flow

### 3.1 Focus -> Exchange

1. Frontend отправляет `POST /api/v1/calendar/events`.
2. Backend при необходимости создаёт Jitsi room.
3. Backend создаёт событие в EWS.
4. Backend сохраняет `meeting_link`.
5. Возвращается event payload + room link.

### 3.2 Exchange -> Focus

1. `SyncWorker` получает окно событий по активным пользователям.
2. Новые события -> создаётся room + `meeting_link`.
3. Изменения -> обновление `meeting_link`.
4. Отсутствующие события в окне -> статус `cancelled`.

---

## 4. Конфигурация

### 4.1 Базовая EWS

- `EXCHANGE_PROVIDER=ews`
- `EXCHANGE_EWS_URL`
- `EXCHANGE_USERNAME`
- `EXCHANGE_PASSWORD`
- `EXCHANGE_DOMAIN`
- `EXCHANGE_AUTH_MODE` (`basic|ntlm|kerberos`)
- `EXCHANGE_CA_CERT_PATH`
- `EXCHANGE_INSECURE_TLS`
- `EXCHANGE_IMPERSONATION`
- `EXCHANGE_TIMEOUT`

### 4.2 Sync worker

- `EXCHANGE_SYNC_ENABLED`
- `EXCHANGE_SYNC_INTERVAL`
- `EXCHANGE_SYNC_LOOKBACK`
- `EXCHANGE_SYNC_LOOKAHEAD`

### 4.3 Kerberos mode

- `EXCHANGE_AUTH_MODE=kerberos`
- `EXCHANGE_KRB5_CONFIG_PATH`
- `EXCHANGE_KRB5_KEYTAB_PATH` (или парольный fallback)
- `EXCHANGE_KRB5_REALM`
- `EXCHANGE_KRB5_SERVICE_PRINCIPAL` (optional)

Готовый пример patch-манифеста: `k8s/exchange-kerberos-example.yaml`.

---

## 5. API endpoints

- `GET /api/v1/calendar/events`
- `POST /api/v1/calendar/events` (`Idempotency-Key` optional)
- `PUT /api/v1/calendar/events/{id}`
- `DELETE /api/v1/calendar/events/{id}`
- `GET /api/v1/admin/exchange/settings`
- `PUT /api/v1/admin/exchange/settings`
- `POST /api/v1/admin/exchange/test-connection`

Persisted настройки Exchange теперь хранятся в БД (`exchange_settings`) и управляются из `frontend-admin` страницы Integrations.

---

## 6. Ограничения текущей реализации

- Polling sync (не push subscription из Exchange).
- Конфликт-резолюция упрощённая (last write/last seen).
- Kerberos реализован через SPNEGO клиент без отдельного Kerberos proxy слоя.
