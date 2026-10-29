package agentplanhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	PlanService          agentports.AgentPlanService
	PreviewService       agentruntimeports.ProposalPreviewService `optional:"true"`
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	plans    agentports.AgentPlanService
	previews agentruntimeports.ProposalPreviewService
	eh       *helpers.ErrorHandler
	pm       *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		plans:    p.PlanService,
		previews: p.PreviewService,
		eh:       p.ErrorHandler,
		pm:       p.PermissionMiddleware,
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
	api.GET(
		"/:planID/preview/",
		h.pm.RequirePermission(permission.ResourceAgentProposal.String(), permission.OpRead),
		h.preview,
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

	plan, err := h.plans.GetByID(c.Request.Context(), agentports.GetAgentPlanByIDRequest{
		ID:         planID,
		TenantInfo: tenantFromAuthContext(authCtx),
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, plan)
}

type resolvePlanRequest struct {
	Decision      agentdecision.DecisionType `json:"decision"`
	ReasonCode    string                     `json:"reasonCode"`
	Note          string                     `json:"note"`
	PreviewDigest string                     `json:"previewDigest"`
}

func actorFromAuthContext(authCtx *authctx.AuthContext) permissionports.RequestActor {
	return permissionports.RequestActor{
		PrincipalType:  kernelports.PrincipalType(authCtx.PrincipalType),
		PrincipalID:    authCtx.PrincipalID,
		UserID:         authCtx.UserID,
		APIKeyID:       authCtx.APIKeyID,
		BusinessUnitID: authCtx.BusinessUnitID,
		OrganizationID: authCtx.OrganizationID,
	}
}

func (h *Handler) preview(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	planID, err := pulid.Parse(c.Param("planID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if h.previews == nil {
		h.eh.HandleError(c, errortypes.NewBusinessError(
			"Previews of proposed changes are not available",
		))
		return
	}

	plan, err := h.plans.GetByID(c.Request.Context(), agentports.GetAgentPlanByIDRequest{
		ID:         planID,
		TenantInfo: tenantFromAuthContext(authCtx),
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := actorFromAuthContext(authCtx)
	preview, err := h.previews.ForPlan(c.Request.Context(), &agentruntimeports.PlanPreviewRequest{
		Plan:   plan,
		Viewer: &agentruntimeports.PreviewViewer{Actor: &actor},
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, preview)
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

	actor := actorFromAuthContext(authCtx)
	plan, err := h.plans.Decide(c.Request.Context(), &agentports.DecideAgentPlanRequest{
		PlanID:        planID,
		Decision:      body.Decision,
		ReasonCode:    body.ReasonCode,
		Note:          body.Note,
		PreviewDigest: body.PreviewDigest,
		TenantInfo:    tenantFromAuthContext(authCtx),
	}, &actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, plan)
}
