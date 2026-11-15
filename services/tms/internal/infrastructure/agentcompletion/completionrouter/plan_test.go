package completionrouter

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requireTrenovaAIRefusal(t *testing.T, err error) {
	t.Helper()

	restriction, ok := errors.AsType[*errortypes.PlanRestrictionError](err)
	require.True(t, ok, "expected a plan restriction, got %v", err)
	assert.Equal(t, "Trenova AI", restriction.AddOn)
	require.ErrorIs(t, err, llmports.ErrNoProviderConfigured,
		"off the plan nothing may be attempted, as with no provider")
}

func TestCompleteStructured_OnABaseOnlyPlanCallsNoModel(t *testing.T) {
	t.Parallel()

	server, calls := chatServer(t, 200, `{"answer":"ok"}`)
	svc := newTestService(t, openAIChatProvider("local", server.URL, 10))
	svc.Features = plantest.BaseOnly()

	_, err := svc.CompleteStructured(t.Context(), generalRequest())

	requireTrenovaAIRefusal(t, err)
	assert.Zero(t, calls.Load())
}

func TestStreamChat_OnABaseOnlyPlanCallsNoModel(t *testing.T) {
	t.Parallel()

	server, calls := chatServer(t, 200, "hello")
	svc := newTestService(t, chatProvider("desk", server.URL, 10))
	svc.Features = plantest.BaseOnly()

	_, err := svc.CompleteChat(t.Context(), chatRequest(""))

	requireTrenovaAIRefusal(t, err)
	assert.Zero(t, calls.Load())
}

func TestSubmitBatch_OnABaseOnlyPlanSubmitsNothing(t *testing.T) {
	t.Parallel()

	server, calls := chatServer(t, 200, "{}")
	svc := newTestService(t, openAIChatProvider("local", server.URL, 10))
	svc.Features = plantest.BaseOnly()

	_, err := svc.SubmitBatch(t.Context(), &llmports.BatchCompletionRequest{
		TenantInfo: generalRequest().TenantInfo,
		Task:       aitask.TaskGeneral,
		Items:      []llmports.BatchCompletionItem{{CustomID: "one", Request: *generalRequest()}},
	})

	requireTrenovaAIRefusal(t, err)
	assert.Zero(t, calls.Load())
}

func TestSubmitBackground_OnABaseOnlyPlanSubmitsNothing(t *testing.T) {
	t.Parallel()

	server, calls := chatServer(t, 200, `{"answer":"ok"}`)
	svc := newTestService(t, openAIChatProvider("local", server.URL, 10))
	svc.Features = plantest.BaseOnly()

	_, err := svc.SubmitBackground(t.Context(), generalRequest())

	requireTrenovaAIRefusal(t, err)
	assert.Zero(t, calls.Load())
}

func TestEmbed_OnABaseOnlyPlanEmbedsNothing(t *testing.T) {
	t.Parallel()

	endpoint := newEmbeddingEndpoint(t)
	svc := newEmbeddingService(t)
	svc.Features = plantest.BaseOnly()
	withProviders(
		svc,
		embeddingRouterProvider(t, svc, "embedder", endpoint.server.URL, "m", "", 10),
	)

	_, err := svc.Embed(t.Context(), &aiusageports.EmbedRequest{
		TenantInfo: embedTenant(),
		Purpose:    aiusageports.EmbeddingPurposeDocument,
		Inputs:     numberedInputs(3),
	})
	requireTrenovaAIRefusal(t, err)

	_, err = svc.ConfiguredModelKey(t.Context(), embedTenant())
	requireTrenovaAIRefusal(t, err)

	assert.Empty(t, endpoint.batches())
}

func TestNovaReady_OnABaseOnlyPlanIsNotReady(t *testing.T) {
	t.Parallel()

	svc := novaService(t, &fakeRepo{nova: opusDeskProvider()})
	svc.Features = plantest.BaseOnly()

	err := svc.NovaReady(t.Context(), availabilityTenant(), aitask.TaskPaletteAnswer)

	requireTrenovaAIRefusal(t, err)
}

// Document Intelligence reads documents with the organization's model
// provider on its own: extraction runs with that add-on and without Trenova
// AI, and nothing else does.
func TestCandidatesFor_ExtractionRunsWithDocumentIntelligenceAlone(t *testing.T) {
	t.Parallel()

	nova := opusDeskProvider()
	svc := novaService(t, &fakeRepo{nova: nova})
	svc.Features = plantest.Without(
		platformcatalog.FeatureAIDesk,
		platformcatalog.FeatureAIWatchtower,
		platformcatalog.FeatureAITriage,
		platformcatalog.FeatureAgentAutomation,
		platformcatalog.FeatureAIAuditTrail,
	)

	candidates, err := svc.CandidatesFor(t.Context(), novaRun(aitask.TaskDocumentExtraction))
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	assert.Equal(t, nova.ID, candidates[0].ID)

	_, err = svc.CandidatesFor(t.Context(), novaRun(aitask.TaskDailyBriefing))
	requireTrenovaAIRefusal(t, err)
}

func TestCandidatesFor_ExtractionIsRefusedWithoutDocumentIntelligence(t *testing.T) {
	t.Parallel()

	svc := novaService(t, &fakeRepo{nova: opusDeskProvider()})
	svc.Features = plantest.Without(platformcatalog.FeatureDocumentIntelligence)

	_, err := svc.CandidatesFor(t.Context(), novaRun(aitask.TaskDocumentExtraction))

	restriction, ok := errors.AsType[*errortypes.PlanRestrictionError](err)
	require.True(t, ok, "expected a plan restriction, got %v", err)
	assert.Equal(t, "Document Intelligence", restriction.AddOn)
}

func TestCandidatesFor_WithEveryFeatureOnRoutesAsBefore(t *testing.T) {
	t.Parallel()

	nova := opusDeskProvider()
	svc := novaService(t, &fakeRepo{nova: nova})

	for _, task := range aitask.AllTasks() {
		if !task.Nova() {
			continue
		}
		candidates, err := svc.CandidatesFor(t.Context(), novaRun(task))
		require.NoErrorf(t, err, "%s", task)
		require.Lenf(t, candidates, 1, "%s", task)
	}
}
