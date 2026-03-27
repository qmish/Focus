package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Config хранит конфигурацию приложения
type Config struct {
	Env       string
	Server    ServerConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	Auth      AuthConfig
	WebSocket WebSocketConfig
	Keycloak  KeycloakConfig
	Jitsi     JitsiConfig
	Exchange  ExchangeConfig
	Log       LogConfig
}

// AuthConfig конфигурация сессионных токенов API/WS
type AuthConfig struct {
	SessionSecret            string
	SessionTokenLifetime     time.Duration
	SessionValidationSecrets []string
	RequiredAudience         string
	ServiceAudiences         []string
	ServiceScopes            []string
}

// WebSocketConfig конфигурация websocket безопасности.
type WebSocketConfig struct {
	AllowedOrigins   []string
	StrictRoomAccess bool
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
	ServerURL          string
	InternalURL        string // optional: internal cluster URL for OIDC discovery
	Realm              string
	ClientID           string
	ClientSecret       string
	RedirectURL        string
	GroupPolicyMapping string
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

// ExchangeConfig конфигурация on-prem Exchange/OWA (EWS).
type ExchangeConfig struct {
	Provider       string
	EWSURL         string
	Username       string
	Password       string
	Domain         string
	AuthMode       string
	CACertPath     string
	InsecureTLS    bool
	Krb5ConfigPath string
	Krb5KeytabPath string
	Krb5Realm      string
	Krb5SPN        string
	Impersonation  bool
	Timeout        time.Duration
	SyncEnabled    bool
	SyncInterval   time.Duration
	SyncLookback   time.Duration
	SyncLookahead  time.Duration
}

// LogConfig конфигурация логирования
type LogConfig struct {
	Level  string
	Format string
}

// Load загружает конфигурацию из переменных окружения
func Load() *Config {
	env := getEnv("ENV", "development")
	strictRoomAccessDefault := true
	if env == "development" {
		strictRoomAccessDefault = false
	}
	cfg := &Config{
		Env: env,
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
		Auth: AuthConfig{
			SessionSecret:            getEnv("SESSION_SECRET", "dev-session-secret-change-me"),
			SessionTokenLifetime:     getDurationEnv("AUTH_SESSION_TOKEN_LIFETIME", 24*time.Hour),
			SessionValidationSecrets: getListEnv("AUTH_SESSION_VALIDATION_SECRETS", nil),
			RequiredAudience:         getEnv("AUTH_REQUIRED_AUDIENCE", "focus-frontend"),
			ServiceAudiences:         getListEnv("AUTH_SERVICE_AUDIENCES", []string{"focus-service"}),
			ServiceScopes:            getListEnv("AUTH_SERVICE_SCOPES", []string{"focus.service"}),
		},
		WebSocket: WebSocketConfig{
			AllowedOrigins: getListEnv("WS_ALLOWED_ORIGINS", []string{
				"http://localhost:3000",
				"http://localhost:3001",
				"http://localhost:5173",
				"http://localhost:5174",
			}),
			StrictRoomAccess: getBoolEnv("WS_STRICT_ROOM_ACCESS", strictRoomAccessDefault),
		},
		Keycloak: KeycloakConfig{
			ServerURL:          getEnv("KEYCLOAK_URL", "http://localhost:8180"),
			InternalURL:        getEnv("KEYCLOAK_INTERNAL_URL", ""),
			Realm:              getEnv("KEYCLOAK_REALM", "company"),
			ClientID:           getEnv("KEYCLOAK_CLIENT_ID", "messenger-api"),
			ClientSecret:       getEnv("KEYCLOAK_CLIENT_SECRET", ""),
			RedirectURL:        getEnv("KEYCLOAK_REDIRECT_URL", "http://localhost:8080/api/v1/auth/callback"),
			GroupPolicyMapping: getEnv("KEYCLOAK_GROUP_POLICY_MAPPING", ""),
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
			Provider:       getEnv("EXCHANGE_PROVIDER", "ews"),
			EWSURL:         getEnv("EXCHANGE_EWS_URL", ""),
			Username:       getEnv("EXCHANGE_USERNAME", ""),
			Password:       getEnv("EXCHANGE_PASSWORD", ""),
			Domain:         getEnv("EXCHANGE_DOMAIN", ""),
			AuthMode:       strings.ToLower(getEnv("EXCHANGE_AUTH_MODE", "basic")),
			CACertPath:     getEnv("EXCHANGE_CA_CERT_PATH", ""),
			InsecureTLS:    getBoolEnv("EXCHANGE_INSECURE_TLS", false),
			Krb5ConfigPath: getEnv("EXCHANGE_KRB5_CONFIG_PATH", ""),
			Krb5KeytabPath: getEnv("EXCHANGE_KRB5_KEYTAB_PATH", ""),
			Krb5Realm:      getEnv("EXCHANGE_KRB5_REALM", ""),
			Krb5SPN:        getEnv("EXCHANGE_KRB5_SERVICE_PRINCIPAL", ""),
			Impersonation:  getBoolEnv("EXCHANGE_IMPERSONATION", true),
			Timeout:        getDurationEnv("EXCHANGE_TIMEOUT", 15*time.Second),
			SyncEnabled:    getBoolEnv("EXCHANGE_SYNC_ENABLED", false),
			SyncInterval:   getDurationEnv("EXCHANGE_SYNC_INTERVAL", 2*time.Minute),
			SyncLookback:   getDurationEnv("EXCHANGE_SYNC_LOOKBACK", 12*time.Hour),
			SyncLookahead:  getDurationEnv("EXCHANGE_SYNC_LOOKAHEAD", 14*24*time.Hour),
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
	}

	return cfg
}

// ValidateSecurity проверяет минимальные security-инварианты для токенов и секретов.
func (c *Config) ValidateSecurity() error {
	sessionSecret := strings.TrimSpace(c.Auth.SessionSecret)
	jitsiSecret := strings.TrimSpace(c.Jitsi.AppSecret)

	if sessionSecret == "" {
		return fmt.Errorf("SESSION_SECRET must not be empty")
	}

	if jitsiSecret == "" {
		return fmt.Errorf("JITSI_APP_SECRET must not be empty")
	}

	if sessionSecret == jitsiSecret {
		return fmt.Errorf("SESSION_SECRET must be different from JITSI_APP_SECRET")
	}

	if c.Env != "development" {
		weakValues := map[string]struct{}{
			"secret":                       {},
			"changeme":                     {},
			"dev-session-secret-change-me": {},
			"change_me":                    {},
		}
		if _, weak := weakValues[strings.ToLower(sessionSecret)]; weak {
			return fmt.Errorf("SESSION_SECRET is too weak for %s environment", c.Env)
		}
	}

	if c.Auth.SessionTokenLifetime < 15*time.Minute {
		return fmt.Errorf("AUTH_SESSION_TOKEN_LIFETIME must be at least 15m")
	}
	if c.Env != "development" && len(c.WebSocket.AllowedOrigins) == 0 {
		return fmt.Errorf("WS_ALLOWED_ORIGINS must not be empty in %s environment", c.Env)
	}

	return nil
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

func getBoolEnv(key string, defaultValue bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return defaultValue
	}
	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return defaultValue
	}
}

func getListEnv(key string, defaultValue []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	rawParts := strings.Split(value, ",")
	result := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		result = append(result, item)
	}
	if len(result) == 0 {
		return defaultValue
	}
	return result
}
