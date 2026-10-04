package cloudsignuphandler

import (
	"net/http"
	"sort"
	"time"

	"github.com/emoss08/trenova/internal/api/csrf"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/sessioncookie"
	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	signupLimitPeriod     = time.Hour
	signupLimitPrefix     = "cloud_signup:ip:"
	verifyLimitPrefix     = "cloud_signup_verify:ip:"
	verifyLimitMultiplier = 10
)

type Params struct {
	fx.In

	Service        services.CloudSignupService
	Catalog        *platformplan.Catalog
	RateLimitStore repositories.RateLimitStore
	Config         *config.Config
	ErrorHandler   *helpers.ErrorHandler
	Logger         *zap.Logger
}

type Handler struct {
	service services.CloudSignupService
	catalog *platformplan.Catalog
	store   repositories.RateLimitStore
	cfg     *config.Config
	eh      *helpers.ErrorHandler
	l       *zap.Logger
}

func New(p Params) *Handler {
	logger := p.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Handler{
		service: p.Service,
		catalog: p.Catalog,
		store:   p.RateLimitStore,
		cfg:     p.Config,
		eh:      p.ErrorHandler,
		l:       logger.Named("handler.cloud-signup"),
	}
}

type PublicConfigResponse struct {
	PlatformMode     string          `json:"platformMode"`
	SignupEnabled    bool            `json:"signupEnabled"`
	TurnstileSiteKey string          `json:"turnstileSiteKey"`
	TermsURL         string          `json:"termsUrl"`
	PrivacyURL       string          `json:"privacyUrl"`
	FreePlan         FreePlanSummary `json:"freePlan"`
}

type FreePlanSummary struct {
	Limits map[string]int64 `json:"limits"`
}

func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	system := rg.Group("/system")
	system.GET("/public-config", h.publicConfig)

	cloud := rg.Group("/cloud/signups")
	cloud.Use(h.requireSignupEnabled())
	cloud.POST("", h.limitSignups(signupLimitPrefix, 1), h.signup)
	cloud.POST("/resend", h.limitSignups(signupLimitPrefix, 1), h.resend)
	cloud.POST("/verify", h.limitSignups(verifyLimitPrefix, verifyLimitMultiplier), h.verify)
}

func (h *Handler) publicConfig(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=60")
	c.JSON(http.StatusOK, h.buildPublicConfig())
}

func (h *Handler) buildPublicConfig() *PublicConfigResponse {
	platform := &h.cfg.Platform
	cloud := &platform.Cloud
	signupEnabled := h.service != nil && h.service.Enabled()

	resp := &PublicConfigResponse{
		PlatformMode:  string(platform.GetMode()),
		SignupEnabled: signupEnabled,
		FreePlan:      FreePlanSummary{Limits: map[string]int64{}},
	}

	if !platform.IsCloud() {
		return resp
	}

	resp.TermsURL = cloud.Signup.GetTermsURL()
	resp.PrivacyURL = cloud.Signup.GetPrivacyURL()
	if signupEnabled && cloud.Turnstile.Enabled {
		resp.TurnstileSiteKey = cloud.Turnstile.SiteKey
	}

	if h.catalog != nil {
		if plan := h.catalog.FreeDemo(); plan != nil {
			resp.FreePlan.Limits = limitsOf(plan)
		}
	}

	return resp
}

func limitsOf(plan *platformplan.Plan) map[string]int64 {
	keys := plan.MeterKeys()
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	limits := make(map[string]int64, len(keys))
	for _, key := range keys {
		if limit, ok := plan.Limit(key); ok {
			limits[string(key)] = limit.Max
		}
	}

	return limits
}

func (h *Handler) signup(c *gin.Context) {
	var req services.CloudSignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	resp, err := h.service.Signup(c.Request.Context(), &req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, resp)
}

func (h *Handler) resend(c *gin.Context) {
	var req services.CloudSignupResendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	resp, err := h.service.Resend(c.Request.Context(), &req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, resp)
}

func (h *Handler) verify(c *gin.Context) {
	var req services.CloudSignupVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	resp, err := h.service.Verify(c.Request.Context(), &req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	authctx.SetAuthContext(
		c,
		resp.User.ID,
		resp.User.BusinessUnitID,
		resp.User.CurrentOrganizationID,
	)

	sessioncookie.Set(c, &h.cfg.Security.Session, resp.SessionToken, resp.ExpiresAt)
	resp.CSRFToken = csrf.Token(resp.SessionID, h.cfg.Security.Session.Secret)

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) requireSignupEnabled() gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.service == nil || !h.service.Enabled() {
			h.eh.HandleError(c, errortypes.NewNotFoundError("Signup is not available"))
			return
		}

		c.Next()
	}
}

func (h *Handler) limitSignups(prefix string, multiplier int) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.store == nil {
			c.Next()
			return
		}

		perHour := h.cfg.Platform.Cloud.Signup.GetPerIPPerHour() * multiplier
		decisions, err := h.store.Check(c.Request.Context(), []repositories.RateLimitRequest{{
			Key: prefix + c.ClientIP(),
			Policy: repositories.RateLimitPolicy{
				Rate:   perHour,
				Period: signupLimitPeriod,
				Burst:  perHour,
			},
			Cost: 1,
		}})
		if err != nil {
			h.l.Warn("signup rate limit check failed; allowing the request", zap.Error(err))
			c.Next()
			return
		}

		if len(decisions) > 0 && !decisions[0].Allowed {
			h.eh.HandleError(c, errortypes.NewRateLimitError(
				"",
				"Too many signup attempts from this network. Try again later.",
			).WithRetryAfter(decisions[0].RetryAfter))
			return
		}

		c.Next()
	}
}
