package completionjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/jobs/completionjobs/completioncontract"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/modelcall"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/briefingports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"go.uber.org/fx"
)

type ActivitiesParams struct {
	fx.In

	Completion llmports.CompletionService
	Providers  aiproviderports.AIProviderProbe
	Briefings  briefingports.BriefingService
}

type Activities struct {
	completion llmports.CompletionService
	providers  aiproviderports.AIProviderProbe
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
	req *llmports.StructuredCompletionRequest,
) (*llmports.StructuredCompletionResult, error) {
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
	payload *completioncontract.TestAIProviderPayload,
) (*aiproviderports.TestAIProviderResult, error) {
	stop := modelcall.Heartbeat(ctx)
	defer stop()

	return a.providers.RunTest(ctx, payload.Request)
}

// ListAIProviderModelsActivity asks an endpoint which models it serves.
func (a *Activities) ListAIProviderModelsActivity(
	ctx context.Context,
	payload *completioncontract.ListAIProviderModelsPayload,
) ([]aiproviderports.AIProviderModelOption, error) {
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
	payload *completioncontract.TestAIProviderDraftPayload,
) (*aiproviderports.AIProviderDraftTestResult, error) {
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
	payload *completioncontract.WriteBriefingPayload,
) (*briefingports.WriteBriefingResult, error) {
	stop := modelcall.Heartbeat(ctx)
	defer stop()

	return a.briefings.WriteForDay(ctx, payload.Request)
}
