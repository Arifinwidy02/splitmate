package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Arifinwidy02/splitmate-backend/internal/auth"
	"github.com/Arifinwidy02/splitmate-backend/internal/config"
	"github.com/Arifinwidy02/splitmate-backend/internal/database"
	"github.com/Arifinwidy02/splitmate-backend/internal/server"
	"github.com/Arifinwidy02/splitmate-backend/internal/session"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	tokenService := session.NewTokenServiceWithDefaults([]byte(cfg.JWTSecret))
	sessionRepo := session.NewRepository(pool)
	sessionService := session.NewService(tokenService, sessionRepo)

	var oauth *auth.OAuthConfig
	if cfg.GoogleOAuthEnabled() {
		oauth = &auth.OAuthConfig{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			RedirectURL:  cfg.OAuthRedirectURL,
			AppBaseURL:   cfg.AppBaseURL,
		}
	} else {
		logger.Warn("google oauth not configured; google sign in will be disabled")
	}

	httpServer := &http.Server{
		Addr: ":" + strconv.Itoa(cfg.Port),
		Handler: server.New(server.Dependencies{
			Pool:          pool,
			Session:       sessionService,
			SecureCookies: cfg.AppEnv == "production",
			OAuth:         oauth,
			AppBaseURL:    cfg.AppBaseURL,
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server starting", "addr", httpServer.Addr, "app_env", cfg.AppEnv)
		errCh <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		logger.Error("server failed", "error", err)
		os.Exit(1)
	case <-signalCtx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped")
}
