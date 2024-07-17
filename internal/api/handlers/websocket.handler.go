// Copyright (c) 2024 Trenova Technologies, LLC
//
// Licensed under the Business Source License 1.1 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://trenova.app/pricing/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// Key Terms:
// - Non-production use only
// - Change Date: 2026-11-16
// - Change License: GNU General Public License v2 or later
//
// For full license text, see the LICENSE file in the root directory.

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
	h.logger.Info().Msg("Websocket handler stopped")
}
