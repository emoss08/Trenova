package accountingdriftservice

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/watchtowersources"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeWatchtower struct {
	services.WatchtowerProjector
	mu   sync.Mutex
	open map[string]services.WatchtowerItemInput
}

func (f *fakeWatchtower) Upsert(_ context.Context, input services.WatchtowerItemInput) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.open[input.SourceID] = input
}

func (f *fakeWatchtower) Resolve(
	_ context.Context,
	_ pagination.TenantInfo,
	_ watchtower.SourceKind,
	sourceID string,
) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.open, sourceID)
}

func (f *fakeFindings) ListAttention(
	_ context.Context,
	req *repositories.ListAccountingDriftAttentionRequest,
) ([]repositories.AccountingDriftAttentionGroup, error) {
	groups := map[accountingsync.DriftKind]*repositories.AccountingDriftAttentionGroup{}
	out := []repositories.AccountingDriftAttentionGroup{}
	for _, row := range f.all() {
		if !row.IsOpen() || row.ConnectionID != req.ConnectionID ||
			row.DetectedAt > req.DetectedBefore {
			continue
		}
		group, ok := groups[row.Kind]
		if !ok {
			out = append(out, repositories.AccountingDriftAttentionGroup{
				Kind:             row.Kind,
				OldestDetectedAt: row.DetectedAt,
				SampleID:         row.ID,
			})
			group = &out[len(out)-1]
			groups[row.Kind] = group
		}
		group.Count++
	}
	return out, nil
}

func TestAFindingOlderThanADayReachesWatchtowerUntilItResolves(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	board := &fakeWatchtower{open: map[string]services.WatchtowerItemInput{}}
	h.svc.watchtower = board
	h.synced(accountingsync.SyncObjectInvoice, "101", 125_000)
	h.provider(accountingsync.SyncObjectInvoice, "101", "1200.00")
	h.reconcile(t)
	finish := func() {
		require.NoError(t, h.svc.FinishCheck(t.Context(), &services.FinishAccountingDriftCheckRequest{
			TenantInfo:   h.tenant,
			ConnectionID: h.conn.ID,
		}))
	}

	finish()
	assert.Empty(t, board.open, "a finding is given a day before it is raised")

	h.now = h.now.Add(25 * time.Hour)
	finish()
	sourceID := watchtowersources.AccountingDriftAttentionSourceID(h.conn, accountingsync.DriftAmountMismatch)
	require.Contains(t, board.open, sourceID)
	item := board.open[sourceID]
	assert.Equal(t, agent.SubjectAccountingDrift, item.SubjectType)
	assert.Equal(t, agent.EventAccountingDriftDetected, item.EventKind)
	assert.Equal(t, watchtowersources.AccountingDriftPath, item.Path)
	assert.Contains(t, item.Title, "1 document")

	h.provider(accountingsync.SyncObjectInvoice, "101", "1250.00")
	h.reconcile(t)
	finish()
	assert.Empty(t, board.open)
}
