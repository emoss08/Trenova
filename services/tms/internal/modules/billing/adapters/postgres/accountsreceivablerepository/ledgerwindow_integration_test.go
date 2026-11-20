//go:build integration

package accountsreceivablerepository

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/customerledgerrepository"
	"github.com/emoss08/trenova/internal/modules/billing/billingtest"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger/customerledgerdb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func ledgerAt(year int, month time.Month, day, hour, minute, second int) int64 {
	return time.Date(year, month, day, hour, minute, second, 0, time.UTC).Unix()
}

type ledgerFixture struct {
	date   int64
	line   int
	amount int64
}

func fullLedger(
	entries []ledgerFixture,
	start, through int64,
) (int64, []int64, []int64) {
	opening := int64(0)
	var amounts, running []int64
	balance := int64(0)
	for _, e := range entries {
		if e.date > through {
			continue
		}
		balance += e.amount
		if e.date < start {
			opening += e.amount
			continue
		}
		amounts = append(amounts, e.amount)
		running = append(running, balance)
	}
	return opening, amounts, running
}

func TestCustomerLedgerWindowMatchesTheWholeLedger(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	defer cleanup()

	conn := postgres.WrapDB(db)
	logger := zap.NewNop()
	repo := New(Params{DB: conn, Logger: logger})
	ledgerRepo := customerledgerrepository.New(
		customerledgerrepository.Params{DB: conn, Logger: logger},
	)
	seeded := billingtest.SeededTenant(t, ctx, db)
	ctx, org, user, customerID := seeded.Ctx, seeded.Org, seeded.User, seeded.Shipment.CustomerID
	tenant := pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}

	fixtures := []ledgerFixture{
		{date: ledgerAt(2001, time.November, 3, 9, 0, 0), line: 1, amount: 5000},
		{date: ledgerAt(2001, time.November, 20, 15, 0, 0), line: 1, amount: -2000},
		{date: ledgerAt(2001, time.December, 31, 23, 59, 59), line: 1, amount: 700},
		{date: ledgerAt(2002, time.January, 1, 0, 0, 0), line: 1, amount: 300},
		{date: ledgerAt(2002, time.January, 10, 12, 0, 0), line: 1, amount: 400},
		{date: ledgerAt(2002, time.January, 10, 12, 0, 0), line: 2, amount: -100},
		{date: ledgerAt(2002, time.January, 15, 0, 0, 0), line: 1, amount: 250},
		{date: ledgerAt(2002, time.January, 20, 8, 0, 0), line: 1, amount: -600},
		{date: ledgerAt(2002, time.January, 20, 8, 0, 0), line: 2, amount: 75},
		{date: ledgerAt(2002, time.February, 2, 10, 0, 0), line: 1, amount: 900},
	}
	rows := make([]*customerledger.CustomerLedgerEntry, 0, len(fixtures))
	for i, f := range fixtures {
		rows = append(rows, &customerledger.CustomerLedgerEntry{
			OrganizationID:   org.ID,
			BusinessUnitID:   org.BusinessUnitID,
			CustomerID:       customerID,
			SourceObjectType: "CustomerPayment",
			SourceObjectID:   pulid.MustNew("cpay_").String(),
			SourceEventType:  accountingcontrol.JournalSourceEventCustomerPaymentPosted.String(),
			DocumentNumber:   "HIST-" + string(rune('A'+i)),
			TransactionDate:  f.date,
			LineNumber:       f.line,
			AmountMinor:      f.amount,
			CreatedByID:      user.ID,
		})
	}
	require.NoError(t, ledgerRepo.AppendEntries(ctx, rows))

	start := ledgerAt(2002, time.January, 15, 0, 0, 0)
	through := ledgerAt(2002, time.January, 31, 23, 59, 59)
	wantOpening, wantAmounts, wantRunning := fullLedger(fixtures, start, through)
	require.Equal(t, int64(4300), wantOpening)
	require.Equal(t, []int64{250, -600, 75}, wantAmounts)
	require.Equal(t, []int64{4550, 3950, 4025}, wantRunning)

	balanceBefore := func(before int64) int64 {
		t.Helper()
		balance, err := repo.GetCustomerLedgerBalance(
			ctx,
			customerledgerports.GetCustomerLedgerBalanceRequest{
				TenantInfo: tenant,
				CustomerID: customerID,
				Before:     before,
			},
		)
		require.NoError(t, err)
		return balance
	}
	window := func(
		after *customerledgerports.ARLedgerPosition,
		limit int,
	) []*customerledgerports.ARLedgerEntry {
		t.Helper()
		entries, err := repo.ListCustomerLedgerWindow(
			ctx,
			customerledgerports.ListCustomerLedgerWindowRequest{
				TenantInfo: tenant,
				CustomerID: customerID,
				From:       start,
				Through:    through,
				After:      after,
				Limit:      limit,
			},
		)
		require.NoError(t, err)
		return entries
	}

	t.Run("the opening balance plus the window is the whole ledger", func(t *testing.T) {
		opening := balanceBefore(start)
		assert.Equal(t, wantOpening, opening)

		entries := window(nil, 100)
		amounts := make([]int64, 0, len(entries))
		running := make([]int64, 0, len(entries))
		balance := opening
		for _, entry := range entries {
			balance += entry.AmountMinor
			amounts = append(amounts, entry.AmountMinor)
			running = append(running, balance)
		}
		assert.Equal(t, wantAmounts, amounts)
		assert.Equal(t, wantRunning, running)
	})

	t.Run("the opening balance at any instant matches the whole ledger", func(t *testing.T) {
		for _, before := range []int64{
			ledgerAt(2001, time.November, 1, 0, 0, 0),
			ledgerAt(2001, time.November, 20, 15, 0, 0),
			ledgerAt(2001, time.November, 20, 15, 0, 1),
			ledgerAt(2002, time.January, 1, 0, 0, 0),
			ledgerAt(2002, time.January, 1, 0, 0, 1),
			ledgerAt(2002, time.January, 10, 12, 0, 0),
			ledgerAt(2002, time.January, 10, 12, 0, 1),
			ledgerAt(2002, time.March, 1, 0, 0, 0),
		} {
			want, _, _ := fullLedger(fixtures, before, before)
			assert.Equal(t, want, balanceBefore(before), time.Unix(before, 0).UTC())
		}
	})

	t.Run("a page ends where the next begins", func(t *testing.T) {
		first := window(nil, 2)
		require.Len(t, first, 2)
		assert.Equal(t, int64(250), first[0].AmountMinor)
		assert.Equal(t, int64(-600), first[1].AmountMinor)
		assert.Equal(t, int64(1), first[1].Position.LineNumber)

		rest := window(&first[1].Position, 2)
		require.Len(t, rest, 1)
		assert.Equal(t, int64(75), rest[0].AmountMinor)
		assert.Equal(t, int64(2), rest[0].Position.LineNumber)
	})

	t.Run("the monthly balances follow every write", func(t *testing.T) {
		cle := customerledgerdb.CustomerLedgerEntries
		_, err := cle.
			Where(cle.CustomerID.Eq(customerID), cle.DocumentNumber.Eq("HIST-B")).
			Delete(ctx, db)
		require.NoError(t, err)
		_, err = cle.
			Where(cle.CustomerID.Eq(customerID), cle.DocumentNumber.Eq("HIST-C")).
			Update(ctx, db, cle.TransactionDate.To(ledgerAt(2002, time.January, 16, 0, 0, 0)))
		require.NoError(t, err)

		mb := customerledgerdb.MonthlyBalances
		months, err := mb.
			Where(
				mb.CustomerID.Eq(customerID),
				mb.MonthStart.Lt(ledgerAt(2002, time.March, 1, 0, 0, 0)),
			).
			Order(mb.MonthStart.Asc()).
			Limit(10).
			List(ctx, db)
		require.NoError(t, err)
		got := make(map[int64]int64, len(months))
		for _, m := range months {
			got[m.MonthStart] = m.AmountMinor
		}
		assert.Equal(t, map[int64]int64{
			ledgerAt(2001, time.November, 1, 0, 0, 0): 5000,
			ledgerAt(2001, time.December, 1, 0, 0, 0): 0,
			ledgerAt(2002, time.January, 1, 0, 0, 0):  1025,
			ledgerAt(2002, time.February, 1, 0, 0, 0): 900,
		}, got)

		assert.Equal(t, int64(5600), balanceBefore(start))
		assert.Equal(t, int64(5000), balanceBefore(ledgerAt(2002, time.January, 1, 0, 0, 0)))
	})
}
