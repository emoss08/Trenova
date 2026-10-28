package accountingdriftservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingdrift"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingdriftports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (f *fakeConnections) GetByType(
	_ context.Context,
	_ accountingconnectionports.GetAccountingConnectionRequest,
) (*accountingconnection.AccountingConnection, error) {
	return f.current(), nil
}

func (f *fakeFindings) Summarize(
	_ context.Context,
	req *accountingdriftports.SummarizeAccountingDriftRequest,
) (*accountingdriftports.AccountingDriftSummary, error) {
	summary := &accountingdriftports.AccountingDriftSummary{}
	for _, row := range f.all() {
		if row.ConnectionID != req.ConnectionID {
			continue
		}
		if row.IsOpen() {
			summary.Open++
			continue
		}
		if row.ResolvedAt != nil && *row.ResolvedAt >= req.ResolvedSince {
			summary.ResolvedSince++
		}
	}
	return summary, nil
}

func (f *fakeFindings) ListConnection(
	_ context.Context,
	req *accountingdriftports.ListAccountingDriftFindingsConnectionRequest,
) (*pagination.CursorListResult[*accountingdrift.AccountingDriftFinding], error) {
	f.mu.Lock()
	f.lastList = req
	f.mu.Unlock()
	return &pagination.CursorListResult[*accountingdrift.AccountingDriftFinding]{
		Items: f.all(),
	}, nil
}

type fakeChecker struct {
	checked []pulid.ID
}

func (f *fakeChecker) CheckNow(_ context.Context, _ pagination.TenantInfo, id pulid.ID) error {
	f.checked = append(f.checked, id)
	return nil
}

func TestOverviewCountsFindingsAndStatesTheTolerance(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	checked := h.now.Unix() - 3600
	h.conn.DriftCheckedAt = &checked
	h.synced(accountingsync.SyncObjectInvoice, "101", 125_000)
	h.provider(accountingsync.SyncObjectInvoice, "101", "1200.00")
	h.reconcile(t)

	overview, err := h.svc.Overview(
		t.Context(),
		&accountingdriftports.AccountingDriftOverviewRequest{
			TenantInfo:      h.tenant,
			IntegrationType: integration.TypeQuickBooksOnline,
		},
	)
	require.NoError(t, err)

	assert.Equal(t, h.conn.ID, overview.ConnectionID)
	assert.Equal(t, "QuickBooks Online", overview.ProviderName)
	assert.Equal(t, &checked, overview.CheckedAt)
	assert.Equal(t, int64(500), overview.ToleranceMinor)
	assert.Equal(t, "USD", overview.CurrencyCode)
	assert.Equal(t, 1, overview.Summary.Open)
}

func TestListScopesTheTableToTheConnection(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	objectID := pulid.MustNew("inv_")

	_, err := h.svc.List(t.Context(), &accountingdriftports.ListAccountingDriftFindingsRequest{
		TenantInfo:      h.tenant,
		IntegrationType: integration.TypeQuickBooksOnline,
		Statuses:        []accountingdrift.DriftStatus{accountingdrift.DriftStatusOpen},
		Kinds:           []accountingdrift.DriftKind{accountingdrift.DriftAmountMismatch},
		ObjectID:        objectID,
		Search:          "INV-1",
	})
	require.NoError(t, err)

	req := h.findings.lastList
	require.NotNil(t, req)
	assert.Equal(t, h.conn.ID, req.ConnectionID)
	assert.Equal(t, h.tenant, req.Filter.TenantInfo)
	assert.Equal(t, []accountingdrift.DriftStatus{accountingdrift.DriftStatusOpen}, req.Statuses)
	assert.Equal(t, []accountingdrift.DriftKind{accountingdrift.DriftAmountMismatch}, req.Kinds)
	assert.Equal(t, objectID, req.ObjectID)
	assert.Equal(t, "INV-1", req.Search)
}

func TestCheckNowStartsTheRunForASyncingConnection(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	checker := &fakeChecker{}
	h.svc.checker = checker

	overview, err := h.svc.CheckNow(t.Context(), h.tenant, integration.TypeQuickBooksOnline)
	require.NoError(t, err)

	assert.Equal(t, []pulid.ID{h.conn.ID}, checker.checked)
	assert.Equal(t, h.conn.ID, overview.ConnectionID)

	paused := h.now.Unix()
	h.conn.PausedAt = &paused
	_, err = h.svc.CheckNow(t.Context(), h.tenant, integration.TypeQuickBooksOnline)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Len(t, checker.checked, 1)
}

func TestCheckNowSaysSoWithoutBackgroundWork(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	_, err := h.svc.CheckNow(t.Context(), h.tenant, integration.TypeQuickBooksOnline)

	assert.True(t, errortypes.IsBusinessError(err))
}
