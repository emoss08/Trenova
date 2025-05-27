package streamingutils

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/pkg/ctx"
	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

// SSEConfig contains configuration options for Server-Sent Events streaming
type SSEConfig struct {
	// PollInterval is how often to poll for new data (default: 2 seconds)
	PollInterval time.Duration
	// StreamTimeout is the maximum duration for a stream (default: 30 minutes)
	StreamTimeout time.Duration
	// SendHeartbeat indicates whether to send heartbeat events (default: true)
	SendHeartbeat bool
}

// DefaultSSEConfig returns a default SSE configuration
func DefaultSSEConfig() SSEConfig {
	return SSEConfig{
		PollInterval:  2 * time.Second,
		StreamTimeout: 30 * time.Minute,
		SendHeartbeat: true,
	}
}

// DataFetcher is a function type that fetches data for streaming
// It should return new items and the latest timestamp, or an error
type DataFetcher[T any] func(ctx context.Context, reqCtx *ctx.RequestContext, lastTimestamp int64) ([]T, int64, error)

// ItemProcessor is a function type that processes items to determine if they're new
// It should return true if the item is newer than lastTimestamp
type ItemProcessor[T any] func(item T, lastTimestamp int64) (isNew bool, itemTimestamp int64)

// SSEStreamer handles Server-Sent Events streaming with generic data types
type SSEStreamer[T any] struct {
	config        SSEConfig
	dataFetcher   DataFetcher[T]
	itemProcessor ItemProcessor[T]
}

// NewSSEStreamer creates a new SSE streamer with the given configuration and data fetcher
func NewSSEStreamer[T any](
	config SSEConfig,
	dataFetcher DataFetcher[T],
	itemProcessor ItemProcessor[T],
) *SSEStreamer[T] {
	return &SSEStreamer[T]{
		config:        config,
		dataFetcher:   dataFetcher,
		itemProcessor: itemProcessor,
	}
}

// Stream starts the SSE stream on the given Fiber context
func (s *SSEStreamer[T]) Stream(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return err
	}

	// Set SSE headers
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	c.Status(fiber.StatusOK).Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		// Track last timestamp to only send new entries
		lastTimestamp := time.Now().Unix()

		// Send initial connection event
		s.sendEvent(w, "connected", map[string]string{"status": "connected"})
		if err := w.Flush(); err != nil {
			fmt.Printf("Error while flushing initial connection: %v. Closing connection.\n", err)
			return
		}

		// Create a ticker for polling
		ticker := time.NewTicker(s.config.PollInterval)
		defer ticker.Stop()

		// Create a timeout for the entire stream
		streamTimeout := time.After(s.config.StreamTimeout)

		for {
			select {
			case <-streamTimeout:
				fmt.Println("Stream timeout reached, closing connection")
				return
			case <-ticker.C:
				// Fetch new data
				items, latestTimestamp, err := s.dataFetcher(context.Background(), reqCtx, lastTimestamp)
				if err != nil {
					// Send error event but continue streaming
					s.sendEvent(w, "error", map[string]string{"error": "Failed to fetch data"})
					if err := w.Flush(); err != nil {
						fmt.Printf("Error while flushing error: %v. Closing connection.\n", err)
						return
					}
					continue
				}

				// Process and filter new items
				newItems := make([]T, 0)
				newLastTimestamp := lastTimestamp

				for _, item := range items {
					isNew, itemTimestamp := s.itemProcessor(item, lastTimestamp)
					if isNew {
						newItems = append(newItems, item)
						if itemTimestamp > newLastTimestamp {
							newLastTimestamp = itemTimestamp
						}
					}
				}

				// Update last timestamp if we found newer items
				if newLastTimestamp > lastTimestamp {
					lastTimestamp = newLastTimestamp
				} else if latestTimestamp > lastTimestamp {
					lastTimestamp = latestTimestamp
				}

				// Send new items
				if len(newItems) > 0 {
					for _, item := range newItems {
						s.sendEvent(w, "new-entry", item)
					}
				}

				// Send heartbeat to keep connection alive
				if s.config.SendHeartbeat {
					heartbeat := map[string]string{"timestamp": time.Now().Format(time.RFC3339)}
					s.sendEvent(w, "heartbeat", heartbeat)
				}

				// Flush all data to client
				if err := w.Flush(); err != nil {
					fmt.Printf("Error while flushing: %v. Closing connection.\n", err)
					return
				}
			}
		}
	}))

	return nil
}

// sendEvent sends a Server-Sent Event with the given event type and data
func (s *SSEStreamer[T]) sendEvent(w *bufio.Writer, eventType string, data any) {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, dataJSON)
}

// StreamWithSimplePoller is a convenience function for simple polling scenarios
// where you just want to fetch all recent items and filter by timestamp
func StreamWithSimplePoller[T any](
	c *fiber.Ctx,
	config SSEConfig,
	fetchFunc func(ctx context.Context, reqCtx *ctx.RequestContext) ([]T, error),
	timestampFunc func(item T) int64,
) error {
	dataFetcher := func(ctx context.Context, reqCtx *ctx.RequestContext, lastTimestamp int64) ([]T, int64, error) {
		items, err := fetchFunc(ctx, reqCtx)
		if err != nil {
			return nil, lastTimestamp, err
		}

		// Find the latest timestamp
		latestTimestamp := lastTimestamp
		for _, item := range items {
			if ts := timestampFunc(item); ts > latestTimestamp {
				latestTimestamp = ts
			}
		}

		return items, latestTimestamp, nil
	}

	itemProcessor := func(item T, lastTimestamp int64) (bool, int64) {
		itemTimestamp := timestampFunc(item)
		return itemTimestamp > lastTimestamp, itemTimestamp
	}

	streamer := NewSSEStreamer(config, dataFetcher, itemProcessor)
	return streamer.Stream(c)
}