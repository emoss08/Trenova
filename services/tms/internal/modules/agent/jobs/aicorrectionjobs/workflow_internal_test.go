package aicorrectionjobs

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

func workItem(id string) temporaljobs.TenantWorkItem {
	return temporaljobs.TenantWorkItem{
		OrganizationID: pulid.ID(id),
		BusinessUnitID: pulid.ID("bu_" + id),
	}
}

func TestAICorrectionRetentionWorkflow_PurgesEachOrganization(t *testing.T) {
	t.Parallel()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	var a *Activities
	env.OnActivity(a.ListAICorrectionOrganizationsActivity, mock.Anything, mock.Anything).
		Return(&temporaljobs.TenantPage{Tenants: []temporaljobs.TenantWorkItem{
			workItem("org_a"), workItem("org_b"),
		}}, nil).
		Once()

	var mu sync.Mutex
	instants := make(map[int64]struct{})
	env.OnActivity(a.PurgeOrganizationAICorrectionsActivity, mock.Anything, mock.Anything).
		Return(func(
			_ context.Context,
			input *OrganizationAICorrectionInput,
		) (*OrganizationAICorrectionResult, error) {
			mu.Lock()
			instants[input.Now] = struct{}{}
			mu.Unlock()
			if input.OrganizationID == "org_b" {
				return nil, temporal.NewNonRetryableApplicationError(
					"broken",
					"Broken",
					errors.New("broken"),
				)
			}

			return &OrganizationAICorrectionResult{Purged: 7}, nil
		})

	env.ExecuteWorkflow(AICorrectionRetentionWorkflow, (*AICorrectionRetentionInput)(nil))

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result AICorrectionRetentionResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, 1, result.OrganizationsProcessed)
	assert.Equal(t, int64(7), result.Purged)
	assert.Equal(t, []string{"org_b"}, result.FailedOrganizations)
	mu.Lock()
	defer mu.Unlock()
	assert.Len(t, instants, 1, "every organization is purged as of one instant")
}

func TestSchedules_MatchTheirWorkflowSignatures(t *testing.T) {
	t.Parallel()

	for _, s := range scheduleProviderWith(true).GetSchedules() {
		require.NoError(t, s.Validate(), s.ID)
	}
}

func TestSchedules_ExtractionDriftOnlyWithDocumentExtraction(t *testing.T) {
	t.Parallel()

	ids := func(p *ScheduleProvider) []string {
		var out []string
		for _, s := range p.GetSchedules() {
			out = append(out, s.ID)
		}
		return out
	}

	assert.Contains(t, ids(scheduleProviderWith(true)), "extraction-accuracy-drift")
	assert.NotContains(t, ids(scheduleProviderWith(false)), "extraction-accuracy-drift")
	assert.Contains(t, ids(scheduleProviderWith(false)), "ai-correction-retention")
}

func scheduleProviderWith(extraction bool) *ScheduleProvider {
	cfg := &config.Config{}
	cfg.AI.DocumentExtraction = extraction
	return NewScheduleProvider(&cfg.AI)
}
