package services

import (
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

type WebSocketService interface {
	HandleConnection(conn *websocket.Conn)
	HandleWebSocket(c *fiber.Ctx) error
	BroadcastToUser(userID string, content any)
	BroadcastToOrg(orgID string, content any)
	BroadcastToRoom(roomID string, content any)
}

type WebSocketMessage struct {
	Type     string `json:"type"`
	ClientID string `json:"clientId"`
	Content  string `json:"content,omitempty"`
}

type WebSocketConnection struct {
	ID             string
	UserID         pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	Conn           *websocket.Conn
	Send           chan []byte
	Rooms          []string
}
