package services

import (
	"context"

	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gorilla/websocket"

	"github.com/gin-gonic/gin"
)

type WebSocketService interface {
	HandleWebSocket(c *gin.Context)
	BroadcastToUser(ctx context.Context, userID string, content any)
	BroadcastToOrg(ctx context.Context, orgID string, content any)
	BroadcastToRoom(ctx context.Context, roomID string, content any)
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
