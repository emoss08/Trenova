package agentactivitysummaryservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const now = int64(1_800_000_000)

type fakeRepo struct {
	totals *repositories.AgentActivityTotals
	asked  []repositories.AgentActivityTotalsRequest
}

func (f *fakeRepo) Totals(
	_ context.Context,
	req repositories.AgentActivityTotalsRequest,
) (*repositories.AgentActivityTotals, error) {
	f.asked = append(f.asked, req)

	return f.totals, nil
}

func serviceWith(totals *repositories.AgentActivityTotals) (*Service, *fakeRepo) {
	repo := &fakeRepo{totals: totals}

	return &Service{repo: repo, now: func() int64 { return now }}, repo
}

func TestSummaryCountsTheDayAndTheWeeksDecisions(t *testing.T) {
	t.Parallel()

	oldest := now - 3600
	svc, repo := serviceWith(&repositories.AgentActivityTotals{
		Runs:              14,
		RunsFailed:        2,
		RunsWorking:       1,
		RunsAwaiting:      3,
		PendingProposals:  4,
		OldestPendingAt:   &oldest,
		OpenExceptions:    2,
		DecisionsAccepted: 6,
		DecisionsModified: 1,
		DecisionsRejected: 1,
	})
	tenant := pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"}

	summary, err := svc.Summary(t.Context(), &services.AgentActivitySummaryRequest{
		TenantInfo: tenant,
		Since:      now - 9*3600,
	})
	require.NoError(t, err)

	assert.Equal(t, 14, summary.Runs)
	assert.Equal(t, 2, summary.RunsFailed)
	assert.Equal(t, 4, summary.PendingProposals)
	assert.Equal(t, &oldest, summary.OldestPendingAt)
	assert.Equal(t, 8, summary.Decided)
	require.NotNil(t, summary.ApprovedAsProposed)
	assert.InDelta(t, 0.75, *summary.ApprovedAsProposed, 1e-9)
	require.Len(t, repo.asked, 1)
	assert.Equal(t, tenant, repo.asked[0].TenantInfo)
	assert.Equal(t, now-9*3600, repo.asked[0].RunsSince)
	assert.Equal(t, now-7*24*3600, repo.asked[0].DecisionsSince)
}

func TestSummaryHasNoApprovalShareWithoutDecisions(t *testing.T) {
	t.Parallel()

	svc, _ := serviceWith(&repositories.AgentActivityTotals{})

	summary, err := svc.Summary(t.Context(), &services.AgentActivitySummaryRequest{Since: now})
	require.NoError(t, err)
	assert.Zero(t, summary.Decided)
	assert.Nil(t, summary.ApprovedAsProposed)
}

func TestSummaryRefusesAWindowInTheFutureOrTooFarBack(t *testing.T) {
	t.Parallel()

	svc, repo := serviceWith(&repositories.AgentActivityTotals{})

	for _, since := range []int64{now + 1, now - 32*24*3600} {
		_, err := svc.Summary(t.Context(), &services.AgentActivitySummaryRequest{Since: since})
		var validation *errortypes.Error
		require.ErrorAs(t, err, &validation)
		assert.Equal(t, "since", validation.Field)
	}
	assert.Empty(t, repo.asked)
}
