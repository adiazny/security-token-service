package handlers

import "github.com/gofiber/fiber/v2"

func GetJWKS(c *fiber.Ctx) error {
	// TODO: Retrieve public keys from Key Manager
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"keys": []interface{}{},
	})
}
