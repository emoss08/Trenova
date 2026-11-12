//go:build integration

package billingqueuerepository

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// reviewItem picks one of the tenant's items under review with no due marker.
func reviewItem(t *testing.T, f *syncFixture) pulid.ID {
	t.Helper()

	var id pulid.ID
	require.NoError(t, keel.Pool(f.db).QueryRow(f.ctx,
		`UPDATE billing_queue_items SET recheck_due_at = NULL
		WHERE id = (
			SELECT id FROM billing_queue_items
			WHERE organization_id = $1 AND business_unit_id = $2
				AND status IN ('ReadyForReview', 'InReview')
			ORDER BY id LIMIT 1
		)
		RETURNING id`,
		f.tenant.OrgID, f.tenant.BuID,
	).Scan(&id), "the seed has an item under review")

	return id
}

func dueAt(t *testing.T, f *syncFixture, id pulid.ID) *int64 {
	t.Helper()

	var due *int64
	require.NoError(t, keel.Pool(f.db).QueryRow(f.ctx,
		`SELECT recheck_due_at FROM billing_queue_items WHERE id = $1`, id,
	).Scan(&due))

	return due
}

// A request that lands while an item's checks run must survive the settle,
// or the change it carries is never checked: every request moves the marker
// forward, and the settle clears only the value it claimed.
func TestSettleRecheckKeepsAnItemRequestedWhileItsChecksRan(t *testing.T) {
	t.Parallel()

	f := setupSyncFixture(t)
	id := reviewItem(t, f)
	now := timeutils.NowUnix()

	require.NoError(t, f.repo.MarkRecheckDue(f.ctx, &billingqueueports.MarkRecheckDueRequest{
		TenantInfo: f.tenant, ItemIDs: []pulid.ID{id}, At: now,
	}))
	due, err := f.repo.ListRecheckDue(f.ctx, &billingqueueports.ListRecheckDueRequest{
		TenantInfo: f.tenant, Now: now, Limit: 100_000,
	})
	require.NoError(t, err)
	var claimed int64
	for _, item := range due {
		if item.ID == id {
			claimed = item.DueAt
		}
	}
	require.NotZero(t, claimed, "the marked item is due")

	require.NoError(t, f.repo.MarkRecheckDue(f.ctx, &billingqueueports.MarkRecheckDueRequest{
		TenantInfo: f.tenant, ItemIDs: []pulid.ID{id}, At: now,
	}), "requested again in the same second")
	require.NoError(t, f.repo.SettleRecheck(f.ctx, &billingqueueports.SettleRecheckRequest{
		TenantInfo: f.tenant, ItemID: id, ClaimedDueAt: claimed,
	}))

	left := dueAt(t, f, id)
	require.NotNil(t, left, "the second request keeps the item due")
	assert.Greater(t, *left, claimed)

	require.NoError(t, f.repo.SettleRecheck(f.ctx, &billingqueueports.SettleRecheckRequest{
		TenantInfo: f.tenant, ItemID: id, ClaimedDueAt: *left,
	}))
	assert.Nil(t, dueAt(t, f, id), "settling what it claimed clears it")
}

// A failed item waits out its retry rather than being claimed again at once.
func TestSettleRecheckPushesAFailedItemBack(t *testing.T) {
	t.Parallel()

	f := setupSyncFixture(t)
	id := reviewItem(t, f)
	now := timeutils.NowUnix()

	require.NoError(t, f.repo.MarkRecheckDue(f.ctx, &billingqueueports.MarkRecheckDueRequest{
		TenantInfo: f.tenant, ItemIDs: []pulid.ID{id}, At: now,
	}))
	claimed := dueAt(t, f, id)
	require.NotNil(t, claimed)
	retryAt := now + 300
	require.NoError(t, f.repo.SettleRecheck(f.ctx, &billingqueueports.SettleRecheckRequest{
		TenantInfo: f.tenant, ItemID: id, ClaimedDueAt: *claimed, RetryAt: &retryAt,
	}))

	due, err := f.repo.ListRecheckDue(f.ctx, &billingqueueports.ListRecheckDueRequest{
		TenantInfo: f.tenant, Now: now, Limit: 100_000,
	})
	require.NoError(t, err)
	for _, item := range due {
		assert.NotEqual(t, id, item.ID, "an item waiting out its retry is not due")
	}
}

// The stale sweep marks only items under review, so it never marks an item
// a recheck cannot settle, and the tenants it then lists are the ones with
// due items.
func TestMarkStaleRecheckDueMarksOnlyItemsUnderReview(t *testing.T) {
	t.Parallel()

	f := setupSyncFixture(t)
	var postedID pulid.ID
	require.NoError(t, keel.Pool(f.db).QueryRow(f.ctx,
		`UPDATE billing_queue_items SET lane = 'Checking', checked_at = NULL, recheck_due_at = NULL
		WHERE id = (
			SELECT id FROM billing_queue_items
			WHERE organization_id = $1 AND business_unit_id = $2 AND status = 'Posted'
			ORDER BY id LIMIT 1
		)
		RETURNING id`,
		f.tenant.OrgID, f.tenant.BuID,
	).Scan(&postedID), "the seed has a posted item to drift")
	reviewID := reviewItem(t, f)
	_, err := keel.Pool(f.db).Exec(f.ctx,
		`UPDATE billing_queue_items SET checked_at = NULL WHERE id = $1`, reviewID)
	require.NoError(t, err)

	now := timeutils.NowUnix()
	marked, err := f.repo.MarkStaleRecheckDue(f.ctx, &billingqueueports.MarkStaleRecheckDueRequest{
		StaleBefore: now, Now: now, Limit: 100_000,
	})
	require.NoError(t, err)
	assert.Positive(t, marked)
	assert.Nil(t, dueAt(t, f, postedID), "a posted item left in a review lane is not marked")
	assert.NotNil(t, dueAt(t, f, reviewID), "an unchecked item under review is marked")

	tenants, err := f.repo.ListRecheckDueTenants(f.ctx, &billingqueueports.ListRecheckDueTenantsRequest{
		Now: now, Limit: 1000,
	})
	require.NoError(t, err)
	assert.Contains(t, tenants, f.tenant)
}
