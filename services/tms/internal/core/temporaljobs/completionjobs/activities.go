package completionjobs

import (
	"context"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"go.uber.org/fx"
)

type ActivitiesParams struct {
	fx.In

	Completion serviceports.CompletionService
	Providers  serviceports.AIProviderProbe
	Briefings  serviceports.BriefingService
}

type Activities struct {
	completion serviceports.CompletionService
	providers  serviceports.AIProviderProbe
	briefings  serviceports.BriefingService
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		completion: p.Completion,
		providers:  p.Providers,
		briefings:  p.Briefings,
	}
}

// CompleteStructuredActivity asks the model one structured question.
func (a *Activities) CompleteStructuredActivity(
	ctx context.Context,
	req *serviceports.StructuredCompletionRequest,
) (*serviceports.StructuredCompletionResult, error) {
	stop := modelcall.Heartbeat(ctx)
	defer stop()

	result, err := a.completion.CompleteStructured(ctx, req)
	if err != nil {
		return nil, modelcall.Classify(err)
	}

	return result, nil
}

// TestAIProviderActivity probes a provider and records the outcome on it. The
// probe reports a failed call as its result, which is what an administrator
// testing the connection wants to read, so only a failure to probe at all is
// an error.
func (a *Activities) TestAIProviderActivity(
	ctx context.Context,
	payload *TestAIProviderPayload,
) (*serviceports.TestAIProviderResult, error) {
	stop := modelcall.Heartbeat(ctx)
	defer stop()

	return a.providers.RunTest(ctx, payload.Request)
}

// WriteBriefingActivity writes a day's briefing for the roles asked for.
func (a *Activities) WriteBriefingActivity(
	ctx context.Context,
	payload *WriteBriefingPayload,
) (*serviceports.WriteBriefingResult, error) {
	stop := modelcall.Heartbeat(ctx)
	defer stop()

	return a.briefings.WriteForDay(ctx, payload.Request)
}
