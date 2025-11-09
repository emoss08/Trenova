package streaming

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	authCtx "github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Config *config.Config
	Logger *zap.Logger
	Tracer *observability.TracerProvider
}

type Service struct {
	l               *zap.Logger
	tracer          *observability.TracerProvider
	mu              sync.RWMutex
	streams         map[string]*StreamManager
	config          *config.StreamingConfig
	userConnections map[string]int // userID -> connection count
	connectionMu    sync.RWMutex
}

type Client struct {
	ID           string
	OrgID        string
	BuID         string
	UserID       string
	LastSeen     time.Time
	ctx          context.Context
	cancel       context.CancelFunc
	sendQueue    chan SSEMessage
	sendTimeout  time.Duration
	isSlowClient bool
	errorCount   int
	closed       bool
	closedMu     sync.Mutex
}

type SSEMessage struct {
	Event string
	Data  any
}

func NewService(p ServiceParams) services.StreamingService {
	return &Service{
		l:               p.Logger.With(zap.String("service", "streaming")),
		tracer:          p.Tracer,
		streams:         make(map[string]*StreamManager),
		config:          &p.Config.Streaming,
		userConnections: make(map[string]int),
		connectionMu:    sync.RWMutex{},
	}
}

func (s *Service) StreamData(c *gin.Context, streamKey string) {
	s.l.Info("StreamData called",
		zap.String("streamKey", streamKey),
		zap.String("path", c.Request.URL.Path),
		zap.String("method", c.Request.Method),
	)

	authContext := authCtx.GetAuthContext(c)
	if authContext == nil {
		s.l.Error("No auth context found")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	s.l.Info("Auth context found",
		zap.String("orgID", authContext.OrganizationID.String()),
		zap.String("buID", authContext.BusinessUnitID.String()),
		zap.String("userID", authContext.UserID.String()),
	)

	tenantStreamKey := fmt.Sprintf(
		"%s:%s:%s",
		streamKey,
		authContext.OrganizationID.String(),
		authContext.BusinessUnitID.String(),
	)

	s.l.Info("Tenant stream key created", zap.String("tenantStreamKey", tenantStreamKey))

	userID := authContext.UserID.String()
	s.connectionMu.Lock()
	currentConnections := s.userConnections[userID]
	if currentConnections >= s.config.MaxConnectionsPerUser {
		s.connectionMu.Unlock()
		s.l.Warn("Max connections per user exceeded",
			zap.String("userID", userID),
			zap.Int("current", currentConnections),
			zap.Int("max", s.config.MaxConnectionsPerUser),
		)
		c.AbortWithStatus(http.StatusTooManyRequests)
		return
	}
	s.userConnections[userID]++
	s.connectionMu.Unlock()

	defer func() {
		s.connectionMu.Lock()
		s.userConnections[userID]--
		if s.userConnections[userID] <= 0 {
			delete(s.userConnections, userID)
		}
		s.connectionMu.Unlock()
	}()

	streamMgr := s.getOrCreateStreamManager(tenantStreamKey)
	s.l.Info("Stream manager obtained", zap.Int("currentClients", streamMgr.getClientCount()))

	if streamMgr.getClientCount() >= s.config.MaxConnections {
		s.l.Error(
			"Too many active connections for stream",
			zap.Int("maxConnections", s.config.MaxConnections),
			zap.String("streamKey", streamKey),
			zap.Int("currentConnections", streamMgr.getClientCount()),
		)
		c.AbortWithStatus(http.StatusTooManyRequests)
		return
	}

	clientID := fmt.Sprintf("%s-%s-%s-%d",
		authContext.OrganizationID.String(),
		authContext.BusinessUnitID.String(),
		authContext.UserID.String(),
		time.Now().UnixNano(),
	)

	s.l.Info("Setting up SSE headers", zap.String("clientID", clientID))

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")
	c.Writer.Header().Set("X-Accel-Buffering", "no") // Disable proxy buffering

	s.l.Info("Starting SSE stream using Gin's native Stream method")

	streamMgr.handleNewClient(c, authContext, clientID)
	s.l.Info("handleNewClient returned", zap.String("clientID", clientID))
}

func (s *Service) GetActiveStreams(streamKey string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if streamMgr, exists := s.streams[streamKey]; exists {
		return streamMgr.getClientCount()
	}

	return 0
}

func (s *Service) BroadcastToStream(streamKey, orgID, buID string, data any) error {
	log := s.l.With(zap.String("operation", "BroadcastToStream"))
	tenantStreamKey := fmt.Sprintf("%s:%s:%s", streamKey, orgID, buID)

	log.Info("BroadcastToStream called",
		zap.String("streamKey", streamKey),
		zap.String("orgID", orgID),
		zap.String("buID", buID),
		zap.String("tenantKey", tenantStreamKey),
	)

	s.mu.RLock()
	streamMgr, exists := s.streams[tenantStreamKey]
	clientCount := 0
	if exists {
		clientCount = streamMgr.getClientCount()
	}
	s.mu.RUnlock()

	if !exists {
		log.Warn(
			"No active stream found for broadcast - no clients connected",
			zap.String("streamKey", streamKey),
			zap.String("tenantKey", tenantStreamKey),
		)
		return nil
	}

	log.Info("Broadcasting to active stream",
		zap.String("streamKey", streamKey),
		zap.Int("activeClients", clientCount),
	)

	streamMgr.broadcastDataUpdate(data)

	log.Info(
		"Data broadcasted to stream successfully",
		zap.String("streamKey", streamKey),
		zap.String("tenantKey", tenantStreamKey),
		zap.Int("clientCount", clientCount),
	)

	return nil
}

func (s *Service) Shutdown() error {
	log := s.l.With(zap.String("operation", "shutdown"))
	s.mu.Lock()
	defer s.mu.Unlock()

	for key, streamMgr := range s.streams {
		streamMgr.shutdown()
		log.Info("Shutting down stream", zap.String("streamKey", key))
		delete(s.streams, key)
	}

	return nil
}

func (s *Service) getOrCreateStreamManager(streamKey string) *StreamManager {
	s.mu.Lock()
	defer s.mu.Unlock()

	if streamMgr, exists := s.streams[streamKey]; exists {
		return streamMgr
	}

	streamMgr := &StreamManager{
		clients:        make(map[string]*Client),
		config:         s.config,
		tracer:         s.tracer,
		idleTimeout:    30 * time.Minute,
		lastDataChange: time.Now(),
		userConnCount:  make(map[string]int),
		metrics: StreamMetrics{
			ActiveConnections: 0,
			TotalConnections:  0,
		},
	}

	s.streams[streamKey] = streamMgr
	return streamMgr
}
