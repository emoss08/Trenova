//go:build integration

package accountingsyncrepository

import (
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment/customerpaymentdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement/carriersettlementdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement/driversettlementdb"
	"github.com/emoss08/trenova/pkg/dbscope"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/customerpaymentrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/invoicerepository"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestExchangeRateStamps_LandOnTheDocumentTheyName(t *testing.T) {
	seeded := newSeededTenant(t)
	ctx, db, tenant, userID := seeded.ctx, seeded.kdb, seeded.tenant, seeded.userID
	org := struct{ ID, BusinessUnitID pulid.ID }{tenant.OrgID, tenant.BuID}
	shp := seeded.shipments(t, 1)[0]
	carrierID, workerID := seeded.carrier(t).ID, seeded.worker(t).ID
	other := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	conn := seeded.conn
	logger := zap.NewNop()
	invoices := invoicerepository.New(invoicerepository.Params{DB: conn, Logger: logger})
	payments := customerpaymentrepository.New(customerpaymentrepository.Params{DB: conn, Logger: logger})
	payables := NewPayablesRepository(PayablesParams{DB: conn, Logger: logger})

	const dated = int64(1_790_000_000)
	rate := decimal.RequireFromString("0.731200000000")
	paidRate := decimal.RequireFromString("0.728800000000")

	queue := &billingqueue.BillingQueueItem{
		OrganizationID:   org.ID,
		BusinessUnitID:   org.BusinessUnitID,
		ShipmentID:       shp.ID,
		BillToCustomerID: shp.CustomerID,
		Number:           "INV-FX-1",
		Status:           billingqueue.StatusPosted,
		BillType:         billingqueue.BillTypeInvoice,
	}
	err := billingqueuedb.BillingQueueItems.Create(ctx, db, queue)
	require.NoError(t, err)
	posted := dated + 100
	inv, err := invoices.Create(ctx, &invoice.Invoice{
		OrganizationID:     org.ID,
		BusinessUnitID:     org.BusinessUnitID,
		BillingQueueItemID: queue.ID,
		ShipmentID:         shp.ID,
		CustomerID:         shp.CustomerID,
		Number:             "INV-FX-1",
		BillType:           billingqueue.BillTypeInvoice,
		Status:             invoice.StatusPosted,
		PostedAt:           &posted,
		PaymentTerm:        invoice.PaymentTermNet30,
		CurrencyCode:       "CAD",
		InvoiceDate:        dated,
		ShipmentProNumber:  shp.ProNumber,
		ShipmentBOL:        shp.BOL,
		BillToName:         "Test Customer",
		SubtotalAmount:     decimal.New(150000, -2),
		OtherAmount:        decimal.Zero,
		TotalAmount:        decimal.New(150000, -2),
		AppliedAmount:      decimal.Zero,
		SettlementStatus:   invoice.SettlementStatusUnpaid,
		DisputeStatus:      invoice.DisputeStatusNone,
	})
	require.NoError(t, err)
	require.False(t, inv.ExchangeRate.Valid)

	t.Run("invoice", func(t *testing.T) {
		require.NoError(t, invoices.StampExchangeRate(ctx, &kernelports.StampExchangeRateRequest{
			TenantInfo: tenant, ID: inv.ID, Rate: rate, Date: dated,
		}))
		stored, getErr := invoices.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{ID: inv.ID, TenantInfo: tenant})
		require.NoError(t, getErr)
		require.True(t, stored.ExchangeRate.Valid)
		assert.True(t, rate.Equal(stored.ExchangeRate.Decimal))
		require.NotNil(t, stored.ExchangeRateDate)
		assert.Equal(t, dated, *stored.ExchangeRateDate)

		err = invoices.StampExchangeRate(dbscope.WithTenant(ctx, other.DBTenant()), &kernelports.StampExchangeRateRequest{
			TenantInfo: other, ID: inv.ID, Rate: paidRate, Date: dated,
		})
		require.Error(t, err, "another tenant cannot stamp the invoice")
		assert.True(t, errortypes.IsNotFoundError(err))
	})

	t.Run("customer payment", func(t *testing.T) {
		payment := &customerpayment.Payment{
			ID:                   pulid.MustNew("cpay_"),
			OrganizationID:       org.ID,
			BusinessUnitID:       org.BusinessUnitID,
			CustomerID:           shp.CustomerID,
			PaymentDate:          dated,
			AccountingDate:       dated,
			AmountMinor:          1_000,
			UnappliedAmountMinor: 1_000,
			Status:               customerpayment.StatusPosted,
			PaymentMethod:        customerpayment.MethodACH,
			CurrencyCode:         "CAD",
			CreatedByID:          userID,
		}
		insertErr := customerpaymentdb.Payments.Create(ctx, db, payment)
		require.NoError(t, insertErr)

		require.NoError(t, payments.StampExchangeRate(ctx, &kernelports.StampExchangeRateRequest{
			TenantInfo: tenant, ID: payment.ID, Rate: rate, Date: dated,
		}))
		stored, getErr := payments.GetByID(ctx, customerpaymentports.GetCustomerPaymentByIDRequest{
			ID: payment.ID, TenantInfo: tenant,
		})
		require.NoError(t, getErr)
		require.True(t, stored.ExchangeRate.Valid)
		assert.True(t, rate.Equal(stored.ExchangeRate.Decimal))
		require.NotNil(t, stored.ExchangeRateDate)
		assert.Equal(t, dated, *stored.ExchangeRateDate)

		err = payments.StampExchangeRate(dbscope.WithTenant(ctx, other.DBTenant()), &kernelports.StampExchangeRateRequest{
			TenantInfo: other, ID: payment.ID, Rate: paidRate, Date: dated,
		})
		require.Error(t, err)
		assert.True(t, errortypes.IsNotFoundError(err))
	})

	t.Run("carrier settlement keeps a posted and a paid stamp apart", func(t *testing.T) {
		paidAt := dated + 8*86400
		settlement := &carriersettlement.CarrierSettlement{
			ID:               pulid.MustNew("carstl_"),
			OrganizationID:   org.ID,
			BusinessUnitID:   org.BusinessUnitID,
			CarrierID:        carrierID,
			SettlementNumber: "CS-FX-1",
			Status:           carriersettlement.StatusPaid,
			PeriodStart:      dated - 7*86400,
			PeriodEnd:        dated,
			PayDate:          dated + 15*86400,
			NetPayableMinor:  165000,
			CurrencyCode:     "CAD",
			PostedAt:         &posted,
			PaidAt:           &paidAt,
		}
		insertErr := carriersettlementdb.CarrierSettlements.Create(ctx, db, settlement)
		require.NoError(t, insertErr)

		require.NoError(t, payables.StampExchangeRate(ctx, &accountingsyncports.StampPayableExchangeRateRequest{
			TenantInfo: tenant, Kind: accountingsyncports.PayableCarrier, ID: settlement.ID, Rate: rate, Date: dated,
		}))
		require.NoError(t, payables.StampExchangeRate(ctx, &accountingsyncports.StampPayableExchangeRateRequest{
			TenantInfo: tenant, Kind: accountingsyncports.PayableCarrier, ID: settlement.ID, Paid: true,
			Rate: paidRate, Date: paidAt,
		}))
		stored, getErr := payables.GetSettlement(ctx, &accountingsyncports.GetPayableSettlementRequest{
			TenantInfo: tenant, Kind: accountingsyncports.PayableCarrier, ID: settlement.ID,
		})
		require.NoError(t, getErr)
		require.True(t, stored.ExchangeRate.Valid)
		assert.True(t, rate.Equal(stored.ExchangeRate.Decimal))
		assert.Equal(t, dated, *stored.ExchangeRateDate)
		require.True(t, stored.PaidExchangeRate.Valid)
		assert.True(t, paidRate.Equal(stored.PaidExchangeRate.Decimal))
		assert.Equal(t, paidAt, *stored.PaidExchangeRateDate)

		err = payables.StampExchangeRate(ctx, &accountingsyncports.StampPayableExchangeRateRequest{
			TenantInfo: tenant, Kind: accountingsyncports.PayableDriver, ID: settlement.ID, Rate: rate, Date: dated,
		})
		require.Error(t, err, "a carrier settlement is not a driver settlement")
		assert.True(t, errortypes.IsNotFoundError(err))
	})

	t.Run("driver settlement", func(t *testing.T) {
		settlement := &driversettlement.Settlement{
			ID:               pulid.MustNew("dstl_"),
			OrganizationID:   org.ID,
			BusinessUnitID:   org.BusinessUnitID,
			WorkerID:         workerID,
			SettlementNumber: "DS-FX-1",
			Status:           driversettlement.StatusPosted,
			Classification:   driverpay.PayeeClassificationOwnerOperator,
			PeriodStart:      dated - 7*86400,
			PeriodEnd:        dated,
			PayDate:          dated + 15*86400,
			NetPayMinor:      210000,
			CurrencyCode:     "CAD",
			PostedAt:         &posted,
		}
		insertErr := driversettlementdb.Settlements.Create(ctx, db, settlement)
		require.NoError(t, insertErr)

		require.NoError(t, payables.StampExchangeRate(ctx, &accountingsyncports.StampPayableExchangeRateRequest{
			TenantInfo: tenant, Kind: accountingsyncports.PayableDriver, ID: settlement.ID, Rate: rate, Date: dated,
		}))
		stored, getErr := payables.GetSettlement(ctx, &accountingsyncports.GetPayableSettlementRequest{
			TenantInfo: tenant, Kind: accountingsyncports.PayableDriver, ID: settlement.ID,
		})
		require.NoError(t, getErr)
		require.True(t, stored.ExchangeRate.Valid)
		assert.True(t, rate.Equal(stored.ExchangeRate.Decimal))
		assert.False(t, stored.PaidExchangeRate.Valid)

		err = payables.StampExchangeRate(ctx, &accountingsyncports.StampPayableExchangeRateRequest{
			TenantInfo: tenant, Kind: accountingsyncports.PayableKind("Vendor"), ID: settlement.ID, Rate: rate, Date: dated,
		})
		require.Error(t, err, "an unknown payable kind is refused")
	})
}
