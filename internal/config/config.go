package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv              string
	HTTPAddr            string
	LogLevel            slog.Level
	DatabaseURL         string
	MigrationsDir       string
	DBMaxConns          int32
	DBMinConns          int32
	DBMaxConnLifetime   time.Duration
	DBConnectTimeout    time.Duration
	HTTPReadTimeout     time.Duration
	HTTPWriteTimeout    time.Duration
	HTTPIdleTimeout     time.Duration
	HTTPShutdownTimeout time.Duration
	HTTPRequestTimeout  time.Duration
	HTTPMaxBodyBytes    int64
	CORSAllowedOrigins  []string
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:              envOr("APP_ENV", "development"),
		HTTPAddr:            envOr("HTTP_ADDR", ":8080"),
		DatabaseURL:         strings.TrimSpace(os.Getenv("DATABASE_URL")),
		MigrationsDir:       envOr("MIGRATIONS_DIR", "migrations"),
		DBMaxConns:          20,
		DBMinConns:          2,
		DBMaxConnLifetime:   30 * time.Minute,
		DBConnectTimeout:    5 * time.Second,
		HTTPReadTimeout:     5 * time.Second,
		HTTPWriteTimeout:    15 * time.Second,
		HTTPIdleTimeout:     60 * time.Second,
		HTTPShutdownTimeout: 10 * time.Second,
		HTTPRequestTimeout:  10 * time.Second,
		HTTPMaxBodyBytes:    1 << 20,
	}
	switch cfg.AppEnv {
	case "development", "test", "production":
	default:
		return Config{}, fmt.Errorf("APP_ENV must be development, test, or production")
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if err := validateDatabaseURL(cfg.DatabaseURL); err != nil {
		return Config{}, err
	}

	var err error
	if cfg.DBMaxConns, err = envInt32("DB_MAX_CONNS", int32(cfg.DBMaxConns)); err != nil {
		return Config{}, err
	}
	if cfg.DBMinConns, err = envInt32("DB_MIN_CONNS", int32(cfg.DBMinConns)); err != nil {
		return Config{}, err
	}
	if cfg.DBMinConns < 0 || cfg.DBMaxConns < 1 || cfg.DBMinConns > cfg.DBMaxConns {
		return Config{}, fmt.Errorf("DB_MIN_CONNS must be non-negative and no greater than positive DB_MAX_CONNS")
	}
	if cfg.DBMaxConnLifetime, err = envDuration("DB_MAX_CONN_LIFETIME", cfg.DBMaxConnLifetime); err != nil {
		return Config{}, err
	}
	if cfg.DBMaxConnLifetime <= 0 {
		return Config{}, fmt.Errorf("DB_MAX_CONN_LIFETIME must be positive")
	}
	if cfg.DBConnectTimeout, err = envDuration("DB_CONNECT_TIMEOUT", cfg.DBConnectTimeout); err != nil {
		return Config{}, err
	}
	if cfg.DBConnectTimeout <= 0 {
		return Config{}, fmt.Errorf("DB_CONNECT_TIMEOUT must be positive")
	}
	if cfg.HTTPReadTimeout, err = envDuration("HTTP_READ_TIMEOUT", cfg.HTTPReadTimeout); err != nil {
		return Config{}, err
	}
	if cfg.HTTPWriteTimeout, err = envDuration("HTTP_WRITE_TIMEOUT", cfg.HTTPWriteTimeout); err != nil {
		return Config{}, err
	}
	if cfg.HTTPIdleTimeout, err = envDuration("HTTP_IDLE_TIMEOUT", cfg.HTTPIdleTimeout); err != nil {
		return Config{}, err
	}
	if cfg.HTTPShutdownTimeout, err = envDuration("HTTP_SHUTDOWN_TIMEOUT", cfg.HTTPShutdownTimeout); err != nil {
		return Config{}, err
	}
	if cfg.HTTPRequestTimeout, err = envDuration("HTTP_REQUEST_TIMEOUT", cfg.HTTPRequestTimeout); err != nil {
		return Config{}, err
	}
	if cfg.HTTPReadTimeout <= 0 || cfg.HTTPWriteTimeout <= 0 || cfg.HTTPIdleTimeout <= 0 ||
		cfg.HTTPShutdownTimeout <= 0 || cfg.HTTPRequestTimeout <= 0 {
		return Config{}, fmt.Errorf("HTTP timeout values must be positive")
	}
	if cfg.HTTPMaxBodyBytes, err = envInt64("HTTP_MAX_BODY_BYTES", cfg.HTTPMaxBodyBytes); err != nil {
		return Config{}, err
	}
	if cfg.HTTPMaxBodyBytes < 1 {
		return Config{}, fmt.Errorf("HTTP_MAX_BODY_BYTES must be positive")
	}
	if cfg.LogLevel, err = parseLogLevel(envOr("LOG_LEVEL", "info")); err != nil {
		return Config{}, err
	}
	originsValue, configuredOrigins := os.LookupEnv("CORS_ALLOWED_ORIGINS")
	if !configuredOrigins || strings.TrimSpace(originsValue) == "" {
		if cfg.AppEnv == "production" {
			return Config{}, fmt.Errorf("CORS_ALLOWED_ORIGINS must be explicitly configured in production")
		}
		originsValue = "https://www.zaminflow.uz,https://zaminflow.uz,http://localhost:3000,http://localhost:5173"
	}
	cfg.CORSAllowedOrigins, err = parseOrigins(originsValue)
	if err != nil {
		return Config{}, err
	}
	if cfg.AppEnv == "production" {
		for _, origin := range cfg.CORSAllowedOrigins {
			parsed, parseErr := url.Parse(origin)
			if parseErr != nil {
				return Config{}, fmt.Errorf("parse production CORS origin %q: %w", origin, parseErr)
			}
			if parsed.Scheme != "https" {
				return Config{}, fmt.Errorf("production CORS origin %q must use https", origin)
			}
		}
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func envInt32(key string, fallback int32) (int32, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	return int32(parsed), nil
}

func envInt64(key string, fallback int64) (int64, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	return parsed, nil
}

func envDuration(key string, fallback time.Duration) (time.Duration, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration: %w", key, err)
	}
	return parsed, nil
}

func parseLogLevel(value string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("LOG_LEVEL must be debug, info, warn, or error")
	}
}

func parseOrigins(value string) ([]string, error) {
	parts := strings.Split(value, ",")
	origins := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		origin := strings.TrimSpace(part)
		if origin == "" {
			return nil, fmt.Errorf("CORS_ALLOWED_ORIGINS must not contain empty entries")
		}
		parsed, err := url.Parse(origin)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") ||
			parsed.Host == "" || parsed.Path != "" || parsed.RawQuery != "" ||
			parsed.Fragment != "" || parsed.User != nil || origin == "*" {
			return nil, fmt.Errorf("CORS_ALLOWED_ORIGINS contains invalid origin %q", origin)
		}
		if _, exists := seen[origin]; exists {
			return nil, fmt.Errorf("CORS_ALLOWED_ORIGINS contains duplicate origin %q", origin)
		}
		seen[origin] = struct{}{}
		origins = append(origins, origin)
	}
	return origins, nil
}

func validateDatabaseURL(value string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("DATABASE_URL is invalid: %w", err)
	}
	if (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") || parsed.Host == "" || parsed.Path == "" {
		return fmt.Errorf("DATABASE_URL must be a PostgreSQL URL with a host and database name")
	}
	return nil
}
