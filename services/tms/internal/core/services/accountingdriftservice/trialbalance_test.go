package accountingdriftservice

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/accounttype"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func (f *fakeReader) ReadTrialBalance(
	_ context.Context,
	req *services.ReadTrialBalanceRequest,
) ([]services.AccountingTrialBalanceRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.trialReq = append(f.trialReq, *req)
	if f.failWith != nil {
		return nil, f.failWith
	}
	return slices.Clone(f.trial), nil
}

func (f *fakeReader) trialRequests() []services.ReadTrialBalanceRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.trialReq)
}

func (f *fakeRecords) CountByStatus(
	context.Context,
	repositories.AccountingSyncConnectionRequest,
) ([]repositories.AccountingSyncStatusCount, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	counts := map[accountingsync.SyncStatus]int{}
	for _, row := range f.rows {
		counts[row.Status]++
	}
	out := make([]repositories.AccountingSyncStatusCount, 0, len(counts))
	for status, count := range counts {
		out = append(out, repositories.AccountingSyncStatusCount{Status: status, Count: count})
	}
	return out, nil
}

type trialLedger struct {
	mu       sync.Mutex
	opening  []repositories.LedgerAccountBalance
	during   []repositories.LedgerAccountBalance
	income   []repositories.LedgerAccountBalance
	start    int64
	requests []repositories.SumLedgerRequest
}

func (f *trialLedger) GetJournal(
	context.Context,
	*repositories.GetLedgerJournalRequest,
) (*repositories.LedgerJournal, error) {
	return nil, nil
}

func (f *trialLedger) ListJournals(
	context.Context,
	*repositories.ListLedgerJournalsRequest,
) ([]*repositories.LedgerJournal, error) {
	return nil, nil
}

func (f *trialLedger) ListActiveAccounts(
	context.Context,
	*repositories.ListLedgerAccountsRequest,
) ([]repositories.LedgerAccount, error) {
	return nil, nil
}

func (f *trialLedger) SumLines(
	_ context.Context,
	req *repositories.SumLedgerRequest,
) ([]repositories.LedgerAccountBalance, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, *req)
	switch {
	case req.IncludeClosing:
		return slices.Clone(f.opening), nil
	case req.From != nil && *req.From == f.start:
		return slices.Clone(f.during), nil
	default:
		return slices.Clone(f.income), nil
	}
}

func (f *trialLedger) sums() []repositories.SumLedgerRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.requests)
}

type trialMappings struct {
	repositories.AccountingMappingRepository
	rows []*accountingsync.AccountingMapping
}

func (f *trialMappings) ListByConnection(
	_ context.Context,
	req *repositories.ListAccountingMappingsRequest,
) ([]*accountingsync.AccountingMapping, error) {
	out := []*accountingsync.AccountingMapping{}
	for _, row := range f.rows {
		if slices.Contains(req.TargetTypes, row.TargetType) {
			out = append(out, row)
		}
	}
	return out, nil
}

type trialReferences struct {
	repositories.AccountingReferenceObjectRepository
	rows []*accountingsync.AccountingReferenceObject
}

func (f *trialReferences) GetByExternalIDs(
	_ context.Context,
	req *repositories.GetAccountingReferenceObjectsRequest,
) ([]*accountingsync.AccountingReferenceObject, error) {
	out := []*accountingsync.AccountingReferenceObject{}
	for _, row := range f.rows {
		if row.Kind == req.Kind && slices.Contains(req.ExternalIDs, row.ExternalID) {
			out = append(out, row)
		}
	}
	return out, nil
}

type trialOrganizations struct {
	repositories.OrganizationRepository
	timezone string
}

func (f trialOrganizations) GetByID(
	_ context.Context,
	req repositories.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	return &tenant.Organization{ID: req.TenantInfo.OrgID, Timezone: f.timezone}, nil
}

type trialFixture struct {
	h        *harness
	ledger   *trialLedger
	mappings *trialMappings
	ar       pulid.ID
	cash     pulid.ID
	prepaid  pulid.ID
	revenue  pulid.ID
	fuel     pulid.ID
	retained pulid.ID
}

func balance(id pulid.ID, code, name string, category accounttype.Category, net int64) repositories.LedgerAccountBalance {
	out := repositories.LedgerAccountBalance{
		AccountID:   id,
		AccountCode: code,
		AccountName: name,
		Category:    category.String(),
	}
	if net >= 0 {
		out.DebitMinor = net
	} else {
		out.CreditMinor = -net
	}
	return out
}

func glMapping(
	connectionID, accountID pulid.ID,
	label, externalID, externalName string,
) *accountingsync.AccountingMapping {
	return &accountingsync.AccountingMapping{
		ID:              pulid.MustNew("acctm_"),
		ConnectionID:    connectionID,
		TargetType:      accountingsync.TargetGLAccount,
		TrenovaObjectID: accountID,
		TargetLabel:     label,
		ExternalID:      externalID,
		ExternalName:    externalName,
		State:           accountingsync.MappingStateConfirmed,
	}
}

func providerRow(externalID, name, debit, credit string) services.AccountingTrialBalanceRow {
	row := services.AccountingTrialBalanceRow{AccountExternalID: externalID, AccountName: name}
	if debit != "" {
		row.Debit = decimal.RequireFromString(debit)
	}
	if credit != "" {
		row.Credit = decimal.RequireFromString(credit)
	}
	return row
}

func newTrialFixture(t *testing.T) *trialFixture {
	t.Helper()
	h := newHarness(t)
	h.conns.conn.SyncMode = accountingsync.SyncModeLedger
	h.conns.conn.LedgerGranularity = accountingsync.LedgerDetailed
	h.conns.conn.ExternalFiscalYearStartMonth = 1

	f := &trialFixture{
		h:        h,
		ar:       pulid.MustNew("gla_"),
		cash:     pulid.MustNew("gla_"),
		prepaid:  pulid.MustNew("gla_"),
		revenue:  pulid.MustNew("gla_"),
		fuel:     pulid.MustNew("gla_"),
		retained: pulid.MustNew("gla_"),
	}
	f.ledger = &trialLedger{start: *h.conn.SyncStartDate}
	h.controls.cash = f.cash
	f.ledger.during = []repositories.LedgerAccountBalance{
		balance(f.ar, "1100", "Accounts receivable", accounttype.CategoryAsset, 150_000),
		balance(f.cash, "1000", "Operating cash", accounttype.CategoryAsset, 50_000),
		balance(f.revenue, "4000", "Freight revenue", accounttype.CategoryRevenue, -999_999),
		balance(f.retained, "3000", "Retained earnings", accounttype.CategoryEquity, -7_000),
	}
	f.ledger.income = []repositories.LedgerAccountBalance{
		balance(f.revenue, "4000", "Freight revenue", accounttype.CategoryRevenue, -120_000),
		balance(f.fuel, "4010", "Fuel surcharge revenue", accounttype.CategoryRevenue, -80_000),
		balance(f.ar, "1100", "Accounts receivable", accounttype.CategoryAsset, 999_999),
	}
	f.mappings = &trialMappings{rows: []*accountingsync.AccountingMapping{
		glMapping(h.conn.ID, f.ar, "GL account 1100 Accounts receivable", "qb-ar", "Accounts Receivable (A/R)"),
		{
			ID:           pulid.MustNew("acctm_"),
			ConnectionID: h.conn.ID,
			TargetType:   accountingsync.TargetAccountRole,
			TrenovaKey:   accountingsync.AccountRoleDeposit,
			TargetLabel:  "Deposit account",
			ExternalID:   "qb-bank",
			ExternalName: "Checking (renamed since)",
			State:        accountingsync.MappingStateConfirmed,
		},
		{
			ID:              pulid.MustNew("acctm_"),
			ConnectionID:    h.conn.ID,
			TargetType:      accountingsync.TargetGLAccount,
			TrenovaObjectID: pulid.MustNew("gla_"),
			TargetLabel:     "GL account 5000 Fuel expense",
			ExternalID:      "qb-fuel",
			ExternalName:    "Fuel",
			State:           accountingsync.MappingStateProposed,
		},
		glMapping(h.conn.ID, f.prepaid, "GL account 1200 Prepaid expenses", "qb-prepaid", "Prepaid"),
		glMapping(h.conn.ID, f.revenue, "GL account 4000 Freight revenue", "qb-income", "Freight income"),
		glMapping(h.conn.ID, f.fuel, "GL account 4010 Fuel surcharge revenue", "qb-income", "Freight income"),
		glMapping(h.conn.ID, f.retained, "GL account 3000 Retained earnings", "qb-re", "Retained Earnings"),
		{
			ID:           pulid.MustNew("acctm_"),
			ConnectionID: h.conn.ID,
			TargetType:   accountingsync.TargetCustomer,
			ExternalID:   "qb-cust",
			State:        accountingsync.MappingStateConfirmed,
		},
	}}
	h.reader.trial = []services.AccountingTrialBalanceRow{
		providerRow("qb-ar", "Accounts Receivable (A/R)", "1500.00", ""),
		providerRow("qb-bank", "Checking", "400.00", ""),
		providerRow("qb-income", "Freight income", "", "2000.00"),
		providerRow("qb-prepaid", "Prepaid", "25.00", ""),
		providerRow("qb-re", "Retained Earnings", "", "999.00"),
		providerRow("qb-fuel", "Fuel", "10.00", ""),
	}
	references := &trialReferences{rows: []*accountingsync.AccountingReferenceObject{
		{Kind: accountingsync.ReferenceKindAccount, ExternalID: "qb-re", AccountType: "Equity", AccountSubType: "RetainedEarnings"},
		{Kind: accountingsync.ReferenceKindAccount, ExternalID: "qb-bank", AccountType: "Bank"},
	}}
	h.svc = New(Params{
		Logger:            zap.NewNop(),
		DB:                dbtest.NopConnection{},
		Connections:       h.conns,
		ConnectionService: h.connSvc,
		Records:           h.records,
		Findings:          h.findings,
		Source:            h.source,
		Ledger:            f.ledger,
		Mappings:          f.mappings,
		References:        references,
		Organizations:     trialOrganizations{timezone: "UTC"},
		InvoiceRepo:       h.invRepo,
		Controls:          h.controls,
		Invoices:          h.invoices,
		Payments:          h.payments,
		Permissions:       h.perms,
		AuditService:      h.audit,
		Dispatcher:        h.kicks,
		Publisher:         h.events,
	})
	h.svc.now = func() time.Time { return h.now }
	return f
}

func (f *trialFixture) reconcile(t *testing.T) *services.AccountingDriftBalanceResult {
	t.Helper()
	result, err := f.h.svc.ReconcileBalances(t.Context(), &services.ReconcileAccountingDriftBalancesRequest{
		TenantInfo:   f.h.tenant,
		ConnectionID: f.h.conn.ID,
		EventBudget:  20,
	})
	require.NoError(t, err)
	return result
}

func openByObject(h *harness) map[pulid.ID]*accountingsync.AccountingDriftFinding {
	out := map[pulid.ID]*accountingsync.AccountingDriftFinding{}
	for _, finding := range h.findings.open() {
		out[finding.ObjectID] = finding
	}
	return out
}

func TestTrialBalanceComparesEveryMappedProviderAccount(t *testing.T) {
	t.Parallel()

	f := newTrialFixture(t)

	result := f.reconcile(t)

	assert.False(t, result.More)
	assert.Equal(t, 4, result.Customers, "receivables, cash, prepaid and income; retained earnings is left out")
	assert.Equal(t, 2, result.Opened)

	requests := f.h.reader.trialRequests()
	require.Len(t, requests, 1)
	assert.Equal(t, "2026-01-01", requests[0].StartDate, "the provider's fiscal year to date")
	assert.Equal(t, "2026-03-09", requests[0].EndDate)
	assert.Equal(t, "token", requests[0].Auth.AccessToken)

	open := openByObject(f.h)
	require.Len(t, open, 2)
	cash := open[f.cash]
	require.NotNil(t, cash)
	assert.Equal(t, accountingsync.DriftTrialBalanceMismatch, cash.Kind)
	assert.Equal(t, accountingsync.DriftObjectGLAccount, cash.ObjectType)
	assert.Equal(t, "Checking", cash.ObjectNumber, "the provider's own name for the account wins")
	assert.Equal(t, "qb-bank", cash.ExternalID)
	assert.Equal(t, int64(50_000), *cash.TrenovaMinor)
	assert.Equal(t, int64(40_000), *cash.ProviderMinor)
	assert.Equal(t, "USD", cash.CurrencyCode)
	require.Len(t, cash.Detail, 1)
	assert.Equal(t, "1000 Operating cash", cash.Detail[0].ObjectNumber)

	prepaid := open[f.prepaid]
	require.NotNil(t, prepaid, "an account Trenova never touched still counts when the provider holds a balance")
	assert.Equal(t, int64(0), *prepaid.TrenovaMinor)
	assert.Equal(t, int64(2_500), *prepaid.ProviderMinor)
	assert.Equal(t, "Prepaid", prepaid.ObjectNumber)

	assert.NotContains(t, open, f.ar)
	assert.NotContains(t, open, f.revenue, "income sums from the fiscal year start and two accounts share one")
	assert.NotContains(t, open, f.fuel)
	assert.NotContains(t, open, f.retained)
	for _, finding := range open {
		assert.NotEqual(t, "qb-fuel", finding.ExternalID, "a proposed mapping is not yet a mapping")
	}
}

func TestTrialBalanceNamesTheLowestCodedAccountAndEveryShare(t *testing.T) {
	t.Parallel()

	f := newTrialFixture(t)
	f.h.reader.trial[2] = providerRow("qb-income", "Freight income", "", "1500.00")

	f.reconcile(t)

	income := openByObject(f.h)[f.revenue]
	require.NotNil(t, income)
	assert.Equal(t, int64(-200_000), *income.TrenovaMinor)
	assert.Equal(t, int64(-150_000), *income.ProviderMinor)
	require.Len(t, income.Detail, 2)
	assert.Equal(t, "4000 Freight revenue", income.Detail[0].ObjectNumber)
	assert.Equal(t, int64(-120_000), income.Detail[0].TrenovaMinor)
	assert.Equal(t, "4010 Fuel surcharge revenue", income.Detail[1].ObjectNumber)
	assert.Equal(t, int64(-80_000), income.Detail[1].TrenovaMinor)
}

func TestTrialBalanceResolvesAFindingOnceTheBooksAgree(t *testing.T) {
	t.Parallel()

	f := newTrialFixture(t)
	f.reconcile(t)
	require.Contains(t, openByObject(f.h), f.cash)

	f.h.reader.trial[1] = providerRow("qb-bank", "Checking", "500.00", "")
	result := f.reconcile(t)

	assert.Equal(t, 1, result.Resolved)
	assert.NotContains(t, openByObject(f.h), f.cash)
	assert.Contains(t, openByObject(f.h), f.prepaid)
}

func TestTrialBalanceCountsOpeningBalancesWhenTheyWereSent(t *testing.T) {
	t.Parallel()

	f := newTrialFixture(t)
	sent := f.h.now.Add(-24 * time.Hour).Unix()
	f.h.conns.conn.LedgerOpeningBalancesSentAt = &sent
	f.ledger.opening = []repositories.LedgerAccountBalance{
		balance(f.cash, "1000", "Operating cash", accounttype.CategoryAsset, -10_000),
		balance(f.revenue, "4000", "Freight revenue", accounttype.CategoryRevenue, -555_555),
	}

	f.reconcile(t)

	assert.NotContains(t, openByObject(f.h), f.cash,
		"cash is its opening balance plus what was sent since")
	sums := f.ledger.sums()
	require.Len(t, sums, 3)
	assert.True(t, sums[1].IncludeClosing)
	assert.Equal(t, *f.h.conn.SyncStartDate, sums[1].Before)
	fiscalStart := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()
	require.NotNil(t, sums[2].From)
	assert.Equal(t, fiscalStart, *sums[2].From)
}

func TestTrialBalanceCountsIncomeOnlyFromTheStartWithoutOpeningBalances(t *testing.T) {
	t.Parallel()

	f := newTrialFixture(t)
	start := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC).Unix()
	f.h.conns.conn.SyncStartDate = &start
	f.ledger.start = start

	f.reconcile(t)

	sums := f.ledger.sums()
	require.Len(t, sums, 2)
	require.NotNil(t, sums[1].From)
	assert.Equal(t, start, *sums[1].From,
		"income before the start date never reached the provider")
	for _, sum := range sums {
		assert.False(t, sum.IncludeClosing)
	}
}

func TestTrialBalanceWaitsWhileEntriesAreStillOnTheirWay(t *testing.T) {
	t.Parallel()

	f := newTrialFixture(t)
	queued := accountingsync.NewAccountingSyncRecord(&accountingsync.NewSyncRecord{
		TenantInfo:   f.h.tenant,
		ConnectionID: f.h.conn.ID,
		Key: accountingsync.SyncRecordKey{
			ObjectType: accountingsync.SyncObjectJournalEntry,
			ObjectID:   pulid.MustNew("je_"),
			Operation:  accountingsync.SyncOperationCreate,
		},
		At: f.h.now.Unix(),
	})
	f.h.records.add(queued)

	result := f.reconcile(t)

	assert.Equal(t, 1, result.Skipped)
	assert.Empty(t, f.h.reader.trialRequests())
	assert.Empty(t, f.h.findings.all())
}

func TestTrialBalanceIsSkippedWhenTheBooksKeepAnotherCurrency(t *testing.T) {
	t.Parallel()

	f := newTrialFixture(t)
	f.h.conns.conn.ExternalHomeCurrency = "CAD"
	f.h.controls.functional = "USD"

	result := f.reconcile(t)

	assert.Equal(t, 1, result.Skipped)
	assert.Empty(t, f.h.reader.trialRequests())
}

func TestDocumentConnectionsNeverReadTheTrialBalance(t *testing.T) {
	t.Parallel()

	f := newTrialFixture(t)
	f.h.conns.conn.SyncMode = accountingsync.SyncModeDocument
	f.h.conns.conn.LedgerGranularity = ""

	f.reconcile(t)

	assert.Empty(t, f.h.reader.trialRequests())
	assert.Empty(t, f.ledger.sums())
}
