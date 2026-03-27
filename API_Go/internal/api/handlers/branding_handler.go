package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/qmish/focus-api/internal/models"
)

// AppSettingsGetter reads appearance settings for branding merge.
type AppSettingsGetter interface {
	Get(ctx context.Context) (*models.AppSetting, error)
}

// JitsiBrandingHandler serves dynamic branding payload for jitsi fork.
type JitsiBrandingHandler struct {
	appSettings AppSettingsGetter
}

// NewJitsiBrandingHandler creates branding handler instance.
func NewJitsiBrandingHandler() *JitsiBrandingHandler {
	return &JitsiBrandingHandler{}
}

// SetAppSettingsGetter injects app settings for dynamic conference theming.
func (h *JitsiBrandingHandler) SetAppSettingsGetter(getter AppSettingsGetter) {
	h.appSettings = getter
}

// DynamicBranding GET /api/v1/branding/jitsi
func (h *JitsiBrandingHandler) DynamicBranding(w http.ResponseWriter, r *http.Request) {
	theme := map[string]string{
		"palette.ui01":     "#0B1220",
		"palette.ui02":     "#111827",
		"palette.action01": "#0EA5E9",
		"palette.text01":   "#F9FAFB",
	}

	appName := "Focus Meet"
	logoURL := "/pics/image34.png"

	if h.appSettings != nil {
		if settings, err := h.appSettings.Get(r.Context()); err == nil && settings != nil {
			if settings.ConferenceThemeJSON != "" && settings.ConferenceThemeJSON != "{}" {
				var dbTheme map[string]string
				if json.Unmarshal([]byte(settings.ConferenceThemeJSON), &dbTheme) == nil {
					for k, v := range dbTheme {
						theme[k] = v
					}
				}
			}
			if settings.BrandingProductName != "" {
				appName = settings.BrandingProductName + " Meet"
			}
			if settings.BrandingLogoURL != "" && settings.BrandingLogoURL != "/logo.png" {
				logoURL = settings.BrandingLogoURL
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"appName":            appName,
		"defaultLanguage":    "ru",
		"dynamicBrandingUrl": "/api/v1/branding/jitsi",
		"logoImageUrl":       logoURL,
		"watermarkImageUrl":  "/pics/image34.png",
		"backgroundImageUrl": "/pics/image29.png",
		"faviconUrl":         "/pics/image28.png",
		"customTheme":        theme,
		"customIcons": map[string]string{
			"mic":          "/pics/image16.png",
			"camera":       "/pics/image17.png",
			"hangup":       "/pics/image16.png",
			"participants": "/pics/image17.png",
		},
	})
}
