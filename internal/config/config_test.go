package config

import (
	"log/slog"
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
	t.Setenv("DATABASE_URL", "postgres://user:password@localhost:5432/zaminflow")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://zaminflow.uz,http://localhost:3000")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "must use https") {
		t.Fatalf("Load() error = %v, want production HTTPS origin error", err)
	}
}
