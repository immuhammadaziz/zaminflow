package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/zaminflow/zaminflow-api/internal/config"
)

func main() {
	if err := run(); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	flags := flag.NewFlagSet("migrate", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	direction := flags.String("direction", "up", "migration direction: up or down (one migration)")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return fmt.Errorf("parse migration options: %w", err)
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %v", flags.Args())
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()
	pingCtx, cancelPing := context.WithTimeout(context.Background(), cfg.DBConnectTimeout)
	defer cancelPing()
	if err := db.PingContext(pingCtx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	switch *direction {
	case "up":
		err = goose.Up(db, cfg.MigrationsDir)
	case "down":
		err = goose.Down(db, cfg.MigrationsDir)
	default:
		return fmt.Errorf("migration direction must be up or down")
	}
	if err != nil {
		return fmt.Errorf("run %s migrations: %w", *direction, err)
	}
	slog.Info("database migration command completed", "direction", *direction, "directory", cfg.MigrationsDir)
	return nil
}
