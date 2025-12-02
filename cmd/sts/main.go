package main

import (
	"log"

	"github.com/adiazny/security-token-service/internal/api"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	app := fiber.New(fiber.Config{
		AppName: "Security Token Service",
	})

	app.Use(logger.New())

	api.SetupRoutes(app)

	// TODO: implement TLS
	log.Fatal(app.Listen(":3000"))
}
