package agentproposalhandler

import (
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
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

	ProposalService      agentports.AgentProposalService
	DecisionService      agentports.AgentDecisionService
	PreviewService       agentruntimeports.ProposalPreviewService `optional:"true"`
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	proposalService agentports.AgentProposalService
	decisionService agentports.AgentDecisionService
	previews        agentruntimeports.ProposalPreviewService
	eh              *helpers.ErrorHandler
	pm              *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		proposalService: p.ProposalService,
		decisionService: p.DecisionService,
		previews:        p.PreviewService,
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
	api.GET(
		"/:proposalID/preview/",
		h.pm.RequirePermission(permission.ResourceAgentProposal.String(), permission.OpRead),
		h.preview,
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
	Decision      agentdecision.DecisionType `json:"decision"`
	Modifications map[string]any             `json:"modifications"`
	ReasonCode    string                     `json:"reasonCode"`
	Note          string                     `json:"note"`
	PreviewDigest string                     `json:"previewDigest"`
}

func (h *Handler) preview(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	proposalID, err := pulid.Parse(c.Param("proposalID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	modifications, err := previewModifications(c.Query("modifications"))
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

	tenant := pagination.FromAuthAsUser(authCtx)
	proposal, err := h.proposalService.GetByID(
		c.Request.Context(),
		agentports.GetAgentProposalByIDRequest{ID: proposalID, TenantInfo: &tenant},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := requestActorFromAuthContext(authCtx)
	preview, err := h.previews.ForProposal(
		c.Request.Context(),
		&agentruntimeports.ProposalPreviewRequest{
			Proposal:      proposal,
			Modifications: modifications,
			Viewer:        &agentruntimeports.PreviewViewer{Actor: &actor},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, preview)
}

func previewModifications(raw string) (map[string]any, error) {
	if raw == "" {
		return nil, nil //nolint:nilnil // no modifications previews the change as proposed
	}

	var modifications map[string]any
	if err := sonic.UnmarshalString(raw, &modifications); err != nil {
		return nil, errortypes.NewValidationError(
			"modifications",
			errortypes.ErrInvalid,
			"Modifications must be a JSON object of the parameters changed",
		)
	}

	return modifications, nil
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
	outcome, err := h.decisionService.DecideWithOutcome(
		c.Request.Context(),
		&agentports.DecideAgentProposalRequest{
			ProposalID:    proposalID,
			Decision:      body.Decision,
			Modifications: body.Modifications,
			ReasonCode:    body.ReasonCode,
			Note:          body.Note,
			PreviewDigest: body.PreviewDigest,
			TenantInfo:    pagination.FromAuthAsUser(authCtx),
		},
		&actor,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, outcome.Result(proposalID))
}
