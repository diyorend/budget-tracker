package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/redis/go-redis/v9"

	"github.com/diyorend/budget-tracker/internal/alert"
	"github.com/diyorend/budget-tracker/internal/config"
	"github.com/diyorend/budget-tracker/internal/handler"
	"github.com/diyorend/budget-tracker/internal/middleware"
	"github.com/diyorend/budget-tracker/internal/repository"
	"github.com/diyorend/budget-tracker/internal/service"
)

func main() {
	// --- Config ---
	cfg := config.Load()

	// --- Structured logging ---
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// --- Database ---
	ctx := context.Background()
	dbpool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}
	defer dbpool.Close()

	if err := dbpool.Ping(ctx); err != nil {
		slog.Error("database ping failed", "err", err)
		os.Exit(1)
	}
	slog.Info("database connected")

	// --- Redis ---
	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		slog.Error("invalid redis URL", "err", err)
		os.Exit(1)
	}
	rdb := redis.NewClient(redisOpts)
	if err := rdb.Ping(ctx).Err(); err != nil {
		slog.Error("redis ping failed", "err", err)
		os.Exit(1)
	}
	defer rdb.Close()
	slog.Info("redis connected")

	// --- Repositories ---
	// Concrete types here; everything downstream (services) accepts the
	// repository.XStore interfaces, so these just happen to satisfy them.
	userRepo := repository.NewUserRepo(dbpool)
	txRepo := repository.NewTransactionRepo(dbpool)
	budgetRepo := repository.NewBudgetRepo(dbpool)

	// --- Alert broker ---
	broker := alert.NewBroker(rdb)

	// --- Services ---
	authSvc := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTExpiryHours)
	txSvc := service.NewTransactionService(txRepo, budgetRepo, broker)
	budgetSvc := service.NewBudgetService(budgetRepo, txRepo)

	// --- Handlers ---
	authHandler := handler.NewAuthHandler(authSvc)
	txHandler := handler.NewTransactionHandler(txSvc)
	budgetHandler := handler.NewBudgetHandler(budgetSvc)
	wsHandler := handler.NewWSHandler(broker, budgetSvc)

	// --- Echo router ---
	e := echo.New()
	e.HideBanner = true

	// Global middleware
	e.Use(echomiddleware.Logger())
	e.Use(echomiddleware.Recover())
	e.Use(echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:5173", "http://localhost:3000", "http://localhost"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAuthorization},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
	}))

	// Health check (no auth — used by Docker healthcheck)
	e.GET("/health", func(c echo.Context) error {
		if err := dbpool.Ping(c.Request().Context()); err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "db_down"})
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Public routes
	e.POST("/api/auth/register", authHandler.Register)
	e.POST("/api/auth/login", authHandler.Login)

	// Protected routes
	api := e.Group("/api", middleware.JWT(authSvc))
	api.GET("/transactions", txHandler.List)
	api.POST("/transactions", txHandler.Create)
	api.POST("/budgets", budgetHandler.Upsert)
	api.GET("/budgets/status", budgetHandler.GetStatus)

	// WebSocket — JWT middleware now also accepts ?token=, see internal/middleware/jwt.go
	e.GET("/ws", wsHandler.Connect, middleware.JWT(authSvc))

	// --- Start broker in background ---
	brokerCtx, brokerCancel := context.WithCancel(ctx)
	defer brokerCancel()
	go broker.Run(brokerCtx)

	// --- Start server ---
	go func() {
		slog.Info("server starting", "port", cfg.Port)
		if err := e.Start(":" + cfg.Port); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
		}
	}()

	// --- Graceful shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down...")
	brokerCancel()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutCancel()

	if err := e.Shutdown(shutCtx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
	slog.Info("server stopped")
}
