package agentexceptionhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentexception"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentexceptionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              agentexceptionports.AgentExceptionService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service agentexceptionports.AgentExceptionService
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

func requestActorFromAuthContext(authCtx *authctx.AuthContext) permissionports.RequestActor {
	return permissionports.RequestActor{
		PrincipalType:  kernelports.PrincipalType(authCtx.PrincipalType),
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
		func() (*pagination.ListResult[*agentexception.AgentException], error) {
			return h.service.List(
				c.Request.Context(),
				&agentexceptionports.ListAgentExceptionRequest{Filter: req},
			)
		},
	)
}

type resolveExceptionRequest struct {
	ResolutionState agentexception.ResolutionState `json:"resolutionState"`
	ResolutionNotes string                         `json:"resolutionNotes"`
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
		&agentexceptionports.ResolveAgentExceptionRequest{
			ID:              exceptionID,
			ResolutionState: body.ResolutionState,
			ResolutionNotes: body.ResolutionNotes,
			TenantInfo:      pagination.FromAuthAsUser(authCtx),
		},
		&actor,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, updated)
}
