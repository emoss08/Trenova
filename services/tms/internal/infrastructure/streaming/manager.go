package streaming

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	authCtx "github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/gin-gonic/gin"
)

type StreamMetrics struct {
	ActiveConnections int   `json:"activeConnections"`
	TotalConnections  int64 `json:"totalConnections"`
}

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

func (sm *StreamManager) handleNewClient(
	c *gin.Context,
	authContext *authCtx.AuthContext,
	clientID string,
) {
	fmt.Printf("[StreamManager] handleNewClient called for clientID: %s\n", clientID)

	requestCtx := c.Request.Context()

	var clientCtx context.Context
	var clientCancel context.CancelFunc

	if sm.config.StreamTimeout > 0 {
		clientCtx, clientCancel = context.WithTimeout(requestCtx, sm.config.StreamTimeout)
		fmt.Printf(
			"[StreamManager] Client context created with timeout: %v\n",
			sm.config.StreamTimeout,
		)
	} else {
		clientCtx, clientCancel = context.WithCancel(requestCtx)
		fmt.Printf("[StreamManager] Client context created without timeout\n")
	}
	defer clientCancel()

	client := &Client{
		ID:           clientID,
		OrgID:        authContext.OrganizationID.String(),
		BuID:         authContext.BusinessUnitID.String(),
		UserID:       authContext.UserID.String(),
		LastSeen:     time.Now(),
		ctx:          clientCtx,
		cancel:       clientCancel,
		sendQueue:    make(chan SSEMessage, 100),
		sendTimeout:  100 * time.Millisecond,
		isSlowClient: false,
		errorCount:   0,
		closed:       false,
	}

	fmt.Printf("[StreamManager] Client struct created: %+v\n", client)

	sm.addClient(client)
	fmt.Printf("[StreamManager] Client added to manager, total clients: %d\n", sm.getClientCount())

	defer func() {
		fmt.Printf("[StreamManager] Removing client: %s\n", clientID)
		sm.removeClient(clientID)
	}()

	sm.ensureStreamRunning()
	fmt.Printf("[StreamManager] Stream running status: %v\n", sm.isRunning)

	// Send initial connection event
	connectEvent := map[string]any{
		"status":    "connected",
		"timestamp": time.Now().Unix(),
	}
	fmt.Printf("[StreamManager] Sending initial connection event\n")
	sm.sendEventToClientAsync(client, "connected", connectEvent)
	fmt.Printf("[StreamManager] Connection event sent\n")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	fmt.Printf("[StreamManager] Starting Gin SSE stream for: %s\n", clientID)

	// Use Gin's Stream method to handle SSE properly
	c.Stream(func(_ io.Writer) bool {
		select {
		case <-clientCtx.Done():
			fmt.Printf(
				"[StreamManager] Client context cancelled: %s, error: %v\n",
				clientID,
				clientCtx.Err(),
			)
			return false
		case <-ticker.C:
			fmt.Printf("[StreamManager] Sending ping to client: %s\n", clientID)
			c.SSEvent("ping", map[string]any{
				"timestamp": time.Now().Unix(),
			})

			sm.mu.Lock()
			if cl, exists := sm.clients[clientID]; exists {
				cl.LastSeen = time.Now()
				fmt.Printf("[StreamManager] Updated LastSeen for client: %s\n", clientID)
			}
			sm.mu.Unlock()
			return true
		case msg, ok := <-client.sendQueue:
			if !ok {
				fmt.Printf("[StreamManager] Send queue closed for client: %s\n", clientID)
				return false
			}
			fmt.Printf("[StreamManager] Sending event '%s' to client: %s\n", msg.Event, clientID)
			c.SSEvent(msg.Event, msg.Data)
			return true
		}
	})

	fmt.Printf("[StreamManager] Stream ended for client: %s\n", clientID)
}

func (sm *StreamManager) ensureStreamRunning() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.isRunning {
		fmt.Printf("Stream already running: %v\n", sm)
		return
	}

	fmt.Printf("Starting CDC-based stream: %v\n", sm)
	sm.isRunning = true
	sm.ctx, sm.cancel = context.WithCancel(context.Background())
}

func (sm *StreamManager) sendEventToClientAsync(client *Client, eventType string, data any) {
	fmt.Printf("[AsyncSend] Sending event '%s' to client: %s\n", eventType, client.ID)

	client.closedMu.Lock()
	if client.closed {
		client.closedMu.Unlock()
		fmt.Printf("[AsyncSend] Client %s is closed, skipping send\n", client.ID)
		return
	}
	client.closedMu.Unlock()

	msg := SSEMessage{
		Event: eventType,
		Data:  data,
	}

	select {
	case <-client.ctx.Done():
		fmt.Printf("[AsyncSend] Client context done, not sending to %s\n", client.ID)
		return
	case client.sendQueue <- msg:
		fmt.Printf("[AsyncSend] Message queued successfully for client %s\n", client.ID)
	case <-time.After(client.sendTimeout):
		client.isSlowClient = true
		fmt.Printf("[AsyncSend] Client %s queue full, marked as slow\n", client.ID)
	}
}

func (sm *StreamManager) addClient(client *Client) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.clients[client.ID] = client
	sm.metrics.ActiveConnections++
	sm.metrics.TotalConnections++
}

func (sm *StreamManager) removeClient(clientID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if client, exists := sm.clients[clientID]; exists {
		client.closedMu.Lock()
		client.closed = true
		client.closedMu.Unlock()

		client.cancel()

		delete(sm.clients, clientID)
		sm.metrics.ActiveConnections--
	}
}

func (sm *StreamManager) getClientCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.clients)
}

func (sm *StreamManager) broadcastDataUpdate(data any) {
	sm.mu.RLock()
	clients := make([]*Client, 0, len(sm.clients))
	for _, client := range sm.clients {
		clients = append(clients, client)
	}
	sm.mu.RUnlock()

	for _, client := range clients {
		sm.sendEventToClientAsync(client, "new-entry", data)
	}
}

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
