mod commands;

use std::sync::Mutex;
use tauri::Manager;

use commands::AuthState;

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_notification::init())
        .plugin(tauri_plugin_deep_link::init())
        .plugin(tauri_plugin_store::Builder::default().build())
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_os::init())
        .setup(|app| {
            app.manage(AuthState {
                code_verifier: Mutex::new(None),
                redirect_uri: Mutex::new(None),
            });
            Ok(())
        })
        .invoke_handler(tauri::generate_handler![
            commands::prepare_oauth_url,
            commands::exchange_code,
        ])
        .run(tauri::generate_context!())
        .expect("error while running Focus mobile app");
}
