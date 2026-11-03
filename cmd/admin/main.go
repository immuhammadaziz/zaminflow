package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/zaminflow/zaminflow-api/internal/auth"
	"github.com/zaminflow/zaminflow-api/internal/config"
	"github.com/zaminflow/zaminflow-api/internal/platform/db"
)

func main() {
	if err := run(); err != nil {
		slog.Error("admin command failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) != 2 || os.Args[1] != "create" {
		return errors.New("usage: admin create (set ADMIN_EMAIL, ADMIN_FULL_NAME, and ADMIN_PASSWORD_FILE in production)")
	}
	email := strings.TrimSpace(os.Getenv("ADMIN_EMAIL"))
	fullName := strings.TrimSpace(os.Getenv("ADMIN_FULL_NAME"))
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	password, err := config.SecretValue("ADMIN_PASSWORD", cfg.AppEnv)
	if err != nil {
		return err
	}
	if email == "" || fullName == "" || password == "" {
		return errors.New("ADMIN_EMAIL, ADMIN_FULL_NAME, and ADMIN_PASSWORD are required")
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	startupCtx, cancel := context.WithTimeout(context.Background(), cfg.DBConnectTimeout)
	defer cancel()
	pool, err := db.Open(startupCtx, cfg)
	if err != nil {
		return err
	}
	defer pool.Close()
	service, err := auth.NewService(startupCtx, pool, cfg, logger)
	if err != nil {
		return err
	}
	if err := service.CreateAdmin(startupCtx, email, password, fullName); err != nil {
		return fmt.Errorf("provision administrator: %w", err)
	}
	logger.Info("administrator provisioned; sign in and complete MFA enrollment")
	return nil
}
