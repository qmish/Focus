//! Mobile-команды для Tauri-приложения Focus.
//!
//! Основные отличия от desktop-варианта:
//! * OAuth callback приходит через deep-link (`focus://auth/callback`),
//!   а не через локальный TCP-сервер (на iOS/Android это запрещено).
//! * Браузер OAuth открывается через системный шеллер (на Android — Custom Tab,
//!   на iOS — SFSafariViewController). Для каркаса достаточно открыть `_blank`
//!   через WebView.

use base64::{engine::general_purpose::URL_SAFE_NO_PAD, Engine};
use rand::Rng;
use serde::Serialize;
use sha2::{Digest, Sha256};
use std::sync::Mutex;
use tauri::{command, AppHandle, Emitter, Manager};

/// Состояние PKCE-сессии. Храним verifier в памяти процесса; на Android
/// процесс может быть убит, но при resume Tauri восстановит state из Activity.
pub struct AuthState {
    pub code_verifier: Mutex<Option<String>>,
    pub redirect_uri: Mutex<Option<String>>,
}

#[derive(Serialize, Clone)]
pub struct OAuthStartedPayload {
    pub auth_url: String,
}

fn gen_verifier() -> String {
    let mut rng = rand::thread_rng();
    let bytes: Vec<u8> = (0..32).map(|_| rng.gen::<u8>()).collect();
    URL_SAFE_NO_PAD.encode(&bytes)
}

fn gen_challenge(verifier: &str) -> String {
    let hash = Sha256::digest(verifier.as_bytes());
    URL_SAFE_NO_PAD.encode(hash)
}

/// Готовит OAuth-URL для Keycloak с PKCE и сохраняет verifier во AuthState.
/// Возвращает URL — webview сама выполнит navigate (через JS) или
/// мы пошлём событие `oauth-url-ready`, чтобы фронт открыл его в Custom Tab.
#[command]
pub async fn prepare_oauth_url(
    app: AppHandle,
    keycloak_url: String,
    realm: String,
    client_id: String,
    redirect_uri: String,
) -> Result<String, String> {
    let verifier = gen_verifier();
    let challenge = gen_challenge(&verifier);

    let state = app.state::<AuthState>();
    *state.code_verifier.lock().map_err(|e| e.to_string())? = Some(verifier);
    *state.redirect_uri.lock().map_err(|e| e.to_string())? = Some(redirect_uri.clone());

    let auth_url = format!(
        "{}/realms/{}/protocol/openid-connect/auth?\
client_id={}&redirect_uri={}&response_type=code&scope=openid+profile+email&\
code_challenge={}&code_challenge_method=S256",
        keycloak_url.trim_end_matches('/'),
        realm,
        urlencoding::encode(&client_id),
        urlencoding::encode(&redirect_uri),
        challenge
    );

    let _ = app.emit(
        "oauth-url-ready",
        OAuthStartedPayload {
            auth_url: auth_url.clone(),
        },
    );

    Ok(auth_url)
}

/// Обмен authorization code на токены. Вызывается после того, как
/// deep-link plugin отдал callback URL с параметром `code`.
#[command]
pub async fn exchange_code(
    app: AppHandle,
    keycloak_url: String,
    realm: String,
    client_id: String,
    code: String,
) -> Result<serde_json::Value, String> {
    let state = app.state::<AuthState>();
    let verifier = state
        .code_verifier
        .lock()
        .map_err(|e| e.to_string())?
        .clone()
        .ok_or_else(|| "PKCE verifier missing".to_string())?;
    let redirect_uri = state
        .redirect_uri
        .lock()
        .map_err(|e| e.to_string())?
        .clone()
        .ok_or_else(|| "redirect_uri missing".to_string())?;

    let token_url = format!(
        "{}/realms/{}/protocol/openid-connect/token",
        keycloak_url.trim_end_matches('/'),
        realm
    );
    let body = format!(
        "grant_type=authorization_code&client_id={}&code={}&redirect_uri={}&code_verifier={}",
        urlencoding::encode(&client_id),
        urlencoding::encode(&code),
        urlencoding::encode(&redirect_uri),
        urlencoding::encode(&verifier),
    );

    let client = reqwest::Client::new();
    let resp = client
        .post(token_url)
        .header("Content-Type", "application/x-www-form-urlencoded")
        .body(body)
        .send()
        .await
        .map_err(|e| e.to_string())?;
    if !resp.status().is_success() {
        let status = resp.status();
        let text = resp.text().await.unwrap_or_default();
        return Err(format!("Token exchange failed ({}): {}", status, text));
    }
    let json: serde_json::Value = resp.json().await.map_err(|e| e.to_string())?;
    Ok(json)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn verifier_has_expected_length() {
        let v = gen_verifier();
        // 32 байта в URL_SAFE_NO_PAD = 43 символа
        assert_eq!(v.len(), 43);
    }

    #[test]
    fn challenge_is_deterministic() {
        let v = "test-verifier".to_string();
        let c1 = gen_challenge(&v);
        let c2 = gen_challenge(&v);
        assert_eq!(c1, c2);
        // SHA-256 → 32 байта → 43 символа в URL_SAFE_NO_PAD
        assert_eq!(c1.len(), 43);
    }
}
