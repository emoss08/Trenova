//go:build integration

package customerledgerrepository

import (
	"testing"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/billing/billingtest"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger/customerledgerdb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func at(year int, month time.Month, day, hour, minute, second int) int64 {
	return time.Date(year, month, day, hour, minute, second, 0, time.UTC).Unix()
}

func TestLedgerSumsMatchTheWholeLedger(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	defer cleanup()

	repo := New(Params{DB: postgres.WrapDB(db), Logger: zap.NewNop()})
	seeded := billingtest.SeededTenant(t, ctx, db)
	ctx, tenant := seeded.Ctx, seeded.Info
	customerID := seeded.Shipment.CustomerID

	invA := pulid.MustNew("inv_")
	invB := pulid.MustNew("inv_")
	entry := func(
		date int64,
		amount int64,
		sourceType, sourceID string,
		related pulid.ID,
	) *customerledger.CustomerLedgerEntry {
		return &customerledger.CustomerLedgerEntry{
			OrganizationID:   tenant.OrgID,
			BusinessUnitID:   tenant.BuID,
			CustomerID:       customerID,
			SourceObjectType: sourceType,
			SourceObjectID:   sourceID,
			SourceEventType:  accountingcontrol.JournalSourceEventCustomerPaymentPosted.String(),
			RelatedInvoiceID: related,
			DocumentNumber:   "HIST",
			TransactionDate:  date,
			LineNumber:       1,
			AmountMinor:      amount,
			CreatedByID:      tenant.UserID,
		}
	}
	payment := func() string { return pulid.MustNew("cpay_").String() }
	require.NoError(t, repo.AppendEntries(ctx, []*customerledger.CustomerLedgerEntry{
		entry(at(2001, time.October, 5, 9, 0, 0), 10000, "Invoice", invA.String(), ""),
		entry(at(2001, time.November, 20, 15, 0, 0), -4000, "CustomerPayment", payment(), invA),
		entry(at(2001, time.December, 31, 23, 59, 59), 25000, "Invoice", invB.String(), ""),
		entry(at(2002, time.January, 1, 0, 0, 0), -1500, "CustomerPayment", payment(), ""),
		entry(at(2002, time.January, 14, 8, 0, 0), -6000, "CustomerPayment", payment(), invA),
		entry(at(2002, time.February, 2, 10, 0, 0), -5000, "CustomerPayment", payment(), invB),
	}))

	cle := customerledgerdb.CustomerLedgerEntries
	wholeAsOf := func(asOf int64) int64 {
		t.Helper()
		total, err := keel.IntoOne[int64](ctx, db, cle.
			Pick(keel.NumericOf(cle.AmountMinor.Sum()).Coalesce(0)).
			Where(cle.TransactionDate.Lte(asOf)))
		require.NoError(t, err)
		return total
	}
	wholeCustomer := func(invoiceIDs ...pulid.ID) int64 {
		t.Helper()
		ids := make([]string, 0, len(invoiceIDs))
		for _, id := range invoiceIDs {
			ids = append(ids, id.String())
		}
		total, err := keel.IntoOne[int64](ctx, db, cle.
			Pick(keel.NumericOf(cle.AmountMinor.Sum()).Coalesce(0)).
			Where(
				cle.CustomerID.Eq(customerID),
				keel.When(len(ids) > 0, keel.Or(
					cle.RelatedInvoiceID.In(invoiceIDs...),
					keel.And(
						cle.RelatedInvoiceID.IsNull(),
						cle.SourceObjectType.Eq("Invoice"),
						cle.SourceObjectID.In(ids...),
					),
				)),
			))
		require.NoError(t, err)
		return total
	}

	t.Run("the balance as of any instant is every entry up to it", func(t *testing.T) {
		for _, asOf := range []int64{
			at(2001, time.September, 30, 0, 0, 0),
			at(2001, time.October, 5, 9, 0, 0),
			at(2001, time.October, 31, 23, 59, 59),
			at(2001, time.November, 20, 14, 59, 59),
			at(2001, time.December, 31, 23, 59, 58),
			at(2001, time.December, 31, 23, 59, 59),
			at(2002, time.January, 1, 0, 0, 0),
			at(2002, time.January, 31, 23, 59, 59),
			at(2002, time.February, 15, 0, 0, 0),
			at(2030, time.June, 30, 23, 59, 59),
		} {
			got, err := repo.SumBalanceAsOf(ctx, customerledgerports.SumCustomerLedgerBalanceRequest{
				TenantInfo: tenant,
				AsOfDate:   asOf,
			})
			require.NoError(t, err)
			assert.Equal(t, wholeAsOf(asOf), got, time.Unix(asOf, 0).UTC())
		}
	})

	t.Run("a customer's balance is every entry it has", func(t *testing.T) {
		balances, err := repo.SumCustomerBalances(ctx, customerledgerports.SumCustomerBalancesRequest{
			TenantInfo:  tenant,
			CustomerIDs: []pulid.ID{customerID, pulid.MustNew("cus_")},
		})
		require.NoError(t, err)
		assert.Equal(t, wholeCustomer(), balances[customerID])
		assert.Equal(t, int64(18500), balances[customerID])
		assert.Len(t, balances, 1)
	})

	t.Run("the invoice balances follow every write", func(t *testing.T) {
		readInvoices := func() map[pulid.ID]int64 {
			t.Helper()
			ib := customerledgerdb.InvoiceBalances
			rows, err := ib.
				Where(ib.CustomerID.Eq(customerID)).
				Order(ib.InvoiceID.Asc()).
				Limit(10).
				List(ctx, db)
			require.NoError(t, err)
			got := make(map[pulid.ID]int64, len(rows))
			for _, row := range rows {
				got[row.InvoiceID] = row.AmountMinor
			}
			return got
		}
		assert.Equal(t, map[pulid.ID]int64{
			invA: wholeCustomer(invA),
			invB: wholeCustomer(invB),
		}, readInvoices())
		assert.Equal(t, int64(0), wholeCustomer(invA), "invoice A is settled")

		_, err := cle.
			Where(cle.CustomerID.Eq(customerID), cle.RelatedInvoiceID.Eq(invB)).
			Update(ctx, db, cle.RelatedInvoiceID.To(invA))
		require.NoError(t, err)
		_, err = cle.
			Where(cle.CustomerID.Eq(customerID), cle.SourceObjectID.Eq(invB.String())).
			Delete(ctx, db)
		require.NoError(t, err)

		assert.Equal(t, map[pulid.ID]int64{
			invA: wholeCustomer(invA),
			invB: 0,
		}, readInvoices())
		assert.Equal(t, int64(-5000), wholeCustomer(invA))

		balances, err := repo.SumCustomerBalances(ctx, customerledgerports.SumCustomerBalancesRequest{
			TenantInfo:  tenant,
			CustomerIDs: []pulid.ID{customerID},
		})
		require.NoError(t, err)
		assert.Equal(t, wholeCustomer(), balances[customerID])

		got, err := repo.SumBalanceAsOf(ctx, customerledgerports.SumCustomerLedgerBalanceRequest{
			TenantInfo: tenant,
			AsOfDate:   at(2002, time.January, 31, 23, 59, 59),
		})
		require.NoError(t, err)
		assert.Equal(t, wholeAsOf(at(2002, time.January, 31, 23, 59, 59)), got)
	})
}
