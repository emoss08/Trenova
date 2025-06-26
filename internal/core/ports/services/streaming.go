package services

import (
	"context"

	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/gofiber/fiber/v2"
)

// StreamingService defines the interface for real-time streaming services
type StreamingService interface {
	// StreamData provides real-time streaming of data changes via SSE
	StreamData(
		ctx *fiber.Ctx,
		streamKey string,
		dataFetcher DataFetcher,
		timestampExtractor TimestampExtractor,
	) error

	// BroadcastToStream immediately broadcasts data to all clients of a specific stream
	BroadcastToStream(streamKey string, orgID, buID string, data any) error

	// GetActiveStreams returns the number of active streams for a given key
	GetActiveStreams(streamKey string) int

	// Shutdown gracefully shuts down all active streams
	Shutdown() error
}

// DataFetcher is a function that fetches data for streaming (used for fallback polling)
type DataFetcher func(ctx context.Context, reqCtx *appctx.RequestContext) (any, error)

// TimestampExtractor extracts timestamp from data items for change detection
type TimestampExtractor func(item any) int64

// StreamConfig contains configuration for streaming
type StreamConfig struct {
	// PollInterval is the interval between data fetches
	PollInterval int64 `json:"pollInterval"`
	// MaxConnections is the maximum number of concurrent connections per stream
	MaxConnections int `json:"maxConnections"`
	// StreamTimeout is the maximum duration for a stream connection
	StreamTimeout int64 `json:"streamTimeout"`
	// EnableHeartbeat enables periodic heartbeat messages
	EnableHeartbeat bool `json:"enableHeartbeat"`
}

// StreamMetrics contains metrics for a streaming endpoint
type StreamMetrics struct {
	// ActiveConnections is the current number of active connections
	ActiveConnections int `json:"activeConnections"`
	// TotalConnections is the total number of connections since startup
	TotalConnections int64 `json:"totalConnections"`
	// DataFetchErrors is the number of data fetch errors
	DataFetchErrors int64 `json:"dataFetchErrors"`
	// LastDataFetch is the timestamp of the last successful data fetch
	LastDataFetch int64 `json:"lastDataFetch"`
}
