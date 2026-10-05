package supportaccesshandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	supportctx "github.com/emoss08/trenova/internal/cloud/supportaccess"
	"github.com/emoss08/trenova/internal/cloud/supportaccess/supportaccessmiddleware"
	"github.com/emoss08/trenova/internal/cloud/supportaccess/supportaccessservice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

var errFromSupportSession = errortypes.NewAuthorizationError(
	"Support access can only be changed by the organization's own administrators",
)

type Params struct {
	fx.In

	Service              *supportaccessservice.Service
	Config               *config.Config
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *supportaccessservice.Service
	session *config.SessionConfig
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		session: &p.Config.Security.Session,
		eh:      p.ErrorHandler,
		pm:      p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterProtectedRoutes(rg *gin.RouterGroup) {
	customer := rg.Group("/support-access")
	customer.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceOrganization.String(), permission.OpRead),
		h.getGrantState,
	)
	customer.POST(
		"/grant/",
		h.pm.RequirePermission(permission.ResourceOrganization.String(), permission.OpUpdate),
		h.createGrant,
	)
	customer.POST(
		"/grant/revoke/",
		h.pm.RequirePermission(permission.ResourceOrganization.String(), permission.OpUpdate),
		h.revokeGrant,
	)

	staff := rg.Group("/support")
	staff.GET("/profile/", h.getStaffProfile)
	staff.GET("/organizations/", h.listGrantedOrganizations)
	staff.POST("/sessions/", h.startSession)
	staff.GET("/sessions/current/", h.getCurrentSession)
	staff.POST("/sessions/current/elevate/", h.elevate)
	staff.POST("/sessions/current/drop-elevation/", h.dropElevation)
	staff.POST("/sessions/current/end/", h.endSession)
}

func tenantOf(c *gin.Context) pagination.TenantInfo {
	authCtx := authctx.GetAuthContext(c)
	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}

func (h *Handler) rejectSupportSession(c *gin.Context) bool {
	if _, active := supportctx.ActiveFromGin(c); active {
		h.eh.HandleError(c, errFromSupportSession)
		return true
	}
	return false
}

func (h *Handler) staffOf(c *gin.Context) (supportaccessservice.StaffContext, bool) {
	authCtx := authctx.GetAuthContext(c)
	if authCtx.PrincipalType != authctx.PrincipalTypeUser || authCtx.SessionID.IsNil() {
		h.eh.HandleError(c, errortypes.NewAuthorizationError(
			"Support sessions need a signed-in staff member",
		))
		return supportaccessservice.StaffContext{}, false
	}

	return supportaccessmiddleware.StaffContextOf(authCtx), true
}

func (h *Handler) cookie(c *gin.Context) string {
	value, err := c.Cookie(h.service.CookieName())
	if err != nil {
		return ""
	}
	return value
}

func (h *Handler) getGrantState(c *gin.Context) {
	state, err := h.service.GrantState(c.Request.Context(), tenantOf(c))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, state)
}

func (h *Handler) createGrant(c *gin.Context) {
	if h.rejectSupportSession(c) {
		return
	}

	var req supportaccessservice.CreateGrantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = tenantOf(c)

	grant, err := h.service.CreateGrant(c.Request.Context(), &req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, grant)
}

func (h *Handler) revokeGrant(c *gin.Context) {
	if h.rejectSupportSession(c) {
		return
	}

	if err := h.service.RevokeGrant(c.Request.Context(), tenantOf(c)); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) getStaffProfile(c *gin.Context) {
	staff, ok := h.staffOf(c)
	if !ok {
		return
	}

	profile, err := h.service.StaffProfile(c.Request.Context(), staff)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, profile)
}

func (h *Handler) listGrantedOrganizations(c *gin.Context) {
	staff, ok := h.staffOf(c)
	if !ok {
		return
	}

	organizations, err := h.service.ListGrantedOrganizations(c.Request.Context(), staff)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"items": organizations})
}

func (h *Handler) startSession(c *gin.Context) {
	staff, ok := h.staffOf(c)
	if !ok {
		return
	}

	var req supportaccessservice.StartSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.Staff = staff
	req.ClientIP = c.ClientIP()
	req.UserAgent = c.Request.UserAgent()

	started, err := h.service.StartSession(c.Request.Context(), &req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	supportaccessmiddleware.SetCookie(
		c,
		h.session,
		h.service.CookieName(),
		started.Token,
		int(max(started.Session.ExpiresAt-timeutils.NowUnix(), 1)),
	)
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusCreated, started.Session)
}

func (h *Handler) getCurrentSession(c *gin.Context) {
	staff, ok := h.staffOf(c)
	if !ok {
		return
	}

	current, err := h.service.CurrentSession(c.Request.Context(), staff, h.cookie(c))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if !current.Active && h.cookie(c) != "" {
		supportaccessmiddleware.ClearCookie(c, h.session, h.service.CookieName())
	}

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, current)
}

func (h *Handler) elevate(c *gin.Context) {
	staff, ok := h.staffOf(c)
	if !ok {
		return
	}

	var req supportaccessservice.ElevateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	view, err := h.service.Elevate(c.Request.Context(), staff, h.cookie(c), &req)
	if err != nil {
		h.handleSessionError(c, err)
		return
	}

	c.JSON(http.StatusOK, view)
}

func (h *Handler) dropElevation(c *gin.Context) {
	staff, ok := h.staffOf(c)
	if !ok {
		return
	}

	view, err := h.service.DropElevation(c.Request.Context(), staff, h.cookie(c))
	if err != nil {
		h.handleSessionError(c, err)
		return
	}

	c.JSON(http.StatusOK, view)
}

func (h *Handler) endSession(c *gin.Context) {
	staff, ok := h.staffOf(c)
	if !ok {
		return
	}

	if err := h.service.EndSession(c.Request.Context(), staff, h.cookie(c)); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	supportaccessmiddleware.ClearCookie(c, h.session, h.service.CookieName())
	c.Status(http.StatusNoContent)
}

func (h *Handler) handleSessionError(c *gin.Context, err error) {
	if _, ended := supportaccessservice.AsSessionEnded(err); ended {
		supportaccessmiddleware.ClearCookie(c, h.session, h.service.CookieName())
		c.Header(supportaccessmiddleware.HeaderSessionState, supportaccessmiddleware.StateEnded)
		h.eh.HandleError(c, errortypes.NewAuthorizationError("Your Trenova support session has ended"))
		return
	}

	h.eh.HandleError(c, err)
}
