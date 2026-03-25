# Security Findings Triage (critical/high)

Дата: 25 марта 2026 г.

## Источники findings

- `govulncheck` (Go dependencies/code paths)
- `npm audit --omit=dev --audit-level=high` (frontend/frontend-admin prod deps)
- Trivy fs/image scan (HIGH/CRITICAL)
- SARIF отчеты Trivy, загружаемые в GitHub Security tab

## Политика блокировки релиза

- Любой `HIGH`/`CRITICAL` finding в CI security gates блокирует pipeline.
- Исключения допускаются только через явное risk-acceptance решение и отдельную задачу с дедлайном.

## Процесс triage

1. Собрать findings из CI (logs + SARIF).
2. Для каждого finding зафиксировать:
   - компонент,
   - severity,
   - exploitability в нашем контуре,
   - remediation plan и owner.
3. Создать issue/задачу на исправление.
4. После исправления повторно прогнать security gates.
5. Обновить статус в release notes и roadmap.

## SLA по исправлениям

- `CRITICAL`: до 24 часов.
- `HIGH`: до 72 часов.

## Артефакты

- Security Tab (SARIF)
- CI logs
- `docs/Security_Review_Auth_WS_Webhooks.md`
- `docs/RBAC_ABAC_Review.md`
