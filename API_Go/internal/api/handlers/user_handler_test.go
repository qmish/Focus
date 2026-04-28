package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchUsers_EmptyQuery(t *testing.T) {
	handler := NewUserHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/search", nil)
	rr := httptest.NewRecorder()
	handler.SearchUsers(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result []interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &result))
	assert.Empty(t, result)
}

func TestSearchUsers_InvalidRoomID(t *testing.T) {
	handler := NewUserHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/search?q=test&room_id=not-uuid", nil)
	rr := httptest.NewRecorder()
	handler.SearchUsers(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
