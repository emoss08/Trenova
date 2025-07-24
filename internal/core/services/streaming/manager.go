/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package streaming

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/rs/zerolog/log"
)

// StreamMetrics contains metrics for a CDC streaming endpoint
type StreamMetrics struct {
	// ActiveConnections is the current number of active connections
	ActiveConnections int `json:"activeConnections"`
	// TotalConnections is the total number of connections since startup
	TotalConnections int64 `json:"totalConnections"`
}

// StreamManager manages CDC streaming connections for a specific stream key
type StreamManager struct {
	mu             sync.RWMutex
	clients        map[string]*Client
	isRunning      bool
	ctx            context.Context
	cancel         context.CancelFunc
	config         *config.StreamingConfig
	metrics        StreamMetrics
	lastDataChange time.Time
	idleTimeout    time.Duration
}

// handleNewClient handles a new client connection
func (sm *StreamManager) handleNewClient(
	ctx context.Context,
	reqCtx *appctx.RequestContext,
	clientID string,
	writer *bufio.Writer,
) {
	// Use context without timeout if StreamTimeout is 0
	var clientCtx context.Context
	var clientCancel context.CancelFunc

	if sm.config.StreamTimeout > 0 {
		clientCtx, clientCancel = context.WithTimeout(ctx, sm.config.StreamTimeout)
	} else {
		// No timeout - connection will be managed by client disconnect detection
		clientCtx, clientCancel = context.WithCancel(ctx)
	}
	defer clientCancel()

	client := &Client{
		ID:           clientID,
		Writer:       writer,
		OrgID:        reqCtx.OrgID.String(),
		BuID:         reqCtx.BuID.String(),
		UserID:       reqCtx.UserID.String(),
		LastSeen:     time.Now(),
		ctx:          clientCtx,
		cancel:       clientCancel,
		sendQueue:    make(chan []byte, 100), // Buffer for message queue
		sendTimeout:  100 * time.Millisecond,
		isSlowClient: false,
		errorCount:   0,
		closed:       false,
	}

	// Add client to manager
	sm.addClient(client)
	defer sm.removeClient(clientID)

	// Mark stream as running for CDC event handling
	sm.ensureStreamRunning()

	// Send initial connection event directly to avoid race condition
	connectEvent := map[string]any{
		"status":    "connected",
		"timestamp": time.Now().Unix(),
	}
	sm.sendEventDirectly(client, "connected", connectEvent)

	if err := writer.Flush(); err != nil {
		log.Error().Err(err).
			Interface("client", client).
			Msg("Error flushing initial connection")
		return
	}

	// Start message sender goroutine AFTER initial connection is established
	go sm.clientMessageSender(client)

	// Keep connection alive with periodic heartbeats
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-clientCtx.Done():
			log.Error().Interface("client", client).Msg("Client context cancelled or timed out")
			return
		case <-ticker.C:
			// Send a ping to detect if connection is still alive
			sm.sendEventDirectly(client, "ping", map[string]any{
				"timestamp": time.Now().Unix(),
			})

			if err := writer.Flush(); err != nil {
				// Check if it's a connection closed error (normal when client disconnects)
				if isConnectionClosed(err) {
					log.Debug().Str("clientID", client.ID).Msg("Client disconnected (normal)")
				} else {
					log.Warn().Err(err).Str("clientID", client.ID).Msg("Client connection error")
				}
				return
			}

			// Update last seen
			sm.mu.Lock()
			if c, exists := sm.clients[clientID]; exists {
				c.LastSeen = time.Now()
			}
			sm.mu.Unlock()
		}
	}
}

// clientMessageSender handles message sending for a specific client with quality detection
func (sm *StreamManager) clientMessageSender( //nolint:gocognit // we need to keep this function long
	client *Client,
) {
	defer func() {
		// Mark client as closed and close the channel safely
		client.closedMu.Lock()
		if !client.closed {
			client.closed = true
			close(client.sendQueue)
		}
		client.closedMu.Unlock()

		if r := recover(); r != nil {
			log.Error().Interface("client", client).Msgf("Client sender panic recovered: %v", r)
		}
	}()

	for {
		select {
		case <-client.ctx.Done():
			return
		case message, ok := <-client.sendQueue:
			if !ok {
				// Channel was closed
				log.Debug().Str("clientID", client.ID).Msg("Client send queue closed")
				return
			}

			// Try to send with timeout
			done := make(chan error, 1)
			go func() {
				defer func() {
					if r := recover(); r != nil {
						done <- fmt.Errorf("panic in write: %v", r)
					}
				}()
				_, err := client.Writer.Write(message)
				if err == nil {
					err = client.Writer.Flush()
				}
				done <- err
			}()

			select {
			case err := <-done:
				if err != nil {
					client.errorCount++
					if client.errorCount > 3 {
						client.isSlowClient = true
					}
					log.Error().Interface("client", client).Err(err).Msg("Error sending to client")
					return
				}
				// Reset error count on success
				client.errorCount = 0
			case <-time.After(client.sendTimeout):
				client.isSlowClient = true
				log.Warn().Interface("client", client).Msg("Client is slow, timeout reached")
				// * Continue trying to send, but mark as slow
			case <-client.ctx.Done():
				log.Warn().Interface("client", client).Msg("Client disconnected while sending")
				// * Client disconnected while sending
				return
			}
		}
	}
}

// ensureStreamRunning marks the stream as running for CDC event handling
func (sm *StreamManager) ensureStreamRunning() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.isRunning {
		log.Debug().Msg("Stream already running")
		return
	}

	log.Debug().Msg("Starting CDC-based stream")
	sm.isRunning = true
	sm.ctx, sm.cancel = context.WithCancel(context.Background())
}

// sendEventToClientAsync sends an event to a client asynchronously via queue
func (sm *StreamManager) sendEventToClientAsync(client *Client, eventType string, data any) {
	// Check if client is closed
	client.closedMu.Lock()
	if client.closed {
		client.closedMu.Unlock()
		return
	}
	client.closedMu.Unlock()

	dataJSON, err := sonic.Marshal(data)
	if err != nil {
		fmt.Printf("Error marshaling data for client %s: %v\n", client.ID, err)
		return
	}

	message := fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, dataJSON)

	select {
	case <-client.ctx.Done():
		// Client disconnected
		return
	case client.sendQueue <- []byte(message):
		// Message queued successfully
	case <-time.After(client.sendTimeout):
		// Client queue is full, mark as slow
		client.isSlowClient = true
		fmt.Printf("Client %s queue full, marked as slow\n", client.ID)
	}
}

// sendEventDirectly sends an event directly to the client's writer (for initial setup)
func (sm *StreamManager) sendEventDirectly(client *Client, eventType string, data any) {
	dataJSON, err := sonic.Marshal(data)
	if err != nil {
		fmt.Printf("Error marshaling data for client %s: %v\n", client.ID, err)
		return
	}
	message := fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, dataJSON)
	_, _ = client.Writer.WriteString(message)
}

// addClient adds a client to the stream manager
func (sm *StreamManager) addClient(client *Client) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.clients[client.ID] = client
	sm.metrics.ActiveConnections++
	sm.metrics.TotalConnections++
}

// removeClient removes a client from the stream manager
func (sm *StreamManager) removeClient(clientID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if client, exists := sm.clients[clientID]; exists {
		// Mark as closed to prevent any further sends
		client.closedMu.Lock()
		client.closed = true
		client.closedMu.Unlock()

		// Cancel the client context
		client.cancel()

		// Remove from manager
		delete(sm.clients, clientID)
		sm.metrics.ActiveConnections--
	}
}

// getClientCount returns the number of connected clients
func (sm *StreamManager) getClientCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.clients)
}

// broadcastDataUpdate immediately broadcasts a data update to all connected clients
func (sm *StreamManager) broadcastDataUpdate(data any) {
	sm.mu.RLock()
	clients := make([]*Client, 0, len(sm.clients))
	for _, client := range sm.clients {
		if !client.isSlowClient {
			clients = append(clients, client)
		}
	}
	sm.mu.RUnlock()

	// Immediately broadcast to all clients
	for _, client := range clients {
		sm.sendEventToClientAsync(client, "new-entry", data)
	}
}

// shutdown gracefully shuts down the stream manager
func (sm *StreamManager) shutdown() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.cancel != nil {
		sm.cancel()
	}

	for _, client := range sm.clients {
		client.cancel()
	}
	sm.clients = make(map[string]*Client)
	sm.isRunning = false
}

// isConnectionClosed checks if an error indicates a closed connection
func isConnectionClosed(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// Check for common connection closed error patterns
	return strings.Contains(errStr, "connection closed") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "use of closed") ||
		strings.Contains(errStr, "connection refused")
}
