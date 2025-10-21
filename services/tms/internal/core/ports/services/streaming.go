package services

import "github.com/gin-gonic/gin"

type StreamingService interface {
	StreamData(c *gin.Context, streamKey string)
	BroadcastToStream(streamKey string, orgID, buID string, data any) error
	GetActiveStreams(streamKey string) int
	Shutdown() error
}
