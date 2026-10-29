package invoicevalidation

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/testutil/mocks/fiscalperiodportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestValidatorValidateCreateRejectsHeaderLineTotalMismatch(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	validator := NewValidator(ValidatorParams{Logger: zap.NewNop()})
	entity := &invoice.Invoice{
		OrganizationID:     orgID,
		BusinessUnitID:     buID,
		BillingQueueItemID: pulid.MustNew("bqi_"),
		ShipmentID:         pulid.MustNew("shp_"),
		CustomerID:         pulid.MustNew("cus_"),
		Number:             "INV-1001",
		BillType:           billingqueue.BillTypeInvoice,
		Status:             invoice.StatusDraft,
		PaymentTerm:        invoice.PaymentTermNet30,
		CurrencyCode:       "USD",
		InvoiceDate:        1_700_000_000,
		BillToName:         "Acme Logistics",
		SubtotalAmount:     decimal.NewFromInt(100),
		OtherAmount:        decimal.NewFromInt(150),
		TotalAmount:        decimal.NewFromInt(250),
		SettlementStatus:   invoice.SettlementStatusUnpaid,
		DisputeStatus:      invoice.DisputeStatusNone,
		Lines: []*invoice.InvoiceLine{
			{
				LineNumber:  1,
				Type:        invoice.InvoiceLineTypeFreight,
				Description: "Freight charge",
				Quantity:    decimal.NewFromInt(1),
				UnitPrice:   decimal.NewFromInt(100),
				Amount:      decimal.NewFromInt(100),
			},
			{
				LineNumber:  2,
				Type:        invoice.InvoiceLineTypeAccessorial,
				Description: "Detention",
				Quantity:    decimal.NewFromInt(2),
				UnitPrice:   decimal.RequireFromString("37.50"),
				Amount:      decimal.NewFromInt(75),
			},
		},
	}

	multiErr := validator.ValidateCreate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "otherAmount")
	assertErrorField(t, multiErr, "totalAmount")
}

func TestValidatorValidatePost_BlocksLockedPeriodPosting(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	inv := &invoice.Invoice{
		ID:             pulid.MustNew("inv_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		ShipmentID:     pulid.MustNew("shp_"),
		BillType:       billingqueue.BillTypeInvoice,
		TotalAmount:    decimal.NewFromInt(100),
	}

	accountingRepo := tenantportsmock.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().
		GetByOrgID(mock.Anything, orgID).
		Return(&accountingcontrol.AccountingControl{
			AccountingBasis:           accountingcontrol.AccountingBasisAccrual,
			RevenueRecognitionPolicy:  accountingcontrol.RevenueRecognitionOnInvoicePost,
			LockedPeriodPostingPolicy: accountingcontrol.LockedPeriodPostingPolicyBlockSubledgerAllowManualJe,
			ReconciliationMode:        accountingcontrol.ReconciliationModeDisabled,
			DefaultRevenueAccountID:   pulid.MustNew("gla_"),
			DefaultARAccountID:        pulid.MustNew("gla_"),
		}, nil)

	fiscalPeriodRepo := fiscalperiodportsmock.NewMockFiscalPeriodRepository(t)
	fiscalPeriodRepo.EXPECT().
		GetPeriodByDate(mock.Anything, fiscalperiodports.GetPeriodByDateRequest{
			OrgID: orgID,
			BuID:  buID,
			Date:  1_700_000_000,
		}).
		Return(&fiscalperiod.FiscalPeriod{Status: fiscalperiod.StatusLocked}, nil)

	validator := NewValidator(
		ValidatorParams{
			Logger:           zap.NewNop(),
			AccountingRepo:   accountingRepo,
			FiscalPeriodRepo: fiscalPeriodRepo,
		},
	)

	multiErr := validator.ValidatePost(t.Context(), inv, pagination.TenantInfo{
		OrgID: orgID,
		BuID:  buID,
	}, 1_700_000_000)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "postedAt")
}

func TestValidatorValidatePostRequiresDefaultLedgerAccounts(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	inv := &invoice.Invoice{
		ID:             pulid.MustNew("inv_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		BillType:       billingqueue.BillTypeInvoice,
	}
	accountingRepo := tenantportsmock.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().
		GetByOrgID(mock.Anything, orgID).
		Return(&accountingcontrol.AccountingControl{
			AccountingBasis:          accountingcontrol.AccountingBasisAccrual,
			RevenueRecognitionPolicy: accountingcontrol.RevenueRecognitionOnInvoicePost,
			ReconciliationMode:       accountingcontrol.ReconciliationModeDisabled,
		}, nil)
	fiscalPeriodRepo := fiscalperiodportsmock.NewMockFiscalPeriodRepository(t)
	validator := NewValidator(
		ValidatorParams{
			Logger:           zap.NewNop(),
			AccountingRepo:   accountingRepo,
			FiscalPeriodRepo: fiscalPeriodRepo,
		},
	)

	multiErr := validator.ValidatePost(t.Context(), inv, pagination.TenantInfo{
		OrgID: orgID,
		BuID:  buID,
	}, 1_700_000_000)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "accountingControl")
}

func TestValidatorValidatePostRequiresFiscalPeriod(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	now := int64(1_700_000_000)
	inv := &invoice.Invoice{
		ID:             pulid.MustNew("inv_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		BillType:       billingqueue.BillTypeInvoice,
	}
	accountingRepo := tenantportsmock.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().
		GetByOrgID(mock.Anything, orgID).
		Return(&accountingcontrol.AccountingControl{
			AccountingBasis:          accountingcontrol.AccountingBasisAccrual,
			RevenueRecognitionPolicy: accountingcontrol.RevenueRecognitionOnInvoicePost,
			ReconciliationMode:       accountingcontrol.ReconciliationModeDisabled,
			DefaultRevenueAccountID:  pulid.MustNew("gla_"),
			DefaultARAccountID:       pulid.MustNew("gla_"),
		}, nil)
	fiscalPeriodRepo := fiscalperiodportsmock.NewMockFiscalPeriodRepository(t)
	fiscalPeriodRepo.EXPECT().
		GetPeriodByDate(mock.Anything, fiscalperiodports.GetPeriodByDateRequest{
			OrgID: orgID,
			BuID:  buID,
			Date:  now,
		}).
		Return(nil, errortypes.NewNotFoundError("fiscal period not found"))
	validator := NewValidator(
		ValidatorParams{
			Logger:           zap.NewNop(),
			AccountingRepo:   accountingRepo,
			FiscalPeriodRepo: fiscalPeriodRepo,
		},
	)

	multiErr := validator.ValidatePost(t.Context(), inv, pagination.TenantInfo{
		OrgID: orgID,
		BuID:  buID,
	}, now)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "postedAt")
}

func assertErrorField(t *testing.T, multiErr *errortypes.MultiError, field string) {
	t.Helper()

	require.NotNil(t, multiErr)
	for _, err := range multiErr.Errors {
		if err.Field == field {
			return
		}
	}

	t.Fatalf("expected validation error for field %q, got %#v", field, multiErr.Errors)
}
