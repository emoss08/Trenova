package websocket

import (
	"context"
	"fmt"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/cache/redis"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	LC     fx.Lifecycle
	Logger *logger.Logger
	Redis  *redis.Client
}

type Service struct {
	l          *zerolog.Logger
	redis      *redis.Client
	serverID   string // Unique ID for this server instance
	clients    map[*Client]bool
	userIndex  map[string]map[*Client]bool // userID -> multiple clients for multi-tab support
	orgIndex   map[string]map[*Client]bool // orgID -> clients for org broadcasts
	roomIndex  map[string]map[*Client]bool // roomID -> clients for room broadcasts
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	mu         sync.RWMutex
}

type Client struct {
	service *Service
	conn    *websocket.Conn
	send    chan []byte
	userID  pulid.ID
	orgID   pulid.ID
	roomID  string
}

type Message struct {
	Type     string `json:"type"`
	Target   string `json:"target"`   // "user", "org", or "room"
	TargetID string `json:"targetId"` // userID, orgID, or roomID
	UserID   string `json:"userId"`
	OrgID    string `json:"orgId"`
	ServerID string `json:"serverId,omitempty"` // Server that published the message
	Content  any    `json:"content"`
}

type BroadcastRequest struct {
	Type     string `json:"type"`     // "userBroadcast" or "orgBroadcast"
	TargetID string `json:"targetId"` // userID or orgID
	Message  any    `json:"message"`
}

func NewService(p ServiceParams) services.WebSocketService {
	log := p.Logger.With().
		Str("service", "websocket").
		Logger()

	s := &Service{
		l:          &log,
		redis:      p.Redis,
		serverID:   pulid.MustNew("srv_").String(), // Unique server ID
		clients:    make(map[*Client]bool),
		userIndex:  make(map[string]map[*Client]bool),
		orgIndex:   make(map[string]map[*Client]bool),
		roomIndex:  make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 1000), // Buffered channel for better performance
	}

	p.LC.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go s.run()
			go s.ListenToRedis()
			s.l.Info().Msg("🚀 websocket service started")
			return nil
		},
		OnStop: func(context.Context) error {
			s.disconnectAllClients()
			s.l.Info().Msg("🔴 websocket service stopped")
			return nil
		},
	})

	return s
}

// Main service loop to handle client registration/unregistration and broadcasts
func (s *Service) run() {
	for {
		select {
		case client := <-s.register:
			s.registerClient(client)

		case client := <-s.unregister:
			s.unregisterClient(client)

		case message := <-s.broadcast:
			s.handleBroadcast(message)
		}
	}
}

func (s *Service) registerClient(c *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.clients[c] = true

	// Add to user index (support multiple clients per user)
	if s.userIndex[c.userID.String()] == nil {
		s.userIndex[c.userID.String()] = make(map[*Client]bool)
	}
	s.userIndex[c.userID.String()][c] = true

	// Add to org index
	if s.orgIndex[c.orgID.String()] == nil {
		s.orgIndex[c.orgID.String()] = make(map[*Client]bool)
	}
	s.orgIndex[c.orgID.String()][c] = true

	// Add to room index if room ID is provided
	if c.roomID != "" {
		if s.roomIndex[c.roomID] == nil {
			s.roomIndex[c.roomID] = make(map[*Client]bool)
		}
		s.roomIndex[c.roomID][c] = true

		// Store room membership in Redis
		if err := s.redis.SAdd(context.Background(), "room:"+c.roomID+":users", c.userID.String()); err != nil {
			s.l.Error().Err(err).Msg("failed to add user to room in Redis")
		}
	}

	// Store in Redis for cross-service coordination
	var remoteAddr string
	if c.conn != nil && c.conn.RemoteAddr() != nil {
		remoteAddr = c.conn.RemoteAddr().String()
	} else {
		remoteAddr = fmt.Sprintf("unknown_%s", pulid.MustNew("conn_").String())
	}

	if err := s.redis.SAdd(
		context.Background(),
		fmt.Sprintf("user:%s:connections", c.userID.String()),
		remoteAddr,
	); err != nil {
		s.l.Error().Err(err).Msg("failed to add user connection to Redis")
	}

	if err := s.redis.SAdd(
		context.Background(),
		fmt.Sprintf("org:%s:clients", c.orgID.String()),
		c.userID.String(),
	); err != nil {
		s.l.Error().Err(err).Msg("failed to add user to org in Redis")
	}

	s.l.Info().
		Str("user_id", c.userID.String()).
		Str("org_id", c.orgID.String()).
		Str("room_id", c.roomID).
		Int("user_connections", len(s.userIndex[c.userID.String()])).
		Msg("client registered")
}

func (s *Service) unregisterClient(c *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.clients[c]; !ok {
		return
	}

	delete(s.clients, c)

	// Only close the channel if it's not already closed
	select {
	case <-c.send:
		// Channel already closed
	default:
		close(c.send)
	}

	// Track remaining connections before removing
	remainingConnections := 0

	// Remove from user index
	userClients := s.userIndex[c.userID.String()]
	if userClients != nil {
		delete(userClients, c)
		remainingConnections = len(userClients)
		if remainingConnections == 0 {
			delete(s.userIndex, c.userID.String())
			// Only remove from Redis if this was the last connection
			if err := s.redis.SRem(
				context.Background(),
				fmt.Sprintf("org:%s:clients", c.orgID.String()),
				c.userID.String(),
			); err != nil {
				s.l.Error().Err(err).Msg("failed to remove user from org in Redis")
			}
		}
	}

	// Remove from org index
	orgClients := s.orgIndex[c.orgID.String()]
	if orgClients != nil {
		delete(orgClients, c)
		if len(orgClients) == 0 {
			delete(s.orgIndex, c.orgID.String())
		}
	}

	// Remove from room index
	if c.roomID != "" {
		roomClients := s.roomIndex[c.roomID]
		if roomClients != nil {
			delete(roomClients, c)
			if len(roomClients) == 0 {
				delete(s.roomIndex, c.roomID)
				if err := s.redis.SRem(context.Background(), "room:"+c.roomID+":users", c.userID.String()); err != nil {
					s.l.Error().Err(err).Msg("failed to remove user from room in Redis")
				}
			}
		}
	}

	// Remove connection from Redis
	if c.conn != nil && c.conn.RemoteAddr() != nil {
		if err := s.redis.SRem(
			context.Background(),
			fmt.Sprintf("user:%s:connections", c.userID.String()),
			c.conn.RemoteAddr().String(),
		); err != nil {
			s.l.Error().Err(err).Msg("failed to remove user connection from Redis")
		}
	}

	s.l.Info().
		Str("user_id", c.userID.String()).
		Str("org_id", c.orgID.String()).
		Str("room_id", c.roomID).
		Int("remaining_connections", remainingConnections).
		Msg("client unregistered")
}

func (s *Service) handleBroadcast(message []byte) {
	var msg Message
	if err := sonic.Unmarshal(message, &msg); err != nil {
		s.l.Error().Err(err).Msg("failed to unmarshal message")
		return
	}

	// Skip messages from our own server to prevent duplicates
	if msg.ServerID == s.serverID {
		s.l.Debug().
			Str("server_id", msg.ServerID).
			Str("target", msg.Target).
			Str("target_id", msg.TargetID).
			Msg("skipping message from own server")
		return
	}

	// Wrap the content in the format expected by the frontend
	wrappedForClient := map[string]any{
		"type":      "notification",
		"data":      msg.Content,
		"timestamp": timeutils.NowUnix(),
	}

	contentBytes, err := sonic.Marshal(wrappedForClient)
	if err != nil {
		s.l.Error().Err(err).Msg("failed to marshal content")
		return
	}

	switch msg.Target {
	case "user":
		s.sendToUser(msg.TargetID, contentBytes)
	case "org":
		s.sendToOrg(msg.TargetID, contentBytes)
	case "room":
		s.sendToRoom(msg.TargetID, contentBytes)
	}
}

func (s *Service) sendToUser(userID string, message []byte) {
	s.mu.RLock()
	clients := s.userIndex[userID]
	s.mu.RUnlock()

	for client := range clients {
		select {
		case client.send <- message:
		default:
			// Channel full, remove client
			go s.unregisterClient(client)
		}
	}
}

func (s *Service) sendToOrg(orgID string, message []byte) {
	s.mu.RLock()
	clients := s.orgIndex[orgID]
	s.mu.RUnlock()

	for client := range clients {
		select {
		case client.send <- message:
		default:
			// Channel full, remove client
			go s.unregisterClient(client)
		}
	}
}

func (s *Service) sendToRoom(roomID string, message []byte) {
	s.mu.RLock()
	clients := s.roomIndex[roomID]
	s.mu.RUnlock()

	for client := range clients {
		select {
		case client.send <- message:
		default:
			// Channel full, remove client
			go s.unregisterClient(client)
		}
	}
}

// Public methods for external broadcasting
func (s *Service) BroadcastToUser(userID string, content any) {
	// The content should already be a notification object
	// Just wrap it for internal routing
	msg := Message{
		Type:     "notification",
		Target:   "user",
		TargetID: userID,
		ServerID: s.serverID, // Add server ID to prevent processing our own messages
		Content:  content,
	}

	s.l.Info().
		Str("user_id", userID).
		Str("server_id", s.serverID).
		Interface("content", content).
		Msg("broadcasting to user")

	// Check if we have local connections for this user
	s.mu.RLock()
	hasLocalConnections := len(s.userIndex[userID]) > 0
	s.mu.RUnlock()

	if hasLocalConnections {
		// Send directly to local connections
		wrappedForClient := map[string]any{
			"type":      "notification",
			"data":      content,
			"timestamp": timeutils.NowUnix(),
		}

		clientData, _ := sonic.Marshal(wrappedForClient)
		s.sendToUser(userID, clientData)
	}

	// Always publish to Redis for other instances
	data, err := sonic.Marshal(msg)
	if err != nil {
		s.l.Error().Err(err).Msg("failed to marshal broadcast message")
		return
	}

	if pubErr := s.redis.Publish(context.Background(), "broadcast:user:"+userID, data).Err(); pubErr != nil {
		s.l.Error().Err(pubErr).Msg("failed to publish to Redis")
	}
}

func (s *Service) BroadcastToRoom(roomID string, content any) {
	// The content should already be a notification object
	// Just wrap it for internal routing
	msg := Message{
		Type:     "notification",
		Target:   "room",
		TargetID: roomID,
		ServerID: s.serverID, // Add server ID to prevent processing our own messages
		Content:  content,
	}

	// Check if we have local connections for this room
	s.mu.RLock()
	hasLocalConnections := len(s.roomIndex[roomID]) > 0
	s.mu.RUnlock()

	if hasLocalConnections {
		// Send directly to local connections
		wrappedForClient := map[string]any{
			"type":      "notification",
			"data":      content,
			"timestamp": timeutils.NowUnix(),
		}

		clientData, _ := sonic.Marshal(wrappedForClient)
		s.sendToRoom(roomID, clientData)
	}

	// Always publish to Redis for other instances
	data, err := sonic.Marshal(msg)
	if err != nil {
		s.l.Error().Err(err).Msg("failed to marshal broadcast message")
		return
	}

	if pubErr := s.redis.Publish(context.Background(), "broadcast:room:"+roomID, data).Err(); pubErr != nil {
		s.l.Error().Err(pubErr).Msg("failed to publish to Redis")
	}
}

func (s *Service) BroadcastToOrg(orgID string, content any) {
	// The content should already be a notification object
	// Just wrap it for internal routing
	msg := Message{
		Type:     "notification",
		Target:   "org",
		TargetID: orgID,
		ServerID: s.serverID, // Add server ID to prevent processing our own messages
		Content:  content,
	}

	// Check if we have local connections for this org
	s.mu.RLock()
	hasLocalConnections := len(s.orgIndex[orgID]) > 0
	s.mu.RUnlock()

	if hasLocalConnections {
		// Send directly to local connections
		wrappedForClient := map[string]any{
			"type":      "notification",
			"data":      content,
			"timestamp": timeutils.NowUnix(),
		}

		clientData, _ := sonic.Marshal(wrappedForClient)
		s.sendToOrg(orgID, clientData)
	}

	// Always publish to Redis for other instances
	data, err := sonic.Marshal(msg)
	if err != nil {
		s.l.Error().Err(err).Msg("failed to marshal broadcast message")
		return
	}

	if pubErr := s.redis.Publish(context.Background(), "broadcast:org:"+orgID, data).Err(); pubErr != nil {
		s.l.Error().Err(pubErr).Msg("failed to publish to Redis")
	}
}

func (s *Service) ListenToRedis() {
	// Use PSubscribe for pattern matching on broadcast channels
	pubsub := s.redis.PSubscribe(context.Background(), "broadcast:*")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for msg := range ch {
		s.broadcast <- []byte(msg.Payload)
	}
}

// WebSocket connection handler
func (s *Service) HandleWebSocket(c *fiber.Ctx) error {
	// Check if this is a websocket upgrade request
	if websocket.IsWebSocketUpgrade(c) {
		reqCtx, err := appctx.WithRequestContext(c)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to get request context",
			})
		}

		// Optional room ID from query param (for joining specific rooms)
		roomID := c.Query("room", "")

		c.Locals("userID", reqCtx.UserID.String())
		c.Locals("orgID", reqCtx.OrgID.String())
		c.Locals("roomID", roomID)

		s.l.Info().
			Str("user_id", reqCtx.UserID.String()).
			Str("org_id", reqCtx.OrgID.String()).
			Str("room_id", roomID).
			Msg("upgrading to websocket")

		return c.Next()
	}

	return fiber.ErrUpgradeRequired
}

func (s *Service) HandleConnection(conn *websocket.Conn) {
	// Extract parameters from query string
	userIDStr, ok := conn.Locals("userID").(string)
	if !ok {
		s.l.Error().Msg("userID not found in locals")
		_ = conn.Close()
		return
	}

	orgIDStr, ok := conn.Locals("orgID").(string)
	if !ok {
		s.l.Error().Msg("orgID not found in locals")
		_ = conn.Close()
		return
	}

	roomID, _ := conn.Locals("roomID").(string) // Optional

	if userIDStr == "" || orgIDStr == "" {
		s.l.Error().Msg("missing required parameters: user_id and org_id")
		_ = conn.Close()
		return
	}

	// Parse PULID strings
	userID, err := pulid.Parse(userIDStr)
	if err != nil {
		s.l.Error().Err(err).Msg("invalid user_id format")
		_ = conn.Close()
		return
	}

	orgID, err := pulid.Parse(orgIDStr)
	if err != nil {
		s.l.Error().Err(err).Msg("invalid org_id format")
		_ = conn.Close()
		return
	}

	client := &Client{
		service: s,
		conn:    conn,
		send:    make(chan []byte, 256),
		userID:  userID,
		orgID:   orgID,
		roomID:  roomID,
	}

	// Register client and start pumps
	s.register <- client
	go s.writePump(client)
	s.readPump(client) // This blocks until connection closes
}

func (s *Service) readPump(c *Client) {
	defer func() {
		s.unregister <- c
		_ = c.conn.Close()
	}()

	for {
		_, messageData, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				s.l.Error().Err(err).Msg("websocket error")
			}
			break
		}

		msg := new(Message)
		if err = sonic.Unmarshal(messageData, &msg); err != nil {
			s.l.Error().Err(err).Msg("failed to unmarshal client message")
			continue
		}

		// Handle ping/pong messages
		if msg.Type == "ping" {
			// Respond with pong immediately
			pongMsg := map[string]any{
				"type": "pong",
				"data": map[string]any{
					"timestamp": timeutils.NowUnix(),
					"received":  msg.Content,
				},
			}
			pongBytes, _ := sonic.Marshal(pongMsg)
			select {
			case c.send <- pongBytes:
			default:
				// Channel full, client is not reading
				s.l.Warn().
					Str("user_id", c.userID.String()).
					Msg("failed to send pong, client send channel full")
			}
			continue
		}

		// Set sender info
		msg.UserID = c.userID.String()
		msg.OrgID = c.orgID.String()

		// Handle different message types
		switch msg.Target {
		case "room":
			if c.roomID != "" {
				msg.TargetID = c.roomID
				s.BroadcastToRoom(c.roomID, msg.Content)
			}
		case "org":
			msg.TargetID = c.orgID.String()
			s.BroadcastToOrg(c.orgID.String(), msg.Content)
		case "user":
			// Allow users to send direct messages to other users
			if msg.TargetID != "" {
				s.BroadcastToUser(msg.TargetID, msg.Content)
			}
		}
	}
}

func (s *Service) writePump(c *Client) {
	defer func() {
		_ = c.conn.Close()
	}()

	for message := range c.send {
		_ = c.conn.WriteMessage(websocket.TextMessage, message)
	}
	// Channel closed, send close message
	_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
}

func (s *Service) disconnectAllClients() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for client := range s.clients {
		_ = client.conn.Close()
	}

	s.clients = make(map[*Client]bool)
	s.userIndex = make(map[string]map[*Client]bool)
	s.orgIndex = make(map[string]map[*Client]bool)
	s.roomIndex = make(map[string]map[*Client]bool)
	s.l.Info().Msg("all clients disconnected")
}

// HTTP API handlers for server-side broadcasting
func (s *Service) HandleBroadcast(c *fiber.Ctx) error {
	var req BroadcastRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid JSON payload",
		})
	}

	switch req.Type {
	case "user_broadcast":
		s.BroadcastToUser(req.TargetID, req.Message)
		return c.JSON(fiber.Map{
			"status":  "success",
			"message": fmt.Sprintf("Message sent to user %s", req.TargetID),
		})

	case "org_broadcast":
		s.BroadcastToOrg(req.TargetID, req.Message)
		return c.JSON(fiber.Map{
			"status":  "success",
			"message": fmt.Sprintf("Message sent to organization %s", req.TargetID),
		})

	case "room_broadcast":
		s.BroadcastToRoom(req.TargetID, req.Message)
		return c.JSON(fiber.Map{
			"status":  "success",
			"message": fmt.Sprintf("Message sent to room %s", req.TargetID),
		})

	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid broadcast type. Use 'user_broadcast', 'org_broadcast', or 'room_broadcast'",
		})
	}
}

// Get organization members
func (s *Service) HandleOrgMembers(c *fiber.Ctx) error {
	orgID := c.Query("org_id")
	if orgID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "org_id query parameter is required",
		})
	}

	members, err := s.redis.SMembers(context.Background(), fmt.Sprintf("org:%s:clients", orgID))
	if err != nil {
		s.l.Error().Err(err).Msg("failed to get organization members")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch organization members",
		})
	}

	return c.JSON(fiber.Map{
		"org_id":  orgID,
		"members": members,
	})
}

// Get room members
func (s *Service) HandleRoomMembers(c *fiber.Ctx) error {
	roomID := c.Query("room_id")
	if roomID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "room_id query parameter is required",
		})
	}

	members, err := s.redis.SMembers(context.Background(), "room:"+roomID+":users")
	if err != nil {
		s.l.Error().Err(err).Msg("failed to get room members")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch room members",
		})
	}

	return c.JSON(fiber.Map{
		"room_id": roomID,
		"members": members,
	})
}
