# Сборка Android-приложения Focus

Документ описывает каркас Android-сборки Focus Messenger Mobile в рамках
этапа 5 «Мобильные приложения и push-инфраструктура».

## Архитектура

Mobile-клиент построен на **Tauri 2 Mobile**:
* Rust-крейт `mobile/src-tauri/` (член Cargo workspace).
* Frontend — общий `frontend/` (PWA-сборка), монтируется в WebView.
* OAuth (Keycloak PKCE) — через системный браузер (Chrome Custom Tabs)
  с deep-link `focus://auth/callback`.
* Push — Web Push в WebView + опциональный FCM канал (заглушка-каркас).

## Требования

| Компонент           | Версия |
|---------------------|--------|
| Android API min     | **26** (Android 8.0) |
| Android API target  | 34     |
| Build Tools         | 34.0.0 |
| NDK                 | 27.0.12077973 |
| JDK                 | 17     |
| Rust                | stable + Android targets |
| Node.js             | ≥ 20   |
| `tauri-cli`         | `^2.0` |

## Локальная сборка (Linux/macOS)

```bash
# 1. Установить Rust-таргеты для Android
rustup target add aarch64-linux-android armv7-linux-androideabi \
    x86_64-linux-android i686-linux-android

# 2. Установить Tauri CLI
cargo install tauri-cli@^2.0 --locked

# 3. Подготовить frontend (PWA)
cd frontend && npm ci && npm run build && cd ..

# 4. Инициализировать Android-проект (один раз)
cd mobile/src-tauri
cargo tauri android init --ci
cd ../..

# 5. Применить наш AndroidManifest шаблон (intent-filter, permissions)
bash mobile/scripts/apply-android-template.sh

# 6. Собрать debug APK
cd mobile/src-tauri
cargo tauri android build --debug --apk
```

Готовый APK будет в `mobile/src-tauri/gen/android/app/build/outputs/apk/`.

## CI

Workflow `.github/workflows/mobile-android.yml`:
* триггеры: push в `master`/`dev` (только при изменении `mobile/**`,
  `frontend/**`), а также `workflow_dispatch`;
* `ubuntu-latest`, JDK 17, Android SDK + NDK через `android-actions/setup-android@v3`;
* генерируется debug keystore прямо в CI;
* собирается реальный APK и публикуется как артефакт `focus-mobile-debug-apk`
  (срок хранения 14 дней);
* запускаются rust unit-тесты `mobile/src-tauri`.

## Релизная подпись

Для production:
1. Сгенерировать release keystore:
   ```bash
   keytool -genkey -v -keystore focus-release.jks \
     -alias focus -keyalg RSA -keysize 2048 -validity 10000
   ```
2. Положить в Github Secrets:
   * `ANDROID_KEYSTORE_BASE64` — `base64 focus-release.jks`
   * `ANDROID_KEYSTORE_PASSWORD`
   * `ANDROID_KEY_ALIAS`
   * `ANDROID_KEY_PASSWORD`
3. Создать workflow `mobile-release-android.yml` с подменой keystore
   и сборкой `--release --aab`.

## OAuth deep-link

* AndroidManifest содержит intent-filter для `focus://auth` (custom scheme,
  для dev) и `https://auth.focus.local/callback` (App Links для prod).
* `tauri-plugin-deep-link` принимает callback и отправляет событие
  `oauth-callback` в WebView. Frontend вызывает Rust-команду
  `exchange_code` (см. `mobile/src-tauri/src/commands.rs`).

## Push на Android

* Сейчас работает Web Push в WebView (см. `docs/Push_Notifications.md`).
* В будущем для нативного FCM:
  - добавить Firebase config в `mobile/src-tauri/google-services.json`;
  - в backend выставить `PUSH_FCM_ENABLED=true` и заменить заглушку
    `internal/push/fcm.go` на реализацию через FCM HTTP v1 API.

## Минимальные API-уровни

`compileSdk = 34`, `minSdk = 26`, `targetSdk = 34`. Минимальная версия
обусловлена требованиями Tauri 2 Mobile и WebView, плюс совместимостью
с Notification API без legacy-патчей.
