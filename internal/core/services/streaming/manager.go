package streaming

import (
	"bufio"
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/rs/zerolog/log"
)

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

// StreamManager manages connections for a specific stream key
type StreamManager struct {
	mu            sync.RWMutex
	clients       map[string]*Client
	lastTimestamp int64
	sentItems     map[string]int64 // Track sent items by ID with timestamp
	isRunning     bool
	ctx           context.Context
	cancel        context.CancelFunc
	dataFetcher   services.DataFetcher
	timestampFunc services.TimestampExtractor
	config        *config.StreamingConfig
	metrics       StreamMetrics
	// Circuit breaker and resilience
	consecutiveErrors    int
	maxConsecutiveErrors int
	backoffDuration      time.Duration
	// Lifecycle management
	lastDataChange time.Time
	idleTimeout    time.Duration
	// Memory management
	maxSentItems int
	lastCleanup  time.Time
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

	// Start the data polling stream if not already running
	sm.ensureStreamRunning(reqCtx)

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
					log.Debug().Str("client_id", client.ID).Msg("Client disconnected (normal)")
				} else {
					log.Warn().Err(err).Str("client_id", client.ID).Msg("Client connection error")
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

// ensureStreamRunning starts the data polling stream if not already running
func (sm *StreamManager) ensureStreamRunning(reqCtx *appctx.RequestContext) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.isRunning {
		return
	}

	sm.isRunning = true
	sm.ctx, sm.cancel = context.WithCancel(context.Background())

	go sm.runDataStream(reqCtx)
}

// runDataStream runs the main data streaming loop
func (sm *StreamManager) runDataStream( //nolint:gocognit // we need to keep this function long
	reqCtx *appctx.RequestContext,
) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Stream panic recovered: %v\n", r)
			// Could implement restart logic here
		}
		sm.mu.Lock()
		sm.isRunning = false
		sm.mu.Unlock()
	}()

	ticker := time.NewTicker(sm.config.PollInterval)
	defer ticker.Stop()

	// Remove the stream timeout - streams should run as long as clients are connected
	for {
		select {
		case <-sm.ctx.Done():
			return
		case <-ticker.C:
			// Check if we have any clients
			clientCount := sm.getClientCount()
			if clientCount == 0 {
				// Check idle timeout
				if time.Since(sm.lastDataChange) > sm.idleTimeout {
					fmt.Println("Stream idle and no clients, shutting down")
					return
				}
				fmt.Println("No clients connected, stopping data stream")
				return
			}

			// Fetch new data with circuit breaker pattern
			data, err := sm.dataFetcher(sm.ctx, reqCtx)
			if err != nil {
				sm.mu.Lock()
				sm.consecutiveErrors++
				sm.metrics.DataFetchErrors++
				sm.mu.Unlock()

				// Implement exponential backoff
				if sm.consecutiveErrors >= sm.maxConsecutiveErrors {
					backoffDuration := time.Duration(math.Min(
						float64(
							sm.backoffDuration,
						)*math.Pow(
							2,
							float64(sm.consecutiveErrors-sm.maxConsecutiveErrors),
						),
						float64(30*time.Second),
					))
					fmt.Printf("Circuit breaker activated, backing off for %v\n", backoffDuration)
					time.Sleep(backoffDuration)
				}

				sm.broadcastError(fmt.Sprintf("Failed to fetch data: %v", err))
				continue
			}

			// Reset consecutive errors on successful fetch
			sm.mu.Lock()
			sm.consecutiveErrors = 0
			sm.backoffDuration = sm.config.PollInterval
			sm.metrics.LastDataFetch = time.Now().Unix()
			sm.lastDataChange = time.Now()
			sm.mu.Unlock()

			// Process data and broadcast new items with permission filtering
			sm.processAndBroadcastDataWithPermissions(data)

			// Send heartbeat if enabled
			if sm.config.EnableHeartbeat {
				sm.broadcastHeartbeat()
			}

			// Clean up disconnected clients
			sm.cleanupDisconnectedClients()
		}
	}
}

// processAndBroadcastDataWithPermissions processes data and broadcasts with permission filtering
func (sm *StreamManager) processAndBroadcastDataWithPermissions( //nolint:gocognit // we need to keep this function long
	data any,
) {
	// Group items by user permissions
	clientItems := make(map[string][]any)

	// Get all clients first
	sm.mu.RLock()
	clients := make(map[string]*Client)
	for id, client := range sm.clients {
		clients[id] = client
	}
	sm.mu.RUnlock()

	// Process data and filter by permissions
	var allNewItems []any
	var maxTimestamp = sm.lastTimestamp

	// Handle different data types and collect new items
	switch v := data.(type) {
	case []any:
		for _, item := range v {
			timestamp := sm.timestampFunc(item)
			itemID := sm.getItemID(item)

			// Check if item is new (timestamp >= lastTimestamp AND not already sent)
			_, alreadySent := sm.sentItems[itemID]
			if timestamp >= sm.lastTimestamp && !alreadySent {
				sm.sentItems[itemID] = timestamp
				if timestamp > maxTimestamp {
					maxTimestamp = timestamp
				}

				// Check which users should see this item
				for clientID := range clients {
					clientItems[clientID] = append(clientItems[clientID], item)
				}
				allNewItems = append(allNewItems, item)
			}
		}
	default:
		timestamp := sm.timestampFunc(data)
		itemID := sm.getItemID(data)

		// Check if item is new (timestamp >= lastTimestamp AND not already sent)
		_, alreadySent := sm.sentItems[itemID]
		if timestamp >= sm.lastTimestamp && !alreadySent {
			sm.sentItems[itemID] = timestamp
			if timestamp > maxTimestamp {
				maxTimestamp = timestamp
			}

			// Check which users should see this item
			for clientID := range clients {
				clientItems[clientID] = append(clientItems[clientID], data)
			}
			allNewItems = append(allNewItems, data)
		}
	}

	// Update timestamp
	if len(allNewItems) > 0 {
		sm.lastTimestamp = maxTimestamp

		// Send filtered data to each user
		for clientID, items := range clientItems {
			sm.mu.RLock()
			client, exists := sm.clients[clientID]
			sm.mu.RUnlock()

			if exists && !client.isSlowClient {
				for _, item := range items {
					sm.sendEventToClientAsync(client, "new-entry", item)
				}
			}
		}

		// Clean up old sent items map to prevent memory leak
		sm.cleanupOldSentItems()
	}
}

// getItemID extracts a unique identifier from an item
func (sm *StreamManager) getItemID(item any) string {
	// Try to extract ID from common struct patterns
	if v, ok := item.(interface{ GetID() string }); ok {
		return v.GetID()
	}

	// For shipment domain objects, use reflection to get ID field
	if shp, ok := item.(*shipment.Shipment); ok {
		return shp.ID.String()
	}

	// Fallback: use timestamp + hash of the object
	timestamp := sm.timestampFunc(item)
	return fmt.Sprintf("%d_%p", timestamp, item)
}

// cleanupOldSentItems removes old entries from sentItems map to prevent memory leaks
func (sm *StreamManager) cleanupOldSentItems() {
	now := time.Now()

	// Only cleanup every 5 minutes to avoid overhead
	if now.Sub(sm.lastCleanup) < 5*time.Minute {
		return
	}
	sm.lastCleanup = now

	// Remove items older than 10 minutes
	cutoffTime := now.Add(-10 * time.Minute).Unix()

	for id, timestamp := range sm.sentItems {
		if timestamp < cutoffTime {
			delete(sm.sentItems, id)
		}
	}

	// If still too large, remove oldest entries
	if len(sm.sentItems) > sm.maxSentItems {
		// Create slice of items with timestamps for sorting
		type itemWithTime struct {
			id        string
			timestamp int64
		}

		items := make([]itemWithTime, 0, len(sm.sentItems))
		for id, timestamp := range sm.sentItems {
			items = append(items, itemWithTime{id: id, timestamp: timestamp})
		}

		// Sort by timestamp (oldest first) using a more efficient approach
		for i := range len(items) - 1 {
			for j := i + 1; j < len(items); j++ {
				if items[i].timestamp > items[j].timestamp {
					items[i], items[j] = items[j], items[i]
				}
			}
		}

		// Remove oldest entries until we're under the limit
		removeCount := len(sm.sentItems) - sm.maxSentItems/2 // Remove to half capacity
		for i := 0; i < removeCount && i < len(items); i++ {
			delete(sm.sentItems, items[i].id)
		}
	}
}

// broadcastHeartbeat sends heartbeat to all clients
func (sm *StreamManager) broadcastHeartbeat() {
	heartbeat := map[string]any{
		"timestamp": time.Now().Unix(),
	}

	sm.mu.RLock()
	clients := make([]*Client, 0, len(sm.clients))
	for _, client := range sm.clients {
		clients = append(clients, client)
	}
	sm.mu.RUnlock()

	for _, client := range clients {
		sm.sendEventToClient(client, "heartbeat", heartbeat)
		if err := client.Writer.Flush(); err != nil {
			if isConnectionClosed(err) {
				log.Debug().
					Str("client_id", client.ID).
					Msg("Client disconnected during heartbeat (normal)")
			} else {
				log.Warn().Err(err).Str("client_id", client.ID).Msg("Error flushing heartbeat to client")
			}
			sm.removeClient(client.ID)
		}
	}
}

// broadcastError broadcasts an error to all clients
func (sm *StreamManager) broadcastError(errorMsg string) {
	errorData := map[string]any{
		"error":     errorMsg,
		"timestamp": time.Now().Unix(),
	}

	sm.mu.RLock()
	clients := make([]*Client, 0, len(sm.clients))
	for _, client := range sm.clients {
		clients = append(clients, client)
	}
	sm.mu.RUnlock()

	for _, client := range clients {
		sm.sendEventToClient(client, "error", errorData)
		if err := client.Writer.Flush(); err != nil {
			if isConnectionClosed(err) {
				log.Debug().
					Str("client_id", client.ID).
					Msg("Client disconnected during error broadcast (normal)")
			} else {
				log.Warn().Err(err).Str("client_id", client.ID).Msg("Error flushing error to client")
			}
			sm.removeClient(client.ID)
		}
	}
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

// sendEventToClient sends an event to a specific client with proper channel state checking
func (sm *StreamManager) sendEventToClient(client *Client, eventType string, data any) {
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

	// Check if client context is still valid
	select {
	case <-client.ctx.Done():
		// Client is disconnected, don't attempt to send
		return
	default:
		// Client is still connected, proceed
	}

	// Try to queue the message with panic recovery
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Stream panic recovered: %v\n", r)
			}
		}()

		select {
		case client.sendQueue <- []byte(message):
			// Message queued successfully
		case <-time.After(10 * time.Millisecond):
			// Quick timeout - if queue is full, mark as slow and skip
			client.isSlowClient = true
		case <-client.ctx.Done():
			// Client disconnected while waiting
			return
		}
	}()
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

// cleanupDisconnectedClients removes inactive clients
func (sm *StreamManager) cleanupDisconnectedClients() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	for clientID, client := range sm.clients {
		if now.Sub(client.LastSeen) > 2*time.Minute {
			client.cancel()
			delete(sm.clients, clientID)
			sm.metrics.ActiveConnections--
		}
	}
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
