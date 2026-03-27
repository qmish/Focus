package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/auth"
	"github.com/qmish/focus-api/internal/bots"
	"github.com/qmish/focus-api/internal/exchange"
	"github.com/qmish/focus-api/internal/models"
	"github.com/qmish/focus-api/internal/repository"
	"github.com/qmish/focus-api/internal/webhooks"
	"golang.org/x/crypto/bcrypt"
)

// AdminHandler обработчики для админ-панели
type AdminHandler struct {
	userRepo             adminUserRepository
	roomRepo             adminRoomRepository
	messageRepo          adminMessageRepository
	inviteRepo           adminInviteRepository
	inviteMailer         inviteMailer
	inviteBaseURL        string
	botSettingsRepo      adminBotSettingsRepository
	exchangeSettingsRepo adminExchangeSettingsRepository
	webhookRepo          adminWebhookRepository
	botRepo              adminBotRepository
	authAuditRepo        adminAuthAuditRepository
	calendarAuditRepo    adminCalendarAuditRepository
}

type adminUserRepository interface {
	Create(ctx context.Context, user *models.User) error
	List(ctx context.Context, limit, offset int) ([]*models.User, error)
	Count(ctx context.Context) (int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type adminRoomRepository interface {
	List(ctx context.Context, limit, offset int) ([]*models.Room, error)
	Count(ctx context.Context) (int64, error)
	CountByType(ctx context.Context, roomType models.RoomType) (int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Room, error)
	Delete(ctx context.Context, id uuid.UUID) error
	CountParticipants(ctx context.Context, roomID uuid.UUID) (int64, error)
}

type adminMessageRepository interface {
	CountSince(ctx context.Context, since time.Time) (int64, error)
}

type adminInviteRepository interface {
	Create(ctx context.Context, invite *models.AdminInvite) error
	List(ctx context.Context, limit, offset int) ([]*models.AdminInvite, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.AdminInvite, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (*models.AdminInvite, error)
	Update(ctx context.Context, invite *models.AdminInvite) error
	MarkSent(ctx context.Context, id uuid.UUID) error
}

type inviteMailer interface {
	SendInvite(email, inviteURL string) error
}

type adminBotSettingsRepository interface {
	List(ctx context.Context) ([]*models.BotSetting, error)
	Create(ctx context.Context, setting *models.BotSetting) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.BotSetting, error)
	Update(ctx context.Context, setting *models.BotSetting) error
}

type adminExchangeSettingsRepository interface {
	Get(ctx context.Context) (*models.ExchangeSetting, error)
	Upsert(ctx context.Context, settings *models.ExchangeSetting) error
}

type adminWebhookRepository interface {
	ListRecentDeliveries(ctx context.Context, limit int, onlyFailed bool) ([]*webhooks.WebhookDelivery, error)
}

type adminBotRepository interface {
	ListCommandEvents(ctx context.Context, limit int, onlyFailed bool) ([]*bots.BotCommandEvent, error)
}

type adminAuthAuditRepository interface {
	ListAuthAuditEvents(ctx context.Context, limit int, onlyFailed bool) ([]*models.AuthAuditEvent, error)
}

type adminCalendarAuditRepository interface {
	ListCalendarAuditEvents(ctx context.Context, limit int, onlyFailed bool) ([]*models.CalendarAuditEvent, error)
}

// NewAdminHandler создаёт новый AdminHandler
func NewAdminHandler(userRepo adminUserRepository, roomRepo adminRoomRepository) *AdminHandler {
	return &AdminHandler{
		userRepo: userRepo,
		roomRepo: roomRepo,
	}
}

// SetWebhookRepository sets optional webhook repository for admin visibility endpoints.
func (h *AdminHandler) SetWebhookRepository(webhookRepo adminWebhookRepository) {
	h.webhookRepo = webhookRepo
}

// SetBotRepository sets optional bot repository for admin bot error visibility.
func (h *AdminHandler) SetBotRepository(botRepo adminBotRepository) {
	h.botRepo = botRepo
}

// SetMessageRepository sets optional message repository for dashboard stats.
func (h *AdminHandler) SetMessageRepository(messageRepo adminMessageRepository) {
	h.messageRepo = messageRepo
}

// SetInviteRepository sets optional repository for admin invite flows.
func (h *AdminHandler) SetInviteRepository(inviteRepo adminInviteRepository) {
	h.inviteRepo = inviteRepo
}

// SetInviteMailer sets optional SMTP mailer and URL base for invite links.
func (h *AdminHandler) SetInviteMailer(mailer inviteMailer, inviteBaseURL string) {
	h.inviteMailer = mailer
	h.inviteBaseURL = strings.TrimSpace(inviteBaseURL)
}

// SetBotSettingsRepository sets optional repository for persistent bot settings.
func (h *AdminHandler) SetBotSettingsRepository(repo adminBotSettingsRepository) {
	h.botSettingsRepo = repo
}

// SetExchangeSettingsRepository sets repository for persisted Exchange settings.
func (h *AdminHandler) SetExchangeSettingsRepository(repo adminExchangeSettingsRepository) {
	h.exchangeSettingsRepo = repo
}

// SetAuthAuditRepository sets optional auth audit repository.
func (h *AdminHandler) SetAuthAuditRepository(authAuditRepo adminAuthAuditRepository) {
	h.authAuditRepo = authAuditRepo
}

// SetCalendarAuditRepository sets optional calendar audit repository.
func (h *AdminHandler) SetCalendarAuditRepository(calendarAuditRepo adminCalendarAuditRepository) {
	h.calendarAuditRepo = calendarAuditRepo
}

// requireAdmin middleware для проверки роли администратора
func requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := auth.GetUserClaimsFromContext(r.Context())
		if claims == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		isAdmin := false
		for _, role := range claims.Roles {
			if role == "admin" {
				isAdmin = true
				break
			}
		}

		if !isAdmin {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		next(w, r)
	}
}

// ListUsers GET /api/v1/admin/users
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	// Проверяем роль администратора
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Получаем пагинацию
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	offset := (page - 1) * perPage

	// Получаем пользователей
	users, err := h.userRepo.List(r.Context(), perPage, offset)
	if err != nil {
		http.Error(w, "failed to get users", http.StatusInternalServerError)
		return
	}

	// Получаем общее количество
	total, err := h.userRepo.Count(r.Context())
	if err != nil {
		http.Error(w, "failed to count users", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"data": users,
		"pagination": map[string]interface{}{
			"page":        page,
			"per_page":    perPage,
			"total":       total,
			"total_pages": (int(total) + perPage - 1) / perPage,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetUser GET /api/v1/admin/users/:id
func (h *AdminHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	// Проверяем роль администратора
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	id := chi.URLParam(r, "id")
	userID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		if err == repository.ErrUserNotFound {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// UpdateUserRoles PUT /api/v1/admin/users/:id/roles
func (h *AdminHandler) UpdateUserRoles(w http.ResponseWriter, r *http.Request) {
	// Проверяем роль администратора
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	id := chi.URLParam(r, "id")
	userID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	var req struct {
		Roles []string `json:"roles"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		if err == repository.ErrUserNotFound {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get user", http.StatusInternalServerError)
		return
	}

	user.Roles = req.Roles
	if err := h.userRepo.Update(r.Context(), user); err != nil {
		http.Error(w, "failed to update user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// CreateUser POST /api/v1/admin/users
func (h *AdminHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req struct {
		Email    string   `json:"email"`
		Name     string   `json:"name"`
		Password string   `json:"password"`
		Roles    []string `json:"roles"`
		IsActive *bool    `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Name = strings.TrimSpace(req.Name)
	if req.Email == "" || req.Name == "" {
		http.Error(w, "email and name are required", http.StatusBadRequest)
		return
	}
	user := &models.User{
		ID:       uuid.New(),
		Email:    req.Email,
		Name:     req.Name,
		Roles:    models.StringArray{"user"},
		IsActive: true,
	}
	if len(req.Roles) > 0 {
		user.Roles = req.Roles
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}
	if strings.TrimSpace(req.Password) != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "failed to hash password", http.StatusInternalServerError)
			return
		}
		user.PasswordHash = string(hash)
	}
	if err := h.userRepo.Create(r.Context(), user); err != nil {
		if err == repository.ErrUserAlreadyExists {
			http.Error(w, "user already exists", http.StatusConflict)
			return
		}
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(user)
}

// PatchUser PATCH /api/v1/admin/users/:id
func (h *AdminHandler) PatchUser(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	id := chi.URLParam(r, "id")
	userID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}
	var req struct {
		Email       *string  `json:"email"`
		Name        *string  `json:"name"`
		AvatarURL   *string  `json:"avatar_url"`
		IsActive    *bool    `json:"is_active"`
		BannedUntil **string `json:"banned_until"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		if err == repository.ErrUserNotFound {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get user", http.StatusInternalServerError)
		return
	}
	if req.Email != nil {
		user.Email = strings.TrimSpace(strings.ToLower(*req.Email))
	}
	if req.Name != nil {
		user.Name = strings.TrimSpace(*req.Name)
	}
	if req.AvatarURL != nil {
		user.AvatarURL = strings.TrimSpace(*req.AvatarURL)
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}
	if req.BannedUntil != nil {
		if *req.BannedUntil == nil || strings.TrimSpace(**req.BannedUntil) == "" {
			user.BannedUntil = nil
		} else {
			t, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(**req.BannedUntil))
			if parseErr != nil {
				http.Error(w, "invalid banned_until format", http.StatusBadRequest)
				return
			}
			tt := t.UTC()
			user.BannedUntil = &tt
		}
	}
	if err := h.userRepo.Update(r.Context(), user); err != nil {
		http.Error(w, "failed to update user", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(user)
}

// DeleteUser DELETE /api/v1/admin/users/:id
func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	id := chi.URLParam(r, "id")
	userID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}
	if err := h.userRepo.Delete(r.Context(), userID); err != nil {
		http.Error(w, "failed to delete user", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// CreateInvite POST /api/v1/admin/invites
func (h *AdminHandler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if h.inviteRepo == nil {
		http.Error(w, "invite repository is not configured", http.StatusServiceUnavailable)
		return
	}
	var req struct {
		Email          string   `json:"email"`
		Roles          []string `json:"roles"`
		ExpiresInHours int      `json:"expires_in_hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" {
		http.Error(w, "email is required", http.StatusBadRequest)
		return
	}
	if req.ExpiresInHours < 1 || req.ExpiresInHours > 720 {
		req.ExpiresInHours = 72
	}
	if len(req.Roles) == 0 {
		req.Roles = []string{"user"}
	}
	rawToken, tokenHash, err := generateInviteToken()
	if err != nil {
		http.Error(w, "failed to generate invite token", http.StatusInternalServerError)
		return
	}
	invite := &models.AdminInvite{
		ID:        uuid.New(),
		Email:     req.Email,
		TokenHash: tokenHash,
		Roles:     req.Roles,
		Status:    models.AdminInviteStatusPending,
		InvitedBy: claims.Email,
		ExpiresAt: time.Now().UTC().Add(time.Duration(req.ExpiresInHours) * time.Hour),
	}
	if err := h.inviteRepo.Create(r.Context(), invite); err != nil {
		http.Error(w, "failed to create invite", http.StatusInternalServerError)
		return
	}
	inviteURL := h.buildInviteURL(rawToken)
	mailSent := false
	if h.inviteMailer != nil {
		if err := h.inviteMailer.SendInvite(req.Email, inviteURL); err == nil {
			_ = h.inviteRepo.MarkSent(r.Context(), invite.ID)
			mailSent = true
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"invite":    invite,
		"inviteUrl": inviteURL,
		"mailSent":  mailSent,
	})
}

// ListInvites GET /api/v1/admin/invites
func (h *AdminHandler) ListInvites(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if h.inviteRepo == nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 200 {
		perPage = 20
	}
	offset := (page - 1) * perPage
	invites, err := h.inviteRepo.List(r.Context(), perPage, offset)
	if err != nil {
		http.Error(w, "failed to list invites", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data": invites,
		"pagination": map[string]interface{}{
			"page":     page,
			"per_page": perPage,
		},
	})
}

// ResendInvite POST /api/v1/admin/invites/:id/resend
func (h *AdminHandler) ResendInvite(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if h.inviteRepo == nil {
		http.Error(w, "invite repository is not configured", http.StatusServiceUnavailable)
		return
	}
	id := chi.URLParam(r, "id")
	inviteID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "invalid invite id", http.StatusBadRequest)
		return
	}
	invite, err := h.inviteRepo.GetByID(r.Context(), inviteID)
	if err != nil {
		if err == repository.ErrAdminInviteNotFound {
			http.Error(w, "invite not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get invite", http.StatusInternalServerError)
		return
	}
	rawToken, tokenHash, err := generateInviteToken()
	if err != nil {
		http.Error(w, "failed to generate invite token", http.StatusInternalServerError)
		return
	}
	invite.TokenHash = tokenHash
	invite.Status = models.AdminInviteStatusPending
	invite.ExpiresAt = time.Now().UTC().Add(72 * time.Hour)
	if err := h.inviteRepo.Update(r.Context(), invite); err != nil {
		http.Error(w, "failed to update invite", http.StatusInternalServerError)
		return
	}
	inviteURL := h.buildInviteURL(rawToken)
	mailSent := false
	if h.inviteMailer != nil {
		if err := h.inviteMailer.SendInvite(invite.Email, inviteURL); err == nil {
			_ = h.inviteRepo.MarkSent(r.Context(), invite.ID)
			mailSent = true
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"invite":    invite,
		"inviteUrl": inviteURL,
		"mailSent":  mailSent,
	})
}

// AcceptInvite POST /api/v1/admin/invites/accept
func (h *AdminHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	if h.inviteRepo == nil {
		http.Error(w, "invite repository is not configured", http.StatusServiceUnavailable)
		return
	}
	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.Token = strings.TrimSpace(req.Token)
	if req.Token == "" {
		http.Error(w, "token is required", http.StatusBadRequest)
		return
	}
	hash := sha256.Sum256([]byte(req.Token))
	tokenHash := fmt.Sprintf("%x", hash[:])
	invite, err := h.inviteRepo.GetByTokenHash(r.Context(), tokenHash)
	if err != nil {
		if err == repository.ErrAdminInviteNotFound {
			http.Error(w, "invite not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get invite", http.StatusInternalServerError)
		return
	}
	if invite.ExpiresAt.Before(time.Now().UTC()) {
		invite.Status = models.AdminInviteStatusExpired
		_ = h.inviteRepo.Update(r.Context(), invite)
		http.Error(w, "invite expired", http.StatusGone)
		return
	}
	now := time.Now().UTC()
	invite.Status = models.AdminInviteStatusAccepted
	invite.AcceptedAt = &now
	if err := h.inviteRepo.Update(r.Context(), invite); err != nil {
		http.Error(w, "failed to accept invite", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"accepted": true,
		"email":    invite.Email,
		"roles":    invite.Roles,
	})
}

// BanUser POST /api/v1/admin/users/:id/ban
func (h *AdminHandler) BanUser(w http.ResponseWriter, r *http.Request) {
	// Проверяем роль администратора
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	id := chi.URLParam(r, "id")
	userID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	var req struct {
		Reason        string `json:"reason"`
		DurationHours int    `json:"duration_hours"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.DurationHours < 0 {
		http.Error(w, "duration_hours must be non-negative", http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		if err == repository.ErrUserNotFound {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get user", http.StatusInternalServerError)
		return
	}

	user.IsActive = false
	var bannedUntil *time.Time
	if req.DurationHours > 0 {
		until := time.Now().UTC().Add(time.Duration(req.DurationHours) * time.Hour)
		bannedUntil = &until
	}
	user.BannedUntil = bannedUntil
	if err := h.userRepo.Update(r.Context(), user); err != nil {
		http.Error(w, "failed to ban user", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"id":           user.ID.String(),
		"banned":       true,
		"reason":       req.Reason,
		"banned_until": bannedUntil,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UnbanUser POST /api/v1/admin/users/:id/unban
func (h *AdminHandler) UnbanUser(w http.ResponseWriter, r *http.Request) {
	// Проверяем роль администратора
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	id := chi.URLParam(r, "id")
	userID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		if err == repository.ErrUserNotFound {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get user", http.StatusInternalServerError)
		return
	}

	user.IsActive = true
	user.BannedUntil = nil
	if err := h.userRepo.Update(r.Context(), user); err != nil {
		http.Error(w, "failed to unban user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     user.ID.String(),
		"banned": false,
	})
}

// ListConferences GET /api/v1/admin/conferences
func (h *AdminHandler) ListConferences(w http.ResponseWriter, r *http.Request) {
	// Проверяем роль администратора
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Получаем пагинацию
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	if h.roomRepo == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
		return
	}

	rooms, err := h.roomRepo.List(r.Context(), perPage, offset)
	if err != nil {
		http.Error(w, "failed to list conferences", http.StatusInternalServerError)
		return
	}

	type conferenceInfo struct {
		ID                string    `json:"id"`
		RoomID            string    `json:"room_id"`
		RoomName          string    `json:"room_name"`
		JitsiRoom         string    `json:"jitsi_room"`
		ParticipantsCount int64     `json:"participants_count"`
		StartedAt         time.Time `json:"started_at"`
		LastActivityAt    time.Time `json:"last_activity_at"`
		Status            string    `json:"status"`
	}

	conferences := make([]conferenceInfo, 0, len(rooms))
	for _, room := range rooms {
		if room == nil || room.Type != models.RoomTypeMeeting {
			continue
		}
		participantsCount, err := h.roomRepo.CountParticipants(r.Context(), room.ID)
		if err != nil {
			participantsCount = 0
		}

		conferences = append(conferences, conferenceInfo{
			ID:                room.ID.String(),
			RoomID:            room.ID.String(),
			RoomName:          room.Name,
			JitsiRoom:         room.JitsiRoomName,
			ParticipantsCount: participantsCount,
			StartedAt:         room.CreatedAt,
			LastActivityAt:    room.UpdatedAt,
			Status:            "active",
		})
	}

	response := map[string]interface{}{
		"data": conferences,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// EndConference POST /api/v1/admin/conferences/:id/end
func (h *AdminHandler) EndConference(w http.ResponseWriter, r *http.Request) {
	// Проверяем роль администратора
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "conference id is required", http.StatusBadRequest)
		return
	}
	roomID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "invalid conference id", http.StatusBadRequest)
		return
	}

	if h.roomRepo == nil {
		http.Error(w, "conference service unavailable", http.StatusServiceUnavailable)
		return
	}

	room, err := h.roomRepo.GetByID(r.Context(), roomID)
	if err != nil {
		if err == repository.ErrRoomNotFound {
			http.Error(w, "conference not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get conference", http.StatusInternalServerError)
		return
	}
	if room.Type != models.RoomTypeMeeting {
		http.Error(w, "room is not a conference", http.StatusBadRequest)
		return
	}

	if err := h.roomRepo.Delete(r.Context(), roomID); err != nil {
		http.Error(w, "failed to end conference", http.StatusInternalServerError)
		return
	}

	endedAt := time.Now().UTC()
	response := map[string]interface{}{
		"id":       roomID.String(),
		"room_id":  roomID.String(),
		"ended":    true,
		"ended_at": endedAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetStats GET /api/v1/admin/stats
func (h *AdminHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	// Проверяем роль администратора
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	ctx := r.Context()

	// Получаем статистику
	var userCount, roomCount, activeConferences, messagesToday int64
	if h.userRepo != nil {
		userCount, _ = h.userRepo.Count(ctx)
	}
	if h.roomRepo != nil {
		roomCount, _ = h.roomRepo.Count(ctx)
		activeConferences, _ = h.roomRepo.CountByType(ctx, models.RoomTypeMeeting)
	}
	if h.messageRepo != nil {
		now := time.Now()
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		messagesToday, _ = h.messageRepo.CountSince(ctx, startOfDay)
	}

	response := map[string]interface{}{
		"users": map[string]interface{}{
			"total": userCount,
		},
		"rooms": map[string]interface{}{
			"total": roomCount,
		},
		"conferences": map[string]interface{}{
			"active": activeConferences,
		},
		"messages": map[string]interface{}{
			"today": messagesToday,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetExchangeSettings GET /api/v1/admin/exchange/settings
func (h *AdminHandler) GetExchangeSettings(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if h.exchangeSettingsRepo == nil {
		http.Error(w, "exchange settings repository is not configured", http.StatusServiceUnavailable)
		return
	}
	settings, err := h.exchangeSettingsRepo.Get(r.Context())
	if err != nil {
		if err == repository.ErrExchangeSettingNotFound {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"configured": false})
			return
		}
		http.Error(w, "failed to get exchange settings", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"configured": true,
		"settings": map[string]interface{}{
			"id":                     settings.ID,
			"ews_url":                settings.EWSURL,
			"username":               settings.Username,
			"domain":                 settings.Domain,
			"auth_mode":              settings.AuthMode,
			"ca_cert_path":           settings.CACertPath,
			"insecure_tls":           settings.InsecureTLS,
			"krb5_config_path":       settings.Krb5ConfigPath,
			"krb5_keytab_path":       settings.Krb5KeytabPath,
			"krb5_realm":             settings.Krb5Realm,
			"krb5_spn":               settings.Krb5SPN,
			"impersonation":          settings.Impersonation,
			"timeout_seconds":        settings.TimeoutSeconds,
			"sync_enabled":           settings.SyncEnabled,
			"sync_interval_seconds":  settings.SyncIntervalS,
			"sync_lookback_seconds":  settings.SyncLookbackS,
			"sync_lookahead_seconds": settings.SyncLookaheadS,
			"updated_by":             settings.UpdatedBy,
			"updated_at":             settings.UpdatedAt,
			"password_set":           strings.TrimSpace(settings.Password) != "",
		},
	})
}

// PutExchangeSettings PUT /api/v1/admin/exchange/settings
func (h *AdminHandler) PutExchangeSettings(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if h.exchangeSettingsRepo == nil {
		http.Error(w, "exchange settings repository is not configured", http.StatusServiceUnavailable)
		return
	}
	var req struct {
		EWSURL         string `json:"ews_url"`
		Username       string `json:"username"`
		Password       string `json:"password"`
		Domain         string `json:"domain"`
		AuthMode       string `json:"auth_mode"`
		CACertPath     string `json:"ca_cert_path"`
		InsecureTLS    bool   `json:"insecure_tls"`
		Krb5ConfigPath string `json:"krb5_config_path"`
		Krb5KeytabPath string `json:"krb5_keytab_path"`
		Krb5Realm      string `json:"krb5_realm"`
		Krb5SPN        string `json:"krb5_spn"`
		Impersonation  bool   `json:"impersonation"`
		TimeoutSeconds int    `json:"timeout_seconds"`
		SyncEnabled    bool   `json:"sync_enabled"`
		SyncIntervalS  int    `json:"sync_interval_seconds"`
		SyncLookbackS  int    `json:"sync_lookback_seconds"`
		SyncLookaheadS int    `json:"sync_lookahead_seconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.EWSURL) == "" || strings.TrimSpace(req.Username) == "" {
		http.Error(w, "ews_url and username are required", http.StatusBadRequest)
		return
	}
	existing, err := h.exchangeSettingsRepo.Get(r.Context())
	if err != nil && err != repository.ErrExchangeSettingNotFound {
		http.Error(w, "failed to read exchange settings", http.StatusInternalServerError)
		return
	}
	settings := &models.ExchangeSetting{ID: "default"}
	if existing != nil {
		settings = existing
	}
	settings.EWSURL = strings.TrimSpace(req.EWSURL)
	settings.Username = strings.TrimSpace(req.Username)
	if strings.TrimSpace(req.Password) != "" {
		settings.Password = req.Password
	}
	settings.Domain = strings.TrimSpace(req.Domain)
	settings.AuthMode = strings.ToLower(strings.TrimSpace(req.AuthMode))
	if settings.AuthMode == "" {
		settings.AuthMode = "basic"
	}
	settings.CACertPath = strings.TrimSpace(req.CACertPath)
	settings.InsecureTLS = req.InsecureTLS
	settings.Krb5ConfigPath = strings.TrimSpace(req.Krb5ConfigPath)
	settings.Krb5KeytabPath = strings.TrimSpace(req.Krb5KeytabPath)
	settings.Krb5Realm = strings.TrimSpace(req.Krb5Realm)
	settings.Krb5SPN = strings.TrimSpace(req.Krb5SPN)
	settings.Impersonation = req.Impersonation
	settings.TimeoutSeconds = req.TimeoutSeconds
	if settings.TimeoutSeconds <= 0 {
		settings.TimeoutSeconds = 15
	}
	settings.SyncEnabled = req.SyncEnabled
	settings.SyncIntervalS = req.SyncIntervalS
	if settings.SyncIntervalS <= 0 {
		settings.SyncIntervalS = 120
	}
	settings.SyncLookbackS = req.SyncLookbackS
	if settings.SyncLookbackS <= 0 {
		settings.SyncLookbackS = 43200
	}
	settings.SyncLookaheadS = req.SyncLookaheadS
	if settings.SyncLookaheadS <= 0 {
		settings.SyncLookaheadS = 1209600
	}
	settings.UpdatedBy = claims.Email
	if err := h.exchangeSettingsRepo.Upsert(r.Context(), settings); err != nil {
		http.Error(w, "failed to update exchange settings", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"updated": true})
}

// TestExchangeConnection POST /api/v1/admin/exchange/test-connection
func (h *AdminHandler) TestExchangeConnection(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if h.exchangeSettingsRepo == nil {
		http.Error(w, "exchange settings repository is not configured", http.StatusServiceUnavailable)
		return
	}
	var req struct {
		TestEmail string `json:"test_email"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	settings, err := h.exchangeSettingsRepo.Get(r.Context())
	if err != nil {
		http.Error(w, "exchange settings not found", http.StatusNotFound)
		return
	}
	client, err := exchange.NewEWSClient(exchange.EWSConfig{
		URL:            settings.EWSURL,
		Username:       settings.Username,
		Password:       settings.Password,
		Domain:         settings.Domain,
		AuthMode:       settings.AuthMode,
		CACertPath:     settings.CACertPath,
		InsecureTLS:    settings.InsecureTLS,
		Krb5ConfigPath: settings.Krb5ConfigPath,
		Krb5KeytabPath: settings.Krb5KeytabPath,
		Krb5Realm:      settings.Krb5Realm,
		Krb5SPN:        settings.Krb5SPN,
		Impersonation:  settings.Impersonation,
		Timeout:        time.Duration(settings.TimeoutSeconds) * time.Second,
	})
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": err.Error()})
		return
	}
	email := strings.TrimSpace(req.TestEmail)
	if email == "" {
		email = settings.Username
	}
	start := time.Now().UTC().Add(-1 * time.Hour)
	end := time.Now().UTC().Add(1 * time.Hour)
	_, err = client.GetEvents(r.Context(), email, start, end)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
}

// ListBots GET /api/v1/admin/bots
func (h *AdminHandler) ListBots(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if h.botSettingsRepo == nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
		return
	}
	settings, err := h.botSettingsRepo.List(r.Context())
	if err != nil {
		http.Error(w, "failed to list bots", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": settings})
}

// CreateBot POST /api/v1/admin/bots
func (h *AdminHandler) CreateBot(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if h.botSettingsRepo == nil {
		http.Error(w, "bot settings repository is not configured", http.StatusServiceUnavailable)
		return
	}
	var req struct {
		Name         string   `json:"name"`
		Description  string   `json:"description"`
		IsEnabled    *bool    `json:"is_enabled"`
		RateLimitMs  int      `json:"rate_limit_ms"`
		AllowedRooms []string `json:"allowed_rooms"`
		CommandsJSON string   `json:"commands_json"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if req.RateLimitMs <= 0 {
		req.RateLimitMs = 2000
	}
	if strings.TrimSpace(req.CommandsJSON) == "" {
		req.CommandsJSON = "[]"
	}
	setting := &models.BotSetting{
		ID:           uuid.New(),
		Name:         req.Name,
		Description:  strings.TrimSpace(req.Description),
		IsEnabled:    true,
		RateLimitMs:  req.RateLimitMs,
		AllowedRooms: req.AllowedRooms,
		CommandsJSON: req.CommandsJSON,
	}
	if req.IsEnabled != nil {
		setting.IsEnabled = *req.IsEnabled
	}
	if err := h.botSettingsRepo.Create(r.Context(), setting); err != nil {
		http.Error(w, "failed to create bot", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(setting)
}

// PatchBot PATCH /api/v1/admin/bots/:id
func (h *AdminHandler) PatchBot(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if h.botSettingsRepo == nil {
		http.Error(w, "bot settings repository is not configured", http.StatusServiceUnavailable)
		return
	}
	id := chi.URLParam(r, "id")
	botID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "invalid bot id", http.StatusBadRequest)
		return
	}
	var req struct {
		Name         *string   `json:"name"`
		Description  *string   `json:"description"`
		RateLimitMs  *int      `json:"rate_limit_ms"`
		AllowedRooms *[]string `json:"allowed_rooms"`
		CommandsJSON *string   `json:"commands_json"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	setting, err := h.botSettingsRepo.GetByID(r.Context(), botID)
	if err != nil {
		if err == repository.ErrBotSettingNotFound {
			http.Error(w, "bot not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get bot", http.StatusInternalServerError)
		return
	}
	if req.Name != nil {
		setting.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		setting.Description = strings.TrimSpace(*req.Description)
	}
	if req.RateLimitMs != nil && *req.RateLimitMs > 0 {
		setting.RateLimitMs = *req.RateLimitMs
	}
	if req.AllowedRooms != nil {
		setting.AllowedRooms = *req.AllowedRooms
	}
	if req.CommandsJSON != nil {
		setting.CommandsJSON = strings.TrimSpace(*req.CommandsJSON)
		if setting.CommandsJSON == "" {
			setting.CommandsJSON = "[]"
		}
	}
	if err := h.botSettingsRepo.Update(r.Context(), setting); err != nil {
		http.Error(w, "failed to update bot", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(setting)
}

// SetBotEnabled POST /api/v1/admin/bots/:id/enable|disable
func (h *AdminHandler) SetBotEnabled(w http.ResponseWriter, r *http.Request, enabled bool) {
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if h.botSettingsRepo == nil {
		http.Error(w, "bot settings repository is not configured", http.StatusServiceUnavailable)
		return
	}
	id := chi.URLParam(r, "id")
	botID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "invalid bot id", http.StatusBadRequest)
		return
	}
	setting, err := h.botSettingsRepo.GetByID(r.Context(), botID)
	if err != nil {
		if err == repository.ErrBotSettingNotFound {
			http.Error(w, "bot not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get bot", http.StatusInternalServerError)
		return
	}
	setting.IsEnabled = enabled
	if err := h.botSettingsRepo.Update(r.Context(), setting); err != nil {
		http.Error(w, "failed to update bot", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         setting.ID.String(),
		"is_enabled": setting.IsEnabled,
	})
}

func (h *AdminHandler) EnableBot(w http.ResponseWriter, r *http.Request) {
	h.SetBotEnabled(w, r, true)
}

func (h *AdminHandler) DisableBot(w http.ResponseWriter, r *http.Request) {
	h.SetBotEnabled(w, r, false)
}

// ListWebhookDeliveries GET /api/v1/admin/webhooks/deliveries
func (h *AdminHandler) ListWebhookDeliveries(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 200 {
		limit = 50
	}

	if h.webhookRepo == nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
		return
	}

	deliveries, err := h.webhookRepo.ListRecentDeliveries(r.Context(), limit, false)
	if err != nil {
		http.Error(w, "failed to list webhook deliveries", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": deliveries})
}

// ListWebhookErrors GET /api/v1/admin/webhooks/errors
func (h *AdminHandler) ListWebhookErrors(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 200 {
		limit = 50
	}

	if h.webhookRepo == nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data":  []interface{}{},
			"total": 0,
		})
		return
	}

	deliveries, err := h.webhookRepo.ListRecentDeliveries(r.Context(), limit, true)
	if err != nil {
		http.Error(w, "failed to list webhook errors", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  deliveries,
		"total": len(deliveries),
	})
}

// ListBotErrors GET /api/v1/admin/bots/errors
func (h *AdminHandler) ListBotErrors(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 200 {
		limit = 50
	}

	if h.botRepo == nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data":  []interface{}{},
			"total": 0,
		})
		return
	}

	events, err := h.botRepo.ListCommandEvents(r.Context(), limit, true)
	if err != nil {
		http.Error(w, "failed to list bot errors", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  events,
		"total": len(events),
	})
}

// ListAuthAuditEvents GET /api/v1/admin/auth/audit
func (h *AdminHandler) ListAuthAuditEvents(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 500 {
		limit = 100
	}
	onlyFailed := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("failed")), "true")

	if h.authAuditRepo == nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data":  []interface{}{},
			"total": 0,
		})
		return
	}

	events, err := h.authAuditRepo.ListAuthAuditEvents(r.Context(), limit, onlyFailed)
	if err != nil {
		http.Error(w, "failed to list auth audit events", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  events,
		"total": len(events),
	})
}

// ListCalendarAuditEvents GET /api/v1/admin/calendar/audit
func (h *AdminHandler) ListCalendarAuditEvents(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 500 {
		limit = 100
	}
	onlyFailed := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("failed")), "true")
	if h.calendarAuditRepo == nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data":  []interface{}{},
			"total": 0,
		})
		return
	}
	events, err := h.calendarAuditRepo.ListCalendarAuditEvents(r.Context(), limit, onlyFailed)
	if err != nil {
		http.Error(w, "failed to list calendar audit events", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  events,
		"total": len(events),
	})
}

func (h *AdminHandler) buildInviteURL(rawToken string) string {
	base := strings.TrimSpace(h.inviteBaseURL)
	if base == "" {
		base = "https://admin.focus.local:30443/invite/accept"
	}
	if strings.Contains(base, "?") {
		return base + "&token=" + rawToken
	}
	return base + "?token=" + rawToken
}

func generateInviteToken() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	return token, fmt.Sprintf("%x", hash[:]), nil
}

// hasRole проверяет наличие роли у пользователя
func hasRole(claims *auth.SessionClaims, role string) bool {
	if claims == nil {
		return false
	}
	for _, r := range claims.Roles {
		if r == role {
			return true
		}
	}
	// Admin endpoints use middleware with role-or-scope semantics.
	// Keep handler-level checks consistent to avoid false 403 for scope-based admins.
	if role == "admin" {
		for _, scope := range claims.AllScopes() {
			if scope == "focus.admin" {
				return true
			}
		}
	}
	return false
}
