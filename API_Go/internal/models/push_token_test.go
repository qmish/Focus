package models

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewWebPushToken(t *testing.T) {
	uid := uuid.New()
	tok := NewWebPushToken(uid, "https://push.example/endpoint", "p256dh-key", "auth-key")

	if tok.ID == uuid.Nil {
		t.Error("expected generated UUID, got nil")
	}
	if tok.UserID != uid {
		t.Errorf("expected UserID %s, got %s", uid, tok.UserID)
	}
	if tok.Platform != PushPlatformWeb {
		t.Errorf("expected Platform=web, got %s", tok.Platform)
	}
	if !tok.IsWebPush() {
		t.Error("IsWebPush() should be true")
	}
	if tok.Endpoint != "https://push.example/endpoint" {
		t.Errorf("unexpected endpoint: %s", tok.Endpoint)
	}
	if tok.P256DHKey != "p256dh-key" || tok.AuthKey != "auth-key" {
		t.Error("expected p256dh/auth to be set")
	}
	if tok.CreatedAt.IsZero() || tok.UpdatedAt.IsZero() {
		t.Error("expected timestamps to be set")
	}
}

func TestNewMobilePushToken(t *testing.T) {
	uid := uuid.New()
	tok := NewMobilePushToken(uid, PushPlatformFCM, "fcm-token-xyz")

	if tok.Platform != PushPlatformFCM {
		t.Errorf("expected Platform=fcm, got %s", tok.Platform)
	}
	if tok.IsWebPush() {
		t.Error("IsWebPush() should be false for FCM")
	}
	if tok.Endpoint != "fcm-token-xyz" {
		t.Errorf("expected endpoint=fcm-token-xyz, got %s", tok.Endpoint)
	}
	if tok.P256DHKey != "" || tok.AuthKey != "" {
		t.Error("FCM token should not have web-push keys")
	}
}

func TestPushToken_TableName(t *testing.T) {
	if (PushToken{}).TableName() != "push_tokens" {
		t.Errorf("unexpected table name: %s", (PushToken{}).TableName())
	}
}
