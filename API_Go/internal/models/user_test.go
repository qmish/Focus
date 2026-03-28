package models

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewUser(t *testing.T) {
	kid := uuid.New()
	u := NewUser(kid, "test@example.com", "Test User")

	if u.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", u.Email)
	}
	if u.Name != "Test User" {
		t.Errorf("expected name Test User, got %s", u.Name)
	}
	if !u.IsActive {
		t.Error("expected user to be active")
	}
	if !u.HasRole("user") {
		t.Error("expected user to have 'user' role")
	}
	if u.KeycloakID == nil {
		t.Fatal("expected KeycloakID to be set")
	}
	if *u.KeycloakID != kid {
		t.Errorf("expected KeycloakID %s, got %s", kid, *u.KeycloakID)
	}
	if u.ID == uuid.Nil {
		t.Error("expected generated UUID, got nil")
	}
}

func TestUser_ProfileFields(t *testing.T) {
	u := &User{
		ID:                       uuid.New(),
		Email:                    "test@example.com",
		Name:                     "Test User",
		Department:               "Engineering",
		Directorate:              "IT",
		Position:                 "Developer",
		Phone:                    "+7 999 123 45 67",
		AboutMe:                  "Hello world",
		VideoStartWithAudioMuted: true,
		VideoStartWithVideoMuted: false,
		VideoDisplayName:         "TestUser",
		VideoDefaultLanguage:     "ru",
	}

	if u.Department != "Engineering" {
		t.Errorf("expected Department 'Engineering', got '%s'", u.Department)
	}
	if u.Directorate != "IT" {
		t.Errorf("expected Directorate 'IT', got '%s'", u.Directorate)
	}
	if u.Position != "Developer" {
		t.Errorf("expected Position 'Developer', got '%s'", u.Position)
	}
	if u.Phone != "+7 999 123 45 67" {
		t.Errorf("expected Phone '+7 999 123 45 67', got '%s'", u.Phone)
	}
	if u.AboutMe != "Hello world" {
		t.Errorf("expected AboutMe 'Hello world', got '%s'", u.AboutMe)
	}
	if !u.VideoStartWithAudioMuted {
		t.Error("expected VideoStartWithAudioMuted to be true")
	}
	if u.VideoStartWithVideoMuted {
		t.Error("expected VideoStartWithVideoMuted to be false")
	}
	if u.VideoDisplayName != "TestUser" {
		t.Errorf("expected VideoDisplayName 'TestUser', got '%s'", u.VideoDisplayName)
	}
	if u.VideoDefaultLanguage != "ru" {
		t.Errorf("expected VideoDefaultLanguage 'ru', got '%s'", u.VideoDefaultLanguage)
	}
}

func TestUser_HasRole(t *testing.T) {
	u := &User{Roles: StringArray{"user", "admin"}}

	if !u.HasRole("user") {
		t.Error("expected HasRole('user') to be true")
	}
	if !u.HasRole("admin") {
		t.Error("expected HasRole('admin') to be true")
	}
	if u.HasRole("superadmin") {
		t.Error("expected HasRole('superadmin') to be false")
	}
}

func TestUser_AddRole(t *testing.T) {
	u := &User{Roles: StringArray{"user"}}
	u.AddRole("admin")

	if len(u.Roles) != 2 {
		t.Errorf("expected 2 roles, got %d", len(u.Roles))
	}

	u.AddRole("admin")
	if len(u.Roles) != 2 {
		t.Errorf("expected 2 roles after duplicate add, got %d", len(u.Roles))
	}
}

func TestUser_AddRole_Empty(t *testing.T) {
	u := &User{}
	u.AddRole("admin")

	if len(u.Roles) != 1 {
		t.Errorf("expected 1 role, got %d", len(u.Roles))
	}
	if !u.HasRole("admin") {
		t.Error("expected HasRole('admin') to be true")
	}
}

func TestStringArray_Value(t *testing.T) {
	a := StringArray{"user", "admin"}
	val, err := a.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "{user,admin}"
	if val != expected {
		t.Errorf("expected %q, got %q", expected, val)
	}
}

func TestStringArray_Value_Nil(t *testing.T) {
	var a StringArray
	val, err := a.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != nil {
		t.Errorf("expected nil, got %v", val)
	}
}

func TestStringArray_Scan(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected StringArray
	}{
		{"nil", nil, nil},
		{"empty string", "", StringArray{}},
		{"empty braces", "{}", StringArray{}},
		{"single", "{user}", StringArray{"user"}},
		{"multiple", "{user,admin}", StringArray{"user", "admin"}},
		{"bytes", []byte("{user,admin}"), StringArray{"user", "admin"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var a StringArray
			if err := a.Scan(tt.input); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.expected == nil {
				if a != nil {
					t.Errorf("expected nil, got %v", a)
				}
				return
			}
			if len(a) != len(tt.expected) {
				t.Errorf("expected %d elements, got %d", len(tt.expected), len(a))
				return
			}
			for i := range a {
				if a[i] != tt.expected[i] {
					t.Errorf("element %d: expected %q, got %q", i, tt.expected[i], a[i])
				}
			}
		})
	}
}

func TestStringArray_Scan_UnsupportedType(t *testing.T) {
	var a StringArray
	err := a.Scan(123)
	if err == nil {
		t.Error("expected error for unsupported type, got nil")
	}
}
