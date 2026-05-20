package middleware

import (
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type RateLimiter struct {
	cfg          config.RateLimitConfig
	errorHandler *helpers.ErrorHandler
	now          func() time.Time

	mu          sync.Mutex
	clients     map[string]*clientRateLimiter
	lastCleanup time.Time
}

type clientRateLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func NewRateLimiter(cfg *config.Config, errorHandler *helpers.ErrorHandler) *RateLimiter {
	rateLimitConfig := config.RateLimitConfig{}
	if cfg != nil {
		rateLimitConfig = cfg.Security.RateLimit
	}

	return &RateLimiter{
		cfg:          rateLimitConfig,
		errorHandler: errorHandler,
		now:          time.Now,
		clients:      make(map[string]*clientRateLimiter),
	}
}

func (m *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !m.cfg.Enabled {
			c.Next()
			return
		}

		limiter := m.limiterFor(c.ClientIP())
		if !limiter.Allow() {
			c.Header("Retry-After", "60")
			m.errorHandler.HandleError(
				c,
				errortypes.NewRateLimitError("request", "Too many requests"),
			)
			return
		}

		c.Next()
	}
}

func (m *RateLimiter) limiterFor(key string) *rate.Limiter {
	now := m.now()

	m.mu.Lock()
	defer m.mu.Unlock()

	m.cleanupLocked(now)

	client, ok := m.clients[key]
	if !ok {
		client = &clientRateLimiter{
			limiter:  rate.NewLimiter(m.rateLimit(), m.cfg.GetBurstSize()),
			lastSeen: now,
		}
		m.clients[key] = client
		return client.limiter
	}

	client.lastSeen = now
	return client.limiter
}

func (m *RateLimiter) cleanupLocked(now time.Time) {
	cleanupInterval := m.cfg.GetCleanupInterval()
	if !m.lastCleanup.IsZero() && now.Sub(m.lastCleanup) < cleanupInterval {
		return
	}

	for key, client := range m.clients {
		if now.Sub(client.lastSeen) > cleanupInterval {
			delete(m.clients, key)
		}
	}
	m.lastCleanup = now
}

func (m *RateLimiter) rateLimit() rate.Limit {
	return rate.Every(time.Minute / time.Duration(m.cfg.GetRequestsPerMinute()))
}
