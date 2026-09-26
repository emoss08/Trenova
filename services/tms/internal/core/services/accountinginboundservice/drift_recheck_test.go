package accountinginboundservice

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDrift struct {
	mu     sync.Mutex
	calls  []services.RecheckAccountingDriftRequest
	failed error
}

func (f *fakeDrift) RecheckDocuments(
	_ context.Context,
	req *services.RecheckAccountingDriftRequest,
) (*services.AccountingDriftBatchResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, *req)
	if f.failed != nil {
		return nil, f.failed
	}
	return &services.AccountingDriftBatchResult{Compared: len(req.Documents)}, nil
}

func withDrift(t *testing.T, h *harness) *fakeDrift {
	t.Helper()
	drift := &fakeDrift{}
	h.svc.drift = drift
	return drift
}

func edited(externalID string) services.AccountingChangedDocument {
	return services.AccountingChangedDocument{
		ObjectTypes: []accountingsync.SyncObjectType{
			accountingsync.SyncObjectInvoice,
			accountingsync.SyncObjectDebitMemo,
		},
		ExternalID: externalID,
		Operation:  services.AccountingChangeUpsert,
	}
}

func TestPollAsksForDocumentsAndComparesTheOnesThatChanged(t *testing.T) {
	t.Parallel()
	h := newHarness(t, accountingsync.InboundPaymentsPropose)
	drift := withDrift(t, h)
	h.connector.pages = append(h.connector.pages, &services.AccountingChangePage{
		Documents:  []services.AccountingChangedDocument{edited("145"), edited("146")},
		NextCursor: "next",
	})

	result, err := h.svc.PollChanges(t.Context(), &services.PollAccountingChangesRequest{
		TenantInfo:   h.tenant,
		ConnectionID: h.conn.ID,
	})
	require.NoError(t, err)

	require.Len(t, h.connector.reads, 1)
	assert.True(t, h.connector.reads[0].Documents)
	require.Len(t, drift.calls, 1)
	assert.Equal(t, h.conn.ID, drift.calls[0].ConnectionID)
	assert.Len(t, drift.calls[0].Documents, 2)
	assert.Equal(t, driftEventsPerPoll, drift.calls[0].EventBudget)
	assert.Equal(t, 2, result.Documents)
}

func TestPollDoesNotAskForDocumentsWithoutADriftCheck(t *testing.T) {
	t.Parallel()
	h := newHarness(t, accountingsync.InboundPaymentsPropose)

	h.poll(t)

	require.Len(t, h.connector.reads, 1)
	assert.False(t, h.connector.reads[0].Documents)
}

func TestPollKeepsItsCursorWhenTheDriftCompareFails(t *testing.T) {
	t.Parallel()
	h := newHarness(t, accountingsync.InboundPaymentsPropose)
	drift := withDrift(t, h)
	drift.failed = errors.New("QuickBooks did not answer")
	h.connector.pages = append(h.connector.pages, &services.AccountingChangePage{
		Documents:  []services.AccountingChangedDocument{edited("145")},
		NextCursor: "next",
	})

	result, err := h.svc.PollChanges(t.Context(), &services.PollAccountingChangesRequest{
		TenantInfo:   h.tenant,
		ConnectionID: h.conn.ID,
	})
	require.NoError(t, err)

	assert.Equal(t, 0, result.Documents)
	assert.Equal(t, "next", h.conns.current().ChangeCursor)
}
