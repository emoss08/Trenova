package agentproposalhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
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

	ProposalService      agentports.AgentProposalService
	DecisionService      agentports.AgentDecisionService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	proposalService agentports.AgentProposalService
	decisionService agentports.AgentDecisionService
	eh              *helpers.ErrorHandler
	pm              *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		proposalService: p.ProposalService,
		decisionService: p.DecisionService,
		eh:              p.ErrorHandler,
		pm:              p.PermissionMiddleware,
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
	api := rg.Group("/agent-proposals")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceAgentProposal.String(), permission.OpRead),
		h.list,
	)
	api.POST(
		"/:proposalID/resolve/",
		h.pm.RequirePermission(permission.ResourceAgentProposal.String(), permission.OpUpdate),
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
		func() (*pagination.ListResult[*agent.AgentProposal], error) {
			return h.proposalService.List(
				c.Request.Context(),
				&agentports.ListAgentProposalRequest{Filter: req},
			)
		},
	)
}

type resolveProposalRequest struct {
	Decision      agent.DecisionType `json:"decision"`
	Modifications map[string]any     `json:"modifications"`
	ReasonCode    string             `json:"reasonCode"`
}

func (h *Handler) resolve(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	proposalID, err := pulid.Parse(c.Param("proposalID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var body resolveProposalRequest
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := requestActorFromAuthContext(authCtx)
	decision, err := h.decisionService.Decide(
		c.Request.Context(),
		&agentports.DecideAgentProposalRequest{
			ProposalID:    proposalID,
			Decision:      body.Decision,
			Modifications: body.Modifications,
			ReasonCode:    body.ReasonCode,
			TenantInfo:    pagination.FromAuthAsUser(authCtx),
		},
		&actor,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, decision)
}
