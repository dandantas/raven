package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dandantas/raven/internal/config"
	"github.com/dandantas/raven/internal/database"
	"github.com/dandantas/raven/internal/handler"
	"github.com/dandantas/raven/internal/scheduler"
	"github.com/dandantas/raven/internal/service"
	"github.com/dandantas/raven/internal/webhook"
	"github.com/dandantas/raven/pkg/middleware"
)

const version = "1.0.0"

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	config.InitLogger(cfg)

	slog.Info("Starting Raven Alert Service", "version", version)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to MongoDB
	db, err := database.Connect(ctx, cfg.MongoURI, cfg.MongoDatabase, cfg.MongoTimeout)
	if err != nil {
		slog.Error("Failed to connect to MongoDB", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Disconnect(context.Background()); err != nil {
			slog.Error("Failed to disconnect from MongoDB", "error", err)
		}
	}()

	// Create indexes
	if err := database.CreateIndexes(ctx, db); err != nil {
		slog.Error("Failed to create indexes", "error", err)
		os.Exit(1)
	}

	// Initialize repositories
	healthCheckRepo := database.NewHealthCheckRepository(db)
	executionRepo := database.NewExecutionRepository(db)
	alertRepo := database.NewAlertRepository(db)
	lockRepo := database.NewLockRepository(db)

	// Initialize services
	healthCheckService := service.NewHealthCheckService(healthCheckRepo)
	executionService := service.NewExecutionService(executionRepo)
	alertService := service.NewAlertService(alertRepo)

	// Initialize HTTP client and webhook dispatcher
	httpClient := service.NewHTTPClient(cfg.DefaultAPITimeout)
	webhookDispatcher := webhook.NewDispatcher(cfg.DefaultWebhookTimeout)

	// Initialize executor
	executor := service.NewExecutor(
		httpClient,
		webhookDispatcher,
		healthCheckRepo,
		executionRepo,
		alertRepo,
	)

	// Initialize async executor
	asyncExecutor := service.NewAsyncExecutor(executor)

	// Initialize scheduler
	sched := scheduler.NewScheduler(cfg, executor, lockRepo, healthCheckRepo)
	sched.Start(ctx)

	// Initialize handlers
	healthCheckHandler := handler.NewHealthCheckHandler(healthCheckService)
	executionHandler := handler.NewExecutionHandler(executor, asyncExecutor)
	historyHandler := handler.NewHistoryHandler(executionService)
	alertHandler := handler.NewAlertHandler(alertService)
	healthHandler := handler.NewHealthHandler(db, version)

	// Create CORS config
	corsConfig := middleware.CORSConfig{
		AllowedOrigins:   cfg.CORSAllowedOrigins,
		AllowedMethods:   cfg.CORSAllowedMethods,
		AllowedHeaders:   cfg.CORSAllowedHeaders,
		AllowCredentials: cfg.CORSAllowCredentials,
		MaxAge:           cfg.CORSMaxAge,
	}

	// Create router
	router := handler.NewRouter(
		healthCheckHandler,
		executionHandler,
		historyHandler,
		alertHandler,
		healthHandler,
		corsConfig,
	)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      router.Handler(),
		ReadTimeout:  cfg.HTTPReadTimeout,
		WriteTimeout: cfg.HTTPWriteTimeout,
	}

	// Start server in goroutine
	go func() {
		slog.Info("Starting HTTP server", "port", cfg.HTTPPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	slog.Info("Received shutdown signal, initiating graceful shutdown")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop scheduler first (wait for in-flight executions)
	slog.Info("Stopping scheduler...")
	sched.Stop(shutdownCtx)

	// Shutdown HTTP server
	slog.Info("Shutting down HTTP server...")
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	}

	slog.Info("Raven Alert Service stopped")
}
