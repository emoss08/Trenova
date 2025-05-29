package middleware

import (
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/session"
	"github.com/emoss08/trenova/internal/core/services/auth"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

const (
	maxSessionAge = 72 * time.Hour
)

type AuthMiddlewareParams struct {
	fx.In

	Logger *logger.Logger
	Config *config.Manager
	Auth   *auth.Service
}

type AuthMiddleware struct {
	auth *auth.Service
	cfg  *config.Manager
	l    *zerolog.Logger
}

func NewAuthMiddleware(p AuthMiddlewareParams) *AuthMiddleware {
	log := p.Logger.With().
		Str("middleware", "auth").
		Logger()

	return &AuthMiddleware{
		auth: p.Auth,
		cfg:  p.Config,
		l:    &log,
	}
}

// Authenticate middleware validates the session and refreshes it if needed
func (m *AuthMiddleware) Authenticate() fiber.Handler {
	return func(c *fiber.Ctx) error {
		log := m.l.With().
			Str("middleware", "auth").
			Str("path", c.Path()).
			Str("method", c.Method()).
			Str("ip", c.IP()).
			Str("userAgent", c.Get("User-Agent")).
			Logger()

		// Get session ID from cookie
		cookie := c.Cookies(m.cfg.Auth().SessionCookieName)
		if cookie == "" {
			log.Debug().Msg("no session cookie found")
			return m.handleAuthError(c, fiber.StatusUnauthorized, "unauthorized")
		}

		// Basic security checks
		if len(cookie) > 128 { // Prevent long cookie attacks
			log.Warn().Str("cookieLength", strconv.Itoa(len(cookie))).Msg("cookie too long")
			return m.handleAuthError(c, fiber.StatusBadRequest, "invalid session")
		}

		// Parse session ID
		sessionID, err := pulid.Parse(cookie)
		if err != nil {
			log.Warn().Err(err).Msg("invalid session ID format")
			return m.handleAuthError(c, fiber.StatusUnauthorized, "invalid session")
		}

		// Get and validate session
		sess, err := m.auth.RefreshSession(c.Context(), sessionID, c.IP(), c.Get("User-Agent"))
		if err != nil {
			// Check if this is a circuit breaker related error
			if strings.Contains(err.Error(), "circuit breaker is open") ||
				strings.Contains(err.Error(), "redis operation timed out") {
				log.Warn().
					Err(err).
					Str("sessionId", sessionID.String()).
					Msg("Redis unavailable during session validation, allowing degraded access")

				// Create a minimal session for degraded operation
				sess = m.createDegradedSession(sessionID, c.IP(), c.Get("User-Agent"))
			} else {
				return m.handleSessionError(c, err, sessionID, &log)
			}
		}

		// Additional security validations
		if err = m.validateSession(sess); err != nil {
			log.Warn().
				Err(err).
				Str("sessionId", sessionID.String()).
				Msg("session validation failed")
			return m.handleAuthError(c, fiber.StatusUnauthorized, "invalid session")
		}

		// Set session in context
		m.setSessionContext(c, sess)

		log.Debug().
			Str("userId", sess.UserID.String()).
			Str("sessionId", sess.ID.String()).
			Str("businessUnitId", sess.BusinessUnitID.String()).
			Str("organizationId", sess.OrganizationID.String()).
			Msg("session authenticated")

		return c.Next()
	}
}

// handleSessionError handles different types of session errors
func (m *AuthMiddleware) handleSessionError(
	c *fiber.Ctx,
	err error,
	sessionID pulid.ID,
	log *zerolog.Logger,
) error {
	switch {
	case eris.Is(err, session.ErrExpired), eris.Is(err, session.ErrNotActive):
		log.Info().Str("sessionId", sessionID.String()).Msg("session expired or inactive")
		m.clearSessionCookie(c)
		return m.handleAuthError(c, fiber.StatusUnauthorized, "session expired")

	case eris.Is(err, session.ErrNotFound):
		log.Info().Str("sessionId", sessionID.String()).Msg("session not found")
		m.clearSessionCookie(c)
		return m.handleAuthError(c, fiber.StatusUnauthorized, "session not found")

	case eris.Is(err, session.ErrIPMismatch):
		log.Warn().
			Str("sessionId", sessionID.String()).
			Str("expectedIP", c.IP()).
			Msg("session IP mismatch")
		return m.handleAuthError(c, fiber.StatusUnauthorized, "invalid session")

	default:
		log.Error().Err(err).Msg("failed to validate session")
		return m.handleAuthError(c, fiber.StatusUnauthorized, "invalid session")
	}
}

// validateSession performs additional security checks on the session
func (m *AuthMiddleware) validateSession(sess *session.Session) error {
	if sess == nil {
		return eris.New("nil session")
	}

	// Ensure critical fields are present
	if sess.UserID == "" || sess.ID == "" {
		return eris.New("invalid session data")
	}

	// Verify session age
	if time.Unix(sess.CreatedAt, 0).Add(maxSessionAge).Before(time.Now()) {
		return eris.New("session too old")
	}

	return nil
}

// setSessionContext sets session data in the request context
func (m *AuthMiddleware) setSessionContext(c *fiber.Ctx, sess *session.Session) {
	c.Locals(appctx.CTXSessionID, sess)
	c.Locals(appctx.CTXUserID, sess.UserID)
	c.Locals(appctx.CTXBusinessUnitID, sess.BusinessUnitID)
	c.Locals(appctx.CTXOrganizationID, sess.OrganizationID)
}

// handleAuthError handles authentication errors with consistent response format
func (m *AuthMiddleware) handleAuthError(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(fiber.Map{
		"error": message,
		"code":  status,
	})
}

// clearSessionCookie clears the session cookie
func (m *AuthMiddleware) clearSessionCookie(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     m.cfg.Auth().SessionCookieName,
		Value:    "",
		Expires:  time.Now().Add(-24 * time.Hour),
		HTTPOnly: true,
		Secure:   m.cfg.Auth().CookieSecure,
		SameSite: m.cfg.Auth().CookieSameSite,
		Domain:   m.cfg.Auth().CookieDomain,
		Path:     m.cfg.Auth().CookiePath,
	})
}

// createDegradedSession creates a temporary session for use when Redis is unavailable
// This allows the application to continue operating in a degraded mode
func (m *AuthMiddleware) createDegradedSession(
	sessionID pulid.ID,
	clientIP, userAgent string,
) *session.Session {
	now := timeutils.NowUnix()

	// Create a temporary session that will allow basic operations
	// Note: This is a security trade-off for availability during Redis outages
	degradedSession := &session.Session{
		ID:             sessionID,
		UserID:         pulid.MustNew("usr_"), // Placeholder - will be limited in scope
		BusinessUnitID: pulid.MustNew("bu_"),  // Placeholder
		OrganizationID: pulid.MustNew("org_"), // Placeholder
		Status:         session.StatusActive,
		IP:             clientIP,
		UserAgent:      userAgent,
		LastAccessedAt: now,
		ExpiresAt:      now + 180, // Very short expiry - 3 minutes
		CreatedAt:      now,
		UpdatedAt:      now,
		Events:         []session.Event{},
	}

	m.l.Warn().
		Str("sessionId", sessionID.String()).
		Str("clientIP", clientIP).
		Int64("expiresAt", degradedSession.ExpiresAt).
		Msg("created degraded session due to Redis unavailability")

	return degradedSession
}
