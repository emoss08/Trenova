package fuelpurchaseservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/fuelpurchaseservice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

// syncQueued runs a feed whose row names a tractor nobody has registered, so the
// row is held. This is the state a discovered card leaves behind, and the state
// working the rows out again has to be able to get out of.
func syncQueued(t *testing.T, h *harness, rows ...[]string) *fuelpurchaseservice.SyncFeedResult {
	t.Helper()

	installFeed(h, &services.FetchFuelTransactionsResult{
		Staged: stagedFrom(feedHeaders, rows...),
		Cards: []services.ProviderCard{
			{LastFour: "4411", Provider: fuelpurchase.CardProviderComdata},
		},
	}, true)

	return syncFeed(t, h)
}

func resolveRows(
	t *testing.T,
	h *harness,
	batchID pulid.ID,
) *fuelpurchaseservice.ResolveRowsResult {
	t.Helper()

	batch := h.repo.batch(batchID)
	result, err := h.svc.ResolveRows(t.Context(), &fuelpurchaseservice.ResolveRowsRequest{
		TenantInfo: h.tenant,
		BatchID:    batch.ID,
		Version:    batch.Version,
		UserID:     h.userID,
	})
	require.NoError(t, err)

	return result
}

// The row a feed could not place is worked out again once the tractor it named
// exists, and posts without the file being fetched a second time.
func TestResolveRowsPostsARowOnceItsTractorIsKnown(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	sync := syncQueued(t, h, feedRow("C-1001", "TRC-UNKNOWN", "4411"))
	require.Equal(t, 1, sync.Queued)
	require.Zero(t, sync.Committed)

	// The unit the feed named is now a tractor on file.
	registered := h.registerTractor("TRC-UNKNOWN")

	result := resolveRows(t, h, sync.BatchID)

	require.Equal(t, 1, result.Reviewed)
	require.Equal(t, 1, result.Resolved)
	require.Equal(t, 1, result.Committed)
	require.Zero(t, result.Queued)

	purchases := h.repo.allPurchases()
	require.Len(t, purchases, 1)
	require.Equal(t, "C-1001", purchases[0].TransactionReference)
	require.Equal(t, registered.ID, purchases[0].TractorID)
}

// A row whose cause has not been fixed stays exactly where it was, so working the
// rows out again is safe to run at any time.
func TestResolveRowsLeavesAStillUnresolvableRowQueued(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	sync := syncQueued(t, h, feedRow("C-1001", "TRC-UNKNOWN", "4411"))

	result := resolveRows(t, h, sync.BatchID)

	require.Equal(t, 1, result.Reviewed)
	require.Zero(t, result.Resolved)
	require.Equal(t, 1, result.Queued)
	require.Empty(t, h.repo.allPurchases())
}

// Rows that already became purchases must survive a second pass untouched; the
// batch is rewritten wholesale, so this is the case that would lose them.
func TestResolveRowsKeepsRowsThatAlreadyPosted(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	sync := syncQueued(t, h,
		feedRow("C-1001", "TRC-001", "4411"),
		feedRow("C-1002", "TRC-UNKNOWN", "4411"),
	)
	require.Equal(t, 1, sync.Committed)
	require.Equal(t, 1, sync.Queued)

	result := resolveRows(t, h, sync.BatchID)

	// Only the held row was looked at again.
	require.Equal(t, 1, result.Reviewed)
	require.Equal(t, 1, result.Queued)

	// The purchase from the first pass is still the only one, and still on file.
	require.Len(t, h.repo.allPurchases(), 1)

	batch := h.repo.batch(sync.BatchID)
	require.Equal(t, 2, batch.RowCount)
	require.Equal(t, 1, batch.CommittedCount)
}

// Working the rows out again must not post a purchase twice, even though the row
// is parsed from the same cells and carries the same reference.
func TestResolveRowsPostsNothingTwice(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	sync := syncQueued(t, h, feedRow("C-1001", "TRC-UNKNOWN", "4411"))

	h.registerTractor("TRC-UNKNOWN")
	require.Equal(t, 1, resolveRows(t, h, sync.BatchID).Committed)

	batch := h.repo.batch(sync.BatchID)
	_, err := h.svc.ResolveRows(t.Context(), &fuelpurchaseservice.ResolveRowsRequest{
		TenantInfo: h.tenant,
		BatchID:    batch.ID,
		Version:    batch.Version,
		UserID:     h.userID,
	})
	require.ErrorContains(t, err, "waiting to be worked out")
	require.Len(t, h.repo.allPurchases(), 1)
}

func TestResolveRowsRefusesAStaleVersion(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	sync := syncQueued(t, h, feedRow("C-1001", "TRC-UNKNOWN", "4411"))

	_, err := h.svc.ResolveRows(t.Context(), &fuelpurchaseservice.ResolveRowsRequest{
		TenantInfo: h.tenant,
		BatchID:    sync.BatchID,
		Version:    h.repo.batch(sync.BatchID).Version + 5,
		UserID:     h.userID,
	})
	require.Error(t, err)
}

func TestResolveRowsRefusesAnotherTenant(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	sync := syncQueued(t, h, feedRow("C-1001", "TRC-UNKNOWN", "4411"))

	_, err := h.svc.ResolveRows(t.Context(), &fuelpurchaseservice.ResolveRowsRequest{
		TenantInfo: h.other,
		BatchID:    sync.BatchID,
		Version:    h.repo.batch(sync.BatchID).Version,
		UserID:     h.userID,
	})
	require.Error(t, err)
}

// The rows are read from their stored cells, so the statement is never
// reinterpreted — only what it is matched against changes.
func TestResolveRowsRereadsTheOriginalCells(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	sync := syncQueued(t, h, feedRow("C-1001", "TRC-UNKNOWN", "4411"))

	h.registerTractor("TRC-UNKNOWN")
	resolveRows(t, h, sync.BatchID)

	purchases := h.repo.allPurchases()
	require.Len(t, purchases, 1)
	require.Equal(t, "125.400", purchases[0].Gallons.StringFixed(3))
	require.Equal(t, int64(50160), purchases[0].TotalAmountMinor)
}
