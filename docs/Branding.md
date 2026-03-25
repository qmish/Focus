# Branding Assets (`pics`)

Документ фиксирует фактический набор UI-ассетов и их назначение для `frontend`, `frontend-admin` и форка `jitsi-meet`.

## Источник ассетов

- Базовая директория: `pics/`
- Текущий формат хранения: плоский (без подпапок)
- Структура назначения (логическая): `logo`, `favicon`, `backgrounds`, `icons`, `avatars`

Полный машинно-читаемый список ассетов и назначений: `docs/branding-manifest.json`.

## Матрица применения

- `frontend`:
  - `logo`, `favicon`
  - `avatars`
  - часть `icons` и `backgrounds`
- `frontend-admin`:
  - `logo`, `favicon`
  - `icons`
- `jitsi-fork`:
  - `logo`
  - `backgrounds`
  - `icons` (toolbar/branding)

## Ограничения и next steps

- `pics/hdphoto1.wdp` (формат `wdp`) конвертирован в `pics/hdphoto1.png` для web UI.
- Для production-пайплайна рекомендована физическая нормализация структуры:
  - `pics/logo/`
  - `pics/favicon/`
  - `pics/backgrounds/`
  - `pics/icons/`
  - `pics/avatars/`

