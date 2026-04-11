package invoiceservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestResolvePaymentTerm(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		customer *customer.Customer
		control  *tenant.BillingControl
		expected string
	}{
		{
			name: "customer override wins",
			customer: &customer.Customer{
				BillingProfile: &customer.CustomerBillingProfile{
					PaymentTerm: customer.PaymentTermNet15,
				},
			},
			control: &tenant.BillingControl{
				DefaultPaymentTerm: tenant.PaymentTermNet60,
			},
			expected: "Net15",
		},
		{
			name: "tenant fallback used when customer term missing",
			customer: &customer.Customer{
				BillingProfile: &customer.CustomerBillingProfile{},
			},
			control: &tenant.BillingControl{
				DefaultPaymentTerm: tenant.PaymentTermNet45,
			},
			expected: "Net45",
		},
		{
			name:     "empty when neither source has a term",
			customer: &customer.Customer{},
			control:  nil,
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, string(resolvePaymentTerm(tc.customer, tc.control)))
		})
	}
}

func TestBuildInvoiceEntityUsesTenantFallbackAndSignsCreditMemoAmounts(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	item := &billingqueue.BillingQueueItem{
		ID:             pulid.MustNew("bqi_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		ShipmentID:     pulid.MustNew("shp_"),
		BillType:       billingqueue.BillTypeCreditMemo,
		Number:         "CM-1001",
	}
	shp := &shipment.Shipment{
		ID:                  item.ShipmentID,
		FreightChargeAmount: decimal.NewNullDecimal(decimal.NewFromInt(100)),
		OtherChargeAmount:   decimal.NewNullDecimal(decimal.NewFromInt(35)),
		TotalChargeAmount:   decimal.NewNullDecimal(decimal.NewFromInt(135)),
		ProNumber:           "PRO123",
		BOL:                 "BOL123",
		ActualDeliveryDate:  int64Ptr(1_700_000_500),
		AdditionalCharges: []*shipment.AdditionalCharge{
			{
				Unit:   2,
				Amount: decimal.NewFromInt(20),
			},
			nil,
		},
	}
	cus := &customer.Customer{
		ID:           pulid.MustNew("cus_"),
		Name:         "Acme Logistics",
		Code:         "ACME",
		AddressLine1: "100 Main",
		City:         "Nashville",
		PostalCode:   "37201",
		BillingProfile: &customer.CustomerBillingProfile{
			BillingCurrency: "CAD",
		},
	}
	control := &tenant.BillingControl{
		DefaultPaymentTerm: tenant.PaymentTermNet45,
	}

	svc := &Service{l: zap.NewNop()}

	entity := svc.buildInvoiceEntity(item, shp, cus, control)

	require.NotNil(t, entity)
	assert.Equal(t, item.Number, entity.Number)
	assert.Equal(t, "Net45", string(entity.PaymentTerm))
	assert.Equal(t, "CAD", entity.CurrencyCode)
	assert.Equal(t, decimal.NewFromInt(-100), entity.SubtotalAmount)
	assert.Equal(t, int64(-10000), entity.SubtotalAmountMinor)
	assert.Equal(t, decimal.NewFromInt(-35), entity.OtherAmount)
	assert.Equal(t, int64(-3500), entity.OtherAmountMinor)
	assert.Equal(t, decimal.NewFromInt(-135), entity.TotalAmount)
	assert.Equal(t, int64(-13500), entity.TotalAmountMinor)
	assert.Len(t, entity.Lines, 2)
	assert.Equal(t, decimal.NewFromInt(-100), entity.Lines[0].Amount)
	assert.Equal(t, int64(-10000), entity.Lines[0].AmountMinor)
	assert.Equal(t, decimal.NewFromInt(-20), entity.Lines[1].Amount)
	assert.Equal(t, int64(-2000), entity.Lines[1].AmountMinor)
	assert.True(t, entity.Lines[1].UnitPrice.Equal(decimal.NewFromInt(-10)))
	assert.Equal(t, shp.ActualDeliveryDate, entity.ServiceDate)
}

func TestShouldAutoPost(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")

	t.Run("organization auto posting requires both org automation and customer opt-in", func(t *testing.T) {
		t.Parallel()

		billingRepo := mocks.NewMockBillingControlRepository(t)
		billingRepo.EXPECT().GetByOrgID(t.Context(), orgID).Return(&tenant.BillingControl{
			InvoicePostingMode: tenant.InvoicePostingModeAutomaticWhenNoBlockingExceptions,
		}, nil)

		svc := &Service{
			l:           zap.NewNop(),
			billingRepo: billingRepo,
		}

		autoPost := svc.shouldAutoPost(t.Context(), orgID, &customer.Customer{
			BillingProfile: &customer.CustomerBillingProfile{AutoBill: true},
		})

		assert.True(t, autoPost)
	})

	t.Run("customer auto bill cannot loosen organization manual review policy", func(t *testing.T) {
		t.Parallel()

		billingRepo := mocks.NewMockBillingControlRepository(t)
		billingRepo.EXPECT().GetByOrgID(t.Context(), orgID).Return(&tenant.BillingControl{
			InvoicePostingMode: tenant.InvoicePostingModeManualReviewRequired,
		}, nil)

		svc := &Service{
			l:           zap.NewNop(),
			billingRepo: billingRepo,
		}

		autoPost := svc.shouldAutoPost(t.Context(), orgID, &customer.Customer{
			BillingProfile: &customer.CustomerBillingProfile{AutoBill: true},
		})

		assert.False(t, autoPost)
	})

	t.Run("customer fallback still works on tenant not found", func(t *testing.T) {
		t.Parallel()

		billingRepo := mocks.NewMockBillingControlRepository(t)
		billingRepo.EXPECT().GetByOrgID(t.Context(), orgID).
			Return(nil, errortypes.NewNotFoundError("billing control not found"))

		svc := &Service{
			l:           zap.NewNop(),
			billingRepo: billingRepo,
		}

		autoPost := svc.shouldAutoPost(t.Context(), orgID, &customer.Customer{
			BillingProfile: &customer.CustomerBillingProfile{AutoBill: true},
		})

		assert.True(t, autoPost)
	})
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

	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(&tenant.AccountingControl{
		LockedPeriodPostingPolicy: tenant.LockedPeriodPostingPolicyBlockSubledgerAllowManualJe,
		ReconciliationMode:        tenant.ReconciliationModeDisabled,
	}, nil)

	fiscalPeriodRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalPeriodRepo.EXPECT().
		GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{
			OrgID: orgID,
			BuID:  buID,
			Date:  1_700_000_000,
		}).
		Return(&fiscalperiod.FiscalPeriod{Status: fiscalperiod.StatusLocked}, nil)

	validator := &Validator{
		l:                zap.NewNop(),
		accountingRepo:   accountingRepo,
		fiscalPeriodRepo: fiscalPeriodRepo,
	}

	multiErr := validator.ValidatePost(t.Context(), inv, pagination.TenantInfo{
		OrgID: orgID,
		BuID:  buID,
	}, 1_700_000_000)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "postedAt")
}

func TestInvoicePostingSourceEvent(t *testing.T) {
	t.Parallel()

	assert.Equal(t, tenant.JournalSourceEventInvoicePosted, invoicePostingSourceEvent(billingqueue.BillTypeInvoice))
	assert.Equal(t, tenant.JournalSourceEventCreditMemoPosted, invoicePostingSourceEvent(billingqueue.BillTypeCreditMemo))
	assert.Equal(t, tenant.JournalSourceEventDebitMemoPosted, invoicePostingSourceEvent(billingqueue.BillTypeDebitMemo))
}

func TestInvoicePostingWorkflow(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	now := int64(1_700_000_000)

	entryStatus, batchStatus, postedAt, postedByID, requiresApproval, isApproved, approvedByID, approvedAt := invoicePostingWorkflow(&tenant.AccountingControl{
		JournalPostingMode:      tenant.JournalPostingModeAutomatic,
		RequireManualJEApproval: true,
	}, userID, now)
	assert.Equal(t, "Posted", entryStatus)
	assert.Equal(t, "Posted", batchStatus)
	require.NotNil(t, postedAt)
	assert.Equal(t, now, *postedAt)
	assert.Equal(t, userID, postedByID)
	assert.False(t, requiresApproval)
	assert.True(t, isApproved)
	assert.Equal(t, userID, approvedByID)
	require.NotNil(t, approvedAt)

	entryStatus, batchStatus, postedAt, postedByID, requiresApproval, isApproved, approvedByID, approvedAt = invoicePostingWorkflow(&tenant.AccountingControl{
		JournalPostingMode:      tenant.JournalPostingModeManual,
		RequireManualJEApproval: true,
	}, userID, now)
	assert.Equal(t, "Pending", entryStatus)
	assert.Equal(t, "Pending", batchStatus)
	assert.Nil(t, postedAt)
	assert.True(t, postedByID.IsNil())
	assert.True(t, requiresApproval)
	assert.False(t, isApproved)
	assert.True(t, approvedByID.IsNil())
	assert.Nil(t, approvedAt)
}

func TestCreateInvoiceJournalPostingBuildsCreditMemoPolarity(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	fyID := pulid.MustNew("fy_")
	periodID := pulid.MustNew("fp_")
	now := int64(1_700_000_000)
	revenueAccountID := pulid.MustNew("gla_")
	arAccountID := pulid.MustNew("gla_")

	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(&tenant.AccountingControl{
		JournalPostingMode:      tenant.JournalPostingModeAutomatic,
		AutoPostSourceEvents:    []tenant.JournalSourceEventType{tenant.JournalSourceEventCreditMemoPosted},
		DefaultRevenueAccountID: revenueAccountID,
		DefaultARAccountID:      arAccountID,
	}, nil)

	fiscalPeriodRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalPeriodRepo.EXPECT().GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: now}).Return(&fiscalperiod.FiscalPeriod{
		ID:           periodID,
		FiscalYearID: fyID,
		Status:       fiscalperiod.StatusOpen,
	}, nil)

	journalRepo := &fakeInvoiceJournalPostingRepository{}
	svc := &Service{
		l:                 zap.NewNop(),
		accountingRepo:    accountingRepo,
		journalRepo:       journalRepo,
		sequenceGenerator: testutil.TestSequenceGenerator{SingleValue: "SEQ-1"},
		validator:         &Validator{fiscalPeriodRepo: fiscalPeriodRepo},
	}

	err := svc.createInvoiceJournalPosting(t.Context(), &invoice.Invoice{
		ID:               pulid.MustNew("inv_"),
		OrganizationID:   orgID,
		BusinessUnitID:   buID,
		CustomerID:       pulid.MustNew("cus_"),
		Number:           "CM-1001",
		BillType:         billingqueue.BillTypeCreditMemo,
		TotalAmountMinor: -13500,
		PostedAt:         &now,
	}, testutil.NewSessionActor(userID, orgID, buID))

	require.NoError(t, err)
	require.NotNil(t, journalRepo.last)
	assert.Equal(t, tenant.JournalSourceEventCreditMemoPosted.String(), journalRepo.last.SourceEventType)
	require.Len(t, journalRepo.last.Lines, 2)
	assert.Equal(t, revenueAccountID, journalRepo.last.Lines[0].GLAccountID)
	assert.Equal(t, int64(13500), journalRepo.last.Lines[0].DebitAmount)
	assert.Equal(t, arAccountID, journalRepo.last.Lines[1].GLAccountID)
	assert.Equal(t, int64(13500), journalRepo.last.Lines[1].CreditAmount)
}

type fakeInvoiceJournalPostingRepository struct {
	last *repositories.CreateJournalPostingParams
}

func (f *fakeInvoiceJournalPostingRepository) CreatePosting(_ context.Context, params repositories.CreateJournalPostingParams) error {
	copyParams := params
	copyParams.Lines = append([]repositories.JournalPostingLine(nil), params.Lines...)
	f.last = &copyParams
	return nil
}

func int64Ptr(v int64) *int64 {
	return &v
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
