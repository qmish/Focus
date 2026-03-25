package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDynamicBranding(t *testing.T) {
	handler := NewJitsiBrandingHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/branding/jitsi", nil)
	rr := httptest.NewRecorder()

	handler.DynamicBranding(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var payload map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &payload)
	require.NoError(t, err)

	assert.Equal(t, "Focus Meet", payload["appName"])
	assert.Equal(t, "ru", payload["defaultLanguage"])
	assert.Equal(t, "/api/v1/branding/jitsi", payload["dynamicBrandingUrl"])
	assert.Contains(t, payload, "customTheme")
	assert.Contains(t, payload, "customIcons")
}
