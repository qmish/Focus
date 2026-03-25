# On-call Runbook и инцидентные инструкции

Дата: 25 марта 2026 г.

## Каналы эскалации

- L1 (дежурный инженер): реакция до 15 минут.
- L2 (backend/devops): подключение до 30 минут.
- L3 (security/архитектор): подключение до 60 минут.

## Severity

- `SEV-1`: полный простой auth/api/chat/call.
- `SEV-2`: деградация ключевого функционала или высокая доля ошибок.
- `SEV-3`: частичная деградация без критического влияния.

## Первичная диагностика

1. Проверить доступность:
   - `/health`, `/ready`, ingress endpoints.
2. Проверить rollout:
   - `kubectl rollout status deployment/api-go -n messenger-prod`
   - `kubectl get pods -n messenger-prod`
3. Проверить метрики/алерты:
   - error rate, p95 latency, pod restarts, DB/Redis availability.
4. Проверить последние релизные изменения:
   - текущий tag, changelog, связанные PR/commit.

## Частые сценарии и действия

### Рост 5xx/timeout в API

- Проверить DB/Redis connectivity.
- Проверить saturation по CPU/memory и HPA.
- При регрессии после релиза — выполнить rollback.

### Проблемы с авторизацией/SSO

- Проверить доступность Keycloak.
- Проверить `SESSION_SECRET` и ротации секретов.
- Проверить clock skew и JWT expiration.

### Проблемы webhook/bot

- Проверить webhook signature validation.
- Проверить admin observability endpoints:
  - `/api/v1/admin/webhooks/errors`
  - `/api/v1/admin/bots/errors`

## Команды rollback (prod)

```bash
kubectl rollout undo deployment/api-go -n messenger-prod
kubectl rollout undo deployment/frontend -n messenger-prod
kubectl rollout undo deployment/frontend-admin -n messenger-prod
```

## Коммуникации

- Обновление статуса инцидента каждые 30 минут (для `SEV-1/2`).
- Постинцидентный разбор обязателен для `SEV-1/2`.
