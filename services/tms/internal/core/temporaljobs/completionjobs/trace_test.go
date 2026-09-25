package completionjobs

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace/aitracetest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/sdk/client"
)

type tracingStarter struct {
	fakeStarter

	startedUnder trace.SpanContext
}

func (s *tracingStarter) StartWorkflow(
	ctx context.Context,
	options client.StartWorkflowOptions,
	workflow any,
	args ...any,
) (client.WorkflowRun, error) {
	s.startedUnder = trace.SpanContextFromContext(ctx)

	return s.fakeStarter.StartWorkflow(ctx, options, workflow, args...)
}

func TestDispatcher_AOneShotCallIsAJobOfItsOwnLinkedToTheRequest(t *testing.T) {
	t.Parallel()
	aitracetest.Install()

	starter := &tracingStarter{fakeStarter: fakeStarter{get: func(context.Context, any) error {
		return nil
	}}}
	req := request()
	req.Attribution.Feature = aiusage.FeatureTableQuery
	ctx, caller := otel.Tracer("completionjobs-test").Start(t.Context(), "POST /graphql")

	_, err := (&Dispatcher{workflows: starter}).CompleteStructured(ctx, req)
	caller.End()
	require.NoError(t, err)

	require.True(t, starter.startedUnder.IsValid())
	assert.NotEqual(t, caller.SpanContext().TraceID(), starter.startedUnder.TraceID(),
		"the job is a trace of its own")
	job := aitracetest.One(t, starter.startedUnder.TraceID(),
		"trenova.ai.job "+string(aiusage.FeatureTableQuery))
	assert.False(t, job.Parent.IsValid())
	assert.Equal(t, starter.startedUnder.SpanID(), job.SpanContext.SpanID(),
		"the workflow is started under the job")
	require.Len(t, job.Links, 1)
	assert.Equal(t, caller.SpanContext().SpanID(), job.Links[0].SpanContext.SpanID())
	assert.Contains(t, job.Attributes,
		aitrace.TenantOrganizationID.String(req.TenantInfo.OrgID.String()))
}

func TestDispatcher_AFailedJobSaysSo(t *testing.T) {
	t.Parallel()
	aitracetest.Install()

	starter := &tracingStarter{fakeStarter: fakeStarter{get: func(context.Context, any) error {
		return errors.New("provider unreachable")
	}}}

	d := &Dispatcher{workflows: starter}
	_, err := d.Test(
		t.Context(),
		repositories.GetAIProviderByIDRequest{ID: pulid.MustNew("aiprv_")},
	)
	require.Error(t, err)

	job := aitracetest.One(t, starter.startedUnder.TraceID(), "trenova.ai.job provider_test")
	assert.Equal(t, codes.Error, job.Status.Code)
}
