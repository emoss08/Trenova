package completionjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/briefingports"
	"go.uber.org/fx"
)

type ActivitiesParams struct {
	fx.In

	Completion agentports.CompletionService
	Providers  agentports.AIProviderProbe
	Briefings  briefingports.BriefingService
}

type Activities struct {
	completion agentports.CompletionService
	providers  agentports.AIProviderProbe
	briefings  briefingports.BriefingService
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
	req *agentports.StructuredCompletionRequest,
) (*agentports.StructuredCompletionResult, error) {
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
) (*agentports.TestAIProviderResult, error) {
	stop := modelcall.Heartbeat(ctx)
	defer stop()

	return a.providers.RunTest(ctx, payload.Request)
}

// ListAIProviderModelsActivity asks an endpoint which models it serves.
func (a *Activities) ListAIProviderModelsActivity(
	ctx context.Context,
	payload *ListAIProviderModelsPayload,
) ([]agentports.AIProviderModelOption, error) {
	stop := modelcall.Heartbeat(ctx)
	defer stop()

	models, err := a.providers.RunListModels(ctx, payload.Request)
	if err != nil {
		return nil, modelcall.Classify(err)
	}

	return models, nil
}

// TestAIProviderDraftActivity probes an unsaved endpoint and records nothing.
// Like a saved provider's test, a failed call is the result.
func (a *Activities) TestAIProviderDraftActivity(
	ctx context.Context,
	payload *TestAIProviderDraftPayload,
) (*agentports.AIProviderDraftTestResult, error) {
	stop := modelcall.Heartbeat(ctx)
	defer stop()

	result, err := a.providers.RunTestDraft(ctx, payload.Request)
	if err != nil {
		return nil, modelcall.Classify(err)
	}

	return result, nil
}

// WriteBriefingActivity writes a day's briefing for the roles asked for.
func (a *Activities) WriteBriefingActivity(
	ctx context.Context,
	payload *WriteBriefingPayload,
) (*briefingports.WriteBriefingResult, error) {
	stop := modelcall.Heartbeat(ctx)
	defer stop()

	return a.briefings.WriteForDay(ctx, payload.Request)
}
