//go:build integration

package accountingsyncrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/customerpaymentrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/invoicerepository"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestExchangeRateStamps_LandOnTheDocumentTheyName(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	registry := seeder.NewRegistry()
	seeds.Register(registry)
	_, err := seeder.NewEngine(
		db,
		registry,
		&config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}},
	).Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	var org candidateOrg
	require.NoError(t, db.NewSelect().Table("organizations").Column("id", "business_unit_id").Limit(1).Scan(ctx, &org))
	var userID pulid.ID
	require.NoError(t, db.NewSelect().Table("users").Column("id").
		Where("current_organization_id = ?", org.ID).Limit(1).Scan(ctx, &userID))
	var shp candidateShipment
	require.NoError(t, db.NewSelect().Table("shipments").
		Column("id", "customer_id", "pro_number", "bol").
		Where("organization_id = ?", org.ID).
		Where("business_unit_id = ?", org.BusinessUnitID).
		Order("id").Limit(1).Scan(ctx, &shp))
	var carrierID, workerID pulid.ID
	require.NoError(t, db.NewSelect().Table("carriers").Column("id").
		Where("organization_id = ?", org.ID).Limit(1).Scan(ctx, &carrierID))
	require.NoError(t, db.NewSelect().Table("workers").Column("id").
		Where("organization_id = ?", org.ID).Limit(1).Scan(ctx, &workerID))

	tenant := pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}
	other := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	conn := postgres.NewTestConnection(db)
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
	_, err = db.NewInsert().Model(queue).Exec(ctx)
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
		require.NoError(t, invoices.StampExchangeRate(ctx, &repositories.StampExchangeRateRequest{
			TenantInfo: tenant, ID: inv.ID, Rate: rate, Date: dated,
		}))
		stored, getErr := invoices.GetByID(ctx, repositories.GetInvoiceByIDRequest{ID: inv.ID, TenantInfo: tenant})
		require.NoError(t, getErr)
		require.True(t, stored.ExchangeRate.Valid)
		assert.True(t, rate.Equal(stored.ExchangeRate.Decimal))
		require.NotNil(t, stored.ExchangeRateDate)
		assert.Equal(t, dated, *stored.ExchangeRateDate)

		err = invoices.StampExchangeRate(ctx, &repositories.StampExchangeRateRequest{
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
		_, insertErr := db.NewInsert().Model(payment).Exec(ctx)
		require.NoError(t, insertErr)

		require.NoError(t, payments.StampExchangeRate(ctx, &repositories.StampExchangeRateRequest{
			TenantInfo: tenant, ID: payment.ID, Rate: rate, Date: dated,
		}))
		stored, getErr := payments.GetByID(ctx, repositories.GetCustomerPaymentByIDRequest{
			ID: payment.ID, TenantInfo: tenant,
		})
		require.NoError(t, getErr)
		require.True(t, stored.ExchangeRate.Valid)
		assert.True(t, rate.Equal(stored.ExchangeRate.Decimal))
		require.NotNil(t, stored.ExchangeRateDate)
		assert.Equal(t, dated, *stored.ExchangeRateDate)

		err = payments.StampExchangeRate(ctx, &repositories.StampExchangeRateRequest{
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
		_, insertErr := db.NewInsert().Model(settlement).Exec(ctx)
		require.NoError(t, insertErr)

		require.NoError(t, payables.StampExchangeRate(ctx, &repositories.StampPayableExchangeRateRequest{
			TenantInfo: tenant, Kind: repositories.PayableCarrier, ID: settlement.ID, Rate: rate, Date: dated,
		}))
		require.NoError(t, payables.StampExchangeRate(ctx, &repositories.StampPayableExchangeRateRequest{
			TenantInfo: tenant, Kind: repositories.PayableCarrier, ID: settlement.ID, Paid: true,
			Rate: paidRate, Date: paidAt,
		}))
		stored, getErr := payables.GetSettlement(ctx, &repositories.GetPayableSettlementRequest{
			TenantInfo: tenant, Kind: repositories.PayableCarrier, ID: settlement.ID,
		})
		require.NoError(t, getErr)
		require.True(t, stored.ExchangeRate.Valid)
		assert.True(t, rate.Equal(stored.ExchangeRate.Decimal))
		assert.Equal(t, dated, *stored.ExchangeRateDate)
		require.True(t, stored.PaidExchangeRate.Valid)
		assert.True(t, paidRate.Equal(stored.PaidExchangeRate.Decimal))
		assert.Equal(t, paidAt, *stored.PaidExchangeRateDate)

		err = payables.StampExchangeRate(ctx, &repositories.StampPayableExchangeRateRequest{
			TenantInfo: tenant, Kind: repositories.PayableDriver, ID: settlement.ID, Rate: rate, Date: dated,
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
		_, insertErr := db.NewInsert().Model(settlement).Exec(ctx)
		require.NoError(t, insertErr)

		require.NoError(t, payables.StampExchangeRate(ctx, &repositories.StampPayableExchangeRateRequest{
			TenantInfo: tenant, Kind: repositories.PayableDriver, ID: settlement.ID, Rate: rate, Date: dated,
		}))
		stored, getErr := payables.GetSettlement(ctx, &repositories.GetPayableSettlementRequest{
			TenantInfo: tenant, Kind: repositories.PayableDriver, ID: settlement.ID,
		})
		require.NoError(t, getErr)
		require.True(t, stored.ExchangeRate.Valid)
		assert.True(t, rate.Equal(stored.ExchangeRate.Decimal))
		assert.False(t, stored.PaidExchangeRate.Valid)

		err = payables.StampExchangeRate(ctx, &repositories.StampPayableExchangeRateRequest{
			TenantInfo: tenant, Kind: repositories.PayableKind("Vendor"), ID: settlement.ID, Rate: rate, Date: dated,
		})
		require.Error(t, err, "an unknown payable kind is refused")
	})
}
