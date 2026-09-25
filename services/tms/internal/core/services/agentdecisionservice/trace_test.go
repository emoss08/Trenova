package agentdecisionservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/internal/core/services/agentshadow"
	"github.com/emoss08/trenova/internal/core/services/proposalexecutor"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace/aitracetest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type traceControls struct {
	repositories.AgentControlRepository
}

func (traceControls) GetOrCreate(context.Context, pagination.TenantInfo) (*tenant.AgentControl, error) {
	return &tenant.AgentControl{}, nil
}

type traceRuns struct {
	repositories.AgentRunRepository
}

func (traceRuns) GetByID(
	_ context.Context,
	req repositories.GetAgentRunByIDRequest,
) (*agent.AgentRun, error) {
	return &agent.AgentRun{ID: req.ID}, nil
}

type traceProposals struct {
	repositories.AgentProposalRepository

	proposal *agent.AgentProposal
	executed []repositories.RecordAgentProposalExecutionRequest
}

func (p *traceProposals) GetByID(
	context.Context,
	repositories.GetAgentProposalByIDRequest,
) (*agent.AgentProposal, error) {
	return p.proposal, nil
}

func (p *traceProposals) UpdateStatus(
	context.Context,
	repositories.UpdateAgentProposalStatusRequest,
) (*agent.AgentProposal, error) {
	return p.proposal, nil
}

func (p *traceProposals) RecordExecution(
	_ context.Context,
	req repositories.RecordAgentProposalExecutionRequest,
) (*agent.AgentProposal, error) {
	p.executed = append(p.executed, req)

	return p.proposal, nil
}

type traceDecisions struct {
	repositories.AgentDecisionRepository

	created *agent.AgentDecision
}

func (d *traceDecisions) Create(
	_ context.Context,
	decision *agent.AgentDecision,
) (*agent.AgentDecision, error) {
	decision.ID = pulid.MustNew("ad_")
	d.created = decision

	return decision, nil
}

type traceAudit struct {
	services.AuditService
}

func (traceAudit) LogAction(*services.LogActionParams, ...services.LogOption) error { return nil }

type traceVersions struct{ versions []int64 }

func (v *traceVersions) Version(
	context.Context,
	pagination.TenantInfo,
	services.ToolTarget,
) (int64, error) {
	version := v.versions[0]
	if len(v.versions) > 1 {
		v.versions = v.versions[1:]
	}

	return version, nil
}

type decideHarness struct {
	service   *Service
	proposals *traceProposals
	decisions *traceDecisions
	actor     *services.RequestActor
	proposing trace.SpanContext
}

func newDecideHarness(t *testing.T) *decideHarness {
	t.Helper()
	aitracetest.Install()

	actor := &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
	run := aitrace.AnchorFor(aitrace.AnchorAgentRun, pulid.MustNew("ar_").String())
	_, proposing := otel.Tracer("agentdecisionservice-test").
		Start(aitrace.ContextWithAnchor(t.Context(), run), "execute_tool place_shipment_hold")
	proposing.End()

	proposals := &traceProposals{proposal: &agent.AgentProposal{
		ID:             pulid.MustNew("ap_"),
		OrganizationID: actor.OrganizationID,
		BusinessUnitID: actor.BusinessUnitID,
		RunID:          pulid.MustNew("ar_"),
		ToolName:       "place_shipment_hold",
		ToolParams:     map[string]any{},
		AutonomyTier:   agent.TierPropose,
		Status:         agent.ProposalStatusPending,
		ExpiresAt:      timeutils.NowUnix() + 3600,
		TraceID:        proposing.SpanContext().TraceID().String(),
		SpanID:         proposing.SpanContext().SpanID().String(),
		StepKey:        "step-proposing",
		TargetResource: string(permission.ResourceShipmentMove),
		TargetID:       pulid.MustNew("smv_"),
		TargetVersion:  3,
	}}
	decisions := &traceDecisions{}
	runs := traceRuns{}
	executor := proposalexecutor.New(proposalexecutor.Params{
		Logger: zap.NewNop(),
		Tools: &agentruntimetest.StubActionRegistry{Tools: []services.AgentTool{
			&agentruntimetest.StubActionTool{ToolName: "place_shipment_hold"},
		}},
		ProposalRepo: proposals,
		Permissions:  &agentruntimetest.StubPermissions{},
		AuditService: traceAudit{},
		Versions:     &traceVersions{versions: []int64{3, 4}},
	})

	return &decideHarness{
		service: &Service{
			l:            zap.NewNop(),
			decisionRepo: decisions,
			proposalRepo: proposals,
			runRepo:      runs,
			shadow:       agentshadow.New(agentshadow.Params{Control: traceControls{}, Runs: runs}),
			executor:     executor,
			audit:        traceAudit{},
		},
		proposals: proposals,
		decisions: decisions,
		actor:     actor,
		proposing: proposing.SpanContext(),
	}
}

func (h *decideHarness) decide(
	t *testing.T,
	ctx context.Context,
	decision agent.DecisionType,
) *services.DecisionOutcome {
	t.Helper()

	outcome, err := h.service.DecideWithOutcome(ctx, &services.DecideAgentProposalRequest{
		ProposalID: h.proposals.proposal.ID,
		Decision:   decision,
		ReasonCode: "looks_right",
		TenantInfo: pagination.TenantInfo{
			OrgID: h.actor.OrganizationID,
			BuID:  h.actor.BusinessUnitID,
		},
	}, h.actor)
	require.NoError(t, err)

	return outcome
}

func TestDecide_LinksBackToTheSpanThatProposedIt(t *testing.T) {
	t.Parallel()

	h := newDecideHarness(t)
	ctx, request := otel.Tracer("agentdecisionservice-test").Start(t.Context(), "POST /graphql")
	h.decide(t, ctx, agent.DecisionRejected)
	request.End()

	requestTrace := request.SpanContext().TraceID()
	decide := aitracetest.One(t, requestTrace, "trenova.ai.proposal.decide")
	assert.Equal(t, request.SpanContext().SpanID(), decide.Parent.SpanID())
	require.Len(t, decide.Links, 1)
	assert.Equal(t, h.proposing.TraceID(), decide.Links[0].SpanContext.TraceID(),
		"a decision made hours later still reaches the trace that proposed it")
	assert.Equal(t, h.proposing.SpanID(), decide.Links[0].SpanContext.SpanID())
	for _, want := range []attribute.KeyValue{
		aitrace.AIProposalID.String(h.proposals.proposal.ID.String()),
		aitrace.AIDecision.String(string(agent.DecisionRejected)),
		aitrace.UserID.String(h.actor.UserID.String()),
		aitrace.GenAIToolName.String("place_shipment_hold"),
	} {
		assert.Contains(t, decide.Attributes, want)
	}

	require.NotNil(t, h.decisions.created)
	assert.Equal(t, requestTrace.String(), h.decisions.created.TraceID,
		"the decision records the trace it was made in")
	assert.Empty(t, aitracetest.Named(requestTrace, "trenova.ai.proposal.execute"),
		"a rejection runs nothing")
}

func TestDecide_AnApprovalRunsTheWriteInsideItsExecuteSpan(t *testing.T) {
	t.Parallel()

	h := newDecideHarness(t)
	ctx, request := otel.Tracer("agentdecisionservice-test").Start(t.Context(), "POST /graphql")
	outcome := h.decide(t, ctx, agent.DecisionAccepted)
	request.End()
	require.NoError(t, outcome.ExecutionError)

	requestTrace := request.SpanContext().TraceID()
	decide := aitracetest.One(t, requestTrace, "trenova.ai.proposal.decide")
	execute := aitracetest.One(t, requestTrace, "trenova.ai.proposal.execute")
	tool := aitracetest.One(t, requestTrace, "execute_tool place_shipment_hold")
	write := aitracetest.One(t, requestTrace, "trenova.ai.write shipment_move")

	assert.Equal(t, decide.SpanContext.SpanID(), execute.Parent.SpanID())
	require.Len(t, execute.Links, 1)
	assert.Equal(t, h.proposing.SpanID(), execute.Links[0].SpanContext.SpanID())
	assert.Equal(t, execute.SpanContext.SpanID(), tool.Parent.SpanID())
	assert.Equal(t, tool.SpanContext.SpanID(), write.Parent.SpanID())
	assert.Contains(t, tool.Attributes, aitrace.AIOutcome.String(aitrace.OutcomeRan))
	assert.Contains(t, tool.Attributes, aitrace.AIStepKey.String("step-proposing"))
	assert.Contains(t, write.Attributes, aitrace.AIVersionBefore.Int64(3))
	assert.Contains(t, write.Attributes, aitrace.AIVersionAfter.Int64(4))

	require.Len(t, h.proposals.executed, 1)
	executed := h.proposals.executed[0]
	assert.Equal(t, agent.ProposalStatusExecuted, executed.Status)
	assert.Equal(t, h.actor.UserID, executed.ExecutedByUserID, "the approver ran the write")
	require.NotNil(t, executed.ExecutedTargetVersion)
	assert.Equal(t, int64(4), *executed.ExecutedTargetVersion)
}
