package api

import (
	"github.com/adiazny/security-token-service/internal/api/handlers"
	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App) {
	// Health Endpoints
	app.Get("/livez", handlers.Liveness)
	app.Get("/readyz", handlers.Readiness)

	// JWKS Endpoint
	app.Get("/.well-known/jwks.json", handlers.GetJWKS)

	// API Version 1
	v1 := app.Group("/v1")

	// Token Exchange Endpoint
	v1.Post("/token/exchange", handlers.ExchangeToken)
}
