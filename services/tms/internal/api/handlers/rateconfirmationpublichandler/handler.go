package rateconfirmationpublichandler

import (
	"net/http"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/services/rateconfirmationservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"golang.org/x/time/rate"
)

const (
	throttleRatePerMinute = 10
	throttleBurst         = 5
	throttleIdleEviction  = 30 * time.Minute
	throttleMaxEntries    = 10000
)

type Params struct {
	fx.In

	Service      *rateconfirmationservice.Service
	ErrorHandler *helpers.ErrorHandler
}

type Handler struct {
	service *rateconfirmationservice.Service
	eh      *helpers.ErrorHandler

	mu       sync.Mutex
	visitors map[string]*visitor
	lastGC   time.Time
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func New(p Params) *Handler {
	return &Handler{
		service:  p.Service,
		eh:       p.ErrorHandler,
		visitors: make(map[string]*visitor),
		lastGC:   time.Now(),
	}
}

// RegisterPublicRoutes lives under /rate-confirmation-links rather than
// /rate-confirmations: the authed routes already bind
// /rate-confirmations/:rateConfirmationID, and gin rejects a second wildcard
// name in the same segment position.
func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/rate-confirmation-links")
	api.GET("/:token/", h.preview)
	api.POST("/:token/confirm/", h.confirm)
}

// allow throttles per token on top of the coarse global IP limiter, so one
// leaked link cannot be hammered from many addresses.
func (h *Handler) allow(token string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	if now.Sub(h.lastGC) > throttleIdleEviction {
		for key, v := range h.visitors {
			if now.Sub(v.lastSeen) > throttleIdleEviction {
				delete(h.visitors, key)
			}
		}
		h.lastGC = now
	}

	v, ok := h.visitors[token]
	if !ok {
		if len(h.visitors) >= throttleMaxEntries {
			h.evictOldestLocked()
		}
		v = &visitor{
			limiter: rate.NewLimiter(rate.Limit(throttleRatePerMinute)/60, throttleBurst),
		}
		h.visitors[token] = v
	}
	v.lastSeen = now

	return v.limiter.Allow()
}

// evictOldestLocked frees exactly one slot at the hard cap; the caller must
// hold h.mu.
func (h *Handler) evictOldestLocked() {
	var oldestKey string
	var oldestSeen time.Time
	for key, v := range h.visitors {
		if oldestKey == "" || v.lastSeen.Before(oldestSeen) {
			oldestKey = key
			oldestSeen = v.lastSeen
		}
	}
	if oldestKey != "" {
		delete(h.visitors, oldestKey)
	}
}

func (h *Handler) throttled(c *gin.Context) (string, bool) {
	token := c.Param("token")
	if !h.allow(token) {
		c.Header("Retry-After", "60")
		h.eh.HandleError(c, errortypes.NewRateLimitError("token", "Too many requests"))
		return "", false
	}
	return token, true
}

func (h *Handler) preview(c *gin.Context) {
	token, ok := h.throttled(c)
	if !ok {
		return
	}

	view, err := h.service.PreviewByToken(c.Request.Context(), token)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, view)
}

type confirmRequest struct {
	SignerName  string `json:"signerName"`
	SignerTitle string `json:"signerTitle"`
}

func (h *Handler) confirm(c *gin.Context) {
	token, ok := h.throttled(c)
	if !ok {
		return
	}

	var req confirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = confirmRequest{}
	}

	if err := h.service.ConfirmByToken(
		c.Request.Context(), token, req.SignerName, req.SignerTitle,
	); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "recorded"})
}
