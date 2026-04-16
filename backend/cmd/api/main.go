package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"

	"github.com/RakeshAero/taskflow-rakeshbalu/backend/internal/config"
	"github.com/RakeshAero/taskflow-rakeshbalu/backend/internal/db"
	"github.com/RakeshAero/taskflow-rakeshbalu/backend/internal/handlers"
	authmw "github.com/RakeshAero/taskflow-rakeshbalu/backend/internal/middleware"
	"github.com/RakeshAero/taskflow-rakeshbalu/backend/internal/repository"
)

func main() {
	// ─── 1. Logger ────────────────────────────────────────────────────────────
	// slog is Go's built-in structured logger (like Monolog in PHP).
	// We write JSON lines to stdout so Docker can capture them.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// ─── 2. Load .env ─────────────────────────────────────────────────────────
	// godotenv.Load() reads the .env file and sets OS environment variables.
	// It's safe to ignore the error in production (env vars already injected by Docker).
	if err := godotenv.Load(); err != nil {
		slog.Info("No .env file found, reading from environment")
	}

	// ─── 3. Config ────────────────────────────────────────────────────────────
	// config.Load() reads all required env vars and returns a typed Config struct.
	// It calls log.Fatal internally if any required var is missing.
	cfg := config.Load()

	// ─── 4. Database ──────────────────────────────────────────────────────────
	// db.Connect() opens a connection pool and pings the DB.
	// If the DB isn't reachable, we fail fast here — better than crashing mid-request.
	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer database.Close()
	slog.Info("Database connection established")

	// ─── 5. Repositories ──────────────────────────────────────────────────────
	// Repositories hold all SQL queries. Think of them as your PDO query classes.
	userRepo := repository.NewUserRepository(database)
	projectRepo := repository.NewProjectRepository(database)
	taskRepo := repository.NewTaskRepository(database)

	// // 6. Handlers also Controllers
	authHandler := handlers.NewAuthHandler(userRepo, cfg.JWTSecret)
	projectHandler := handlers.NewProjectHandler(projectRepo)
	taskHandler := handlers.NewTaskHandler(taskRepo, projectRepo)
	healthHandler := handlers.NewHealthHandler(database)


	// ─── 7. Router
	route := chi.NewRouter()
	// Global middleware — runs on every request
	route.Use(middleware.RequestID)  // Adds X-Request-Id header (useful for tracing)
	route.Use(middleware.RealIP)     // Reads real client IP from X-Forwarded-For
	route.Use(requestLogger(logger)) // Our custom structured logger (defined below)
	route.Use(middleware.Recoverer)  // Catches panics and returns 500 instead of crashing

	// ─── 8. Routes
	// Health check — no auth required. Used by Docker and load balancers.
	route.Get("/health", healthHandler.Check)

	// // Public routes — no JWT required
	route.Post("/auth/register", authHandler.Register)
	route.Post("/auth/login", authHandler.Login)

	// Protected routes — JWT required
	// chi.Group lets us apply middleware to a subset of routes only.
	route.Group(func(r chi.Router) {
		// authmw.Authenticate parses the Bearer token and injects user_id into context.
		// Any route inside this group will return 401 if the token is missing or invalid.
		r.Use(authmw.Authenticate(cfg.JWTSecret))

		// Projects
		r.Get("/projects", projectHandler.List)
		r.Post("/projects", projectHandler.Create)
		r.Get("/projects/{id}", projectHandler.Get)
		r.Patch("/projects/{id}", projectHandler.Update)
		r.Delete("/projects/{id}", projectHandler.Delete)

		// Tasks (nested under projects for create/list)
		r.Get("/projects/{id}/tasks", taskHandler.List)
		r.Post("/projects/{id}/tasks", taskHandler.Create)

		// Tasks (standalone for update/delete — task ID is enough)
		r.Patch("/tasks/{id}", taskHandler.Update)
		r.Delete("/tasks/{id}", taskHandler.Delete)
	})

	// ─── 9. HTTP Server + Graceful shutdown ────────────────────────────────────
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           route,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		slog.Info("HTTP server started", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	slog.Info("Shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	slog.Info("Server stopped gracefully")
}

// requestLogger returns a chi middleware that logs every request using slog.
// We write this ourselves because chi's built-in logger uses fmt.Printf,
// not slog — so log lines would be inconsistent with the rest of the app.
func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// chi's WrapResponseWriter lets us capture the status code after
			// the handler runs (http.ResponseWriter doesn't expose it by default).
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			logger.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", middleware.GetReqID(r.Context()),
			)
		})
	}
}
