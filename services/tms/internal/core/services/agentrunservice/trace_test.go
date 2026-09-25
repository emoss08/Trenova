package agentrunservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentjobs"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace/aitracetest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

func TestStartForDefinition_StartsTheWorkflowUnderTheRunsAnchor(t *testing.T) {
	t.Parallel()
	aitracetest.Install()

	def := definitionFixture(true)
	var startedUnder trace.SpanContext
	var payload *agentjobs.AgentRunPayload
	svc := &Service{
		l:           zap.NewNop(),
		validator:   NewValidator(ValidatorParams{}),
		definitions: definitionRepoFor(def),
		repo: &fakeAgentRunRepo{
			create: func(_ context.Context, entity *agent.AgentRun) (*agent.AgentRun, error) {
				return entity, nil
			},
		},
		workflows: &fakeWorkflowStarter{
			enabled: true,
			startWorkflow: func(
				ctx context.Context,
				_ client.StartWorkflowOptions,
				_ any,
				args ...any,
			) (client.WorkflowRun, error) {
				startedUnder = trace.SpanContextFromContext(ctx)
				payload, _ = args[0].(*agentjobs.AgentRunPayload)
				return nil, nil
			},
		},
		audit: &fakeAuditService{},
	}

	ctx, request := otel.Tracer("agentrunservice-test").Start(t.Context(), "POST /agent-runs/")
	run, err := svc.StartForDefinition(ctx, startRequest(def), nil)
	request.End()
	require.NoError(t, err)

	anchor := aitrace.AnchorFor(aitrace.AnchorAgentRun, run.ID.String())
	assert.Equal(t, anchor.TraceID.String(), run.TraceID, "the run records its trace")
	assert.Equal(t, anchor.TraceID, startedUnder.TraceID(),
		"the workflow starts in the run's trace, not the request's")
	assert.Equal(t, anchor.RootSpanID, startedUnder.SpanID())
	require.NotNil(t, payload)
	link, ok := aitrace.LinkFromTraceparent(payload.Origin)
	require.True(t, ok, "the payload carries what started the run")
	assert.Equal(t, request.SpanContext().SpanID(), link.SpanContext.SpanID())
}

func TestStartInline_RecordsTheTraceItRanIn(t *testing.T) {
	t.Parallel()
	aitracetest.Install()

	var created *agent.AgentRun
	svc := &Service{
		l:         zap.NewNop(),
		validator: NewValidator(ValidatorParams{}),
		repo: &fakeAgentRunRepo{
			create: func(_ context.Context, entity *agent.AgentRun) (*agent.AgentRun, error) {
				created = entity
				return entity, nil
			},
		},
		audit: &fakeAuditService{},
	}

	ctx, span := otel.Tracer("agentrunservice-test").Start(t.Context(), "DispatchPlanActivity")
	_, err := svc.StartInline(ctx, &serviceports.StartInlineAgentRunRequest{
		AgentType:   agent.TypeDispatchAssignment,
		SubjectType: agent.SubjectOrganization,
		SubjectID:   testTenant.OrgID,
		TenantInfo:  testTenant,
	}, nil)
	span.End()
	require.NoError(t, err)

	require.NotNil(t, created)
	assert.Equal(t, span.SpanContext().TraceID().String(), created.TraceID)
}
