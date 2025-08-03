/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package websocket

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In

	Logger              *logger.Logger
	NotificationService services.NotificationService
	WebSocketService    services.WebSocketService
}

type Handler struct {
	l                   *zerolog.Logger
	notificationService services.NotificationService
	webSocketService    services.WebSocketService
}

type Message struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

func NewHandler(p HandlerParams) *Handler {
	log := p.Logger.With().
		Str("handler", "websocket").
		Logger()

	return &Handler{
		l:                   &log,
		notificationService: p.NotificationService,
		webSocketService:    p.WebSocketService,
	}
}

func (h *Handler) RegisterRoutes(r fiber.Router) {
	r.Use("/ws", h.webSocketService.HandleWebSocket)
	r.Get("/ws/notifications", websocket.New(h.webSocketService.HandleConnection))

	// Test endpoints for development
	r.Post("/test/notification", h.TestNotification)
	r.Post("/test/org-notification", h.TestOrgNotification)
}
