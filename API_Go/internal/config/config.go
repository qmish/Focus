package config

import (
	"fmt"
	"os"
	"time"
)

// Config хранит конфигурацию приложения
type Config struct {
	Env      string
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Keycloak KeycloakConfig
	Jitsi    JitsiConfig
	Exchange ExchangeConfig
	Log      LogConfig
}

// ServerConfig конфигурация HTTP сервера
type ServerConfig struct {
	Host         string
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DatabaseConfig конфигурация PostgreSQL
type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// RedisConfig конфигурация Redis
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// KeycloakConfig конфигурация Keycloak OIDC
type KeycloakConfig struct {
	ServerURL    string
	Realm        string
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// JitsiConfig конфигурация Jitsi Meet
type JitsiConfig struct {
	BaseURL       string
	AppID         string
	AppSecret     string
	Issuer        string
	Audience      string
	TokenLifetime time.Duration
}

// ExchangeConfig конфигурация MS Exchange
type ExchangeConfig struct {
	TenantID     string
	ClientID     string
	ClientSecret string
}

// LogConfig конфигурация логирования
type LogConfig struct {
	Level  string
	Format string
}

// Load загружает конфигурацию из переменных окружения
func Load() *Config {
	cfg := &Config{
		Env: getEnv("ENV", "development"),
		Server: ServerConfig{
			Host:         getEnv("SERVER_HOST", "0.0.0.0"),
			Port:         getEnv("SERVER_PORT", "8080"),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:  getDurationEnv("SERVER_IDLE_TIMEOUT", 60*time.Second),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "postgres"),
			DBName:          getEnv("DB_NAME", "focus"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    getIntEnv("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getIntEnv("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getDurationEnv("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getIntEnv("REDIS_DB", 0),
		},
		Keycloak: KeycloakConfig{
			ServerURL:    getEnv("KEYCLOAK_URL", "http://localhost:8180"),
			Realm:        getEnv("KEYCLOAK_REALM", "company"),
			ClientID:     getEnv("KEYCLOAK_CLIENT_ID", "messenger-api"),
			ClientSecret: getEnv("KEYCLOAK_CLIENT_SECRET", ""),
			RedirectURL:  getEnv("KEYCLOAK_REDIRECT_URL", "http://localhost:8080/api/v1/auth/callback"),
		},
		Jitsi: JitsiConfig{
			BaseURL:       getEnv("JITSI_BASE_URL", "https://meet.company.com"),
			AppID:         getEnv("JITSI_APP_ID", "jitsi"),
			AppSecret:     getEnv("JITSI_APP_SECRET", "secret"),
			Issuer:        getEnv("JITSI_ISSUER", "jitsi"),
			Audience:      getEnv("JITSI_AUDIENCE", "jitsi"),
			TokenLifetime: getDurationEnv("JITSI_TOKEN_LIFETIME", 8*time.Hour),
		},
		Exchange: ExchangeConfig{
			TenantID:     getEnv("EXCHANGE_TENANT_ID", ""),
			ClientID:     getEnv("EXCHANGE_CLIENT_ID", ""),
			ClientSecret: getEnv("EXCHANGE_CLIENT_SECRET", ""),
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var result int
		fmt.Sscanf(value, "%d", &result)
		return result
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		duration, err := time.ParseDuration(value)
		if err == nil {
			return duration
		}
	}
	return defaultValue
}
