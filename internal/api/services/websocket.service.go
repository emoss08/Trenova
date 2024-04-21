package services

import (
	"sync"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/gofiber/contrib/websocket"
	"github.com/rs/zerolog"
)

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

type WebsocketService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewWebsocketService creates a new comment type service.
func NewWebsocketService(s *api.Server) *WebsocketService {
	return &WebsocketService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

func (ws *WebsocketService) RegisterClient(id string, conn *websocket.Conn) {
	mu.Lock()
	clients[id] = conn
	mu.Unlock()
	ws.Logger.Debug().Msgf("Client %s registered", id)
}

func (ws *WebsocketService) UnregisterClient(id string) {
	mu.Lock()
	if conn, ok := clients[id]; ok {
		_ = conn.Close() // Attempt to close the websocket connection gracefully
		delete(clients, id)
	}
	mu.Unlock()
	ws.Logger.Debug().Msgf("Client %s unregistered", id)
}

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

// NotifyAllClients now excludes the sender to prevent echo loops.
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
