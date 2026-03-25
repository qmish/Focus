# Load Testing Target Profiles (API + Jitsi/JVB)

Дата: 25 марта 2026 г.

## Цель

Стандартизовать целевые нагрузочные профили для stage перед обязательным прогонами в рамках этапа `7.1`.

## Артефакты

- `tests/load/target-profiles.js` — k6 сценарии для API и Jitsi/JVB.
- `.github/workflows/stage-load-profiles.yml` — manual запуск профилей на stage URL.

## Профили

1. `api_read_profile`:
   - ramping VUs: 5 -> 20 -> 50 -> 0
   - проверки: `/health`, `/ready`
2. `jitsi_health_profile`:
   - constant VUs: 15 на 3 минуты
   - проверки: `JITSI_URL/`, `JVB_HEALTH_URL`

## Thresholds

- `http_req_failed < 3%`
- `p95(http_req_duration) < 1200ms`

## Как запускать

1. Запустить GitHub Action `Stage Load Profiles`.
2. Передать stage URL для API/Jitsi/JVB.
3. Сохранить отчет и приложить в release evidence/go-live материалы.

## Ограничения

- Это профильный прогон, не финальный production capacity test.
- Для закрытия пункта roadmap о load-тестах нужен фактический прогон и зафиксированные результаты.
