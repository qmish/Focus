package config

import "testing"

func TestValidateSecurityRejectsSameSecrets(t *testing.T) {
	cfg := &Config{
		Env:  "production",
		Auth: AuthConfig{SessionSecret: "same-secret"},
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
		Env:  "production",
		Auth: AuthConfig{SessionSecret: "prod-session-secret-value"},
		Jitsi: JitsiConfig{
			AppSecret: "prod-jitsi-secret-value",
		},
	}

	if err := cfg.ValidateSecurity(); err != nil {
		t.Fatalf("expected valid security config, got error: %v", err)
	}
}

func TestValidateSecurityAllowsDevelopmentDefaultSessionSecret(t *testing.T) {
	cfg := &Config{
		Env:  "development",
		Auth: AuthConfig{SessionSecret: "dev-session-secret-change-me"},
		Jitsi: JitsiConfig{
			AppSecret: "jitsi-dev-secret",
		},
	}

	if err := cfg.ValidateSecurity(); err != nil {
		t.Fatalf("expected development config to allow default session secret, got error: %v", err)
	}
}
