package api

import (
	"github.com/adiazny/security-token-service/internal/core"
	"github.com/gofiber/fiber/v2"
)

/*
	Driving Adapter
*/

const (
	tokenExchangeEndpoint = "/token/exchange"
	apiV1                 = "/v1"
)

type Controller struct {
	server *fiber.App
	sts    core.TokenExchanger
}

func NewController(server *fiber.App, sts core.TokenExchanger) *Controller {
	return &Controller{server: server, sts: sts}
}

func (c *Controller) SetupRoutes() {
	// API Version 1
	v1 := c.server.Group(apiV1)

	v1.Post(tokenExchangeEndpoint, c.HandleTokenExchange)
}

func (c *Controller) HandleTokenExchange(ctx *fiber.Ctx) error {
	return nil
}
