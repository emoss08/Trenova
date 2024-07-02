package handlers

import "github.com/gofiber/fiber/v2"

// StandardHandler interface for handlers with get, create, and update functions
type StandardHandler interface {
	RegisterRoutes(r fiber.Router)
	Get() fiber.Handler
	Create() fiber.Handler
	Update() fiber.Handler
}

// FlexibleHandler interface for handlers with custom routes
type FlexibleHandler interface {
	RegisterRoutes(r fiber.Router)
}
