package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
)

// GrantTypeTokenExchange is the required value for the grant_type parameter
const GrantTypeTokenExchange = "urn:ietf:params:oauth:grant-type:token-exchange"

// TokenExchangeRequest represents an RFC 8693 Token Exchange Request
// Parameters are sent as application/x-www-form-urlencoded
type TokenExchangeRequest struct {
	// REQUIRED. The value "urn:ietf:params:oauth:grant-type:token-exchange"
	GrantType string `json:"grant_type" form:"grant_type"`

	// OPTIONAL. Indicates the location of the target service or resource.
	// May be repeated.
	Resource string `json:"resource,omitempty" form:"resource"`

	// OPTIONAL. The logical name of the target service where the client intends to use the requested security token.
	// May be repeated.
	Audience string `json:"audience,omitempty" form:"audience"`

	// OPTIONAL. A list of space-delimited, case-sensitive strings.
	Scope []string `json:"scope,omitempty" form:"scope"`

	// OPTIONAL. An identifier for the type of the requested security token.
	RequestedTokenType string `json:"requested_token_type,omitempty" form:"requested_token_type"`

	// REQUIRED. A security token that represents the identity of the party on behalf of whom the request is being made.
	SubjectToken string `json:"subject_token" form:"subject_token"`

	// REQUIRED. An identifier that indicates the type of the security token in the "subject_token" parameter.
	SubjectTokenType string `json:"subject_token_type" form:"subject_token_type"`

	// OPTIONAL. A security token that represents the identity of the acting party.
	ActorToken string `json:"actor_token,omitempty" form:"actor_token"`

	// OPTIONAL. An identifier that indicates the type of the security token in the "actor_token" parameter.
	ActorTokenType string `json:"actor_token_type,omitempty" form:"actor_token_type"`
}

// Validate ensures the request meets RFC 8693 requirements
func (r *TokenExchangeRequest) Validate() error {
	if r.GrantType != GrantTypeTokenExchange {
		return errors.New("invalid grant_type: must be " + GrantTypeTokenExchange)
	}

	if r.SubjectToken == "" {
		return errors.New("subject_token is required")
	}

	if r.SubjectTokenType == "" {
		return errors.New("subject_token_type is required")
	}

	// If actor_token is present, actor_token_type is required
	if r.ActorToken != "" && r.ActorTokenType == "" {
		return errors.New("actor_token_type is required when actor_token is present")
	}

	return nil
}

func ExchangeToken(c *fiber.Ctx) error {
	var req TokenExchangeRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	err := req.Validate()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// TODO:
	// wire up the exchanger

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"access_token":      "mock_token",
		"token_type":        "Bearer",
		"issued_token_type": "urn:ietf:params:oauth:token-type:jwt",
		"expires_in":        3600,
	})
}
