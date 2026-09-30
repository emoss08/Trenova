package accountingmappingservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type fakeLedgerAccounts struct {
	repositories.AccountingLedgerSource

	accounts []repositories.LedgerAccount
	asked    []repositories.ListLedgerAccountsRequest
}

func (f *fakeLedgerAccounts) ListActiveAccounts(
	_ context.Context,
	req *repositories.ListLedgerAccountsRequest,
) ([]repositories.LedgerAccount, error) {
	f.asked = append(f.asked, *req)
	return f.accounts, nil
}

func (h *harness) ledgerMode(t *testing.T) {
	t.Helper()
	h.connections.mu.Lock()
	defer h.connections.mu.Unlock()
	row := h.connections.rows[h.conn.ID]
	row.SyncMode = accountingsync.SyncModeLedger
	row.LedgerGranularity = accountingsync.LedgerDetailed
}

func (h *harness) seedMapping(mapping *accountingsync.AccountingMapping) {
	h.mappings.mu.Lock()
	defer h.mappings.mu.Unlock()
	mapping.ID = pulid.MustNew("acctm_")
	mapping.ConnectionID = h.conn.ID
	mapping.OrganizationID = h.tenant.OrgID
	mapping.BusinessUnitID = h.tenant.BuID
	h.mappings.rows = append(h.mappings.rows, mapping)
}

func TestRescoreOffersEveryActiveGLAccountInLedgerMode(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.ledgerMode(t)
	cash := &glaccount.GLAccount{ID: pulid.MustNew("gla_"), AccountCode: "1000", Name: "Operating cash"}
	fuel := &glaccount.GLAccount{ID: pulid.MustNew("gla_"), AccountCode: "5010", Name: "Fuel expense"}
	accounts := mocks.NewMockGLAccountRepository(t)
	accounts.EXPECT().List(mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*glaccount.GLAccount]{
			Items: []*glaccount.GLAccount{cash, fuel},
			Total: 2,
		}, nil).Once()
	h.svc.glAccounts = accounts

	h.rescore(t)

	row := h.object(t, accountingsync.TargetGLAccount, cash.ID)
	assert.Equal(t, "1000 (Operating cash)", row.TargetLabel)
	h.object(t, accountingsync.TargetGLAccount, fuel.ID)
}

func TestRescoreOffersAnInactiveAccountThatCarriesEntries(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.ledgerMode(t)
	cash := &glaccount.GLAccount{ID: pulid.MustNew("gla_"), AccountCode: "1000", Name: "Operating cash"}
	retired := repositories.LedgerAccount{ID: pulid.MustNew("gla_"), Code: "1050", Name: "Old payroll"}
	accounts := mocks.NewMockGLAccountRepository(t)
	accounts.EXPECT().List(mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*glaccount.GLAccount]{
			Items: []*glaccount.GLAccount{cash},
			Total: 1,
		}, nil).Once()
	h.svc.glAccounts = accounts
	ledger := &fakeLedgerAccounts{accounts: []repositories.LedgerAccount{
		{ID: cash.ID, Code: "1000", Name: "Operating cash"},
		retired,
	}}
	h.svc.ledger = ledger

	h.rescore(t)

	row := h.object(t, accountingsync.TargetGLAccount, retired.ID)
	assert.Equal(t, "1050 (Old payroll)", row.TargetLabel)
	cashRows := 0
	for _, mapping := range h.mappings.rows {
		if mapping.TargetType == accountingsync.TargetGLAccount && mapping.TrenovaObjectID == cash.ID {
			cashRows++
		}
	}
	assert.Equal(t, 1, cashRows, "an active account with entries is offered once")
	require.NotEmpty(t, ledger.asked)
	assert.Nil(t, ledger.asked[0].Since, "every account that ever carried an entry is offered")
}

func TestRescoreLeavesUnusedGLAccountsOutInDocumentMode(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	h.rescore(t)

	assert.Nil(t, h.mappings.byTarget(accountingsync.TargetGLAccount, pulid.MustNew("gla_"), ""))
	for _, row := range h.mappings.rows {
		assert.NotEqual(t, accountingsync.TargetGLAccount, row.TargetType,
			"without payables accounts a document connection maps no GL accounts")
	}
}

func TestLedgerModeRequiresEveryAccountWithActivity(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.ledgerMode(t)
	ar := pulid.MustNew("gla_")
	revenue := pulid.MustNew("gla_")
	cash := pulid.MustNew("gla_")
	ledger := &fakeLedgerAccounts{accounts: []repositories.LedgerAccount{
		{ID: ar, Code: "1100"}, {ID: revenue, Code: "4000"}, {ID: cash, Code: "1000"},
	}}
	h.svc.ledger = ledger
	controls := mocks.NewMockAccountingControlRepository(t)
	controls.EXPECT().GetByOrgID(mock.Anything, h.tenant.OrgID).
		Return(&tenant.AccountingControl{DefaultRevenueAccountID: revenue}, nil)
	h.svc.accountingControls = controls
	h.seedMapping(&accountingsync.AccountingMapping{
		TargetType:      accountingsync.TargetGLAccount,
		TrenovaObjectID: ar,
		ExternalID:      "qb-ar",
		State:           accountingsync.MappingStateConfirmed,
	})
	h.seedMapping(&accountingsync.AccountingMapping{
		TargetType:      accountingsync.TargetGLAccount,
		TrenovaObjectID: cash,
		ExternalID:      "qb-bank",
		State:           accountingsync.MappingStateProposed,
	})
	h.seedMapping(&accountingsync.AccountingMapping{
		TargetType: accountingsync.TargetAccountRole,
		TrenovaKey: accountingsync.AccountRoleRevenue,
		ExternalID: "qb-income",
		State:      accountingsync.MappingStateConfirmed,
	})

	summary, err := h.svc.Summary(t.Context(), h.tenant, integration.TypeQuickBooksOnline)
	require.NoError(t, err)

	assert.Equal(t, 3, summary.RequiredTotal)
	assert.Equal(t, 2, summary.RequiredConfirmed, "receivables by its own mapping, revenue by its role")
	assert.False(t, summary.CanCompleteSetup, "cash is only proposed")
	require.Len(t, ledger.asked, 1)
	assert.Nil(t, ledger.asked[0].Since, "before a start date every account with activity counts")
}

func TestLedgerRequirementCountsFromTheStartDateUnlessOpeningBalancesWereSent(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.ledgerMode(t)
	start := int64(1_790_000_000)
	h.connections.mu.Lock()
	h.connections.rows[h.conn.ID].SyncStartDate = &start
	h.connections.mu.Unlock()
	ledger := &fakeLedgerAccounts{}
	h.svc.ledger = ledger

	summary, err := h.svc.Summary(t.Context(), h.tenant, integration.TypeQuickBooksOnline)
	require.NoError(t, err)
	assert.Equal(t, 0, summary.RequiredTotal)
	assert.True(t, summary.CanCompleteSetup)
	require.Len(t, ledger.asked, 1)
	require.NotNil(t, ledger.asked[0].Since)
	assert.Equal(t, start, *ledger.asked[0].Since)

	h.connections.mu.Lock()
	h.connections.rows[h.conn.ID].LedgerOpeningBalancesSentAt = &start
	h.connections.mu.Unlock()
	_, err = h.svc.Summary(t.Context(), h.tenant, integration.TypeQuickBooksOnline)
	require.NoError(t, err)
	require.Len(t, ledger.asked, 2)
	assert.Nil(t, ledger.asked[1].Since, "opening balances carry every earlier account")
}
