# Release Checklist и критерии go-live

Дата: 25 марта 2026 г.

## Pre-release checklist

- [ ] Все CI quality/security gates зеленые.
- [ ] Пройден smoke (API/WS/admin/webhook/bot).
- [ ] Прогнан актуальный набор e2e для релизного кандидата.
- [ ] Отсутствуют открытые `critical/high` security findings.
- [ ] Проверена совместимость миграций БД и rollback план.
- [ ] Секреты и сертификаты валидны (не истекают в ближайшее окно релиза).
- [ ] Подготовлены release notes на русском языке.

## Stage -> Prod

- [ ] Stage развернут тем же image tag, что планируется в prod.
- [ ] Stage sign-off от backend + frontend + devops.
- [ ] Выполнен prod rollout в согласованное окно.
- [ ] Подтвержден health/readiness после rollout.

## Go-live criteria

- [ ] UAT от пилотной группы завершен успешно.
- [ ] Метрики SLO в норме в течение 24 часов после релиза.
- [ ] On-call команда подтверждает операционную готовность.
- [ ] Формальное согласование владельца продукта получено.

## Post-release

- [ ] Зафиксированы фактические версии и commit/tag.
- [ ] Обновлен changelog/roadmap.
- [ ] Проведен post-release review (при необходимости).
