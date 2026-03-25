package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/api/handlers"
	"github.com/qmish/focus-api/internal/auth"
	"github.com/qmish/focus-api/internal/bots"
	"github.com/qmish/focus-api/internal/config"
	"github.com/qmish/focus-api/internal/database"
	"github.com/qmish/focus-api/internal/exchange"
	"github.com/qmish/focus-api/internal/jitsi"
	"github.com/qmish/focus-api/internal/models"
	"github.com/qmish/focus-api/internal/repository"
	"github.com/qmish/focus-api/internal/webhooks"
	"github.com/qmish/focus-api/internal/websocket"
	"github.com/qmish/focus-api/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	// Загрузка конфигурации
	cfg := config.Load()

	// Инициализация логгера
	logger.Init(cfg.Log.Level, cfg.Log.Format)
	logger.Info("Starting Focus API server",
		zap.String("env", cfg.Env),
		zap.String("host", cfg.Server.Host),
		zap.String("port", cfg.Server.Port),
	)

	if err := cfg.ValidateSecurity(); err != nil {
		logger.Error("Security configuration validation failed", zap.Error(err))
		os.Exit(1)
	}

	// Подключение к базе данных
	db, err := database.NewDatabase(&cfg.Database)
	if err != nil {
		logger.Error("Failed to connect to database", zap.Error(err))
		os.Exit(1)
	}
	defer db.Close()

	// Проверка подключения
	if err := db.Ping(); err != nil {
		logger.Error("Database ping failed", zap.Error(err))
		os.Exit(1)
	}
	logger.Info("Connected to database")

	// Миграция моделей
	if err := db.AutoMigrate(
		&models.User{},
		&models.Room{},
		&models.RoomParticipant{},
		&models.Message{},
		&models.MessageReaction{},
		&models.AuthAuditEvent{},
		&models.RevokedSession{},
		&bots.BotCommandEvent{},
		&webhooks.IncomingEvent{},
	); err != nil {
		logger.Error("Database migration failed", zap.Error(err))
		os.Exit(1)
	}
	logger.Info("Database migrations completed")

	// Инициализация репозиториев
	userRepo := repository.NewUserRepository(db.DB)
	roomRepo := repository.NewRoomRepository(db.DB)
	messageRepo := repository.NewMessageRepository(db.DB)
	webhookRepo := repository.NewWebhookRepository(db.DB)
	botRepo := repository.NewBotRepository(db.DB)
	authAuditRepo := repository.NewAuthAuditRepository(db.DB)
	sessionRevocationRepo := repository.NewSessionRevocationRepository(db.DB)

	// Инициализация Graph API клиента (Exchange)
	var graphClient *exchange.GraphClient
	if cfg.Exchange.TenantID != "" && cfg.Exchange.ClientID != "" {
		graphClient, err = exchange.NewGraphClient(exchange.GraphConfig{
			TenantID:     cfg.Exchange.TenantID,
			ClientID:     cfg.Exchange.ClientID,
			ClientSecret: cfg.Exchange.ClientSecret,
		})
		if err != nil {
			logger.Warn("Failed to create Graph client, calendar features disabled", zap.Error(err))
		} else {
			logger.Info("Graph API client initialized")
		}
	}

	// Инициализация OIDC провайдера
	oidcProvider, err := auth.NewOIDCProvider(auth.OIDCConfig{
		IssuerURL:    fmt.Sprintf("%s/realms/%s", cfg.Keycloak.ServerURL, cfg.Keycloak.Realm),
		ClientID:     cfg.Keycloak.ClientID,
		ClientSecret: cfg.Keycloak.ClientSecret,
		RedirectURL:  cfg.Keycloak.RedirectURL,
		Scopes:       []string{"openid", "profile", "email", "roles"},
	})
	if err != nil {
		logger.Error("Failed to create OIDC provider", zap.Error(err))
		// Продолжаем без OIDC для разработки
	}

	// Инициализация Jitsi генератора токенов
	jitsiGen := jitsi.NewTokenGenerator(
		cfg.Jitsi.BaseURL,
		cfg.Jitsi.AppID,
		cfg.Jitsi.AppSecret,
		cfg.Jitsi.Issuer,
		cfg.Jitsi.Audience,
		cfg.Jitsi.TokenLifetime,
	)

	// Инициализация WebSocket Hub
	wsHub := websocket.NewHub(logger.WithContext(context.Background()))
	wsHub.SetRoomAccessChecker(func(ctx context.Context, userID, roomID string) (bool, error) {
		userUUID, err := uuid.Parse(userID)
		if err != nil {
			return false, nil
		}
		roomUUID, err := uuid.Parse(roomID)
		if err != nil {
			return false, nil
		}
		return roomRepo.IsParticipant(ctx, roomUUID, userUUID)
	})
	go wsHub.Run()

	botUserID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("focus-system-bot"))
	botEngine := bots.NewBotEngineWithDelivery(messageRepo, roomRepo, wsHub, botUserID)
	botEngine.SetRoomRepository(roomRepo)
	botEngine.SetJitsiBaseURL(cfg.Jitsi.BaseURL)
	botEngine.SetRateLimitWindow(2 * time.Second)
	botEngine.SetCommandEventStore(botRepo)
	botEngine.SetCalendarScheduler(&botMeetingScheduler{
		graphClient: graphClient,
		userRepo:    userRepo,
	})

	// Создание handlers
	authHandler := handlers.NewAuthHandler(oidcProvider, userRepo, jitsiGen, cfg, logger.WithContext(context.Background()))
	authHandler.SetAuthAuditRepository(authAuditRepo)
	authHandler.SetSessionRevocationRepository(sessionRevocationRepo)
	roomHandler := handlers.NewRoomHandler(roomRepo, userRepo, jitsiGen)
	messageHandler := handlers.NewMessageHandler(messageRepo, wsHub, botEngine)
	calendarHandler := handlers.NewCalendarHandler(graphClient, roomRepo, jitsiGen)
	adminHandler := handlers.NewAdminHandler(userRepo, roomRepo)
	adminHandler.SetMessageRepository(messageRepo)
	adminHandler.SetWebhookRepository(webhookRepo)
	adminHandler.SetBotRepository(botRepo)
	adminHandler.SetAuthAuditRepository(authAuditRepo)
	brandingHandler := handlers.NewJitsiBrandingHandler()
	// Warm in-memory revocation blacklist from persistent storage for API/WS checks.
	if revokedSessions, err := sessionRevocationRepo.ListActiveRevokedSessions(context.Background(), time.Now(), 100000); err == nil {
		for _, item := range revokedSessions {
			if item == nil {
				continue
			}
			auth.RevokeSession(item.SessionID, item.ExpiresAt)
		}
	}

	inboundWebhookHandler := handlers.NewInboundWebhookHandler(
		webhooks.NewWebhookHandlerWithConfig(cfg.Jitsi.AppSecret, webhookRepo),
	)

	// Создание auth middleware
	authMiddleware := auth.NewAuthMiddlewareWithPolicies(
		[]byte(cfg.Auth.SessionSecret),
		cfg.Auth.RequiredAudience,
		cfg.Auth.ServiceAudiences,
		cfg.Auth.ServiceScopes,
	)
	abacEngine := auth.NewDefaultABACEngine()

	// Создание роутера
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Health check endpoints
	r.Get("/health", healthCheck)
	r.Get("/ready", readinessCheck(db))

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public routes
		r.Route("/branding", func(r chi.Router) {
			r.Get("/jitsi", brandingHandler.DynamicBranding)
		})

		r.Route("/webhooks", func(r chi.Router) {
			r.Post("/jitsi", inboundWebhookHandler.JitsiWebhook)
		})

		r.Route("/auth", func(r chi.Router) {
			r.Get("/login", authHandler.Login)
			r.Get("/callback", authHandler.Callback)
			r.Post("/refresh", authHandler.Refresh)
			r.Post("/logout", authHandler.Logout)
		})

		// WebSocket endpoint
		r.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
			claims, err := websocket.AuthenticateRequest(r, []byte(cfg.Auth.SessionSecret))
			if err != nil {
				if errors.Is(err, websocket.ErrExpiredWebSocketToken) {
					http.Error(w, "token_expired", http.StatusUnauthorized)
					return
				}
				if errors.Is(err, websocket.ErrRevokedWebSocketToken) {
					http.Error(w, "session_revoked", http.StatusUnauthorized)
					return
				}
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			expiresAt := time.Time{}
			if claims.ExpiresAt != nil {
				expiresAt = claims.ExpiresAt.Time
			}
			wsHub.HandleWebSocket(w, r, claims.UserID, expiresAt)
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Middleware)

			// User info
			r.Get("/auth/me", authHandler.Me)

			// Rooms
			r.Route("/rooms", func(r chi.Router) {
				r.Get("/", roomHandler.ListRooms)
				r.Post("/", roomHandler.CreateRoom)
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", roomHandler.GetRoom)
					r.Put("/", roomHandler.UpdateRoom)
					r.Delete("/", roomHandler.DeleteRoom)
					r.Post("/join", roomHandler.JoinRoom)
				})
			})

			// Messages
			r.Route("/messages", func(r chi.Router) {
				r.Get("/", messageHandler.ListMessages)
				r.Post("/", messageHandler.CreateMessage)
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", messageHandler.GetMessage)
					r.Put("/", messageHandler.UpdateMessage)
					r.Delete("/", messageHandler.DeleteMessage)
				})
			})

			// Calendar
			if graphClient != nil {
				r.Route("/calendar", func(r chi.Router) {
					r.Use(auth.RequireAccess(auth.AccessRule{
						AnyRoles: []string{"user", "moderator", "admin", "service"},
						AnyScopes: []string{
							"focus.calendar",
							"exchange.calendar",
							"Calendars.ReadWrite",
						},
					}))
					r.Get("/events", calendarHandler.GetEvents)
					r.Post("/events", calendarHandler.CreateEvent)
					r.Route("/events/:id", func(r chi.Router) {
						r.Put("/", calendarHandler.UpdateEvent)
						r.Delete("/", calendarHandler.DeleteEvent)
					})
				})
			}

			// Admin routes
			r.Route("/admin", func(r chi.Router) {
				r.Use(auth.RequireAccess(auth.AccessRule{
					AnyRoles:  []string{"admin"},
					AnyScopes: []string{"focus.admin"},
				}))

				r.Get("/users", adminHandler.ListUsers)
				r.Get("/users/:id", adminHandler.GetUser)
				r.Put("/users/:id/roles", adminHandler.UpdateUserRoles)
				r.With(auth.RequireABAC(abacEngine, "user.ban", nil)).Post("/users/:id/ban", adminHandler.BanUser)
				r.With(auth.RequireABAC(abacEngine, "user.unban", nil)).Post("/users/:id/unban", adminHandler.UnbanUser)
				r.Get("/conferences", adminHandler.ListConferences)
				r.With(auth.RequireABAC(abacEngine, "conference.end", nil)).Post("/conferences/:id/end", adminHandler.EndConference)
				r.Get("/stats", adminHandler.GetStats)
				r.Get("/webhooks/deliveries", adminHandler.ListWebhookDeliveries)
				r.Get("/webhooks/errors", adminHandler.ListWebhookErrors)
				r.Get("/bots/errors", adminHandler.ListBotErrors)
				r.Get("/auth/audit", adminHandler.ListAuthAuditEvents)
			})
		})
	})

	// Создание HTTP сервера
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Запуск сервера в горутине
	go func() {
		logger.Info("Server starting", zap.String("address", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed", zap.Error(err))
		}
	}()

	// Ожидание сигнала завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("Server stopped")
}

// healthCheck обработчик health check
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}

// readinessCheck обработчик readiness check
func readinessCheck(db *database.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if err := db.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"not ready","database":"disconnected"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	}
}

type botMeetingScheduler struct {
	graphClient *exchange.GraphClient
	userRepo    *repository.UserRepository
}

func (s *botMeetingScheduler) ScheduleMeeting(
	ctx context.Context,
	userID uuid.UUID,
	title string,
	start, end time.Time,
	roomURL string,
) error {
	if s.graphClient == nil || s.userRepo == nil {
		return nil
	}
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	_, err = s.graphClient.CreateEvent(ctx, user.Email, exchange.CalendarEvent{
		Subject:     title,
		Description: "Создано через BotEngine /schedule",
		StartTime:   start,
		EndTime:     end,
		Location:    "Jitsi Meeting",
		JitsiURL:    roomURL,
		Organizer: exchange.EventAttendee{
			Email: user.Email,
			Name:  user.Name,
		},
	})
	return err
}
