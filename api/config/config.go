package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	DBSSLMode      string
	APIPort        string
	FrontendURL    string
	ScanWorkspace  string
	GithubToken    string
	AnthropicKey   string
}

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

// DSN returns the PostgreSQL connection string.
func (c *Config) DSN() string {
	return "postgres://" + c.DBUser + ":" + c.DBPassword +
		"@" + c.DBHost + ":" + c.DBPort +
		"/" + c.DBName + "?sslmode=" + c.DBSSLMode
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
