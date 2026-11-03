package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zaminflow/zaminflow-api/internal/config"
	"github.com/zaminflow/zaminflow-api/internal/httpapi"
	"github.com/zaminflow/zaminflow-api/internal/platform/db"
)

func main() {
	if err := run(); err != nil {
		slog.Error("api stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	startupCtx, cancelStartup := context.WithTimeout(context.Background(), cfg.DBConnectTimeout)
	defer cancelStartup()
	pool, err := db.Open(startupCtx, cfg)
	if err != nil {
		return err
	}
	defer pool.Close()

	handler := httpapi.NewRouter(pool, logger, cfg.CORSAllowedOrigins, cfg.HTTPRequestTimeout, cfg.HTTPMaxBodyBytes)
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: cfg.HTTPReadTimeout,
		ReadTimeout:       cfg.HTTPReadTimeout,
		WriteTimeout:      cfg.HTTPWriteTimeout,
		IdleTimeout:       cfg.HTTPIdleTimeout,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("http server listening", "address", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-shutdownCtx.Done():
		ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTPShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			_ = server.Close()
			return err
		}
		select {
		case err := <-serverErrors:
			if !errors.Is(err, http.ErrServerClosed) {
				return err
			}
		case <-time.After(cfg.HTTPShutdownTimeout):
			return errors.New("http server did not stop before shutdown timeout")
		}
	}
	return nil
}
