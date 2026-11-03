//go:build integration

package billingqueuerepository

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A recheck only marks an item checked while it is under review, so the sweep
// must never pick anything else: an item it cannot mark would be picked again
// on every run, ahead of the items that are waiting to be checked.
func TestListRecheckCandidatesOnlyPicksItemsARecheckCanMark(t *testing.T) {
	t.Parallel()

	f := setupSyncFixture(t)
	var postedID pulid.ID
	require.NoError(t, keel.Pool(f.db).QueryRow(f.ctx,
		`UPDATE billing_queue_items SET lane = 'Checking', checked_at = NULL
		WHERE id = (
			SELECT id FROM billing_queue_items
			WHERE organization_id = $1 AND business_unit_id = $2 AND status = 'Posted'
			ORDER BY id LIMIT 1
		)
		RETURNING id`,
		f.tenant.OrgID, f.tenant.BuID,
	).Scan(&postedID), "the seed has a posted item to drift")

	candidates, err := f.repo.ListRecheckCandidates(f.ctx, &billingqueueports.ListRecheckCandidatesRequest{
		StaleBefore: timeutils.NowUnix(),
		Limit:       100_000,
	})
	require.NoError(t, err)

	ids := make([]pulid.ID, 0, len(candidates))
	for _, candidate := range candidates {
		ids = append(ids, candidate.ID)
	}
	assert.NotContains(t, ids, postedID, "a posted item left in a review lane is not rechecked")

	if len(ids) == 0 {
		return
	}
	statuses := make([]string, 0, len(billingqueue.ReviewStatuses()))
	for _, status := range billingqueue.ReviewStatuses() {
		statuses = append(statuses, string(status))
	}
	var outside int
	require.NoError(t, keel.Pool(f.db).QueryRow(f.ctx,
		`SELECT count(*) FROM billing_queue_items WHERE id = ANY($1) AND NOT (status = ANY($2))`,
		ids, statuses,
	).Scan(&outside))
	assert.Zero(t, outside, "every candidate is under review")
}
