package handlers

import "github.com/gofiber/fiber/v2"

func Liveness(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "ok",
	})
}

func Readiness(c *fiber.Ctx) error {
	// TODO: Check dependencies (Vault, IdP, etc.)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "ready",
	})
}
