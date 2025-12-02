package api

import (
	"github.com/adiazny/security-token-service/internal/api/handlers"
	"github.com/gofiber/fiber/v2"
)

const (
	tokenExchangeEndpoint = "/token/exchange"
	jwksEndpoint          = "/.well-known/jwks.json"
	livenessEndpoint      = "/livez"
	readinessEndpoint     = "/readyz"
)

func SetupRoutes(app *fiber.App) {
	app.Get(livenessEndpoint, handlers.Liveness)
	app.Get(readinessEndpoint, handlers.Readiness)

	app.Get(jwksEndpoint, handlers.GetJWKS)

	// API Version 1
	v1 := app.Group("/v1")

	v1.Post(tokenExchangeEndpoint, handlers.ExchangeToken)
}
