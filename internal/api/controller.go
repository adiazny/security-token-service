package api

import (
	"github.com/adiazny/security-token-service/internal/core"
	"github.com/gofiber/fiber/v2"
)

/*
	Driving Adapter
*/

type Controller struct {
	server *fiber.App
	sts    core.SecurityTokenService
}

func NewController(server *fiber.App, sts core.SecurityTokenService) *Controller {

	return &Controller{server: server, sts: sts}
}

func (c *Controller) HandleTokenExchange(ctx *fiber.Ctx) error {
	return nil
}
