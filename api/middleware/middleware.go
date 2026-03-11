package middleware

import (
	"github.com/gofiber/fiber/v3"                    // Fiber app type used to register middleware.
	"github.com/gofiber/fiber/v3/middleware/cors"    // Browser clients require CORS for cross-origin API calls.
	"github.com/gofiber/fiber/v3/middleware/logger"  // Request logging for debugging and operational visibility.
	"github.com/gofiber/fiber/v3/middleware/recover" // Prevent panics from crashing the server; returns 500 instead.
)

// Setup configures global HTTP middleware for the API server.
//
// Middleware is applied in a single place so behavior is consistent across routes.
// The current stack focuses on resiliency (recover), observability (logger), and
// allowing the frontend app to call the API from a different origin (CORS).
func Setup(app *fiber.App, frontendURL string) {
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{frontendURL},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	}))
}
