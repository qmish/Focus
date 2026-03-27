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
	"github.com/qmish/focus-api/internal/notifications"
	"github.com/qmish/focus-api/internal/repository"
	"github.com/qmish/focus-api/internal/webhooks"
	"github.com/qmish/focus-api/internal/websocket"
	"github.com/qmish/focus-api/pkg/logger"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"go.uber.org/zap"
)

func main() {
	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()

	// Загрузка конфигурации
	cfg := config.Load()

	// Инициализация логгера
	logger.Init(cfg.Log.Level, cfg.Log.Format)
	logger.Info("Starting Focus API server",
		zap.String("env", cfg.Env),
		zap.String("host", cfg.Server.Host),
		zap.String("port", cfg.Server.Port),
		zap.String("swagger_port", cfg.Server.SwaggerPort),
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
		&models.CalendarAuditEvent{},
		&models.MeetingLink{},
		&models.CalendarIdempotencyKey{},
		&models.AdminInvite{},
		&models.BotSetting{},
		&models.ExchangeSetting{},
		&models.AppSetting{},
		&models.AuditLog{},
		&models.ConferencePolicy{},
		&models.RevokedSession{},
		&bots.BotCommandEvent{},
		&webhooks.IncomingEvent{},
		&webhooks.WebhookDelivery{},
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
	calendarAuditRepo := repository.NewCalendarAuditRepository(db.DB)
	meetingLinkRepo := repository.NewMeetingLinkRepository(db.DB)
	calendarIdempotencyRepo := repository.NewCalendarIdempotencyRepository(db.DB)
	adminInviteRepo := repository.NewAdminInviteRepository(db.DB)
	botSettingsRepo := repository.NewBotSettingsRepository(db.DB)
	exchangeSettingsRepo := repository.NewExchangeSettingsRepository(db.DB)
	appSettingsRepo := repository.NewAppSettingsRepository(db.DB)
	auditLogRepo := repository.NewAuditLogRepository(db.DB)
	conferencePolicyRepo := repository.NewConferencePolicyRepository(db.DB)
	sessionRevocationRepo := repository.NewSessionRevocationRepository(db.DB)

	// Инициализация Exchange EWS клиента (on-prem)
	var calendarService exchange.CalendarService
	if cfg.Exchange.Provider == "ews" && cfg.Exchange.EWSURL != "" {
		calendarService, err = exchange.NewEWSClient(exchange.EWSConfig{
			URL:            cfg.Exchange.EWSURL,
			Username:       cfg.Exchange.Username,
			Password:       cfg.Exchange.Password,
			Domain:         cfg.Exchange.Domain,
			AuthMode:       cfg.Exchange.AuthMode,
			CACertPath:     cfg.Exchange.CACertPath,
			InsecureTLS:    cfg.Exchange.InsecureTLS,
			Krb5ConfigPath: cfg.Exchange.Krb5ConfigPath,
			Krb5KeytabPath: cfg.Exchange.Krb5KeytabPath,
			Krb5Realm:      cfg.Exchange.Krb5Realm,
			Krb5SPN:        cfg.Exchange.Krb5SPN,
			Impersonation:  cfg.Exchange.Impersonation,
			Timeout:        cfg.Exchange.Timeout,
		})
		if err != nil {
			logger.Warn("Failed to create Exchange EWS client, calendar features disabled", zap.Error(err))
		} else {
			logger.Info("Exchange EWS client initialized")
		}
	}

	// Инициализация OIDC провайдера
	oidcCfg := auth.OIDCConfig{
		IssuerURL:    fmt.Sprintf("%s/realms/%s", cfg.Keycloak.ServerURL, cfg.Keycloak.Realm),
		ClientID:     cfg.Keycloak.ClientID,
		ClientSecret: cfg.Keycloak.ClientSecret,
		RedirectURL:  cfg.Keycloak.RedirectURL,
		Scopes:       []string{"openid", "profile", "email", "roles"},
	}
	if cfg.Keycloak.InternalURL != "" {
		oidcCfg.DiscoveryURL = fmt.Sprintf("%s/realms/%s", cfg.Keycloak.InternalURL, cfg.Keycloak.Realm)
	}
	oidcProvider, err := auth.NewOIDCProvider(oidcCfg)
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
	wsHub.SetAllowedOrigins(cfg.WebSocket.AllowedOrigins)
	wsHub.SetStrictRoomAccess(cfg.WebSocket.StrictRoomAccess)
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
	ensureBotUser(db, botUserID)
	botEngine := bots.NewBotEngineWithDelivery(messageRepo, roomRepo, wsHub, botUserID)
	botEngine.SetRoomRepository(roomRepo)
	botEngine.SetUserRepository(userRepo)
	botEngine.SetJitsiBaseURL(cfg.Jitsi.BaseURL)
	botEngine.SetRateLimitWindow(2 * time.Second)
	botEngine.SetCommandEventStore(botRepo)
	botEngine.SetCalendarScheduler(&botMeetingScheduler{
		calendarService: calendarService,
		userRepo:        userRepo,
	})
	botEngine.SetBotSettingsProvider(botSettingsRepo)
	if err := botEngine.ReloadSettings(context.Background()); err != nil {
		logger.Warn("Failed to load bot settings from DB", zap.Error(err))
	}

	// Создание handlers
	authHandler := handlers.NewAuthHandler(oidcProvider, userRepo, jitsiGen, cfg, logger.WithContext(context.Background()))
	authHandler.SetAuthAuditRepository(authAuditRepo)
	authHandler.SetSessionRevocationRepository(sessionRevocationRepo)
	roomHandler := handlers.NewRoomHandler(roomRepo, userRepo, jitsiGen)
	messageHandler := handlers.NewMessageHandler(messageRepo, wsHub, botEngine)
	fileHandler := handlers.NewFileHandler("/data/uploads")
	calendarHandler := handlers.NewCalendarHandler(calendarService, roomRepo, jitsiGen)
	calendarHandler.SetCalendarAuditRepository(calendarAuditRepo)
	calendarHandler.SetMeetingLinkRepository(meetingLinkRepo)
	calendarHandler.SetCalendarIdempotencyRepository(calendarIdempotencyRepo)
	adminHandler := handlers.NewAdminHandler(userRepo, roomRepo)
	adminHandler.SetMessageRepository(messageRepo)
	adminHandler.SetInviteRepository(adminInviteRepo)
	adminHandler.SetBotSettingsRepository(botSettingsRepo)
	adminHandler.SetExchangeSettingsRepository(exchangeSettingsRepo)
	adminHandler.SetWebhookRepository(webhookRepo)
	adminHandler.SetBotRepository(botRepo)
	adminHandler.SetBotConfigReloader(botEngine)
	adminHandler.SetAppSettingsRepository(appSettingsRepo)
	adminHandler.SetAuditLogRepository(auditLogRepo)
	adminHandler.SetConferencePolicyRepository(conferencePolicyRepo)
	adminHandler.SetAuthAuditRepository(authAuditRepo)
	adminHandler.SetCalendarAuditRepository(calendarAuditRepo)
	if cfg.Email.SMTPHost != "" && cfg.Email.FromAddress != "" {
		adminHandler.SetInviteMailer(
			notifications.NewSMTPInviteMailer(
				cfg.Email.SMTPHost,
				cfg.Email.SMTPPort,
				cfg.Email.SMTPUser,
				cfg.Email.SMTPPassword,
				cfg.Email.FromAddress,
			),
			cfg.Email.InviteBaseURL,
		)
	} else {
		adminHandler.SetInviteMailer(nil, cfg.Email.InviteBaseURL)
	}
	brandingHandler := handlers.NewJitsiBrandingHandler()
	brandingHandler.SetAppSettingsGetter(appSettingsRepo)
	localAuthHandler := handlers.NewLocalAuthHandler(
		userRepo,
		resolveSessionSecret(cfg),
		resolveSessionLifetimeDuration(cfg),
		logger.WithContext(context.Background()),
	)
	// Warm in-memory revocation blacklist from persistent storage for API/WS checks.
	if revokedSessions, err := sessionRevocationRepo.ListActiveRevokedSessions(context.Background(), time.Now(), 100000); err == nil {
		for _, item := range revokedSessions {
			if item == nil {
				continue
			}
			auth.RevokeSession(item.SessionID, item.ExpiresAt)
		}
	}

	inboundWebhookService := webhooks.NewWebhookHandlerWithConfig(cfg.Jitsi.AppSecret, webhookRepo)
	inboundWebhookService.SetRoomLifecycleRepository(roomRepo)
	inboundWebhookHandler := handlers.NewInboundWebhookHandler(inboundWebhookService)

	if cfg.Exchange.SyncEnabled && calendarService != nil {
		syncWorker := exchange.NewSyncWorker(
			calendarService,
			userRepo,
			roomRepo,
			meetingLinkRepo,
			cfg.Exchange.SyncInterval,
			cfg.Exchange.SyncLookback,
			cfg.Exchange.SyncLookahead,
		)
		go syncWorker.Start(appCtx)
		logger.Info(
			"Exchange sync worker started",
			zap.Duration("interval", cfg.Exchange.SyncInterval),
			zap.Duration("lookback", cfg.Exchange.SyncLookback),
			zap.Duration("lookahead", cfg.Exchange.SyncLookahead),
		)
	}

	// Создание auth middleware
	authMiddleware := auth.NewAuthMiddlewareWithPolicies(
		[]byte(cfg.Auth.SessionSecret),
		cfg.Auth.RequiredAudience,
		cfg.Auth.ServiceAudiences,
		cfg.Auth.ServiceScopes,
	)
	authMiddleware.SetValidationSecrets(cfg.Auth.SessionValidationSecrets)
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
		r.Get("/settings/appearance", adminHandler.GetAppearanceSettings)

		r.Route("/webhooks", func(r chi.Router) {
			r.Post("/jitsi", inboundWebhookHandler.JitsiWebhook)
		})

		r.Route("/auth", func(r chi.Router) {
			r.Get("/login", authHandler.Login)
			r.Get("/callback", authHandler.Callback)
			r.Post("/refresh", authHandler.Refresh)
			r.Post("/logout", authHandler.Logout)
			r.Post("/token-exchange", authHandler.TokenExchange)
			r.Post("/local/register", localAuthHandler.Register)
			r.Post("/local/login", localAuthHandler.Login)
		})
		r.Route("/invites", func(r chi.Router) {
			r.Post("/accept", adminHandler.AcceptInvite)
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

			// Files
			r.Post("/files/upload", fileHandler.Upload)
			r.Get("/files/{fileId}", fileHandler.Download)

			// Calendar
			if calendarService != nil {
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
				r.Post("/users", adminHandler.CreateUser)
				r.Get("/users/:id", adminHandler.GetUser)
				r.Patch("/users/:id", adminHandler.PatchUser)
				r.Delete("/users/:id", adminHandler.DeleteUser)
				r.Put("/users/:id/roles", adminHandler.UpdateUserRoles)
				r.With(auth.RequireABAC(abacEngine, "user.ban", nil)).Post("/users/:id/ban", adminHandler.BanUser)
				r.With(auth.RequireABAC(abacEngine, "user.unban", nil)).Post("/users/:id/unban", adminHandler.UnbanUser)
				r.Get("/invites", adminHandler.ListInvites)
				r.Post("/invites", adminHandler.CreateInvite)
				r.Post("/invites/:id/resend", adminHandler.ResendInvite)
				r.Get("/bots", adminHandler.ListBots)
				r.Post("/bots", adminHandler.CreateBot)
				r.Post("/bots/reload", adminHandler.ReloadBotConfig)
				r.Patch("/bots/:id", adminHandler.PatchBot)
				r.Delete("/bots/:id", adminHandler.DeleteBot)
				r.Post("/bots/:id/enable", adminHandler.EnableBot)
				r.Post("/bots/:id/disable", adminHandler.DisableBot)
				r.Get("/bots/:id/stats", adminHandler.GetBotStats)
				r.Put("/settings/appearance", adminHandler.PutAppearanceSettings)
				r.Get("/exchange/settings", adminHandler.GetExchangeSettings)
				r.Put("/exchange/settings", adminHandler.PutExchangeSettings)
				r.Post("/exchange/test-connection", adminHandler.TestExchangeConnection)
				r.Get("/analytics", adminHandler.GetAnalytics)
				r.Get("/conference/policies", adminHandler.GetConferencePolicies)
				r.Put("/conference/policies", adminHandler.PutConferencePolicies)
				r.Get("/conferences", adminHandler.ListConferences)
				r.With(auth.RequireABAC(abacEngine, "conference.end", nil)).Post("/conferences/:id/end", adminHandler.EndConference)
				r.Get("/stats", adminHandler.GetStats)
				r.Get("/webhooks/deliveries", adminHandler.ListWebhookDeliveries)
				r.Get("/webhooks/errors", adminHandler.ListWebhookErrors)
				r.Get("/bots/errors", adminHandler.ListBotErrors)
				r.Get("/auth/audit", adminHandler.ListAuthAuditEvents)
				r.Get("/calendar/audit", adminHandler.ListCalendarAuditEvents)
				r.Get("/audit", adminHandler.ListAuditLogs)
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
	var swaggerSrv *http.Server
	if cfg.Server.SwaggerPort != "" && cfg.Server.SwaggerPort != cfg.Server.Port {
		swaggerRouter := chi.NewRouter()
		swaggerRouter.Use(middleware.RequestID)
		swaggerRouter.Use(middleware.RealIP)
		swaggerRouter.Use(middleware.Logger)
		swaggerRouter.Use(middleware.Recoverer)
		swaggerRouter.Get("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/yaml")
			http.ServeFile(w, r, "docs/openapi.yaml")
		})
		swaggerRouter.Get("/swagger/*", httpSwagger.Handler(
			httpSwagger.URL("/openapi.yaml"),
			httpSwagger.DocExpansion("list"),
			httpSwagger.DefaultModelsExpandDepth(-1),
		))
		swaggerAddr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.SwaggerPort)
		swaggerSrv = &http.Server{
			Addr:         swaggerAddr,
			Handler:      swaggerRouter,
			ReadTimeout:  cfg.Server.ReadTimeout,
			WriteTimeout: cfg.Server.WriteTimeout,
			IdleTimeout:  cfg.Server.IdleTimeout,
		}
		go func() {
			logger.Info("Swagger server starting", zap.String("address", swaggerAddr))
			if err := swaggerSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("Swagger server failed", zap.Error(err))
			}
		}()
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
	appCancel()

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
		os.Exit(1)
	}
	if swaggerSrv != nil {
		if err := swaggerSrv.Shutdown(ctx); err != nil {
			logger.Error("Swagger server forced to shutdown", zap.Error(err))
			os.Exit(1)
		}
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
	calendarService exchange.CalendarService
	userRepo        *repository.UserRepository
}

func resolveSessionSecret(cfg *config.Config) []byte {
	if cfg == nil || cfg.Auth.SessionSecret == "" {
		return []byte("dev-session-secret-change-me")
	}
	return []byte(cfg.Auth.SessionSecret)
}

func resolveSessionLifetimeDuration(cfg *config.Config) time.Duration {
	if cfg == nil || cfg.Auth.SessionTokenLifetime <= 0 {
		return 24 * time.Hour
	}
	return cfg.Auth.SessionTokenLifetime
}

func ensureBotUser(db *database.Database, botID uuid.UUID) {
	var count int64
	db.DB.Model(&models.User{}).Where("id = ?", botID).Count(&count)
	if count > 0 {
		return
	}
	now := time.Now()
	botUser := &models.User{
		ID:        botID,
		Email:     "bot@focus.local",
		Name:      "Focus Bot",
		Roles:     models.StringArray{"bot"},
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	db.DB.Create(botUser)
}

func (s *botMeetingScheduler) ScheduleMeeting(
	ctx context.Context,
	userID uuid.UUID,
	title string,
	start, end time.Time,
	roomURL string,
) error {
	if s.calendarService == nil || s.userRepo == nil {
		return nil
	}
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	_, err = s.calendarService.CreateEvent(ctx, user.Email, exchange.CalendarEvent{
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
