package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/Arifinwidy02/splitmate-backend/internal/config"
	"github.com/Arifinwidy02/splitmate-backend/internal/database"
	"github.com/Arifinwidy02/splitmate-backend/migrations"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	if cfg.DatabaseURL == "" {
		logger.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	ctx := context.Background()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	list, err := database.ParseMigrations(migrations.Files)
	if err != nil {
		logger.Error("failed to parse migrations", "error", err)
		os.Exit(1)
	}

	if err := database.Migrate(ctx, pool, list, logger); err != nil {
		logger.Error("migration failed", "error", err)
		os.Exit(1)
	}

	logger.Info("migrations up to date")
}
