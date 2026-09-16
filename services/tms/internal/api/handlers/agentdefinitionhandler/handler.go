package agentdefinitionhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
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

	Service              serviceports.AgentDefinitionService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service serviceports.AgentDefinitionService
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

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/agent-definitions")
	resource := permission.ResourceAgentDefinition.String()

	api.GET("/", h.pm.RequirePermission(resource, permission.OpRead), h.list)
	api.GET("/templates/", h.pm.RequirePermission(resource, permission.OpRead), h.templates)
	api.GET("/:agentID/", h.pm.RequirePermission(resource, permission.OpRead), h.get)
	api.POST("/", h.pm.RequirePermission(resource, permission.OpCreate), h.create)
	api.PUT("/:agentID/", h.pm.RequirePermission(resource, permission.OpUpdate), h.update)
	api.DELETE("/:agentID/", h.pm.RequirePermission(resource, permission.OpDelete), h.remove)
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

func tenantFromAuthContext(authCtx *authctx.AuthContext) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	}
}

func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	// The chat picker asks for enabled agents only; the admin list wants all.
	enabledOnly := c.Query("enabledOnly") == "true"

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*agentdefinition.Definition], error) {
			return h.service.List(c.Request.Context(), &repositories.ListAgentDefinitionRequest{
				Filter:      req,
				EnabledOnly: enabledOnly,
			})
		},
	)
}

func (h *Handler) templates(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"templates": h.service.Templates()})
}

func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	agentID, err := pulid.Parse(c.Param("agentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	definition, err := h.service.GetByID(
		c.Request.Context(),
		repositories.GetAgentDefinitionByIDRequest{
			ID:         agentID,
			TenantInfo: tenantFromAuthContext(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, definition)
}

// saveAgentRequest deliberately has no system-prompt field; see the
// agentdefinition domain for why.
type saveAgentRequest struct {
	Name            string               `json:"name"`
	Description     string               `json:"description"`
	Kind            agentdefinition.Kind `json:"kind"`
	Focus           string               `json:"focus"`
	ToolNames       []string             `json:"toolNames"`
	AutonomyCeiling agent.AutonomyTier   `json:"autonomyCeiling"`
	Enabled         bool                 `json:"enabled"`
	Version         int64                `json:"version"`
}

func (r *saveAgentRequest) toServiceRequest(
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) *serviceports.SaveAgentDefinitionRequest {
	return &serviceports.SaveAgentDefinitionRequest{
		ID:              id,
		Name:            r.Name,
		Description:     r.Description,
		Kind:            r.Kind,
		Focus:           r.Focus,
		ToolNames:       r.ToolNames,
		AutonomyCeiling: r.AutonomyCeiling,
		Enabled:         r.Enabled,
		Version:         r.Version,
		TenantInfo:      tenantInfo,
	}
}

func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var body saveAgentRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := requestActorFromAuthContext(authCtx)
	created, err := h.service.Create(
		c.Request.Context(),
		body.toServiceRequest(pulid.Nil, tenantFromAuthContext(authCtx)),
		&actor,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	agentID, err := pulid.Parse(c.Param("agentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var body saveAgentRequest
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := requestActorFromAuthContext(authCtx)
	updated, err := h.service.Update(
		c.Request.Context(),
		body.toServiceRequest(agentID, tenantFromAuthContext(authCtx)),
		&actor,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

func (h *Handler) remove(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	agentID, err := pulid.Parse(c.Param("agentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := requestActorFromAuthContext(authCtx)
	if err = h.service.Delete(c.Request.Context(), repositories.DeleteAgentDefinitionRequest{
		ID:         agentID,
		TenantInfo: tenantFromAuthContext(authCtx),
	}, &actor); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
