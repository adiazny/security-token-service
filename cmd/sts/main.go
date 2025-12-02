package main

import (
	"log"

	"github.com/adiazny/security-token-service/internal/api"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	// 1. Initialize Fiber app
	app := fiber.New(fiber.Config{
		AppName: "Security Token Service",
	})

	// 2. Add Middleware
	app.Use(recover.New())
	app.Use(logger.New())

	// 3. Setup Routes
	api.SetupRoutes(app)

	// 4. Start Server
	log.Fatal(app.Listen(":3000"))
}
