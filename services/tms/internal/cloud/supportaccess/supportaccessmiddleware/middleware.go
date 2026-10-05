package supportaccessmiddleware

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/cloud/domain/supportaccess"
	supportctx "github.com/emoss08/trenova/internal/cloud/supportaccess"
	"github.com/emoss08/trenova/internal/cloud/supportaccess/supportaccessservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

const (
	HeaderSessionState = "X-Trenova-Support-Session"
	StateEnded         = "ended"
	staffRoutePrefix   = "/api/v1/support/"
	authProvider       = "trenova_support"
)

var (
	errSessionEnded = errortypes.NewAuthorizationError(
		"Your Trenova support session has ended. Return to the support console to start another.",
	)
	errReadOnly = errortypes.NewAuthorizationError(
		"This Trenova support session is read-only. Elevate to write to make changes.",
	)
	errDenied = errortypes.NewAuthorizationError(
		"Trenova support sessions cannot do this, even with write access.",
	)
)

type sessionResolver interface {
	Enabled() bool
	CookieName() string
	Resolve(
		ctx context.Context,
		staff supportaccessservice.StaffContext,
		rawToken string,
	) (*supportctx.Active, *supportaccess.Session, error)
	RecordRefusal(ctx context.Context, req *supportaccessservice.RefusalRequest)
}

type Params struct {
	fx.In

	Service      *supportaccessservice.Service
	Config       *config.Config
	ErrorHandler *helpers.ErrorHandler
}

type Middleware struct {
	service sessionResolver
	session *config.SessionConfig
	eh      *helpers.ErrorHandler
}

func New(p Params) *Middleware {
	return &Middleware{
		service: p.Service,
		session: &p.Config.Security.Session,
		eh:      p.ErrorHandler,
	}
}

func (m *Middleware) ProtectedMiddleware() gin.HandlerFunc {
	return m.Handle()
}

func (m *Middleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !m.service.Enabled() {
			c.Next()
			return
		}

		raw, err := c.Cookie(m.service.CookieName())
		if err != nil || raw == "" {
			c.Next()
			return
		}

		route := routeOf(c)
		if strings.HasPrefix(route, staffRoutePrefix) {
			c.Next()
			return
		}

		authCtx := authctx.GetAuthContext(c)
		if authCtx.PrincipalType != authctx.PrincipalTypeUser || authCtx.SessionID.IsNil() {
			c.Next()
			return
		}

		active, _, err := m.service.Resolve(c.Request.Context(), StaffContextOf(authCtx), raw)
		if err != nil {
			var ended *supportaccessservice.SessionEndedError
			if errors.As(err, &ended) {
				ClearCookie(c, m.session, m.service.CookieName())
				c.Header(HeaderSessionState, StateEnded)
				m.eh.HandleError(c, errSessionEnded)
				c.Abort()
				return
			}
			m.eh.HandleError(c, err)
			c.Abort()
			return
		}

		decision := supportctx.DecideRoute(supportctx.RouteRequest{
			Method:      c.Request.Method,
			Route:       route,
			WriteActive: active.WriteActive(),
		})
		c.Header(HeaderSessionState, active.EffectiveMode().String())
		if decision != supportctx.DecisionAllow {
			m.service.RecordRefusal(c.Request.Context(), &supportaccessservice.RefusalRequest{
				Active:  active,
				Method:  c.Request.Method,
				Route:   route,
				Subject: c.Request.Method + " " + route,
				Denied:  decision == supportctx.DecisionDenied,
			})
			m.eh.HandleError(c, refusal(decision))
			c.Abort()
			return
		}

		enterTenant(c, authCtx, active)
		c.Next()
	}
}

func refusal(decision supportctx.Decision) error {
	if decision == supportctx.DecisionDenied {
		return errDenied
	}
	return errReadOnly
}

func enterTenant(c *gin.Context, staff *authctx.AuthContext, active *supportctx.Active) {
	locale, _ := authctx.GetLocale(c)

	authctx.SetSessionAuthContext(c, authctx.SessionAuthContextParams{
		SessionID:             staff.SessionID,
		UserID:                active.PrincipalUserID,
		BusinessUnitID:        active.BusinessUnitID,
		OrganizationID:        active.OrganizationID,
		ActiveRoleIDs:         nil,
		AuthProvider:          authProvider,
		AuthenticatorAAL:      staff.AuthenticatorAAL,
		FederationFAL:         staff.FederationFAL,
		MFAAuthenticatedAt:    staff.MFAAuthenticatedAt,
		LastReauthenticatedAt: staff.LastReauthenticatedAt,
		RiskDecision:          staff.RiskDecision,
		RiskDecisionID:        staff.RiskDecisionID,
		Locale:                locale,
	})
	supportctx.BindActive(c, active)
}

func StaffContextOf(authCtx *authctx.AuthContext) supportaccessservice.StaffContext {
	return supportaccessservice.StaffContext{
		UserID:             authCtx.UserID,
		OrganizationID:     authCtx.OrganizationID,
		BusinessUnitID:     authCtx.BusinessUnitID,
		SessionID:          authCtx.SessionID,
		AuthenticatorAAL:   authCtx.AuthenticatorAAL,
		MFAAuthenticatedAt: authCtx.MFAAuthenticatedAt,
	}
}

func SetCookie(c *gin.Context, cfg *config.SessionConfig, name, value string, maxAge int) {
	c.SetSameSite(cfg.GetSameSite())
	c.SetCookie(name, value, maxAge, cfg.Path, cfg.Domain, cfg.Secure, true)
}

func ClearCookie(c *gin.Context, cfg *config.SessionConfig, name string) {
	c.SetSameSite(cfg.GetSameSite())
	c.SetCookie(name, "", -1, cfg.Path, cfg.Domain, cfg.Secure, true)
	c.Header("Cache-Control", "no-store")
}

func routeOf(c *gin.Context) string {
	if route := c.FullPath(); route != "" {
		return route
	}
	return c.Request.URL.Path
}
