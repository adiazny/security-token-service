package handlers

import "github.com/gofiber/fiber/v2"

type TokenExchangeRequest struct {
	RequestedTokenType string `json:"requested_token_type"`
	Audience           string `json:"audience"`
}

func ExchangeToken(c *fiber.Ctx) error {
	var req TokenExchangeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// TODO:
	// 1. Validate Input
	// 2. Validate Incoming Token (from Header)
	// 3. Fetch Roles (Enrichment)
	// 4. Mint New Token

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"access_token":      "mock_token",
		"token_type":        "Bearer",
		"issued_token_type": "urn:ietf:params:oauth:token-type:jwt",
		"expires_in":        3600,
	})
}
