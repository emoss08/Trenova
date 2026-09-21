package agentplanhandler

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

	PlanService          serviceports.AgentPlanService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	plans serviceports.AgentPlanService
	eh    *helpers.ErrorHandler
	pm    *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		plans: p.PlanService,
		eh:    p.ErrorHandler,
		pm:    p.PermissionMiddleware,
	}
}

// RegisterRoutes puts plans under the proposal permission: a plan is several
// proposals decided at once, and deciding it is deciding them.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/agent-plans")
	api.GET(
		"/:planID/",
		h.pm.RequirePermission(permission.ResourceAgentProposal.String(), permission.OpRead),
		h.get,
	)
	api.POST(
		"/:planID/resolve/",
		h.pm.RequirePermission(permission.ResourceAgentProposal.String(), permission.OpUpdate),
		h.resolve,
	)
}

func tenantFromAuthContext(authCtx *authctx.AuthContext) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}

func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	planID, err := pulid.Parse(c.Param("planID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	plan, err := h.plans.GetByID(c.Request.Context(), repositories.GetAgentPlanByIDRequest{
		ID:         planID,
		TenantInfo: tenantFromAuthContext(authCtx),
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, plan)
}

type resolvePlanRequest struct {
	Decision   agent.DecisionType `json:"decision"`
	ReasonCode string             `json:"reasonCode"`
}

func (h *Handler) resolve(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	planID, err := pulid.Parse(c.Param("planID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var body resolvePlanRequest
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalType(authCtx.PrincipalType),
		PrincipalID:    authCtx.PrincipalID,
		UserID:         authCtx.UserID,
		APIKeyID:       authCtx.APIKeyID,
		BusinessUnitID: authCtx.BusinessUnitID,
		OrganizationID: authCtx.OrganizationID,
	}
	plan, err := h.plans.Decide(c.Request.Context(), &serviceports.DecideAgentPlanRequest{
		PlanID:     planID,
		Decision:   body.Decision,
		ReasonCode: body.ReasonCode,
		TenantInfo: tenantFromAuthContext(authCtx),
	}, &actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, plan)
}
