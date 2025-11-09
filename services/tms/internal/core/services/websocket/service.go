package websocket

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability"
	"github.com/emoss08/trenova/internal/infrastructure/redis"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.opentelemetry.io/otel/attribute"

	authcontext "github.com/emoss08/trenova/internal/api/context"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	LC      fx.Lifecycle
	Logger  *zap.Logger
	Redis   *redis.Connection
	Metrics *observability.MetricsRegistry
	Tracer  *observability.TracerProvider
	Config  *config.Config
}

type Service struct {
	l          *zap.Logger
	redis      *redis.Connection
	metrics    *observability.MetricsRegistry
	tracer     *observability.TracerProvider
	cfg        *config.Config
	serverID   string
	clients    map[*Client]bool
	userIndex  map[string]map[*Client]bool // userID -> multiple clients for multi-tab support
	orgIndex   map[string]map[*Client]bool // orgID -> clients for org broadcasts
	roomIndex  map[string]map[*Client]bool // roomID -> clients for room broadcasts
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	mu         sync.RWMutex
	upgrader   websocket.Upgrader
}

type Client struct {
	service     *Service
	conn        *websocket.Conn
	send        chan []byte
	userID      pulid.ID
	orgID       pulid.ID
	roomID      string
	connectedAt time.Time
	ctx         context.Context
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
	cfg := p.Config.WebSocket

	s := &Service{
		l:          p.Logger.Named("service.websocket"),
		redis:      p.Redis,
		metrics:    p.Metrics,
		tracer:     p.Tracer,
		cfg:        p.Config,
		serverID:   pulid.MustNew("srv_").String(), // Unique server ID
		clients:    make(map[*Client]bool),
		userIndex:  make(map[string]map[*Client]bool),
		orgIndex:   make(map[string]map[*Client]bool),
		roomIndex:  make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 1000), // Buffered channel for better performance
		upgrader: websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool {
				// TODO: Implement proper origin checking in production
				return true
			},
			ReadBufferSize:  cfg.ReadBufferSize,
			WriteBufferSize: cfg.WriteBufferSize,
		},
	}

	p.LC.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go s.run()
			go s.ListenToRedis(ctx)
			go s.startConnectionMonitor(ctx)
			s.l.Info("🚀 websocket service started",
				zap.Int("max_total_connections", p.Config.WebSocket.MaxTotalConnections),
				zap.Int("max_connections_per_user", p.Config.WebSocket.MaxConnectionsPerUser),
			)
			return nil
		},
		OnStop: func(context.Context) error {
			s.disconnectAllClients()
			s.l.Info("🔴 websocket service stopped")
			return nil
		},
	})

	return s
}

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

func (s *Service) checkConnectionLimits(ctx context.Context, c *Client) bool {
	if len(s.clients) >= s.cfg.WebSocket.MaxTotalConnections {
		observability.AddSpanEvent(ctx, "connection_rejected_total_limit",
			attribute.Int("current_connections", len(s.clients)),
			attribute.Int("max_connections", s.cfg.WebSocket.MaxTotalConnections),
		)
		s.l.Warn("connection rejected: total limit reached",
			zap.Int("current", len(s.clients)),
			zap.Int("max", s.cfg.WebSocket.MaxTotalConnections),
		)
		return false
	}

	userConnections := len(s.userIndex[c.userID.String()])
	if userConnections >= s.cfg.WebSocket.MaxConnectionsPerUser {
		observability.AddSpanEvent(ctx, "connection_rejected_user_limit",
			attribute.String("user_id", c.userID.String()),
			attribute.Int("current_user_connections", userConnections),
			attribute.Int("max_user_connections", s.cfg.WebSocket.MaxConnectionsPerUser),
		)
		s.l.Warn("connection rejected: user limit reached",
			zap.String("user_id", c.userID.String()),
			zap.Int("current", userConnections),
			zap.Int("max", s.cfg.WebSocket.MaxConnectionsPerUser),
		)
		return false
	}

	return true
}

func (s *Service) updateClientIndexes(ctx context.Context, c *Client) {
	if s.userIndex[c.userID.String()] == nil {
		s.userIndex[c.userID.String()] = make(map[*Client]bool)
	}
	s.userIndex[c.userID.String()][c] = true

	if s.orgIndex[c.orgID.String()] == nil {
		s.orgIndex[c.orgID.String()] = make(map[*Client]bool)
	}
	s.orgIndex[c.orgID.String()][c] = true

	if c.roomID != "" {
		if s.roomIndex[c.roomID] == nil {
			s.roomIndex[c.roomID] = make(map[*Client]bool)
		}
		s.roomIndex[c.roomID][c] = true
		s.metrics.RecordWSRoomSize(c.roomID, len(s.roomIndex[c.roomID]))

		if err := s.redis.SAdd(ctx, "room:"+c.roomID+":users", c.userID.String()); err != nil {
			s.l.Error("failed to add user to room in Redis", zap.Error(err))
		}
	}

	s.metrics.UpdateWSRooms(len(s.roomIndex))
}

func (s *Service) syncClientToRedis(ctx context.Context, c *Client) {
	var remoteAddr string
	if c.conn != nil && c.conn.RemoteAddr() != nil {
		remoteAddr = c.conn.RemoteAddr().String()
	} else {
		remoteAddr = fmt.Sprintf("unknown_%s", pulid.MustNew("conn_").String())
	}

	if err := s.redis.SAdd(
		ctx,
		fmt.Sprintf("user:%s:connections", c.userID.String()),
		remoteAddr,
	); err != nil {
		s.l.Error("failed to add user connection to Redis", zap.Error(err))
	}

	if err := s.redis.SAdd(
		ctx,
		fmt.Sprintf("org:%s:clients", c.orgID.String()),
		c.userID.String(),
	); err != nil {
		s.l.Error("failed to add user to org in Redis", zap.Error(err))
	}
}

func (s *Service) registerClient(c *Client) {
	ctx, span := s.tracer.StartSpan(c.ctx, "websocket.registerClient")
	defer span.End()

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.checkConnectionLimits(ctx, c) {
		return
	}

	s.clients[c] = true
	s.metrics.RecordWSConnection(c.orgID.String(), c.userID.String())

	s.updateClientIndexes(ctx, c)
	s.syncClientToRedis(ctx, c)

	observability.AddSpanAttributes(ctx,
		attribute.Int("ws.total_connections", len(s.clients)),
		attribute.Int("ws.user_connections", len(s.userIndex[c.userID.String()])),
		attribute.Int("ws.org_connections", len(s.orgIndex[c.orgID.String()])),
		attribute.Int("ws.total_rooms", len(s.roomIndex)),
		attribute.String("user_id", c.userID.String()),
		attribute.String("org_id", c.orgID.String()),
	)

	s.l.Info("client registered",
		zap.String("user_id", c.userID.String()),
		zap.String("org_id", c.orgID.String()),
		zap.String("room_id", c.roomID),
		zap.Int("user_connections", len(s.userIndex[c.userID.String()])),
		zap.Int("total_connections", len(s.clients)),
	)
}

func (s *Service) unregisterClient(c *Client) {
	ctx, span := s.tracer.StartSpan(c.ctx, "websocket.unregisterClient")
	defer span.End()

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.clients[c]; !ok {
		return
	}

	delete(s.clients, c)

	s.metrics.RecordWSDisconnection(c.orgID.String(), c.connectedAt)

	select {
	case <-c.send:
	default:
		close(c.send)
	}

	remainingConnections := 0

	userClients := s.userIndex[c.userID.String()]
	if userClients != nil {
		delete(userClients, c)
		remainingConnections = len(userClients)
		if remainingConnections == 0 {
			delete(s.userIndex, c.userID.String())
			if err := s.redis.SRem(
				ctx,
				fmt.Sprintf("org:%s:clients", c.orgID.String()),
				c.userID.String(),
			); err != nil {
				s.l.Error("failed to remove user from org in Redis", zap.Error(err))
			}
		}
	}

	orgClients := s.orgIndex[c.orgID.String()]
	if orgClients != nil {
		delete(orgClients, c)
		if len(orgClients) == 0 {
			delete(s.orgIndex, c.orgID.String())
		}
	}

	if c.roomID != "" { //nolint:nestif // this is fine
		roomClients := s.roomIndex[c.roomID]
		if roomClients != nil {
			delete(roomClients, c)
			if len(roomClients) == 0 {
				delete(s.roomIndex, c.roomID)
				if err := s.redis.SRem(ctx, "room:"+c.roomID+":users", c.userID.String()); err != nil {
					s.l.Error("failed to remove user from room in Redis", zap.Error(err))
				}
			} else {
				s.metrics.RecordWSRoomSize(c.roomID, len(roomClients))
			}
		}
	}

	s.metrics.UpdateWSRooms(len(s.roomIndex))

	if c.conn != nil && c.conn.RemoteAddr() != nil {
		if err := s.redis.SRem(
			ctx,
			fmt.Sprintf("user:%s:connections", c.userID.String()),
			c.conn.RemoteAddr().String(),
		); err != nil {
			s.l.Error("failed to remove user connection from Redis", zap.Error(err))
		}
	}

	observability.AddSpanAttributes(ctx,
		attribute.Int("ws.total_connections", len(s.clients)),
		attribute.Int("ws.user_remaining_connections", remainingConnections),
		attribute.Int("ws.org_connections", len(s.orgIndex[c.orgID.String()])),
		attribute.Int("ws.total_rooms", len(s.roomIndex)),
		attribute.String("user_id", c.userID.String()),
		attribute.String("org_id", c.orgID.String()),
	)

	s.l.Info("client unregistered",
		zap.String("user_id", c.userID.String()),
		zap.String("org_id", c.orgID.String()),
		zap.String("room_id", c.roomID),
		zap.Int("remaining_connections", remainingConnections),
		zap.Int("total_connections", len(s.clients)),
	)
}

func (s *Service) handleBroadcast(message []byte) {
	var msg Message
	if err := sonic.Unmarshal(message, &msg); err != nil {
		s.l.Error("failed to unmarshal message", zap.Error(err))
		return
	}

	if msg.ServerID == s.serverID {
		s.l.Debug("skipping message from own server",
			zap.String("server_id", msg.ServerID),
			zap.String("target", msg.Target),
			zap.String("target_id", msg.TargetID),
		)
		return
	}

	wrappedForClient := map[string]any{
		"type":      "notification",
		"data":      msg.Content,
		"timestamp": utils.NowUnix(),
	}

	contentBytes, err := sonic.Marshal(wrappedForClient)
	if err != nil {
		s.l.Error("failed to marshal content", zap.Error(err))
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

	recipientCount := 0
	for client := range clients {
		select {
		case client.send <- message:
			recipientCount++
		default:
			go s.unregisterClient(client)
		}
	}

	if recipientCount > 0 {
		s.metrics.RecordWSBroadcast("user", recipientCount)
	}
}

func (s *Service) sendToOrg(orgID string, message []byte) {
	s.mu.RLock()
	clients := s.orgIndex[orgID]
	s.mu.RUnlock()

	recipientCount := 0
	for client := range clients {
		select {
		case client.send <- message:
			recipientCount++
		default:
			go s.unregisterClient(client)
		}
	}

	if recipientCount > 0 {
		s.metrics.RecordWSBroadcast("org", recipientCount)
	}
}

func (s *Service) sendToRoom(roomID string, message []byte) {
	s.mu.RLock()
	clients := s.roomIndex[roomID]
	s.mu.RUnlock()

	recipientCount := 0
	for client := range clients {
		select {
		case client.send <- message:
			recipientCount++
		default:
			go s.unregisterClient(client)
		}
	}

	if recipientCount > 0 {
		s.metrics.RecordWSBroadcast("room", recipientCount)
	}
}

func (s *Service) BroadcastToUser(ctx context.Context, userID string, content any) {
	msg := Message{
		Type:     "notification",
		Target:   "user",
		TargetID: userID,
		ServerID: s.serverID,
		Content:  content,
	}

	s.l.Info("broadcasting to user",
		zap.String("user_id", userID),
		zap.String("server_id", s.serverID),
		zap.Any("content", content),
	)

	s.mu.RLock()
	hasLocalConnections := len(s.userIndex[userID]) > 0
	s.mu.RUnlock()

	if hasLocalConnections {
		wrappedForClient := map[string]any{
			"type":      "notification",
			"data":      content,
			"timestamp": utils.NowUnix(),
		}

		clientData, _ := sonic.Marshal(wrappedForClient)
		s.sendToUser(userID, clientData)
	}

	data, err := sonic.Marshal(msg)
	if err != nil {
		s.l.Error("failed to marshal broadcast message", zap.Error(err))
		return
	}

	if pubErr := s.redis.Publish(ctx, "broadcast:user:"+userID, data); pubErr != nil {
		s.l.Error("failed to publish to Redis", zap.Error(pubErr))
	}
}

func (s *Service) BroadcastToRoom(ctx context.Context, roomID string, content any) {
	msg := Message{
		Type:     "notification",
		Target:   "room",
		TargetID: roomID,
		ServerID: s.serverID,
		Content:  content,
	}

	s.mu.RLock()
	hasLocalConnections := len(s.roomIndex[roomID]) > 0
	s.mu.RUnlock()

	if hasLocalConnections {
		wrappedForClient := map[string]any{
			"type":      "notification",
			"data":      content,
			"timestamp": utils.NowUnix(),
		}

		clientData, _ := sonic.Marshal(wrappedForClient)
		s.sendToRoom(roomID, clientData)
	}

	data, err := sonic.Marshal(msg)
	if err != nil {
		s.l.Error("failed to marshal broadcast message", zap.Error(err))
		return
	}

	if pubErr := s.redis.Publish(ctx, "broadcast:room:"+roomID, data); pubErr != nil {
		s.l.Error("failed to publish to Redis", zap.Error(pubErr))
	}
}

func (s *Service) BroadcastToOrg(ctx context.Context, orgID string, content any) {
	msg := Message{
		Type:     "notification",
		Target:   "org",
		TargetID: orgID,
		ServerID: s.serverID,
		Content:  content,
	}

	s.mu.RLock()
	hasLocalConnections := len(s.orgIndex[orgID]) > 0
	s.mu.RUnlock()

	if hasLocalConnections {
		wrappedForClient := map[string]any{
			"type":      "notification",
			"data":      content,
			"timestamp": utils.NowUnix(),
		}

		clientData, _ := sonic.Marshal(wrappedForClient)
		s.sendToOrg(orgID, clientData)
	}

	data, err := sonic.Marshal(msg)
	if err != nil {
		s.l.Error("failed to marshal broadcast message", zap.Error(err))
		return
	}

	if pubErr := s.redis.Publish(ctx, "broadcast:org:"+orgID, data); pubErr != nil {
		s.l.Error("failed to publish to Redis", zap.Error(pubErr))
	}
}

func (s *Service) ListenToRedis(ctx context.Context) {
	pubsub := s.redis.PSubscribe(ctx, "broadcast:*")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for msg := range ch {
		s.broadcast <- []byte(msg.Payload)
	}
}

func (s *Service) HandleWebSocket(c *gin.Context) {
	reqCtx := authcontext.GetAuthContext(c)

	roomID := c.Query("room")

	s.l.Info("upgrading to websocket",
		zap.String("user_id", reqCtx.UserID.String()),
		zap.String("org_id", reqCtx.OrganizationID.String()),
		zap.String("room_id", roomID),
	)

	conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		s.l.Error("failed to upgrade connection", zap.Error(err))
		s.metrics.RecordWSError("upgrade_failed")
		return
	}

	ctx := c.Request.Context()
	ctx = observability.WithUserID(ctx, reqCtx.UserID.String())
	ctx = observability.WithOrganizationID(ctx, reqCtx.OrganizationID.String())

	client := &Client{
		service:     s,
		conn:        conn,
		send:        make(chan []byte, 256),
		userID:      reqCtx.UserID,
		orgID:       reqCtx.OrganizationID,
		roomID:      roomID,
		connectedAt: time.Now(),
		ctx:         ctx,
	}

	s.register <- client
	go s.writePump(client)
	s.readPump(ctx, client) // This blocks until connection closes
}

func (s *Service) readPump(ctx context.Context, c *Client) {
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
				s.l.Error("websocket error", zap.Error(err))
				s.metrics.RecordWSError("read_error")
			}
			break
		}

		s.metrics.RecordWSMessage("received", "client_message", len(messageData))

		msg := new(Message)
		if err = sonic.Unmarshal(messageData, &msg); err != nil {
			s.l.Error("failed to unmarshal client message", zap.Error(err))
			continue
		}

		if msg.Type == "ping" {
			pingTime := time.Now()
			pongMsg := map[string]any{
				"type": "pong",
				"data": map[string]any{
					"timestamp": utils.NowUnix(),
					"received":  msg.Content,
				},
			}
			pongBytes, _ := sonic.Marshal(pongMsg)
			select {
			case c.send <- pongBytes:
				s.metrics.RecordWSPingLatency(c.userID.String(), time.Since(pingTime).Seconds())
			default:
				s.l.Warn("failed to send pong, client send channel full",
					zap.String("user_id", c.userID.String()),
				)
			}
			continue
		}

		msg.UserID = c.userID.String()
		msg.OrgID = c.orgID.String()

		switch msg.Target {
		case "room":
			if c.roomID != "" {
				msg.TargetID = c.roomID
				s.BroadcastToRoom(ctx, c.roomID, msg.Content)
			}
		case "org":
			msg.TargetID = c.orgID.String()
			s.BroadcastToOrg(ctx, c.orgID.String(), msg.Content)
		case "user":
			if msg.TargetID != "" {
				s.BroadcastToUser(ctx, msg.TargetID, msg.Content)
			}
		}
	}
}

func (s *Service) writePump(c *Client) {
	defer func() {
		_ = c.conn.Close()
	}()

	for message := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			s.l.Error("failed to write message", zap.Error(err))
			s.metrics.RecordWSError("write_error")
			break
		}
		s.metrics.RecordWSMessage("sent", "server_message", len(message))
	}
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
	s.l.Info("all clients disconnected")
}

func (s *Service) HandleBroadcast(ctx context.Context, c *gin.Context) {
	var req BroadcastRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON payload",
		})
		return
	}

	switch req.Type {
	case "user_broadcast":
		s.BroadcastToUser(ctx, req.TargetID, req.Message)
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": fmt.Sprintf("Message sent to user %s", req.TargetID),
		})

	case "org_broadcast":
		s.BroadcastToOrg(ctx, req.TargetID, req.Message)
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": fmt.Sprintf("Message sent to organization %s", req.TargetID),
		})

	case "room_broadcast":
		s.BroadcastToRoom(ctx, req.TargetID, req.Message)
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": fmt.Sprintf("Message sent to room %s", req.TargetID),
		})

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid broadcast type. Use 'user_broadcast', 'org_broadcast', or 'room_broadcast'",
		})
	}
}

func (s *Service) GetOrgMembers(c *gin.Context) {
	orgID := c.Query("org_id")
	if orgID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "org_id query parameter is required",
		})
		return
	}

	members, err := s.redis.SMembers(c.Request.Context(), fmt.Sprintf("org:%s:clients", orgID))
	if err != nil {
		s.l.Error("failed to get organization members", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch organization members",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"org_id":  orgID,
		"members": members,
	})
}

func (s *Service) GetRoomMembers(c *gin.Context) {
	roomID := c.Query("room_id")
	if roomID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "room_id query parameter is required",
		})
		return
	}

	members, err := s.redis.SMembers(c.Request.Context(), "room:"+roomID+":users")
	if err != nil {
		s.l.Error("failed to get room members", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch room members",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"room_id": roomID,
		"members": members,
	})
}

func (s *Service) startConnectionMonitor(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.monitorConnections(ctx)
		}
	}
}

func (s *Service) monitorConnections(ctx context.Context) {
	ctx, span := s.tracer.StartSpan(ctx, "websocket.monitorConnections")
	defer span.End()

	s.mu.RLock()
	totalConnections := len(s.clients)
	totalUsers := len(s.userIndex)
	totalOrgs := len(s.orgIndex)
	totalRooms := len(s.roomIndex)
	s.mu.RUnlock()

	observability.AddSpanAttributes(ctx,
		attribute.Int("ws.monitor.total_connections", totalConnections),
		attribute.Int("ws.monitor.total_users", totalUsers),
		attribute.Int("ws.monitor.total_orgs", totalOrgs),
		attribute.Int("ws.monitor.total_rooms", totalRooms),
	)

	observability.AddSpanEvent(ctx, "connection_monitor_check",
		attribute.Int("total_connections", totalConnections),
		attribute.Int("total_users", totalUsers),
	)

	s.l.Debug("connection monitor check",
		zap.Int("total_connections", totalConnections),
		zap.Int("total_users", totalUsers),
		zap.Int("total_orgs", totalOrgs),
		zap.Int("total_rooms", totalRooms),
	)
}
