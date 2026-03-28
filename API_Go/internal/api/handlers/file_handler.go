package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/auth"
)

var validExtRe = regexp.MustCompile(`^\.[a-zA-Z0-9]{1,10}$`)

const maxUploadSize = 50 << 20 // 50 MB

// FileHandler handles file upload/download
type FileHandler struct {
	uploadDir string
}

// NewFileHandler creates a new FileHandler
func NewFileHandler(uploadDir string) *FileHandler {
	_ = os.MkdirAll(uploadDir, 0755)
	return &FileHandler{uploadDir: uploadDir}
}

// Upload POST /api/v1/files/upload
func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Требуется авторизация", http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		http.Error(w, "Файл слишком большой (макс. 50 МБ)", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Отсутствует поле file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	fileID := uuid.New().String()
	ext := filepath.Ext(header.Filename)
	if !validExtRe.MatchString(ext) {
		http.Error(w, "Некорректное расширение файла", http.StatusBadRequest)
		return
	}
	storedName := fileID + ext

	dstPath := filepath.Join(h.uploadDir, storedName)
	dst, err := os.Create(dstPath)
	if err != nil {
		http.Error(w, "Не удалось сохранить файл", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	written, err := io.Copy(dst, file)
	if err != nil {
		os.Remove(dstPath)
		http.Error(w, "Не удалось сохранить файл", http.StatusInternalServerError)
		return
	}

	mime := header.Header.Get("Content-Type")
	if mime == "" {
		mime = "application/octet-stream"
	}

	resp := map[string]interface{}{
		"file_id":   fileID,
		"file_name": header.Filename,
		"file_size": written,
		"file_mime": mime,
		"url":       fmt.Sprintf("/api/v1/files/%s%s", fileID, ext),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// Download GET /api/v1/files/{fileId}
func (h *FileHandler) Download(w http.ResponseWriter, r *http.Request) {
	fileParam := chi.URLParam(r, "fileId")
	if fileParam == "" {
		http.Error(w, "Отсутствует идентификатор файла", http.StatusBadRequest)
		return
	}

	if strings.Contains(fileParam, "..") || strings.Contains(fileParam, "/") || strings.Contains(fileParam, "\\") {
		http.Error(w, "Некорректный идентификатор файла", http.StatusBadRequest)
		return
	}

	entries, err := os.ReadDir(h.uploadDir)
	if err != nil {
		http.Error(w, "Файл не найден", http.StatusNotFound)
		return
	}

	baseName := strings.TrimSuffix(fileParam, filepath.Ext(fileParam))
	var found string
	for _, e := range entries {
		if strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())) == baseName {
			found = e.Name()
			break
		}
	}

	if found == "" {
		http.Error(w, "Файл не найден", http.StatusNotFound)
		return
	}

	filePath := filepath.Join(h.uploadDir, found)

	disposition := "attachment"
	if r.URL.Query().Get("inline") == "1" {
		disposition = "inline"
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`%s; filename="%s"`, disposition, found))
	http.ServeFile(w, r, filePath)
}
