package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUpdateProfile_RequiresAuth(t *testing.T) {
	req := httptest.NewRequest("PUT", "/api/v1/auth/profile", bytes.NewBufferString(`{"name":"Test"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler := &AuthHandler{}
	handler.UpdateProfile(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestUpdateProfile_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest("PUT", "/api/v1/auth/profile", bytes.NewBufferString(`{invalid}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler := &AuthHandler{}
	handler.UpdateProfile(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth, got %d", rr.Code)
	}
}

func TestMe_RequiresAuth(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	rr := httptest.NewRecorder()

	handler := &AuthHandler{}
	handler.Me(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestMe_RequiresAuth_ResponseBody(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	rr := httptest.NewRecorder()

	handler := &AuthHandler{}
	handler.Me(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}

	body := rr.Body.String()
	if body == "" {
		t.Error("expected non-empty error body")
	}
	var resp map[string]interface{}
	_ = json.Unmarshal([]byte(body), &resp) // допускается plain-text ответ от http.Error
}
