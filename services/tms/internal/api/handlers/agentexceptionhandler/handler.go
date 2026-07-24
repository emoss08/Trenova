package agentexceptionhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              serviceports.AgentExceptionService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service serviceports.AgentExceptionService
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
		pm:      p.PermissionMiddleware,
	}
}

func requestActorFromAuthContext(authCtx *authctx.AuthContext) serviceports.RequestActor {
	return serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalType(authCtx.PrincipalType),
		PrincipalID:    authCtx.PrincipalID,
		UserID:         authCtx.UserID,
		APIKeyID:       authCtx.APIKeyID,
		BusinessUnitID: authCtx.BusinessUnitID,
		OrganizationID: authCtx.OrganizationID,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/agent-exceptions")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceAgentException.String(), permission.OpRead),
		h.list,
	)
	api.POST(
		"/:exceptionID/resolve/",
		h.pm.RequirePermission(permission.ResourceAgentException.String(), permission.OpUpdate),
		h.resolve,
	)
}

func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*agent.AgentException], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListAgentExceptionRequest{Filter: req},
			)
		},
	)
}

type resolveExceptionRequest struct {
	ResolutionState agent.ResolutionState `json:"resolutionState"`
	ResolutionNotes string                `json:"resolutionNotes"`
}

func (h *Handler) resolve(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	exceptionID, err := pulid.Parse(c.Param("exceptionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var body resolveExceptionRequest
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := requestActorFromAuthContext(authCtx)
	updated, err := h.service.Resolve(
		c.Request.Context(),
		&serviceports.ResolveAgentExceptionRequest{
			ID:              exceptionID,
			ResolutionState: body.ResolutionState,
			ResolutionNotes: body.ResolutionNotes,
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
		},
		&actor,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}
