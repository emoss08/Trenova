package retrievaljobs

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/retrievalservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

const (
	testActiveKey  = "api.example.com/embed-small@768"
	testPendingKey = "api.example.com/embed-large@1024"
)

func indexInput() retrievalservice.IndexOrganizationInput {
	return retrievalservice.IndexOrganizationInput{
		OrganizationID: pulid.ID("org_a"),
		BusinessUnitID: pulid.ID("bu_a"),
	}
}

func activePlan() *serviceports.RetrievalIndexPlan {
	return &serviceports.RetrievalIndexPlan{
		Active:         true,
		ActiveModelKey: testActiveKey,
		SourceTypes:    airetrieval.AllSourceTypes(),
	}
}

func newEnv(t *testing.T) *testsuite.TestWorkflowEnvironment {
	t.Helper()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	return env
}

func TestIndexOrganizationWorkflowDrainsTheOutboxThenEnds(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	var a *Activities

	env.OnActivity(a.PlanRetrievalIndexActivity, mock.Anything, mock.Anything).
		Return(activePlan(), nil)

	batches := []serviceports.RetrievalIndexBatchResult{
		{Claimed: 25, Indexed: 24, Failed: 1, ChunksEmbedded: 30,
			CostUSD: decimal.RequireFromString("0.02")},
		{Claimed: 3, Indexed: 3, ChunksEmbedded: 3, CostUSD: decimal.RequireFromString("0.01")},
	}
	env.OnActivity(a.IndexRetrievalBatchActivity, mock.Anything, mock.Anything).
		Return(func(
			_ context.Context,
			input *serviceports.RetrievalIndexBatchRequest,
		) (*serviceports.RetrievalIndexBatchResult, error) {
			assert.Equal(t, testActiveKey, input.ModelKey)
			assert.Equal(t, retrievalservice.DefaultBatchSize, input.Limit)
			if len(batches) == 0 {
				return &serviceports.RetrievalIndexBatchResult{CostUSD: decimal.Zero}, nil
			}
			next := batches[0]
			batches = batches[1:]
			return &next, nil
		})
	env.OnActivity(a.SweepRetrievalSourcesActivity, mock.Anything, mock.Anything).
		Return(func(
			_ context.Context,
			input *OrganizationSweepInput,
		) (*serviceports.RetrievalSweepResult, error) {
			assert.False(t, input.Wake, "the running indexer does not signal itself")
			return &serviceports.RetrievalSweepResult{}, nil
		})

	env.ExecuteWorkflow(IndexOrganizationWorkflow, indexInput())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result IndexOrganizationResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, 2, result.Batches)
	assert.Equal(t, 27, result.Indexed)
	assert.Equal(t, 1, result.Failed)
	assert.Equal(t, 33, result.ChunksEmbedded)
	assert.True(t, result.CostUSD.Equal(decimal.RequireFromString("0.03")))
	assert.Empty(t, result.Stopped)
}

func TestIndexOrganizationWorkflowPicksUpASignalWhileIdle(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	var a *Activities

	env.OnActivity(a.PlanRetrievalIndexActivity, mock.Anything, mock.Anything).
		Return(activePlan(), nil)

	claimed := 0
	signalled := false
	env.OnActivity(a.IndexRetrievalBatchActivity, mock.Anything, mock.Anything).
		Return(func(
			context.Context,
			*serviceports.RetrievalIndexBatchRequest,
		) (*serviceports.RetrievalIndexBatchResult, error) {
			if signalled && claimed == 0 {
				claimed++
				return &serviceports.RetrievalIndexBatchResult{
					Claimed: 1, Indexed: 1, CostUSD: decimal.Zero,
				}, nil
			}
			return &serviceports.RetrievalIndexBatchResult{CostUSD: decimal.Zero}, nil
		})
	env.OnActivity(a.SweepRetrievalSourcesActivity, mock.Anything, mock.Anything).
		Return(&serviceports.RetrievalSweepResult{}, nil)

	env.RegisterDelayedCallback(func() {
		signalled = true
		env.SignalWorkflow(retrievalservice.IndexSignalName, retrievalservice.IndexSignal{
			SourceType: airetrieval.SourceTypeDocument,
			Count:      1,
		})
	}, time.Minute)

	env.ExecuteWorkflow(IndexOrganizationWorkflow, indexInput())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result IndexOrganizationResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, 1, result.Indexed, "work marked while the indexer waited is indexed")
}

func TestIndexOrganizationWorkflowStopsAtTheBudget(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	var a *Activities

	env.OnActivity(a.PlanRetrievalIndexActivity, mock.Anything, mock.Anything).
		Return(activePlan(), nil)
	env.OnActivity(a.IndexRetrievalBatchActivity, mock.Anything, mock.Anything).
		Return(&serviceports.RetrievalIndexBatchResult{
			Claimed: 10, Indexed: 10, BudgetReached: true, CostUSD: decimal.Zero,
		}, nil).
		Once()

	env.ExecuteWorkflow(IndexOrganizationWorkflow, indexInput())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result IndexOrganizationResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, airetrieval.UnavailableReasonBudgetPaused, result.Stopped)
	env.AssertExpectations(t)
}

func TestIndexOrganizationWorkflowEndsWhenIndexingIsUnavailable(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	var a *Activities

	env.OnActivity(a.PlanRetrievalIndexActivity, mock.Anything, mock.Anything).
		Return(&serviceports.RetrievalIndexPlan{Reason: airetrieval.UnavailableReasonNoProvider}, nil)

	env.ExecuteWorkflow(IndexOrganizationWorkflow, indexInput())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result IndexOrganizationResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, airetrieval.UnavailableReasonNoProvider, result.Stopped)
}

func TestIndexOrganizationWorkflowSwapsAFullyIndexedModelAndPurgesTheOldOne(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	var a *Activities

	plan := activePlan()
	plan.PendingModelKey = testPendingKey
	plan.RetiredKeys = []string{"old.example.com/embed@1536"}
	swapped := false
	env.OnActivity(a.PlanRetrievalIndexActivity, mock.Anything, mock.Anything).
		Return(func(context.Context, *PlanInput) (*serviceports.RetrievalIndexPlan, error) {
			if swapped {
				return &serviceports.RetrievalIndexPlan{
					Active:         true,
					ActiveModelKey: testPendingKey,
					SourceTypes:    airetrieval.AllSourceTypes(),
				}, nil
			}
			return plan, nil
		})

	keys := make([]string, 0, 4)
	env.OnActivity(a.IndexRetrievalBatchActivity, mock.Anything, mock.Anything).
		Return(func(
			_ context.Context,
			input *serviceports.RetrievalIndexBatchRequest,
		) (*serviceports.RetrievalIndexBatchResult, error) {
			keys = append(keys, input.ModelKey)
			return &serviceports.RetrievalIndexBatchResult{CostUSD: decimal.Zero}, nil
		})
	env.OnActivity(a.CompleteRetrievalModelChangeActivity, mock.Anything, mock.Anything).
		Return(func(
			_ context.Context,
			input *ModelChangeInput,
		) (*serviceports.RetrievalModelChangeResult, error) {
			assert.Equal(t, testPendingKey, input.PendingModelKey)
			swapped = true
			return &serviceports.RetrievalModelChangeResult{
				Swapped:         true,
				RetiredModelKey: testActiveKey,
			}, nil
		}).
		Once()

	purged := make([]string, 0, 3)
	env.OnActivity(a.PurgeRetrievalModelActivity, mock.Anything, mock.Anything).
		Return(func(
			_ context.Context,
			input *PurgeInput,
		) (*serviceports.RetrievalPurgeResult, error) {
			purged = append(purged, input.ModelKey)
			return &serviceports.RetrievalPurgeResult{Embeddings: 10, Done: true}, nil
		})
	env.OnActivity(a.SweepRetrievalSourcesActivity, mock.Anything, mock.Anything).
		Return(&serviceports.RetrievalSweepResult{}, nil)

	env.ExecuteWorkflow(IndexOrganizationWorkflow, indexInput())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result IndexOrganizationResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.True(t, result.Swapped)
	assert.Contains(t, keys, testActiveKey)
	assert.Contains(t, keys, testPendingKey)
	assert.Equal(t, []string{"old.example.com/embed@1536", testActiveKey}, purged[:2])
}

func TestIndexOrganizationWorkflowContinuesAsNewAfterItsSteps(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	var a *Activities

	env.OnActivity(a.PlanRetrievalIndexActivity, mock.Anything, mock.Anything).
		Return(activePlan(), nil)
	env.OnActivity(a.IndexRetrievalBatchActivity, mock.Anything, mock.Anything).
		Return(&serviceports.RetrievalIndexBatchResult{
			Claimed: 25, Indexed: 25, CostUSD: decimal.Zero,
		}, nil)

	env.ExecuteWorkflow(IndexOrganizationWorkflow, indexInput())

	require.True(t, env.IsWorkflowCompleted())
	var continued *workflow.ContinueAsNewError
	require.ErrorAs(t, env.GetWorkflowError(), &continued)
}

func TestReindexRetrievalSourceWorkflowPagesThroughTheSources(t *testing.T) {
	t.Parallel()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	var a *Activities
	pages := []serviceports.RetrievalReindexPage{
		{Marked: 500, Next: pulid.ID("doc_500")},
		{Marked: 120, Next: pulid.ID("doc_620"), Done: true},
	}
	afters := make([]pulid.ID, 0, 2)
	env.OnActivity(a.ReindexRetrievalPageActivity, mock.Anything, mock.Anything).
		Return(func(
			_ context.Context,
			input *serviceports.RetrievalReindexPageRequest,
		) (*serviceports.RetrievalReindexPage, error) {
			afters = append(afters, input.AfterID)
			assert.Equal(t, airetrieval.SourceTypeDocument, input.SourceType)
			next := pages[0]
			pages = pages[1:]
			return &next, nil
		})

	env.ExecuteWorkflow(ReindexRetrievalSourceWorkflow, retrievalservice.ReindexSourceInput{
		OrganizationID: pulid.ID("org_a"),
		BusinessUnitID: pulid.ID("bu_a"),
		SourceType:     airetrieval.SourceTypeDocument,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result ReindexResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, 620, result.Marked)
	assert.True(t, result.Done)
	assert.Equal(t, []pulid.ID{pulid.Nil, pulid.ID("doc_500")}, afters)
}

func TestRetrievalIndexSweepWorkflowWakesEachOrganization(t *testing.T) {
	t.Parallel()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	var a *Activities
	env.OnActivity(a.ListRetrievalOrganizationsActivity, mock.Anything, mock.Anything).
		Return(&temporaljobs.TenantPage{Tenants: []temporaljobs.TenantWorkItem{
			{OrganizationID: pulid.ID("org_a"), BusinessUnitID: pulid.ID("bu_a")},
			{OrganizationID: pulid.ID("org_b"), BusinessUnitID: pulid.ID("bu_b")},
		}}, nil).
		Once()
	env.OnActivity(a.SweepRetrievalSourcesActivity, mock.Anything, mock.Anything).
		Return(func(
			_ context.Context,
			input *OrganizationSweepInput,
		) (*serviceports.RetrievalSweepResult, error) {
			assert.True(t, input.Wake)
			return &serviceports.RetrievalSweepResult{Marked: 3}, nil
		})

	env.ExecuteWorkflow(RetrievalIndexSweepWorkflow, &SweepInput{})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result SweepResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, 2, result.OrganizationsSwept)
	assert.Equal(t, 6, result.Marked)
	assert.Empty(t, result.FailedOrganizations)
}

func TestSchedulesMatchTheirWorkflowSignatures(t *testing.T) {
	t.Parallel()

	for _, s := range NewScheduleProvider().GetSchedules() {
		require.NoError(t, s.Validate(), s.ID)
	}
}

func TestWorkflowNamesMatchTheServiceNames(t *testing.T) {
	t.Parallel()

	names := make([]string, 0, 3)
	for _, definition := range RegisterWorkflows() {
		names = append(names, definition.Name)
	}
	assert.ElementsMatch(t, []string{
		retrievalservice.IndexOrganizationWorkflowName,
		retrievalservice.ReindexSourceWorkflowName,
		retrievalservice.SweepWorkflowName,
	}, names)
}
