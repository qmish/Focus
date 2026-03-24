package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/qmish/focus-api/internal/config"
	"github.com/qmish/focus-api/internal/database"
	"github.com/qmish/focus-api/internal/jitsi"
	"github.com/qmish/focus-api/internal/models"
	"github.com/qmish/focus-api/internal/repository"
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
	); err != nil {
		logger.Error("Database migration failed", zap.Error(err))
		os.Exit(1)
	}
	logger.Info("Database migrations completed")

	// Инициализация репозиториев
	userRepo := repository.NewUserRepository(db.DB)
	roomRepo := repository.NewRoomRepository(db.DB)

	// Инициализация Jitsi генератора токенов
	jitsiGen := jitsi.NewTokenGenerator(
		cfg.Jitsi.BaseURL,
		cfg.Jitsi.AppID,
		cfg.Jitsi.AppSecret,
		cfg.Jitsi.Issuer,
		cfg.Jitsi.Audience,
		cfg.Jitsi.TokenLifetime,
	)

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
		r.Route("/auth", func(r chi.Router) {
			// TODO: Implement auth handlers
			r.Get("/login", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("Login endpoint - TODO"))
			})
			r.Get("/callback", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("Callback endpoint - TODO"))
			})
		})

		// Protected routes (TODO: Add auth middleware)
		r.Route("/rooms", func(r chi.Router) {
			// r.Use(authMiddleware)
			r.Get("/", listRooms(roomRepo))
			r.Post("/", createRoom(roomRepo, userRepo, jitsiGen))
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", getRoom(roomRepo))
				r.Put("/", updateRoom(roomRepo))
				r.Delete("/", deleteRoom(roomRepo))
				r.Post("/join", joinRoom(roomRepo, jitsiGen))
			})
		})

		r.Route("/messages", func(r chi.Router) {
			// r.Use(authMiddleware)
			r.Get("/", listMessages)
			r.Post("/", createMessage)
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

// listRooms обработчик GET /rooms
func listRooms(repo *repository.RoomRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"rooms":[],"pagination":{"page":1,"per_page":20,"total":0}}`))
	}
}

// createRoom обработчик POST /rooms
func createRoom(repo *repository.RoomRepository, userRepo *repository.UserRepository, jitsiGen *jitsi.TokenGenerator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(`{"error":"not implemented"}`))
	}
}

// getRoom обработчик GET /rooms/{id}
func getRoom(repo *repository.RoomRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(`{"error":"not implemented"}`))
	}
}

// updateRoom обработчик PUT /rooms/{id}
func updateRoom(repo *repository.RoomRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(`{"error":"not implemented"}`))
	}
}

// deleteRoom обработчик DELETE /rooms/{id}
func deleteRoom(repo *repository.RoomRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(`{"error":"not implemented"}`))
	}
}

// joinRoom обработчик POST /rooms/{id}/join
func joinRoom(repo *repository.RoomRepository, jitsiGen *jitsi.TokenGenerator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(`{"error":"not implemented"}`))
	}
}

// listMessages обработчик GET /messages
func listMessages(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte(`{"error":"not implemented"}`))
}

// createMessage обработчик POST /messages
func createMessage(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte(`{"error":"not implemented"}`))
}
