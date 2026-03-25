package handlers

import (
	"encoding/json"
	"net/http"
)

// JitsiBrandingHandler serves dynamic branding payload for jitsi fork.
type JitsiBrandingHandler struct{}

// NewJitsiBrandingHandler creates branding handler instance.
func NewJitsiBrandingHandler() *JitsiBrandingHandler {
	return &JitsiBrandingHandler{}
}

// DynamicBranding GET /api/v1/branding/jitsi
func (h *JitsiBrandingHandler) DynamicBranding(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"appName":            "Focus Meet",
		"defaultLanguage":    "ru",
		"dynamicBrandingUrl": "/api/v1/branding/jitsi",
		"logoImageUrl":       "/pics/image34.png",
		"watermarkImageUrl":  "/pics/image34.png",
		"backgroundImageUrl": "/pics/image29.png",
		"faviconUrl":         "/pics/image28.png",
		"customTheme": map[string]string{
			"palette.ui01":     "#0B1220",
			"palette.ui02":     "#111827",
			"palette.action01": "#0EA5E9",
			"palette.text01":   "#F9FAFB",
		},
		"customIcons": map[string]string{
			"mic":          "/pics/image16.png",
			"camera":       "/pics/image17.png",
			"hangup":       "/pics/image16.png",
			"participants": "/pics/image17.png",
		},
	})
}
