package tenderpublichandler

import (
	"net/http"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/services/tenderservice"
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

	Service      *tenderservice.Service
	ErrorHandler *helpers.ErrorHandler
}

type Handler struct {
	service *tenderservice.Service
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

func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/tender-offers")
	api.GET("/:token/", h.preview)
	api.POST("/:token/accept/", h.accept)
	api.POST("/:token/decline/", h.decline)
}

// allow throttles per token on top of the coarse global IP limiter, so one
// leaked link cannot be hammered from many addresses.
func (h *Handler) allow(token string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	if now.Sub(h.lastGC) > throttleIdleEviction || len(h.visitors) > throttleMaxEntries {
		for key, v := range h.visitors {
			if now.Sub(v.lastSeen) > throttleIdleEviction {
				delete(h.visitors, key)
			}
		}
		h.lastGC = now
	}

	v, ok := h.visitors[token]
	if !ok {
		v = &visitor{
			limiter: rate.NewLimiter(rate.Limit(throttleRatePerMinute)/60, throttleBurst),
		}
		h.visitors[token] = v
	}
	v.lastSeen = now

	return v.limiter.Allow()
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

type declineRequest struct {
	Reason string `json:"reason"`
}

func (h *Handler) accept(c *gin.Context) {
	token, ok := h.throttled(c)
	if !ok {
		return
	}

	if err := h.service.RespondByToken(
		c.Request.Context(), token, tender.ResponseActionAccept, "",
	); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "recorded"})
}

func (h *Handler) decline(c *gin.Context) {
	token, ok := h.throttled(c)
	if !ok {
		return
	}

	var req declineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Reason = ""
	}

	if err := h.service.RespondByToken(
		c.Request.Context(), token, tender.ResponseActionDecline, req.Reason,
	); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "recorded"})
}
