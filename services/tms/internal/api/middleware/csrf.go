package middleware

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/emoss08/trenova/internal/api/csrf"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type CSRFMiddleware struct {
	cfg          *config.Config
	errorHandler *helpers.ErrorHandler
}

func NewCSRFMiddleware(cfg *config.Config, errorHandler *helpers.ErrorHandler) *CSRFMiddleware {
	return &CSRFMiddleware{
		cfg:          cfg,
		errorHandler: errorHandler,
	}
}

type CSRFBrowserGuard struct {
	cfg            *config.Config
	errorHandler   *helpers.ErrorHandler
	logger         *zap.Logger
	trustedOrigins map[string]struct{}
	trustAll       bool
}

func NewCSRFBrowserGuard(
	cfg *config.Config,
	errorHandler *helpers.ErrorHandler,
	logger *zap.Logger,
) *CSRFBrowserGuard {
	trustedOrigins, trustAll := buildTrustedOrigins(cfg)

	return &CSRFBrowserGuard{
		cfg:            cfg,
		errorHandler:   errorHandler,
		logger:         logger.With(zap.String("middleware", "csrf_browser_guard")),
		trustedOrigins: trustedOrigins,
		trustAll:       trustAll,
	}
}

func (m *CSRFMiddleware) RequireToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isSafeMethod(c.Request.Method) {
			c.Next()
			return
		}

		authCtx := authctx.GetAuthContext(c)
		if authCtx == nil || authCtx.PrincipalType != authctx.PrincipalTypeUser {
			c.Next()
			return
		}

		sessionID, err := c.Cookie(m.cfg.Security.Session.Name)
		if err != nil || sessionID == "" {
			m.reject(c)
			return
		}

		token := c.GetHeader(m.cfg.Security.CSRF.HeaderName)
		if token == "" || !csrf.Verify(token, sessionID, m.cfg.Security.Session.Secret) {
			m.reject(c)
			return
		}

		c.Next()
	}
}

func (m *CSRFBrowserGuard) Guard() gin.HandlerFunc {
	return func(c *gin.Context) {
		mode := strings.ToLower(strings.TrimSpace(m.cfg.Security.CSRF.BrowserGuard.Mode))
		if mode == "" {
			mode = "report"
		}
		if mode == "off" || isSafeMethod(c.Request.Method) {
			c.Next()
			return
		}

		violation := m.violation(c)
		if violation == "" {
			c.Next()
			return
		}

		if mode == "report" {
			m.logViolation(c, violation)
			c.Next()
			return
		}

		m.reject(c)
	}
}

func (m *CSRFMiddleware) reject(c *gin.Context) {
	m.errorHandler.HandleError(
		c,
		errortypes.NewAuthorizationError("CSRF token required"),
	)
}

func (m *CSRFBrowserGuard) violation(c *gin.Context) string {
	fetchSite := strings.ToLower(strings.TrimSpace(c.GetHeader("Sec-Fetch-Site")))
	if fetchSite == "cross-site" {
		return "cross-site fetch metadata"
	}

	origin := strings.TrimSpace(c.GetHeader("Origin"))
	if origin != "" {
		if !m.isTrustedOrigin(origin) {
			return "untrusted origin"
		}
		return ""
	}

	if !m.hasSessionCookie(c) {
		return ""
	}

	refererOrigin, ok := requestOrigin(c.GetHeader("Referer"))
	if !ok || !m.isTrustedOrigin(refererOrigin) {
		return "missing or untrusted referer"
	}

	return ""
}

func (m *CSRFBrowserGuard) hasSessionCookie(c *gin.Context) bool {
	sessionID, err := c.Cookie(m.cfg.Security.Session.Name)
	return err == nil && strings.TrimSpace(sessionID) != ""
}

func (m *CSRFBrowserGuard) isTrustedOrigin(origin string) bool {
	if m.trustAll {
		return true
	}

	normalized, ok := requestOrigin(origin)
	if !ok {
		return false
	}

	_, ok = m.trustedOrigins[normalized]
	return ok
}

func (m *CSRFBrowserGuard) logViolation(c *gin.Context, violation string) {
	refererOrigin, _ := requestOrigin(c.GetHeader("Referer"))
	requestID := strings.TrimSpace(c.GetString("request_id"))
	if requestID == "" {
		requestID = strings.TrimSpace(c.GetHeader("X-Request-ID"))
	}

	m.logger.Warn(
		"csrf browser provenance violation",
		zap.String("violation", violation),
		zap.String("requestID", requestID),
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("origin", c.GetHeader("Origin")),
		zap.String("refererOrigin", refererOrigin),
		zap.String("fetchSite", c.GetHeader("Sec-Fetch-Site")),
	)
}

func (m *CSRFBrowserGuard) reject(c *gin.Context) {
	m.errorHandler.HandleError(
		c,
		errortypes.NewAuthorizationError("CSRF browser provenance check failed"),
	)
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}

func buildTrustedOrigins(cfg *config.Config) (map[string]struct{}, bool) {
	origins := cfg.Security.CSRF.TrustedOrigins
	if len(origins) == 0 {
		origins = cfg.Server.CORS.AllowedOrigins
	}

	trustedOrigins := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		if origin == "*" {
			return nil, true
		}
		normalized, ok := requestOrigin(origin)
		if ok {
			trustedOrigins[normalized] = struct{}{}
		}
	}

	return trustedOrigins, false
}

func requestOrigin(raw string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", false
	}
	return u.Scheme + "://" + u.Host, true
}
