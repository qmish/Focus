package config

import (
	"testing"
	"time"
)

func TestValidateSecurityRejectsSameSecrets(t *testing.T) {
	cfg := &Config{
		Env: "production",
		Auth: AuthConfig{
			SessionSecret:        "same-secret",
			SessionTokenLifetime: 24 * time.Hour,
		},
		Jitsi: JitsiConfig{
			AppSecret: "same-secret",
		},
	}

	if err := cfg.ValidateSecurity(); err == nil {
		t.Fatalf("expected error when SESSION_SECRET equals JITSI_APP_SECRET")
	}
}

func TestValidateSecurityAllowsDifferentSecrets(t *testing.T) {
	cfg := &Config{
		Env: "production",
		Auth: AuthConfig{
			SessionSecret:        "prod-session-secret-value",
			SessionTokenLifetime: 24 * time.Hour,
		},
		Jitsi: JitsiConfig{
			AppSecret: "prod-jitsi-secret-value",
		},
		WebSocket: WebSocketConfig{
			AllowedOrigins: []string{"https://chat.company.com"},
		},
	}

	if err := cfg.ValidateSecurity(); err != nil {
		t.Fatalf("expected valid security config, got error: %v", err)
	}
}

func TestValidateSecurityAllowsDevelopmentDefaultSessionSecret(t *testing.T) {
	cfg := &Config{
		Env: "development",
		Auth: AuthConfig{
			SessionSecret:        "dev-session-secret-change-me",
			SessionTokenLifetime: 24 * time.Hour,
		},
		Jitsi: JitsiConfig{
			AppSecret: "jitsi-dev-secret",
		},
	}

	if err := cfg.ValidateSecurity(); err != nil {
		t.Fatalf("expected development config to allow default session secret, got error: %v", err)
	}
}

func TestValidateSecurityRejectsTooShortSessionLifetime(t *testing.T) {
	cfg := &Config{
		Env: "production",
		Auth: AuthConfig{
			SessionSecret:        "prod-session-secret-value",
			SessionTokenLifetime: 5 * time.Minute,
		},
		Jitsi: JitsiConfig{
			AppSecret: "prod-jitsi-secret-value",
		},
		WebSocket: WebSocketConfig{
			AllowedOrigins: []string{"https://chat.company.com"},
		},
	}
	if err := cfg.ValidateSecurity(); err == nil {
		t.Fatalf("expected error for too short session token lifetime")
	}
}

func TestValidateSecurityRejectsEmptyWebSocketOriginsInProd(t *testing.T) {
	cfg := &Config{
		Env: "production",
		Auth: AuthConfig{
			SessionSecret:        "prod-session-secret-value",
			SessionTokenLifetime: 24 * time.Hour,
		},
		Jitsi: JitsiConfig{
			AppSecret: "prod-jitsi-secret-value",
		},
		WebSocket: WebSocketConfig{
			AllowedOrigins: []string{},
		},
	}
	if err := cfg.ValidateSecurity(); err == nil {
		t.Fatalf("expected error when WS_ALLOWED_ORIGINS is empty in production")
	}
}
