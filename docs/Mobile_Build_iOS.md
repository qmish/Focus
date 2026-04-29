# Сборка iOS-приложения Focus

## Архитектура

Tauri 2 Mobile + общий React-фронтенд + общий Rust-крейт workspace.
Bundle ID: `com.focus.messenger.mobile`. Минимальный таргет: **iOS 14.0**.

## Требования

| Компонент       | Версия |
|-----------------|--------|
| macOS           | Sonoma+ (для Xcode 16) |
| Xcode           | ≥ 15.0 |
| iOS deployment  | 14.0 |
| Rust            | stable + iOS targets |
| Node.js         | ≥ 20 |
| `tauri-cli`     | `^2.0` |
| Apple Developer | для подписи и публикации |

## Локальная сборка (MacBook Pro)

```bash
# 1. Rust-таргеты для iOS
rustup target add aarch64-apple-ios aarch64-apple-ios-sim x86_64-apple-ios

# 2. Tauri CLI
cargo install tauri-cli@^2.0 --locked

# 3. Frontend
cd frontend && npm ci && npm run build && cd ..

# 4. Инициализация iOS-проекта
cd mobile/src-tauri
cargo tauri ios init --ci
cd ../..

# 5. Применяем кастомный Info.plist (push, OAuth scheme, ATS)
bash mobile/scripts/apply-ios-template.sh

# 6. Открываем проект в Xcode
open mobile/src-tauri/gen/apple/focus_mobile.xcodeproj
```

В Xcode задайте свою команду разработчика (Signing & Capabilities → Team).
Запустите на симуляторе (Cmd+R) или подключённом устройстве.

CLI-сборка симулятора без подписи:
```bash
cd mobile/src-tauri
cargo tauri ios build --debug --target aarch64-apple-ios-sim
```

## CI

Workflow `.github/workflows/mobile-ios.yml`:
* `macos-latest`, по запросу (`workflow_dispatch`) и при push в `master`
  (только при изменении `mobile/**`);
* собирается без подписи на симуляторе для проверки сборки;
* запускает rust unit-тесты `mobile/src-tauri`.

CI на Windows-окружении не предполагается — iOS требует macOS.

## Подписание и публикация

1. Получить App ID `com.focus.messenger.mobile` в developer.apple.com.
2. Создать Provisioning Profile (Development + Distribution).
3. Включить Capabilities: Push Notifications, Background Modes (remote-notification, voip).
4. В Xcode выполнить Product → Archive → Distribute → App Store Connect.
5. TestFlight → внутреннее тестирование → submission.

## OAuth callback

* `Info.plist` объявляет URL scheme `focus://`.
* `tauri-plugin-deep-link` ловит callback и пробрасывает в Rust.
* Для prod рекомендуется Universal Links на `https://auth.focus.local/callback`
  (требует `apple-app-site-association` файл на домене).

## Push на iOS

* Web Push в WebView (через PWA service worker) **не работает** в нативном WKWebView.
* Для нативных уведомлений нужно подключить APNs:
  - в backend выставить `PUSH_APNS_ENABLED=true`;
  - заменить заглушку `internal/push/apns.go` на реализацию через
    apns2-go (HTTP/2) с собственным сертификатом или authentication key (`.p8`).
* Минимально для бета-стенда достаточно Web Push для PWA-варианта,
  установленной через Safari «Добавить на главный экран» (iOS 16.4+).

## Известные ограничения каркаса

* Иконки в `mobile/src-tauri/icons/` — плейсхолдеры. Для App Store нужны
  AppIcon.appiconset с реальными ассетами 20…1024px.
* Universal Links требуют публичного HTTPS-домена и AASA-файла; для
  локального теста используется custom scheme `focus://`.
