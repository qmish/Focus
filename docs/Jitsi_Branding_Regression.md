# Jitsi Branding Regression

Дата: 25 марта 2026 г.

## Цель

Зафиксировать и регулярно проверять корректность бренд-кастомизаций Jitsi после изменений в `pics`, backend branding endpoint и embed-конфигурации frontend.

## Артефакты

- `tests/smoke/jitsi-branding-regression.sh`
- `.github/workflows/stage-jitsi-branding-regression.yml`
- `tests/e2e/ci-api.spec.ts` (API smoke check branding payload)

## Что проверяем

1. Endpoint `GET /api/v1/branding/jitsi` доступен (HTTP 200).
2. В payload присутствуют ключевые поля:
   - `appName`, `dynamicBrandingUrl`,
   - `logoImageUrl`, `faviconUrl`, `backgroundImageUrl`,
   - `customTheme`, `customIcons`.
3. Ссылки на ассеты используют каталог `pics` (`/pics/...`).

## Запуск

1. GitHub Actions -> `Stage Jitsi Branding Regression`.
2. Передать `api_url` stage-среды.
3. Зафиксировать результат прогона в release evidence.

## Примечание

Конфигурационные ориентиры для параметров (`dynamicBrandingUrl`, `customTheme`, `customIcons`) соответствуют Jitsi Handbook.
