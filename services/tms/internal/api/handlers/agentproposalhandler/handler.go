package agentproposalhandler

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

	ProposalService      serviceports.AgentProposalService
	DecisionService      serviceports.AgentDecisionService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	proposalService serviceports.AgentProposalService
	decisionService serviceports.AgentDecisionService
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
				&repositories.ListAgentProposalRequest{Filter: req},
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
		&serviceports.DecideAgentProposalRequest{
			ProposalID:    proposalID,
			Decision:      body.Decision,
			Modifications: body.Modifications,
			ReasonCode:    body.ReasonCode,
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

	c.JSON(http.StatusOK, decision)
}
