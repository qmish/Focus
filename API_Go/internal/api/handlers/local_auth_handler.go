package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/auth"
	"github.com/qmish/focus-api/internal/models"
	"github.com/qmish/focus-api/internal/repository"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var (
	regLimiterMu sync.Mutex
	regLimiter   = make(map[string][]time.Time)
)

func isRegistrationRateLimited(ip string) bool {
	regLimiterMu.Lock()
	defer regLimiterMu.Unlock()

	now := time.Now()
	window := now.Add(-1 * time.Minute)

	timestamps := regLimiter[ip]
	valid := timestamps[:0]
	for _, t := range timestamps {
		if t.After(window) {
			valid = append(valid, t)
		}
	}
	regLimiter[ip] = valid

	if len(valid) >= 5 {
		return true
	}
	regLimiter[ip] = append(valid, now)
	return false
}

type LocalAuthHandler struct {
	userRepo      *repository.UserRepository
	sessionSecret []byte
	tokenLifetime time.Duration
	logger        *zap.Logger
}

func NewLocalAuthHandler(
	userRepo *repository.UserRepository,
	sessionSecret []byte,
	tokenLifetime time.Duration,
	logger *zap.Logger,
) *LocalAuthHandler {
	return &LocalAuthHandler{
		userRepo:      userRepo,
		sessionSecret: sessionSecret,
		tokenLifetime: tokenLifetime,
		logger:        logger,
	}
}

func (h *LocalAuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	ip := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ip = strings.Split(fwd, ",")[0]
	}
	if isRegistrationRateLimited(strings.TrimSpace(ip)) {
		http.Error(w, "Слишком много попыток регистрации, попробуйте позже", http.StatusTooManyRequests)
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректные данные запроса", http.StatusBadRequest)
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Name = strings.TrimSpace(req.Name)

	if req.Email == "" || req.Password == "" || req.Name == "" {
		http.Error(w, "Требуются email, пароль и имя", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 6 {
		http.Error(w, "Пароль должен содержать не менее 6 символов", http.StatusBadRequest)
		return
	}

	existing, _ := h.userRepo.GetByEmail(r.Context(), req.Email)
	if existing != nil {
		http.Error(w, "Пользователь с таким email уже существует", http.StatusConflict)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		h.logger.Error("failed to hash password", zap.Error(err))
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	now := time.Now()
	user := &models.User{
		ID:           uuid.New(),
		Email:        req.Email,
		Name:         req.Name,
		PasswordHash: string(hash),
		Roles:        models.StringArray{"user"},
		IsActive:     true,
		LastLoginAt:  &now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := h.userRepo.Create(r.Context(), user); err != nil {
		h.logger.Error("failed to create user", zap.Error(err))
		http.Error(w, "Не удалось создать пользователя", http.StatusInternalServerError)
		return
	}

	token, err := h.issueToken(user)
	if err != nil {
		h.logger.Error("failed to generate token", zap.Error(err))
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_in":   int(h.tokenLifetime.Seconds()),
		"user": map[string]interface{}{
			"id":    user.ID.String(),
			"email": user.Email,
			"name":  user.Name,
			"roles": user.Roles,
		},
	})
}

func (h *LocalAuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректные данные запроса", http.StatusBadRequest)
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		http.Error(w, "Требуются email и пароль", http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetByEmail(r.Context(), req.Email)
	if err != nil {
		http.Error(w, "Неверный email или пароль", http.StatusUnauthorized)
		return
	}

	if !user.IsActive {
		http.Error(w, "Учётная запись деактивирована", http.StatusForbidden)
		return
	}

	if user.PasswordHash == "" {
		http.Error(w, "Эта учётная запись использует вход только через SSO", http.StatusBadRequest)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		http.Error(w, "Неверный email или пароль", http.StatusUnauthorized)
		return
	}

	_ = h.userRepo.UpdateLastLogin(r.Context(), user.ID)

	token, err := h.issueToken(user)
	if err != nil {
		h.logger.Error("failed to generate token", zap.Error(err))
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_in":   int(h.tokenLifetime.Seconds()),
		"user": map[string]interface{}{
			"id":    user.ID.String(),
			"email": user.Email,
			"name":  user.Name,
			"roles": user.Roles,
		},
	})
}

func (h *LocalAuthHandler) issueToken(user *models.User) (string, error) {
	sessionID, _ := generateSessionID()
	userInfo := &auth.UserInfo{
		Sub:   user.ID.String(),
		Email: user.Email,
		Name:  user.Name,
		Roles: []string(user.Roles),
	}
	return auth.GenerateSessionJWT(userInfo, sessionID, h.sessionSecret, h.tokenLifetime)
}
