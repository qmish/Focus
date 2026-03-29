use base64::{engine::general_purpose::URL_SAFE_NO_PAD, Engine};
use rand::Rng;
use sha2::{Digest, Sha256};
use std::io::{BufRead, BufReader, Write};
use std::net::TcpListener;
use std::sync::Mutex;
use tauri::{command, AppHandle, Emitter, Manager};

pub struct AuthState {
    pub code_verifier: Mutex<Option<String>>,
    pub callback_redirect_uri: Mutex<Option<String>>,
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

fn extract_code_from_request(request_line: &str) -> Option<String> {
    let path = request_line.split_whitespace().nth(1)?;
    let query = path.split('?').nth(1)?;
    for param in query.split('&') {
        let mut kv = param.splitn(2, '=');
        if kv.next() == Some("code") {
            return kv.next().map(|v| urlencoding::decode(v).unwrap_or_default().into_owned());
        }
    }
    None
}

#[command]
pub async fn open_keycloak_auth(
    app: AppHandle,
    keycloak_url: String,
    realm: String,
    client_id: String,
    _redirect_uri: String,
) -> Result<(), String> {
    let verifier = generate_code_verifier();
    let challenge = generate_code_challenge(&verifier);

    const PREFERRED_PORT: u16 = 17923;
    let listener = TcpListener::bind(format!("127.0.0.1:{}", PREFERRED_PORT))
        .or_else(|_| TcpListener::bind("127.0.0.1:0"))
        .map_err(|e| e.to_string())?;
    let port = listener.local_addr().map_err(|e| e.to_string())?.port();
    let callback_url = format!("http://localhost:{}/auth/callback", port);

    let state = app.state::<AuthState>();
    *state.code_verifier.lock().unwrap() = Some(verifier);
    *state.callback_redirect_uri.lock().unwrap() = Some(callback_url.clone());

    let auth_url = format!(
        "{}/realms/{}/protocol/openid-connect/auth?client_id={}&redirect_uri={}&response_type=code&scope=openid+profile+email&code_challenge={}&code_challenge_method=S256",
        keycloak_url, realm,
        urlencoding::encode(&client_id),
        urlencoding::encode(&callback_url),
        challenge
    );

    open::that(&auth_url).map_err(|e| e.to_string())?;

    let app_handle = app.clone();
    std::thread::spawn(move || {
        listener
            .set_nonblocking(false)
            .ok();
        if let Ok((stream, _)) = listener.accept() {
            let mut reader = BufReader::new(&stream);
            let mut request_line = String::new();
            if reader.read_line(&mut request_line).is_ok() {
                if let Some(code) = extract_code_from_request(&request_line) {
                    let html = "<html><head><meta charset='utf-8'></head><body style='font-family:sans-serif;display:flex;align-items:center;justify-content:center;height:100vh;margin:0;background:#f5f5f5'>\
                        <div style='text-align:center'><h1 style='color:#4caf50'>✓ Авторизация успешна</h1><p>Можете закрыть эту вкладку и вернуться в приложение.</p></div></body></html>";
                    let response = format!(
                        "HTTP/1.1 200 OK\r\nContent-Type: text/html; charset=utf-8\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{}",
                        html.len(),
                        html
                    );
                    let mut writer = stream.try_clone().unwrap_or_else(|_| {
                        std::net::TcpStream::connect("127.0.0.1:1").unwrap()
                    });
                    let _ = writer.write_all(response.as_bytes());
                    let _ = writer.flush();

                    app_handle.emit("auth-code-received", code).ok();
                }
            }
        }
    });

    Ok(())
}

#[command]
pub async fn exchange_auth_code(
    app: AppHandle,
    keycloak_url: String,
    realm: String,
    client_id: String,
    _redirect_uri: String,
    code: String,
) -> Result<serde_json::Value, String> {
    let state = app.state::<AuthState>();
    let verifier = state
        .code_verifier
        .lock()
        .unwrap()
        .take()
        .ok_or_else(|| "No code verifier found".to_string())?;

    let redirect_uri = state
        .callback_redirect_uri
        .lock()
        .unwrap()
        .take()
        .ok_or_else(|| "No callback redirect URI found".to_string())?;

    let token_url = format!(
        "{}/realms/{}/protocol/openid-connect/token",
        keycloak_url, realm
    );

    let client = reqwest::Client::builder()
        .danger_accept_invalid_certs(true)
        .build()
        .map_err(|e| e.to_string())?;
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
