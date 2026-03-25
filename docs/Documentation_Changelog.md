# Documentation Changelog

## 2026-03-25

- Добавлены operational документы:
  - `docs/OnCall_Runbook.md`,
  - `docs/GoLive_Checklist.md`.
- В `docs/Roadmap_v2.md` закрыты 2 подпункта `7.3` (on-call runbook, release checklist/go-live criteria).
- Расширено API e2e покрытие в `tests/e2e/ci-api.spec.ts`:
  - auth redirect flow,
  - unauth room/chat/bot/admin checks,
  - webhook invalid signature check.
- Обновлен CI шаг запуска e2e smoke (`API Smoke|API Flows`).
- Детализирован backlog `7.1` в `docs/Roadmap_v2.md` с отдельным закрытым API-level e2e подпунктом.
- Конвертирован `pics/hdphoto1.wdp` -> `pics/hdphoto1.png` (web-совместимый формат).
- Обновлены `docs/branding-manifest.json` и `docs/Branding.md` под новый background asset.
- Обновлен статус задачи `0.2` в `docs/Roadmap_v2.md`.
- Актуализирован `README.md`:
  - обновлен раздел статуса реализации;
  - удалены устаревшие формулировки о готовности и старые проценты прогресса;
  - добавлена ссылка на `docs/Roadmap_v2.md` как источник правды.
- Актуализирован `ANALYSIS.md`:
  - заменен устаревший snapshot `v0.5.0` на текущий фактический статус;
  - пересобран раздел «что не завершено» по этапам roadmap v2.
- Обновлен `docs/Roadmap.md`:
  - добавлен явный дисклеймер «исторический документ v1»;
  - указана ссылка на `docs/Roadmap_v2.md` как текущий execution-backlog.
- Ранее в этот же день добавлены:
  - `docs/Branding.md`,
  - `docs/branding-manifest.json`,
  - `docs/Runbook_RolloutRollback.md`.
