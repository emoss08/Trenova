package agentproposalhandler_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/agentproposalhandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
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

	sent    *agentports.DecideAgentProposalRequest
	err     error
	execErr error
}

func (d *recordingDecisions) DecideWithOutcome(
	_ context.Context,
	req *agentports.DecideAgentProposalRequest,
	_ *permissionports.RequestActor,
) (*agentports.DecisionOutcome, error) {
	d.sent = req
	if d.err != nil {
		return nil, d.err
	}

	return &agentports.DecisionOutcome{
		Decision: &agentdecision.AgentDecision{
			ID:       pulid.MustNew("adec_"),
			Decision: req.Decision,
		},
		ExecutionError: d.execErr,
	}, nil
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
	agentruntimeports.ProposalPreviewService

	asked *agentruntimeports.ProposalPreviewRequest
}

func (p *digestPreviews) ForProposal(
	_ context.Context,
	req *agentruntimeports.ProposalPreviewRequest,
) (*agentpreview.ProposalPreview, error) {
	p.asked = req

	return &agentpreview.ProposalPreview{
		ProposalID: req.Proposal.ID,
		Tool:       req.Proposal.ToolName,
		Digest:     strings.Repeat("a", 64),
	}, nil
}

func newHandler(
	decisions agentports.AgentDecisionService,
	proposals agentports.AgentProposalService,
	previews agentruntimeports.ProposalPreviewService,
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
			"decision":      agentdecision.DecisionAccepted,
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

func TestResolve_AnApprovalThatRanSaysSo(t *testing.T) {
	t.Parallel()

	proposalID := pulid.MustNew("ap_")
	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/agent-proposals/" + proposalID.String() + "/resolve/").
		WithJSONBody(map[string]any{"decision": agentdecision.DecisionAccepted, "reasonCode": "ok"}).
		WithDefaultAuthContext()
	newHandler(&recordingDecisions{}, &storedProposals{}, nil).
		RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	require.Equal(t, http.StatusOK, ginCtx.ResponseCode(), string(ginCtx.ResponseBody()))
	var body map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&body))
	assert.Equal(t, proposalID.String(), body["proposalId"])
	assert.Equal(t, true, body["executed"])
	assert.Equal(t, false, body["refused"])
	assert.NotContains(t, body, "error")
	decision, ok := body["decision"].(map[string]any)
	require.True(t, ok, "the recorded decision is returned: %v", body["decision"])
	assert.Equal(t, string(agentdecision.DecisionAccepted), decision["decision"])
}

func TestResolve_AnApprovalRefusedOverAChangedRecordIsTheOutcomeNotAnError(t *testing.T) {
	t.Parallel()

	refusal := fmt.Errorf(
		"%w: the shipment S-1 was at version 3 when this was proposed and is at version 5 now, "+
			"so nothing was changed",
		agentplan.ErrTargetChanged,
	)
	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/agent-proposals/" + pulid.MustNew("ap_").String() + "/resolve/").
		WithJSONBody(map[string]any{"decision": agentdecision.DecisionAccepted, "reasonCode": "ok"}).
		WithDefaultAuthContext()
	newHandler(&recordingDecisions{execErr: refusal}, &storedProposals{}, nil).
		RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	require.Equal(t, http.StatusOK, ginCtx.ResponseCode(), string(ginCtx.ResponseBody()))
	var body map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&body))
	assert.Equal(t, false, body["executed"])
	assert.Equal(t, true, body["refused"])
	assert.Equal(t, refusal.Error(), body["error"])
	assert.NotNil(t, body["decision"], "the approval is still recorded")
}

func TestResolve_AGatedApprovalWithoutADigestIsRefused(t *testing.T) {
	t.Parallel()

	decisions := &recordingDecisions{err: agent.ReviewRequiredError()}
	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/agent-proposals/" + pulid.MustNew("ap_").String() + "/resolve/").
		WithJSONBody(map[string]any{"decision": agentdecision.DecisionAccepted, "reasonCode": "ok"}).
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
	assert.NotNil(
		t,
		previews.asked.Viewer.Actor,
		"the preview is for the caller, as the decision is",
	)
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
