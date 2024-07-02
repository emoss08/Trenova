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

func (h *WebsocketHandler) HandleConnection(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		c.Locals("allowed", true)
		return c.Next()
	}

	return fiber.ErrUpgradeRequired
}

func (h *WebsocketHandler) HandleWebsocketConnection(c *websocket.Conn) {
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

func (h *WebsocketHandler) Stop() {
	// Call the service's stop function
	h.service.Stop()
	h.logger.Info().Msg("Websocket handler stopped")
}
