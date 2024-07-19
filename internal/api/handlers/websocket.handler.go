// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

package handlers

import (
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/server"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

type WebsocketHandler struct {
	server              *server.Server
	logger              *zerolog.Logger
	service             *services.WebsocketService
	notificationService *services.UserNotificationService
}

func NewWebsocketHandler(s *server.Server) *WebsocketHandler {
	return &WebsocketHandler{
		server:              s,
		logger:              s.Logger,
		service:             services.NewWebsocketService(s),
		notificationService: services.NewUserNotificationService(s),
	}
}

func (h WebsocketHandler) HandleConnection(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		c.Locals("allowed", true)
		return c.Next()
	}

	return fiber.ErrUpgradeRequired
}

func (h WebsocketHandler) HandleWebsocketConnection(c *websocket.Conn) {
	id := c.Params("id")
	allowed, _ := c.Locals("allowed").(bool)

	if c == nil {
		h.logger.Error().Msg("WebsocketHandler: Connection not allowed")
		return
	}

	if !allowed {
		_ = c.Close()
		h.logger.Error().Msg("WebsocketHandler: Connection not allowed")
		return
	}

	h.service.RegisterClient(id, c)
	defer h.service.UnregisterClient(id)

	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			h.logger.Error().Err(err).Msg("WebsocketHandler: Error reading message")
			break
		}

		serviceMsg := services.Message{
			Type:     "message",
			Content:  string(msg),
			ClientID: id,
		}

		h.service.NotifyAllClients(serviceMsg, id)
	}
}

func (h WebsocketHandler) Stop() {
	// Call the service's stop function
	h.service.Stop()
	h.logger.Debug().Msg("Websocket handler stopped")
}
