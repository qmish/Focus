package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

type DesktopUpdateHandler struct {
	logger        *zap.Logger
	updatesDir    string
	latestVersion string
}

type DesktopUpdateResponse struct {
	Version   string `json:"version"`
	Notes     string `json:"notes"`
	PubDate   string `json:"pub_date"`
	URL       string `json:"url"`
	Signature string `json:"signature"`
}

func NewDesktopUpdateHandler(logger *zap.Logger) *DesktopUpdateHandler {
	return &DesktopUpdateHandler{
		logger:     logger,
		updatesDir: os.Getenv("DESKTOP_UPDATES_DIR"),
	}
}

func (h *DesktopUpdateHandler) CheckUpdate(w http.ResponseWriter, r *http.Request) {
	target := r.PathValue("target")
	arch := r.PathValue("arch")
	currentVersion := r.PathValue("current_version")

	if h.updatesDir == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	manifestPath := filepath.Join(h.updatesDir, "latest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		h.logger.Debug("Нет файла обновлений", zap.Error(err))
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		h.logger.Error("Ошибка парсинга манифеста обновлений", zap.Error(err))
		w.WriteHeader(http.StatusNoContent)
		return
	}

	latestVersion, _ := manifest["version"].(string)
	if latestVersion == "" || latestVersion == currentVersion {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	platforms, _ := manifest["platforms"].(map[string]interface{})
	if platforms == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	platformKey := target + "-" + arch
	platform, ok := platforms[platformKey]
	if !ok {
		h.logger.Debug("Платформа не найдена", zap.String("target", target), zap.String("arch", arch), zap.String("key", platformKey))
		w.WriteHeader(http.StatusNoContent)
		return
	}

	platformMap, _ := platform.(map[string]interface{})
	if platformMap == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	response := DesktopUpdateResponse{
		Version:   latestVersion,
		Notes:     stringVal(manifest, "notes"),
		PubDate:   stringVal(manifest, "pub_date"),
		URL:       stringVal(platformMap, "url"),
		Signature: stringVal(platformMap, "signature"),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func stringVal(m map[string]interface{}, key string) string {
	v, _ := m[key].(string)
	return v
}
