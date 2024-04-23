package handlers

import (
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

// WebsocketHandler is a struct that manages websocket connections and communication between clients.
type WebsocketHandler struct {
	Logger  *zerolog.Logger
	Service *services.WebsocketService
}

// NewWebsocketHandler creates a new handler for managing websocket connections.
// It initializes the handler with a logger and a client to interact with the database.
//
// Parameters:
//
//	logger *zerolog.Logger: A pointer to a logger instance used for logging messages in the handler.
//	client *ent.Client: A pointer to the client instance used to interact with the database.
//
// Returns:
//
//	*WebsocketHandler: A pointer to the newly created WebsocketHandler instance.
func NewWebsocketHandler(logger *zerolog.Logger, client *ent.Client) *WebsocketHandler {
	return &WebsocketHandler{
		Logger:  logger,
		Service: services.NewWebsocketService(client, logger),
	}
}

// HandleConnection checks if the incoming request is a websocket upgrade request.
// If it is, it allows the connection to be upgraded to a websocket connection.
//
// Parameters:
//
//	c *fiber.Ctx: The context object representing the incoming HTTP request.
//
// Returns:
//
//	error: An error if the request is not a websocket upgrade request, nil otherwise.
func (h *WebsocketHandler) HandleConnection(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		c.Locals("allowed", true)
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}

// HandleWebsocketConnection manages the websocket connection with a client.
// It reads messages from the client and broadcasts them to all connected clients.
// The connection is closed when an error occurs or the client disconnects.
//
// Parameters:
//
//	c *websocket.Conn: The websocket connection object representing the connection with the client.
func (h *WebsocketHandler) HandleWebsocketConnection(c *websocket.Conn) {
	id := c.Params("id")
	allowed, _ := c.Locals("allowed").(bool)

	if c == nil {
		h.Logger.Error().Msg("Websocket connection is nil")
	}
	if !allowed {
		_ = c.Close()
		h.Logger.Error().Msg("Websocket connection is not allowed")
		return
	}

	h.Service.RegisterClient(id, c)
	defer h.Service.UnregisterClient(id)

	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			h.Logger.Error().Err(err).Msg("Failed to read message from client")
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
