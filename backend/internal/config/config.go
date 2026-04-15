package config

import (
	"log/slog"
	"os"
)

// Config holds all values the app needs to run.
// Everything comes from environment variables — never hardcoded.
// Think of this like your .env + a typed wrapper in PHP (e.g. Laravel's config/).
type Config struct {
	Port        string // HTTP port the server listens on         e.g. "8080"
	DatabaseURL string // Full Postgres DSN                       e.g. "postgres://user:pass@host:5432/dbname?sslmode=disable"
	JWTSecret   string // Secret key used to sign/verify JWTs     e.g. "some-long-random-string"
	Env         string // "development" | "production"            controls log verbosity etc.
}

// Load reads every required env var and returns a populated Config.
// If a required variable is missing we log a fatal error and exit immediately.
// Optional variables fall back to sensible defaults.
//
// Call this once in main() right after godotenv.Load().
func Load() *Config {
	cfg := &Config{
		// Required — no default, app cannot run without these
		DatabaseURL: requireEnv("DATABASE_URL"),
		JWTSecret:   requireEnv("JWT_SECRET"),

		// Optional — safe defaults provided
		Port: getEnvOrDefault("PORT", "8080"),
		Env:  getEnvOrDefault("APP_ENV", "development"),
	}

	// Warn loudly if someone ships a weak JWT secret.
	// 32 chars ≈ 256 bits — anything shorter is too easy to brute-force.
	if len(cfg.JWTSecret) < 32 {
		slog.Warn("JWT_SECRET is shorter than 32 characters — use a longer secret in production")
	}

	slog.Info("Config loaded",
		"port", cfg.Port,
		"env", cfg.Env,
		// Never log DatabaseURL or JWTSecret — they contain credentials.
	)

	return cfg
}

// requireEnv reads an env var and exits the process if it is empty or unset.
// This gives a clear error at startup instead of a confusing nil-pointer panic later.
func requireEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		// slog.Error then os.Exit — same effect as log.Fatalf but uses structured logger.
		slog.Error("Required environment variable is missing", "key", key)
		os.Exit(1)
	}
	return val
}

// getEnvOrDefault reads an env var and returns fallback if it is empty or unset.
func getEnvOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}