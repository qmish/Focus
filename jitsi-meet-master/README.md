# Focus Jitsi Fork Contract

Этот каталог фиксирует управляемый слой сопровождения форка `jitsi-meet-master` в составе `Focus`.

## Upstream strategy

- Upstream источник: `jitsi/jitsi-meet` (основной репозиторий).
- Частота синхронизации: минимум раз в 2 недели и дополнительно при security hotfix.
- Merge policy: `upstream/master -> focus-fork/main` через отдельный PR с обязательным smoke/regression прогоном.
- Owner: команда `Focus Platform` (Backend + Frontend lead).

## Граница кастомизаций

- `config-only`:
  - `dynamicBrandingUrl`,
  - `customTheme`,
  - `customIcons`,
  - отключение публичных функций по policy.
- `code-level`:
  - только при невозможности реализовать требование через конфиг,
  - любое code-level изменение должно иметь ссылку на задачу и тест-план.

## Брендинг

- Источник ассетов: `pics/` + `docs/branding-manifest.json`.
- Базовые overrides: `jitsi-meet-master/config/custom/branding-overrides.js`.
- Dynamic branding endpoint: `GET /api/v1/branding/jitsi`.
