package config

import (
	"fmt"
	"log/slog"
	"net"
	"net/mail"
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
	AppSecret           string
	MetricsToken        string
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
	SMTPHost            string
	SMTPPort            string
	SMTPUser            string
	SMTPPassword        string
	SMTPRequireTLS      bool
	MailFrom            string
	WebBaseURL          string
	RefreshCookieDomain string
}

func Load() (Config, error) {
	appEnv := envOr("APP_ENV", "development")
	appSecret, err := SecretValue("APP_SECRET", appEnv)
	if err != nil {
		return Config{}, err
	}
	metricsToken, err := SecretValue("METRICS_TOKEN", appEnv)
	if err != nil {
		return Config{}, err
	}
	databaseURL, err := configuredDatabaseURL(appEnv)
	if err != nil {
		return Config{}, err
	}
	smtpUser, err := SecretValue("SMTP_USER", appEnv)
	if err != nil {
		return Config{}, err
	}
	smtpPassword, err := SecretValue("SMTP_PASSWORD", appEnv)
	if err != nil {
		return Config{}, err
	}
	cfg := Config{
		AppEnv:              appEnv,
		HTTPAddr:            envOr("HTTP_ADDR", ":8080"),
		DatabaseURL:         databaseURL,
		AppSecret:           strings.TrimSpace(appSecret),
		MetricsToken:        strings.TrimSpace(metricsToken),
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
		SMTPHost:            envOr("SMTP_HOST", "mailpit"),
		SMTPPort:            envOr("SMTP_PORT", "1025"),
		SMTPUser:            strings.TrimSpace(smtpUser),
		SMTPPassword:        smtpPassword,
		SMTPRequireTLS:      false,
		MailFrom:            envOr("MAIL_FROM", "ZaminFlow <noreply@zaminflow.uz>"),
		WebBaseURL:          envOr("WEB_BASE_URL", "http://localhost:3000"),
		RefreshCookieDomain: strings.TrimSpace(os.Getenv("REFRESH_COOKIE_DOMAIN")),
	}
	switch cfg.AppEnv {
	case "development", "test", "production":
	default:
		return Config{}, fmt.Errorf("APP_ENV must be development, test, or production")
	}
	smtpRequireTLS, err := envBool("SMTP_REQUIRE_TLS", cfg.AppEnv == "production")
	if err != nil {
		return Config{}, err
	}
	cfg.SMTPRequireTLS = smtpRequireTLS
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if len(cfg.AppSecret) < 32 {
		return Config{}, fmt.Errorf("APP_SECRET must contain at least 32 characters")
	}
	if cfg.AppEnv == "production" && cfg.AppSecret == "replace-this-with-a-random-secret-at-least-32-characters" {
		return Config{}, fmt.Errorf("APP_SECRET must be changed from the development placeholder in production")
	}
	if err := validateDatabaseURL(cfg.DatabaseURL); err != nil {
		return Config{}, err
	}

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
	smtpPort, err := strconv.Atoi(cfg.SMTPPort)
	if err != nil || smtpPort < 1 || smtpPort > 65535 {
		return Config{}, fmt.Errorf("SMTP_PORT must be an integer from 1 to 65535")
	}
	if (cfg.SMTPUser == "") != (cfg.SMTPPassword == "") {
		return Config{}, fmt.Errorf("SMTP_USER and SMTP_PASSWORD must either both be set or both be empty")
	}
	if _, err := mail.ParseAddress(cfg.MailFrom); err != nil {
		return Config{}, fmt.Errorf("MAIL_FROM must be a valid email address: %w", err)
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
		if len(cfg.MetricsToken) < 32 {
			return Config{}, fmt.Errorf("METRICS_TOKEN must contain at least 32 characters in production")
		}
		for _, origin := range cfg.CORSAllowedOrigins {
			parsed, parseErr := url.Parse(origin)
			if parseErr != nil {
				return Config{}, fmt.Errorf("parse production CORS origin %q: %w", origin, parseErr)
			}
			if parsed.Scheme != "https" {
				return Config{}, fmt.Errorf("production CORS origin %q must use https", origin)
			}
		}
		if !cfg.SMTPRequireTLS {
			return Config{}, fmt.Errorf("SMTP_REQUIRE_TLS must be enabled in production")
		}
	}
	webURL, err := url.Parse(cfg.WebBaseURL)
	if err != nil || (webURL.Scheme != "http" && webURL.Scheme != "https") ||
		webURL.Host == "" || webURL.User != nil || webURL.RawQuery != "" || webURL.Fragment != "" {
		return Config{}, fmt.Errorf("WEB_BASE_URL must be an absolute http or https URL without credentials, query, or fragment")
	}
	if cfg.AppEnv == "production" && webURL.Scheme != "https" {
		return Config{}, fmt.Errorf("WEB_BASE_URL must use https in production")
	}
	cfg.WebBaseURL = strings.TrimRight(cfg.WebBaseURL, "/")
	return cfg, nil
}

func SecretValue(name, appEnv string) (string, error) {
	filePath := strings.TrimSpace(os.Getenv(name + "_FILE"))
	plainValue, plainSet := os.LookupEnv(name)
	if filePath != "" {
		if plainSet && strings.TrimSpace(plainValue) != "" {
			return "", fmt.Errorf("%s and %s_FILE must not both be set", name, name+"_FILE")
		}
		value, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("read %s_FILE: %w", name, err)
		}
		secret := strings.TrimRight(string(value), "\r\n")
		if strings.TrimSpace(secret) == "" {
			return "", fmt.Errorf("%s_FILE must not be empty", name)
		}
		return secret, nil
	}
	if appEnv == "production" && plainSet && strings.TrimSpace(plainValue) != "" {
		return "", fmt.Errorf("%s must be provided through %s_FILE in production", name, name)
	}
	if appEnv != "development" && appEnv != "test" && appEnv != "production" {
		return "", fmt.Errorf("APP_ENV must be development, test, or production")
	}
	if appEnv == "production" {
		return "", nil
	}
	return plainValue, nil
}

func configuredDatabaseURL(appEnv string) (string, error) {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if appEnv == "production" && databaseURL != "" {
		return "", fmt.Errorf("DATABASE_URL must not be set in production; configure DB_HOST, DB_PORT, DB_NAME, DB_USER_FILE, and DB_PASSWORD_FILE")
	}
	if databaseURL != "" {
		return databaseURL, nil
	}
	host := strings.TrimSpace(os.Getenv("DB_HOST"))
	name := strings.TrimSpace(os.Getenv("DB_NAME"))
	user, err := SecretValue("DB_USER", appEnv)
	if err != nil {
		return "", err
	}
	password, err := SecretValue("DB_PASSWORD", appEnv)
	if err != nil {
		return "", err
	}
	if host == "" && name == "" && user == "" && password == "" && appEnv != "production" {
		return "", nil
	}
	if host == "" || name == "" || user == "" || password == "" {
		return "", fmt.Errorf("DB_HOST, DB_NAME, DB_USER, and DB_PASSWORD are required when DATABASE_URL is not set")
	}
	port := envOr("DB_PORT", "5432")
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return "", fmt.Errorf("DB_PORT must be an integer from 1 to 65535")
	}
	databaseURL = (&url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(user, password),
		Host:     net.JoinHostPort(host, port),
		Path:     "/" + name,
		RawQuery: "sslmode=disable",
	}).String()
	return databaseURL, nil
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

func envBool(key string, fallback bool) (bool, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return false, fmt.Errorf("%s must be true or false: %w", key, err)
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
