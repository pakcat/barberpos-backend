package main

import (
    "context"
    "log/slog"
    "os"
    "os/signal"
    "syscall"

    "barberpos-backend/internal/config"
    "barberpos-backend/internal/db"
    "barberpos-backend/internal/handler"
    "barberpos-backend/internal/server"
)

func main() {
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

    cfg, err := config.Load()
    if err != nil {
        logger.Error("failed to load config", "err", err)
        os.Exit(1)
    }

    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    pg, err := db.New(ctx, cfg)
    if err != nil {
        logger.Error("failed to connect database", "err", err)
        os.Exit(1)
    }
    defer pg.Close()

    healthHandler := handler.HealthHandler{DB: pg}
    router := server.NewRouter(healthHandler)

    if err := server.Start(ctx, cfg, router, logger); err != nil {
        logger.Error("server error", "err", err)
        os.Exit(1)
    }
}
