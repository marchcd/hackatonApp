package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"hackatonApp/internal/bot"
	"hackatonApp/internal/config"
	"hackatonApp/internal/handlers"
	"hackatonApp/internal/ingest"
	"hackatonApp/internal/repository"
	"hackatonApp/internal/service"
	"hackatonApp/internal/source"
	"hackatonApp/internal/source/kultura"
	"hackatonApp/internal/source/timepad"
)

func main() {
	// 1. Logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 2. Config load
	cfg := config.Load()
	slog.Info("Configuration loaded successfully",
		slog.String("port", cfg.ServerPort),
	)
	slog.Info("Checking bot token", "len", len(cfg.MaxBotToken))

	// 3. PostgreSQL initialization
	dbpool, err := repository.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("Failed to connect to PostgreSQL", "error", err)
		os.Exit(1)
	}
	slog.Info("Connected to PostgreSQL successfully")

	// 4. Migration
	slog.Info("Running database migrations...")
	if err := repository.RunMigrations(ctx, dbpool); err != nil {
		slog.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("Migrations applied successfully")

	// Routes
	mainMux := http.NewServeMux()

	repo := repository.NewPostgresRepo(dbpool)
	eventService := service.NewEventService(repo)
	maxBot, err := bot.New(cfg.MaxBotToken, eventService, cfg.MiniAppURL)
	if err != nil {
		slog.Error("max bot init failed", "error", err)
		os.Exit(1)
	}

	eventsHandler := handlers.NewEventService(eventService, cfg.MaxBotToken, cfg.SkipInitDataCheck)
	eventsHandler.RegisterRoutes(mainMux)

	fileServer := http.FileServer(http.Dir("./static"))
	mainMux.Handle("/", fileServer)

	sources := []source.EventSource{
		timepad.NewClient(cfg.TimepadToken),
		kultura.NewClient(cfg.KulturaAPIKey),
	}

	sched := ingest.NewScheduler(sources, repo, cfg.IngestCities, cfg.IngestInterval)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		sched.Run(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		maxBot.Run(ctx)
	}()

	// Creating server configuration
	server := &http.Server{
		Addr:         cfg.ServerPort,
		Handler:      corsMiddleware(mainMux),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	serverErrors := make(chan error, 1)

	go func() {
		slog.Info("Server is starting", slog.String("port", cfg.ServerPort))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	select {
	case err := <-serverErrors:
		slog.Error("Server starting error", "erorr", err)
		stop()
	case <-ctx.Done():
		slog.Info("Shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Couldn't stop server graceffully", "error", err)
		if err := server.Close(); err != nil {
			slog.Error("Couldn't close server forced", "error", err)
		}
	}

	slog.Info("Waiting for background ingest to finish current cycle...")
	wg.Wait()

	slog.Info("Closing PostgreSQL connection pool...")
	dbpool.Close()

	slog.Info("Server stopped gracefully")
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
