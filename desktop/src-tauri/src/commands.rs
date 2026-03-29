use base64::{engine::general_purpose::URL_SAFE_NO_PAD, Engine};
use rand::Rng;
use sha2::{Digest, Sha256};
use std::sync::Mutex;
use tauri::{command, AppHandle, Manager};

pub struct AuthState {
    pub code_verifier: Mutex<Option<String>>,
}

fn generate_code_verifier() -> String {
    let mut rng = rand::thread_rng();
    let bytes: Vec<u8> = (0..32).map(|_| rng.gen::<u8>()).collect();
    URL_SAFE_NO_PAD.encode(&bytes)
}

fn generate_code_challenge(verifier: &str) -> String {
    let hash = Sha256::digest(verifier.as_bytes());
    URL_SAFE_NO_PAD.encode(hash)
}

#[command]
pub async fn open_keycloak_auth(
    app: AppHandle,
    keycloak_url: String,
    realm: String,
    client_id: String,
    redirect_uri: String,
) -> Result<(), String> {
    let verifier = generate_code_verifier();
    let challenge = generate_code_challenge(&verifier);

    let state = app.state::<AuthState>();
    *state.code_verifier.lock().unwrap() = Some(verifier);

    let auth_url = format!(
        "{}/realms/{}/protocol/openid-connect/auth?client_id={}&redirect_uri={}&response_type=code&scope=openid+profile+email&code_challenge={}&code_challenge_method=S256",
        keycloak_url, realm,
        urlencoding::encode(&client_id),
        urlencoding::encode(&redirect_uri),
        challenge
    );

    open::that(&auth_url).map_err(|e| e.to_string())?;
    Ok(())
}

#[command]
pub async fn exchange_auth_code(
    app: AppHandle,
    keycloak_url: String,
    realm: String,
    client_id: String,
    redirect_uri: String,
    code: String,
) -> Result<serde_json::Value, String> {
    let state = app.state::<AuthState>();
    let verifier = state
        .code_verifier
        .lock()
        .unwrap()
        .take()
        .ok_or_else(|| "No code verifier found".to_string())?;

    let token_url = format!(
        "{}/realms/{}/protocol/openid-connect/token",
        keycloak_url, realm
    );

    let client = reqwest::Client::new();
    let resp = client
        .post(&token_url)
        .form(&[
            ("grant_type", "authorization_code"),
            ("client_id", &client_id),
            ("redirect_uri", &redirect_uri),
            ("code", &code),
            ("code_verifier", &verifier),
        ])
        .send()
        .await
        .map_err(|e| e.to_string())?;

    if !resp.status().is_success() {
        let text = resp.text().await.unwrap_or_default();
        return Err(format!("Token exchange failed: {}", text));
    }

    let tokens: serde_json::Value = resp.json().await.map_err(|e| e.to_string())?;
    Ok(tokens)
}

#[command]
pub async fn show_notification(
    app: AppHandle,
    title: String,
    body: String,
    _room_id: String,
) -> Result<(), String> {
    use tauri_plugin_notification::NotificationExt;
    app.notification()
        .builder()
        .title(&title)
        .body(&body)
        .show()
        .map_err(|e| e.to_string())?;
    Ok(())
}

#[command]
pub async fn open_auth(
    _app: AppHandle,
    auth_url: String,
) -> Result<(), String> {
    open::that(&auth_url).map_err(|e| e.to_string())?;
    Ok(())
}
