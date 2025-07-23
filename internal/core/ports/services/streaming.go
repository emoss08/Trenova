// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package services

import (
	"github.com/gofiber/fiber/v2"
)

// StreamingService defines the interface for real-time streaming services via CDC
type StreamingService interface {
	// StreamData establishes an SSE connection for real-time CDC event streaming
	StreamData(ctx *fiber.Ctx, streamKey string) error

	// BroadcastToStream immediately broadcasts CDC events to all clients of a specific stream
	BroadcastToStream(streamKey string, orgID, buID string, data any) error

	// GetActiveStreams returns the number of active streams for a given key
	GetActiveStreams(streamKey string) int

	// Shutdown gracefully shuts down all active streams
	Shutdown() error
}

// StreamConfig contains configuration for CDC-based streaming
type StreamConfig struct {
	// MaxConnections is the maximum number of concurrent connections per stream
	MaxConnections int `json:"maxConnections"`
	// StreamTimeout is the maximum duration for a stream connection (0 for no timeout)
	StreamTimeout int64 `json:"streamTimeout"`
	// EnableHeartbeat enables periodic heartbeat messages to keep connections alive
	EnableHeartbeat bool `json:"enableHeartbeat"`
}

// StreamMetrics contains metrics for a CDC streaming endpoint
type StreamMetrics struct {
	// ActiveConnections is the current number of active connections
	ActiveConnections int `json:"activeConnections"`
	// TotalConnections is the total number of connections since startup
	TotalConnections int64 `json:"totalConnections"`
}
