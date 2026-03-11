package config

import (
	"os" // Environment variables are the primary configuration mechanism.

	"github.com/joho/godotenv" // Dev convenience: load key/value pairs from a local .env file.
)

// Config holds runtime configuration for the API.
//
// Why environment-based config:
// - Fits 12-factor deployments (containers, CI/CD, secrets managers).
// - Keeps secrets (tokens/keys) out of source control.
// - Avoids bespoke config file parsing in the server.
type Config struct {
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	DBSSLMode     string
	APIPort       string
	FrontendURL   string
	ScanWorkspace string
	GithubToken   string
	AnthropicKey  string
}

// Load builds a Config from environment variables (and optionally a `.env` file).
//
// We attempt to load `../.env` because the API code lives in `api/` while local
// developer configs often live at the repo root. In production, this is typically
// a no-op and config comes from the process environment.
func Load() *Config {
	// Load .env from project root (one level up from api/)
	godotenv.Load("../.env")

	return &Config{
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "securescan"),
		DBPassword:    getEnv("DB_PASSWORD", "securescan"),
		DBName:        getEnv("DB_NAME", "securescan"),
		DBSSLMode:     getEnv("DB_SSLMODE", "disable"),
		APIPort:       getEnv("API_PORT", "3000"),
		FrontendURL:   getEnv("FRONTEND_URL", "http://localhost:5173"),
		ScanWorkspace: getEnv("SCAN_WORKSPACE", "/tmp/securescan"),
		GithubToken:   getEnv("GITHUB_TOKEN", ""),
		AnthropicKey:  getEnv("ANTHROPIC_API_KEY", ""),
	}
}

// DSN builds a PostgreSQL connection string from the Config fields.
//
// We keep DSN assembly in code (rather than a single DSN env var) so local defaults
// are ergonomic while still allowing full override via environment variables.
func (c *Config) DSN() string {
	return "postgres://" + c.DBUser + ":" + c.DBPassword +
		"@" + c.DBHost + ":" + c.DBPort +
		"/" + c.DBName + "?sslmode=" + c.DBSSLMode
}

// getEnv returns the environment value for key, or fallback if unset/empty.
//
// Using a helper keeps defaults centralized and avoids scattering os.Getenv calls.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
