package services

import (
	"sync"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/gofiber/contrib/websocket"
	"github.com/rs/zerolog"
)

// Message represents a message sent over the websocket connection.
type Message struct {
	Type     string `json:"type"`
	Content  string `json:"content"`
	ClientID string `json:"clientId,omitempty"` // optional field
}

var (
	// clients stores the active clients with their IDs as keys
	clients = make(map[string]*websocket.Conn)
	// mutex to synchronize access to the clients map
	mu sync.Mutex
)

// WebsocketService is a struct that manages websocket connections and communication between clients.
type WebsocketService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewWebsocketService creates a new websocket service.
// It initializes the service with a client to interact with the database and a logger.
//
// Parameters:
//
//	client *ent.Client: A pointer to the client instance used to interact with the database.
//	logger *zerolog.Logger: A pointer to a logger instance used for logging messages in the service.
//
// Returns:
//
//	*WebsocketService: A pointer to the newly created WebsocketService instance.
func NewWebsocketService(client *ent.Client, logger *zerolog.Logger) *WebsocketService {
	return &WebsocketService{
		Client: client,
		Logger: logger,
	}
}

// RegisterClient registers a new client with the given ID and websocket connection.
//
// Parameters:
//
//	id string: The ID of the client to register.
//	conn *websocket.Conn: The websocket connection object representing the connection with the client.
func (ws *WebsocketService) RegisterClient(id string, conn *websocket.Conn) {
	mu.Lock()
	clients[id] = conn
	mu.Unlock()
	ws.Logger.Debug().Msgf("Client %s registered", id)
}

// UnregisterClient unregisters a client with the given ID.
//
// Parameters:
//
//	id string: The ID of the client to unregister.
func (ws *WebsocketService) UnregisterClient(id string) {
	mu.Lock()
	if conn, ok := clients[id]; ok {
		_ = conn.Close() // Attempt to close the websocket connection gracefully
		delete(clients, id)
	}
	mu.Unlock()
	ws.Logger.Debug().Msgf("Client %s unregistered", id)
}

// NotifyClient sends a message to a specific client.
//
// Parameters:
//
//	clientID string: The ID of the client to send the message to.
//	message Message: The message to send to the client.
func (ws *WebsocketService) NotifyClient(clientID string, message Message) {
	mu.Lock()
	conn, ok := clients[clientID]
	mu.Unlock()
	if ok {
		jsonData, err := sonic.Marshal(message)
		if err != nil {
			ws.Logger.Error().Err(err).Msg("Failed to marshal message")
			return
		}

		if err = conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
			ws.Logger.Error().Err(err).Msgf("Failed to send message to client %s", clientID)
		}
	}
}

// NotifyAllClients sends a message to all connected clients except the sender.
// This method is used to broadcast messages to all clients except the one that initiated the broadcast.
// The senderID parameter is used to exclude the sender from receiving the message.
//
// Parameters:
//
//	msg Message: The message to broadcast to all clients.
//	senderID string: The ID of the client that initiated the broadcast.
func (ws *WebsocketService) NotifyAllClients(msg Message, senderID string) {
	message, err := sonic.Marshal(msg)
	if err != nil {
		ws.Logger.Error().Err(err).Msg("Failed to marshal message")
		return
	}

	mu.Lock()
	for id, conn := range clients {
		if id == senderID { // Skip the sender to avoid sending the message back to them
			continue
		}
		mu.Unlock() // Unlock before sending to reduce lock contention
		if err = conn.WriteMessage(websocket.TextMessage, message); err != nil {
			ws.Logger.Error().Err(err).Msgf("Failed to send message to client %s", id)
		}
		mu.Lock() // Re-lock to continue safely iterating over the map
	}
	mu.Unlock()
}
