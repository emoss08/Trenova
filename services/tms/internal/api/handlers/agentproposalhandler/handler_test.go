package agentproposalhandler_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/agentproposalhandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingDecisions struct {
	agentports.AgentDecisionService

	sent *agentports.DecideAgentProposalRequest
	err  error
}

func (d *recordingDecisions) Decide(
	_ context.Context,
	req *agentports.DecideAgentProposalRequest,
	_ *permissionports.RequestActor,
) (*agent.AgentDecision, error) {
	d.sent = req
	if d.err != nil {
		return nil, d.err
	}

	return &agent.AgentDecision{ID: pulid.MustNew("adec_"), Decision: req.Decision}, nil
}

type storedProposals struct {
	agentports.AgentProposalService

	proposal *agent.AgentProposal
}

func (p *storedProposals) GetByID(
	context.Context,
	agentports.GetAgentProposalByIDRequest,
) (*agent.AgentProposal, error) {
	return p.proposal, nil
}

type digestPreviews struct {
	agentports.ProposalPreviewService

	asked *agentports.ProposalPreviewRequest
}

func (p *digestPreviews) ForProposal(
	_ context.Context,
	req *agentports.ProposalPreviewRequest,
) (*agent.ProposalPreview, error) {
	p.asked = req

	return &agent.ProposalPreview{
		ProposalID: req.Proposal.ID,
		Tool:       req.Proposal.ToolName,
		Digest:     strings.Repeat("a", 64),
	}, nil
}

func newHandler(
	decisions agentports.AgentDecisionService,
	proposals agentports.AgentProposalService,
	previews agentports.ProposalPreviewService,
) *agentproposalhandler.Handler {
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: zap.NewNop(),
		Config: &config.Config{App: config.AppConfig{Debug: true}},
	})

	return agentproposalhandler.New(agentproposalhandler.Params{
		ProposalService: proposals,
		DecisionService: decisions,
		PreviewService:  previews,
		ErrorHandler:    errorHandler,
		PermissionMiddleware: middleware.NewPermissionMiddleware(
			middleware.PermissionMiddlewareParams{
				PermissionEngine: &mocks.AllowAllPermissionEngine{},
				ErrorHandler:     errorHandler,
			},
		),
	})
}

func TestResolve_EchoesThePreviewDigestAndNote(t *testing.T) {
	t.Parallel()

	decisions := &recordingDecisions{}
	proposalID := pulid.MustNew("ap_")
	digest := strings.Repeat("b", 64)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/agent-proposals/" + proposalID.String() + "/resolve/").
		WithJSONBody(map[string]any{
			"decision":      agent.DecisionAccepted,
			"reasonCode":    "looks_right",
			"note":          "Matches the remittance",
			"previewDigest": digest,
		}).
		WithDefaultAuthContext()
	newHandler(decisions, &storedProposals{}, nil).RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	require.Equal(t, http.StatusOK, ginCtx.ResponseCode(), string(ginCtx.ResponseBody()))
	require.NotNil(t, decisions.sent)
	assert.Equal(t, proposalID, decisions.sent.ProposalID)
	assert.Equal(t, digest, decisions.sent.PreviewDigest)
	assert.Equal(t, "Matches the remittance", decisions.sent.Note)
}

func TestResolve_AGatedApprovalWithoutADigestIsRefused(t *testing.T) {
	t.Parallel()

	decisions := &recordingDecisions{err: agent.ReviewRequiredError()}
	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/agent-proposals/" + pulid.MustNew("ap_").String() + "/resolve/").
		WithJSONBody(map[string]any{"decision": agent.DecisionAccepted, "reasonCode": "ok"}).
		WithDefaultAuthContext()
	newHandler(decisions, &storedProposals{}, nil).RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.GreaterOrEqual(t, ginCtx.ResponseCode(), http.StatusBadRequest)
	assert.Contains(t, string(ginCtx.ResponseBody()), "must be reviewed")
}

func TestPreview_ReturnsTheDigestToEcho(t *testing.T) {
	t.Parallel()

	proposal := &agent.AgentProposal{
		ID:          pulid.MustNew("ap_"),
		ToolName:    "receivable_post_customer_payment",
		EgressClass: agent.EgressMoney,
	}
	previews := &digestPreviews{}
	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/agent-proposals/" + proposal.ID.String() + "/preview/").
		WithQuery(map[string]string{"modifications": `{"memo":"Short paid"}`}).
		WithDefaultAuthContext()
	newHandler(&recordingDecisions{}, &storedProposals{proposal: proposal}, previews).
		RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	require.Equal(t, http.StatusOK, ginCtx.ResponseCode(), string(ginCtx.ResponseBody()))
	var body map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&body))
	assert.Equal(t, strings.Repeat("a", 64), body["digest"])
	require.NotNil(t, previews.asked)
	assert.Equal(t, map[string]any{"memo": "Short paid"}, previews.asked.Modifications)
	require.NotNil(t, previews.asked.Viewer)
	assert.NotNil(t, previews.asked.Viewer.Actor, "the preview is for the caller, as the decision is")
}

func TestPreview_RefusesModificationsThatAreNotAnObject(t *testing.T) {
	t.Parallel()

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/agent-proposals/" + pulid.MustNew("ap_").String() + "/preview/").
		WithQuery(map[string]string{"modifications": `["memo"]`}).
		WithDefaultAuthContext()
	newHandler(&recordingDecisions{}, &storedProposals{}, &digestPreviews{}).
		RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.GreaterOrEqual(t, ginCtx.ResponseCode(), http.StatusBadRequest)
}
