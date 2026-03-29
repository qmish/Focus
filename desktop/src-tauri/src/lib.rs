mod commands;
mod tray;

use std::sync::Mutex;
use tauri::{Emitter, Listener, Manager};

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .manage(commands::AuthState {
            code_verifier: Mutex::new(None),
        })
        .plugin(tauri_plugin_single_instance::init(|app, _args, _cwd| {
            if let Some(window) = app.get_webview_window("main") {
                let _ = window.show();
                let _ = window.unminimize();
                let _ = window.set_focus();
            }
        }))
        .plugin(tauri_plugin_autostart::init(
            tauri_plugin_autostart::MacosLauncher::LaunchAgent,
            Some(vec![]),
        ))
        .plugin(tauri_plugin_notification::init())
        .plugin(tauri_plugin_deep_link::init())
        .plugin(tauri_plugin_store::Builder::default().build())
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_updater::Builder::new().build())
        .invoke_handler(tauri::generate_handler![
            commands::show_notification,
            commands::open_auth,
            commands::open_keycloak_auth,
            commands::exchange_auth_code,
        ])
        .setup(|app| {
            let handle = app.handle().clone();
            tray::create_tray(&handle)?;

            let app_handle = app.handle().clone();
            app.listen("deep-link://new-url", move |event: tauri::Event| {
                if let Some(payload) = event
                    .payload()
                    .strip_prefix('"')
                    .and_then(|s| s.strip_suffix('"'))
                {
                    if payload.starts_with("focus://auth/callback") {
                        app_handle
                            .emit("auth-deep-link", payload.to_string())
                            .ok();
                    }
                }
            });

            let main_window = app.get_webview_window("main").unwrap();
            let app_handle_close = app.handle().clone();
            main_window.on_window_event(move |event| {
                if let tauri::WindowEvent::CloseRequested { api, .. } = event {
                    api.prevent_close();
                    if let Some(window) = app_handle_close.get_webview_window("main") {
                        let _ = window.hide();
                    }
                }
            });

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running Focus Desktop");
}
