package config

import (
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseOriginsRejectsWildcardAndPaths(t *testing.T) {
	for _, value := range []string{"*", "https://example.com/path", "http://example.com, "} {
		t.Run(value, func(t *testing.T) {
			if _, err := parseOrigins(value); err == nil {
				t.Fatalf("parseOrigins(%q) unexpectedly succeeded", value)
			}
		})
	}
}

func TestParseOriginsAcceptsExplicitOrigins(t *testing.T) {
	got, err := parseOrigins("https://zaminflow.uz,http://localhost:3000")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "https://zaminflow.uz" || got[1] != "http://localhost:3000" {
		t.Fatalf("origins = %#v", got)
	}
}

func TestParseLogLevel(t *testing.T) {
	level, err := parseLogLevel("warn")
	if err != nil {
		t.Fatal(err)
	}
	if level != slog.LevelWarn {
		t.Fatalf("level = %v, want %v", level, slog.LevelWarn)
	}
	if _, err := parseLogLevel("verbose"); err == nil {
		t.Fatal("invalid log level unexpectedly succeeded")
	}
}

func TestLoadRejectsHTTPOriginInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	setProductionSecrets(t)
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://zaminflow.uz,http://localhost:3000")
	t.Setenv("WEB_BASE_URL", "https://zaminflow.uz")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "must use https") {
		t.Fatalf("Load() error = %v, want production HTTPS origin error", err)
	}
}

func TestLoadRejectsInsecureProductionWebURL(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	setProductionSecrets(t)
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://zaminflow.uz")
	t.Setenv("WEB_BASE_URL", "http://zaminflow.uz")
	t.Setenv("SMTP_REQUIRE_TLS", "true")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "WEB_BASE_URL must use https") {
		t.Fatalf("Load() error = %v, want production HTTPS web URL error", err)
	}
}

func TestSecretValueReadsFileAndTrimsLineEnding(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(path, []byte("secret-value\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TEST_SECRET_FILE", path)
	got, err := SecretValue("TEST_SECRET", "production")
	if err != nil {
		t.Fatal(err)
	}
	if got != "secret-value" {
		t.Fatalf("SecretValue() = %q, want secret-value", got)
	}
}

func TestSecretValueRejectsProductionPlaintext(t *testing.T) {
	t.Setenv("TEST_SECRET", "plaintext-secret")
	if _, err := SecretValue("TEST_SECRET", "production"); err == nil ||
		!strings.Contains(err.Error(), "TEST_SECRET_FILE") {
		t.Fatalf("SecretValue() error = %v, want file-only production error", err)
	}
}

func TestSecretValueRejectsEmptyOrMissingFile(t *testing.T) {
	emptyPath := filepath.Join(t.TempDir(), "empty-secret")
	if err := os.WriteFile(emptyPath, []byte("\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{emptyPath, filepath.Join(t.TempDir(), "missing-secret")} {
		t.Run(filepath.Base(path), func(t *testing.T) {
			t.Setenv("TEST_SECRET_FILE", path)
			if _, err := SecretValue("TEST_SECRET", "production"); err == nil {
				t.Fatal("SecretValue() unexpectedly succeeded")
			}
		})
	}
}

func TestConfiguredDatabaseURLBuildsAndEscapesCredentials(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_HOST", "db.internal")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("DB_NAME", "zaminflow")
	t.Setenv("DB_USER", "user@name")
	t.Setenv("DB_PASSWORD", "p@ss:/?#word")
	got, err := configuredDatabaseURL("development")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	username := parsed.User.Username()
	password, ok := parsed.User.Password()
	if !ok || username != "user@name" || password != "p@ss:/?#word" {
		t.Fatalf("parsed credentials = %q, %q, want original credentials", username, password)
	}
	if parsed.Host != "db.internal:5433" || parsed.Path != "/zaminflow" {
		t.Fatalf("parsed database address = %q%s", parsed.Host, parsed.Path)
	}
}

func TestConfiguredDatabaseURLRequiresFileSecretsInProduction(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://db.internal/zaminflow")
	if _, err := configuredDatabaseURL("production"); err == nil ||
		!strings.Contains(err.Error(), "must not be set in production") {
		t.Fatalf("configuredDatabaseURL() error = %v, want DATABASE_URL rejection", err)
	}

	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_HOST", "db.internal")
	t.Setenv("DB_NAME", "zaminflow")
	t.Setenv("DB_USER", "plaintext")
	if _, err := configuredDatabaseURL("production"); err == nil ||
		!strings.Contains(err.Error(), "DB_USER_FILE") {
		t.Fatalf("configuredDatabaseURL() error = %v, want DB_USER_FILE requirement", err)
	}
}

func setProductionSecrets(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	for name, value := range map[string]string{
		"APP_SECRET":    "test-secret-with-more-than-thirty-two-characters",
		"METRICS_TOKEN": "test-metrics-token-with-more-than-32-characters",
		"DB_USER":       "test-user",
		"DB_PASSWORD":   "test-password",
	} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
		t.Setenv(name+"_FILE", path)
		t.Setenv(name, "")
	}
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_NAME", "zaminflow")
}
