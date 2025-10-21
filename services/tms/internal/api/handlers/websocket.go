package handlers

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type WebSocketHandlerParams struct {
	fx.In

	Service services.WebSocketService
}

type WebSocketHandler struct {
	service services.WebSocketService
}

func NewWebSocketHandler(p WebSocketHandlerParams) *WebSocketHandler {
	return &WebSocketHandler{
		service: p.Service,
	}
}

func (h *WebSocketHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/ws/notifications", h.service.HandleWebSocket)
}
