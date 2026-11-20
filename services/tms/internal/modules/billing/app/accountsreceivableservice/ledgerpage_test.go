package accountsreceivableservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerledgerportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func ledgerEntry(
	customerID pulid.ID,
	date, line int64,
	amount int64,
) *customerledgerports.ARLedgerEntry {
	return &customerledgerports.ARLedgerEntry{
		CustomerID:      customerID,
		TransactionDate: date,
		AmountMinor:     amount,
		Position: customerledgerports.ARLedgerPosition{
			TransactionDate: date,
			LineNumber:      line,
			ID:              pulid.MustNew("cle_"),
		},
	}
}

func TestListCustomerLedgerCarriesTheBalanceAcrossPages(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	customerID := pulid.MustNew("cus_")
	first := []*customerledgerports.ARLedgerEntry{
		ledgerEntry(customerID, 110, 1, 1000),
		ledgerEntry(customerID, 110, 2, -300),
		ledgerEntry(customerID, 150, 1, 50),
	}
	second := []*customerledgerports.ARLedgerEntry{
		ledgerEntry(customerID, 160, 1, -200),
	}

	repo := customerledgerportsmock.NewMockAccountsReceivableRepository(t)
	repo.EXPECT().
		GetCustomerLedgerBalance(mock.Anything, customerledgerports.GetCustomerLedgerBalanceRequest{
			TenantInfo: tenant,
			CustomerID: customerID,
			Before:     100,
		}).
		Return(int64(5000), nil).
		Once()
	repo.EXPECT().
		ListCustomerLedgerWindow(mock.Anything, customerledgerports.ListCustomerLedgerWindowRequest{
			TenantInfo: tenant,
			CustomerID: customerID,
			From:       100,
			Through:    200,
			Limit:      3,
		}).
		Return(first, nil).
		Once()
	svc := &Service{repo: repo}
	req := customerledgerports.ARLedgerPageRequest{
		CustomerID: customerID,
		StartDate:  100,
		EndDate:    200,
		First:      2,
	}

	page, err := svc.ListCustomerLedger(t.Context(), tenant, req)

	require.NoError(t, err)
	assert.Equal(t, int64(5000), page.OpeningBalanceMinor)
	assert.True(t, page.HasNextPage)
	require.Len(t, page.Entries, 2)
	assert.Equal(t, int64(6000), page.Entries[0].RunningBalanceMinor)
	assert.Equal(t, int64(5700), page.Entries[1].RunningBalanceMinor)
	require.NotEmpty(t, page.EndCursor)

	after := first[1].Position
	repo.EXPECT().
		ListCustomerLedgerWindow(mock.Anything, customerledgerports.ListCustomerLedgerWindowRequest{
			TenantInfo: tenant,
			CustomerID: customerID,
			From:       100,
			Through:    200,
			After:      &after,
			Limit:      3,
		}).
		Return(second, nil).
		Once()
	req.After = page.EndCursor

	next, err := svc.ListCustomerLedger(t.Context(), tenant, req)

	require.NoError(t, err)
	assert.Equal(t, int64(5700), next.OpeningBalanceMinor,
		"a later page opens on the balance the page before it ended on")
	assert.False(t, next.HasNextPage)
	assert.Empty(t, next.EndCursor)
	require.Len(t, next.Entries, 1)
	assert.Equal(t, int64(5500), next.Entries[0].RunningBalanceMinor)
}

func TestListCustomerLedgerFromTheFirstEntryOpensAtZero(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	repo := customerledgerportsmock.NewMockAccountsReceivableRepository(t)
	repo.EXPECT().
		ListCustomerLedgerWindow(mock.Anything, mock.MatchedBy(
			func(req customerledgerports.ListCustomerLedgerWindowRequest) bool {
				return req.From == 0 && req.Through > 0 &&
					req.Limit == DefaultLedgerPageSize+1
			},
		)).
		Return([]*customerledgerports.ARLedgerEntry{ledgerEntry(customerID, 10, 1, 400)}, nil).
		Once()
	svc := &Service{repo: repo}

	page, err := svc.ListCustomerLedger(
		t.Context(),
		pagination.TenantInfo{},
		customerledgerports.ARLedgerPageRequest{CustomerID: customerID},
	)

	require.NoError(t, err)
	assert.Zero(t, page.OpeningBalanceMinor)
	assert.False(t, page.HasNextPage)
	require.Len(t, page.Entries, 1)
	assert.Equal(t, int64(400), page.Entries[0].RunningBalanceMinor)
}

func TestListCustomerLedgerCapsThePageSize(t *testing.T) {
	t.Parallel()

	repo := customerledgerportsmock.NewMockAccountsReceivableRepository(t)
	repo.EXPECT().
		ListCustomerLedgerWindow(mock.Anything, mock.MatchedBy(
			func(req customerledgerports.ListCustomerLedgerWindowRequest) bool {
				return req.Limit == MaxLedgerPageSize+1
			},
		)).
		Return(nil, nil).
		Once()
	svc := &Service{repo: repo}

	page, err := svc.ListCustomerLedger(
		t.Context(),
		pagination.TenantInfo{},
		customerledgerports.ARLedgerPageRequest{CustomerID: pulid.MustNew("cus_"), First: 10_000},
	)

	require.NoError(t, err)
	assert.Empty(t, page.Entries)
}

func TestListCustomerLedgerRefusesACursorFromAnotherWindow(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	customerID := pulid.MustNew("cus_")
	issued := customerledgerports.ARLedgerPageRequest{CustomerID: customerID, EndDate: 200}
	cursor := sealLedgerCursor(ledgerCursorBinding(tenant, issued), ledgerCursor{
		through:  200,
		balance:  900,
		position: customerledgerports.ARLedgerPosition{TransactionDate: 10, ID: "cle_1"},
	})
	svc := &Service{repo: customerledgerportsmock.NewMockAccountsReceivableRepository(t)}

	cases := map[string]struct {
		tenant pagination.TenantInfo
		req    customerledgerports.ARLedgerPageRequest
	}{
		"another customer": {
			tenant: tenant,
			req: customerledgerports.ARLedgerPageRequest{
				CustomerID: pulid.MustNew("cus_"), EndDate: 200, After: cursor,
			},
		},
		"another window": {
			tenant: tenant,
			req: customerledgerports.ARLedgerPageRequest{
				CustomerID: customerID, StartDate: 5, EndDate: 200, After: cursor,
			},
		},
		"another organization": {
			tenant: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: tenant.BuID},
			req: customerledgerports.ARLedgerPageRequest{
				CustomerID: customerID, EndDate: 200, After: cursor,
			},
		},
		"an altered cursor": {
			tenant: tenant,
			req: customerledgerports.ARLedgerPageRequest{
				CustomerID: customerID, EndDate: 200, After: cursor + "A",
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := svc.ListCustomerLedger(t.Context(), tc.tenant, tc.req)

			var validation *errortypes.Error
			require.ErrorAs(t, err, &validation)
			assert.Equal(t, "after", validation.Field())
		})
	}
}

func TestListCustomerLedgerChecksTheWindow(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	svc := &Service{repo: customerledgerportsmock.NewMockAccountsReceivableRepository(t)}

	cases := map[string]struct {
		req   customerledgerports.ARLedgerPageRequest
		field string
	}{
		"no customer": {
			req:   customerledgerports.ARLedgerPageRequest{},
			field: "customerId",
		},
		"a negative date": {
			req:   customerledgerports.ARLedgerPageRequest{CustomerID: customerID, StartDate: -1},
			field: "startDate",
		},
		"a start after the end": {
			req: customerledgerports.ARLedgerPageRequest{
				CustomerID: customerID, StartDate: 300, EndDate: 200,
			},
			field: "startDate",
		},
		"a negative page size": {
			req:   customerledgerports.ARLedgerPageRequest{CustomerID: customerID, First: -1},
			field: "first",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := svc.ListCustomerLedger(t.Context(), pagination.TenantInfo{}, tc.req)

			var validation *errortypes.Error
			require.ErrorAs(t, err, &validation)
			assert.Equal(t, tc.field, validation.Field())
		})
	}
}
