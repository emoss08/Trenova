package handlers

import (
	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

type WebsocketHandler struct {
	Server  *api.Server
	Service *services.WebsocketService
}

// NewWebsocketHandler creates a new handler for managing websocket connections.
func NewWebsocketHandler(s *api.Server) *WebsocketHandler {
	return &WebsocketHandler{
		Server:  s,
		Service: services.NewWebsocketService(s),
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
		h.Server.Logger.Error().Msg("Websocket connection is nil")
	}
	if !allowed {
		_ = c.Close()
		h.Server.Logger.Error().Msg("Websocket connection is not allowed")
		return
	}

	h.Service.RegisterClient(id, c)
	defer h.Service.UnregisterClient(id)

	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			h.Server.Logger.Error().Err(err).Msg("Failed to read message from client")
			break
		}

		serviceMsg := services.Message{
			Type:     "message",
			Content:  string(msg),
			ClientID: id,
		}

		h.Service.NotifyAllClients(serviceMsg, id) // Where `id` is the sender's ID
	}
}
