package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joythejaks/palmyield/backend/internal/config"
	"github.com/joythejaks/palmyield/backend/internal/handler"
	authmw "github.com/joythejaks/palmyield/backend/internal/middleware"
	"github.com/joythejaks/palmyield/backend/internal/repository"
	"github.com/joythejaks/palmyield/backend/internal/service"
)

// Per ADR-0004: 15 min access tokens, 30 day rotating refresh tokens.
const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.Load()

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	repo := repository.New(pool)
	authService := service.NewAuthService(repo, cfg.JWTSecret, accessTokenTTL, refreshTokenTTL)
	authHandler := &handler.AuthHandler{Service: authService}

	healthHandler := &handler.HealthHandler{DB: pool}
	r.Get("/healthz", healthHandler.Healthz)
	r.Get("/readyz", healthHandler.Readyz)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)
			r.Post("/logout", authHandler.Logout)

			r.Group(func(r chi.Router) {
				r.Use(authmw.Authenticate(cfg.JWTSecret))
				r.Use(authmw.RequireRole("admin"))
				r.Post("/invite", authHandler.Invite)
			})
		})
	})

	slog.Info("starting server", "port", cfg.Port, "env", cfg.Env)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
