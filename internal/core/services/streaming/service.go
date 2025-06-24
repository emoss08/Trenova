package streaming

import (
	"bufio"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/valyala/fasthttp"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Config services.StreamConfig
	Logger *logger.Logger
}

// Service implements the StreamingService interface
type Service struct {
	l                     *zerolog.Logger
	mu                    sync.RWMutex
	streams               map[string]*StreamManager
	config                services.StreamConfig
	userConnections       map[string]int // userID -> connection count
	connectionMu          sync.RWMutex
	maxConnectionsPerUser int
}

// Client represents a connected streaming client
type Client struct {
	ID       string
	Writer   *bufio.Writer
	OrgID    string
	BuID     string
	UserID   string
	LastSeen time.Time
	ctx      context.Context
	cancel   context.CancelFunc
	// Connection quality tracking
	sendQueue    chan []byte
	sendTimeout  time.Duration
	isSlowClient bool
	errorCount   int
}

// NewService creates a new streaming service
func NewService(p ServiceParams) services.StreamingService {
	log := p.Logger.With().
		Str("service", "streaming").
		Logger()

	return &Service{
		l:                     &log,
		streams:               make(map[string]*StreamManager),
		config:                p.Config,
		userConnections:       make(map[string]int),
		maxConnectionsPerUser: 5, // Configurable limit per user
		connectionMu:          sync.RWMutex{},
	}
}

// StreamData implements the StreamingService interface
func (s *Service) StreamData(
	c *fiber.Ctx,
	streamKey string,
	dataFetcher services.DataFetcher,
	timestampExtractor services.TimestampExtractor,
) error {
	log := s.l.With().Str("operation", "stream_data").Logger()

	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get request context")
		return err
	}

	// Implement tenant isolation with tenant-aware stream keys
	tenantStreamKey := fmt.Sprintf(
		"%s:%s:%s",
		streamKey,
		reqCtx.OrgID.String(),
		reqCtx.BuID.String(),
	)

	// Implement per-user connection rate limiting
	userID := reqCtx.UserID.String()
	s.connectionMu.Lock()
	if s.userConnections[userID] >= s.maxConnectionsPerUser {
		s.connectionMu.Unlock()
		return fiber.NewError(fiber.StatusTooManyRequests, "Too many connections for user")
	}
	s.userConnections[userID]++
	s.connectionMu.Unlock()

	// Ensure cleanup on disconnect
	defer func() {
		s.connectionMu.Lock()
		s.userConnections[userID]--
		if s.userConnections[userID] <= 0 {
			delete(s.userConnections, userID)
		}
		s.connectionMu.Unlock()
	}()

	// Get or create stream manager with tenant-aware key
	streamMgr := s.getOrCreateStreamManager(tenantStreamKey, dataFetcher, timestampExtractor)

	// Check connection limits per stream
	if streamMgr.getClientCount() >= s.config.MaxConnections {
		log.Error().
			Int("max_connections", s.config.MaxConnections).
			Str("stream_key", streamKey).
			Int("current_connections", streamMgr.getClientCount()).
			Msg("Too many active connections for stream")
		return fiber.NewError(fiber.StatusTooManyRequests, "Too many active connections for stream")
	}

	// Generate unique client ID
	clientID := fmt.Sprintf("%s-%s-%s-%d",
		reqCtx.OrgID.String(),
		reqCtx.BuID.String(),
		reqCtx.UserID.String(),
		time.Now().UnixNano(),
	)

	// Extract user context before entering stream writer to avoid segfault
	userCtx := c.UserContext()

	// Set SSE headers
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	c.Status(fiber.StatusOK).
		Context().
		SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
			streamMgr.handleNewClient(userCtx, reqCtx, clientID, w)
		}))

	return nil
}

// GetActiveStreams returns the number of active streams for a given key
func (s *Service) GetActiveStreams(streamKey string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if streamMgr, exists := s.streams[streamKey]; exists {
		return streamMgr.getClientCount()
	}
	return 0
}

// Shutdown gracefully shuts down all active streams
func (s *Service) Shutdown() error {
	log := s.l.With().Str("operation", "shutdown").Logger()
	s.mu.Lock()
	defer s.mu.Unlock()

	for key, streamMgr := range s.streams {
		streamMgr.shutdown()
		log.Info().Str("stream_key", key).Msg("Shutting down stream")
		delete(s.streams, key)
	}

	return nil
}

// getOrCreateStreamManager gets or creates a stream manager for the given key
func (s *Service) getOrCreateStreamManager(
	streamKey string,
	dataFetcher services.DataFetcher,
	timestampExtractor services.TimestampExtractor,
) *StreamManager {
	s.mu.Lock()
	defer s.mu.Unlock()

	if streamMgr, exists := s.streams[streamKey]; exists {
		return streamMgr
	}

	streamMgr := &StreamManager{
		clients:              make(map[string]*Client),
		lastTimestamp:        time.Now().Unix(),
		sentItems:            make(map[string]int64),
		dataFetcher:          dataFetcher,
		timestampFunc:        timestampExtractor,
		config:               s.config,
		maxConsecutiveErrors: 5,
		backoffDuration:      time.Duration(s.config.PollInterval) * time.Millisecond,
		idleTimeout:          30 * time.Minute,
		maxSentItems:         10000,
		lastCleanup:          time.Now(),
		lastDataChange:       time.Now(),
		metrics: services.StreamMetrics{
			ActiveConnections: 0,
			TotalConnections:  0,
			DataFetchErrors:   0,
			LastDataFetch:     0,
		},
	}

	s.streams[streamKey] = streamMgr
	return streamMgr
}
