package supportentryhandler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/api/sessioncookie"
	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/emoss08/trenova/internal/modules/identity/app/supportaccessservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	// maxBody bounds the form a control plane page posts: one assertion of at most
	// supportaccess.MaxAssertionBytes and its field name.
	maxBody       = 8 << 10
	assertionKey  = "assertion"
	formMediaType = "application/x-www-form-urlencoded"
	enteredPage   = `<!doctype html><html lang="en"><head><meta charset="utf-8">` +
		`<meta name="referrer" content="no-referrer">` +
		`<meta http-equiv="refresh" content="0;url=/">` +
		`<title>Trenova support</title></head><body>` +
		`<p>Opening Trenova&hellip; <a href="/">Continue</a></p></body></html>`
	refusedPage = `<!doctype html><html lang="en"><head><meta charset="utf-8">` +
		`<meta name="referrer" content="no-referrer">` +
		`<title>Trenova support</title></head><body>` +
		`<p>Trenova support could not open a session on this instance. ` +
		`Start again from the control plane.</p></body></html>`
)

// The entry route is limited twice: per address, and across the instance, since staff enter
// rarely and every attempt costs a signature check and a database write.
var (
	perAddress  = kernelports.RateLimitPolicy{Rate: 10, Period: time.Minute, Burst: 10}
	perInstance = kernelports.RateLimitPolicy{Rate: 60, Period: time.Minute, Burst: 30}
)

type Entrance interface {
	Enter(
		ctx context.Context,
		req *supportaccessservice.EnterRequest,
	) (*supportaccessservice.EnterResult, error)
}

type Params struct {
	fx.In

	Service  *supportaccessservice.Service
	Limits   kernelports.RateLimitStore
	Managed  *managedconfig.Config
	Security *securityconfig.Config
	Logger   *zap.Logger
}

// Handler is the one way Trenova staff enter a managed instance: a form the control plane's
// admin posts with a single-use support assertion. It is registered only on a managed instance.
type Handler struct {
	entrance Entrance
	limits   kernelports.RateLimitStore
	managed  bool
	session  *securityconfig.SessionConfig
	l        *zap.Logger
}

//nolint:gocritic // fx.In parameter structs are passed by value
func New(p Params) *Handler {
	return &Handler{
		entrance: p.Service,
		limits:   p.Limits,
		managed:  p.Managed.Enabled,
		session:  &p.Security.Session,
		l:        p.Logger.Named("handler.support-entry"),
	}
}

func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.POST("/support/enter", middleware.NewBodyLimit(maxBody), h.enter)
}

func (h *Handler) enter(c *gin.Context) {
	if !h.managed {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	if !h.allow(c) {
		return
	}
	if c.Request.URL.RawQuery != "" {
		h.l.Warn("refused a support entry that carried a query string")
		h.refuse(c, http.StatusForbidden)
		return
	}
	mediaType, _, _ := strings.Cut(c.GetHeader("Content-Type"), ";")
	if !strings.EqualFold(strings.TrimSpace(mediaType), formMediaType) {
		h.refuse(c, http.StatusForbidden)
		return
	}
	if err := c.Request.ParseForm(); err != nil {
		h.refuse(c, http.StatusRequestEntityTooLarge)
		return
	}
	values := c.Request.PostForm[assertionKey]
	if len(values) != 1 || len(c.Request.PostForm) != 1 {
		h.refuse(c, http.StatusForbidden)
		return
	}

	result, err := h.entrance.Enter(c.Request.Context(), &supportaccessservice.EnterRequest{
		Assertion: values[0],
	})
	if err != nil {
		var refusal *supportaccess.RefusalError
		if errors.As(err, &refusal) {
			h.refuse(c, http.StatusForbidden)
			return
		}
		h.l.Error("could not open a support session", zap.Error(err))
		h.refuse(c, http.StatusServiceUnavailable)
		return
	}

	sessioncookie.Set(c, h.session, result.Token, result.ExpiresAt)
	h.page(c, http.StatusOK, enteredPage)
}

// allow applies both limits in one round trip. A limiter that cannot answer refuses: entry is
// rare, and refusing it while Redis is down costs a retry, not a lockout.
func (h *Handler) allow(c *gin.Context) bool {
	decisions, err := h.limits.Check(c.Request.Context(), []kernelports.RateLimitRequest{
		{Key: "support-entry:address:" + c.ClientIP(), Policy: perAddress, Cost: 1},
		{Key: "support-entry:instance", Policy: perInstance, Cost: 1},
	})
	if err != nil {
		h.l.Error("could not check the support entry rate limit", zap.Error(err))
		h.refuse(c, http.StatusServiceUnavailable)
		return false
	}
	for _, decision := range decisions {
		if !decision.Allowed {
			h.l.Warn(
				"refused a support entry over its rate limit",
				zap.String("reason", string(supportaccess.RefusalRateLimited)),
			)
			if seconds := int(decision.RetryAfter.Round(time.Second) / time.Second); seconds > 0 {
				c.Header("Retry-After", strconv.Itoa(seconds))
			}
			h.refuse(c, http.StatusTooManyRequests)
			return false
		}
	}
	if len(decisions) != 2 {
		h.refuse(c, http.StatusServiceUnavailable)
		return false
	}
	return true
}

func (h *Handler) refuse(c *gin.Context, status int) {
	h.page(c, status, refusedPage)
}

func (h *Handler) page(c *gin.Context, status int, body string) {
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.Header("X-Frame-Options", "DENY")
	c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
	c.Data(status, "text/html; charset=utf-8", []byte(body))
	c.Abort()
}
