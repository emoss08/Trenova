package invoicepostingservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/app/journalposting"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceaction"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceledger"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicevalidation"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks/fiscalperiodportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/journalentryportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestInvoicePostingSourceEvent(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		accountingcontrol.JournalSourceEventInvoicePosted,
		invoiceledger.SourceEvent(billingqueue.BillTypeInvoice),
	)
	assert.Equal(
		t,
		accountingcontrol.JournalSourceEventCreditMemoPosted,
		invoiceledger.SourceEvent(billingqueue.BillTypeCreditMemo),
	)
	assert.Equal(
		t,
		accountingcontrol.JournalSourceEventDebitMemoPosted,
		invoiceledger.SourceEvent(billingqueue.BillTypeDebitMemo),
	)
}

func TestInvoicePostingWorkflow(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	now := int64(1_700_000_000)

	posted := journalposting.ResolveWorkflow(
		&accountingcontrol.AccountingControl{
			JournalPostingMode:      accountingcontrol.JournalPostingModeAutomatic,
			RequireManualJEApproval: true,
		},
		userID,
		now,
	)
	assert.Equal(t, "Posted", posted.EntryStatus)
	assert.Equal(t, "Posted", posted.BatchStatus)
	require.NotNil(t, posted.PostedAt)
	assert.Equal(t, now, *posted.PostedAt)
	assert.Equal(t, userID, posted.PostedByID)
	assert.False(t, posted.RequiresApproval)
	assert.True(t, posted.IsApproved)
	assert.Equal(t, userID, posted.ApprovedByID)
	require.NotNil(t, posted.ApprovedAt)

	pending := journalposting.ResolveWorkflow(
		&accountingcontrol.AccountingControl{
			JournalPostingMode:      accountingcontrol.JournalPostingModeManual,
			RequireManualJEApproval: true,
		},
		userID,
		now,
	)
	assert.Equal(t, "Pending", pending.EntryStatus)
	assert.Equal(t, "Pending", pending.BatchStatus)
	assert.Nil(t, pending.PostedAt)
	assert.True(t, pending.PostedByID.IsNil())
	assert.True(t, pending.RequiresApproval)
	assert.False(t, pending.IsApproved)
	assert.True(t, pending.ApprovedByID.IsNil())
	assert.Nil(t, pending.ApprovedAt)
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

	accountingRepo := tenantportsmock.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().
		GetByOrgID(mock.Anything, orgID).
		Return(&accountingcontrol.AccountingControl{
			AccountingBasis:          accountingcontrol.AccountingBasisAccrual,
			RevenueRecognitionPolicy: accountingcontrol.RevenueRecognitionOnInvoicePost,
			JournalPostingMode:       accountingcontrol.JournalPostingModeAutomatic,
			AutoPostSourceEvents: []accountingcontrol.JournalSourceEventType{
				accountingcontrol.JournalSourceEventCreditMemoPosted,
			},
			DefaultRevenueAccountID: revenueAccountID,
			DefaultARAccountID:      arAccountID,
		}, nil)

	fiscalPeriodRepo := fiscalperiodportsmock.NewMockFiscalPeriodRepository(t)
	fiscalPeriodRepo.EXPECT().
		GetPeriodByDate(mock.Anything, fiscalperiodports.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: now}).
		Return(&fiscalperiod.FiscalPeriod{
			ID:           periodID,
			FiscalYearID: fyID,
			Status:       fiscalperiod.StatusOpen,
		}, nil)

	journalRepo := journalentryportsmock.NewMockJournalPostingRepository(t)
	var lastParams *journalentryports.CreateJournalPostingParams
	journalRepo.EXPECT().
		CreatePosting(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, params journalentryports.CreateJournalPostingParams) error {
			copyParams := params
			copyParams.Lines = append([]journalentryports.JournalPostingLine(nil), params.Lines...)
			lastParams = &copyParams
			return nil
		}).
		Once()
	svc := &Service{
		l:                 zap.NewNop(),
		accountingRepo:    accountingRepo,
		journalRepo:       journalRepo,
		sequenceGenerator: testutil.TestSequenceGenerator{SingleValue: "SEQ-1"},
		fiscalPeriodRepo:  fiscalPeriodRepo,
		validator: invoicevalidation.NewValidator(
			invoicevalidation.ValidatorParams{Logger: zap.NewNop()},
		),
		rec: invoiceaction.NewRecorder(
			invoiceaction.RecorderParams{Logger: zap.NewNop()},
		),
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
	require.NotNil(t, lastParams)
	assert.Equal(
		t,
		accountingcontrol.JournalSourceEventCreditMemoPosted.String(),
		lastParams.SourceEventType,
	)
	require.Len(t, lastParams.Lines, 2)
	assert.Equal(t, revenueAccountID, lastParams.Lines[0].GLAccountID)
	assert.Equal(t, int64(13500), lastParams.Lines[0].DebitAmount)
	assert.Equal(t, arAccountID, lastParams.Lines[1].GLAccountID)
	assert.Equal(t, int64(13500), lastParams.Lines[1].CreditAmount)
}

func TestCreateInvoiceJournalPostingSkipsWhenRecognitionPolicyDoesNotAllowInvoicePosting(
	t *testing.T,
) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	now := int64(1_700_000_000)
	accountingRepo := tenantportsmock.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().
		GetByOrgID(mock.Anything, orgID).
		Return(&accountingcontrol.AccountingControl{
			AccountingBasis:          accountingcontrol.AccountingBasisCash,
			RevenueRecognitionPolicy: accountingcontrol.RevenueRecognitionOnCashReceipt,
			JournalPostingMode:       accountingcontrol.JournalPostingModeAutomatic,
			AutoPostSourceEvents: []accountingcontrol.JournalSourceEventType{
				accountingcontrol.JournalSourceEventCustomerPaymentPosted,
			},
		}, nil)
	journalRepo := journalentryportsmock.NewMockJournalPostingRepository(t)
	svc := &Service{
		l:                 zap.NewNop(),
		accountingRepo:    accountingRepo,
		journalRepo:       journalRepo,
		sequenceGenerator: testutil.TestSequenceGenerator{SingleValue: "SEQ-1"},
		fiscalPeriodRepo:  fiscalperiodportsmock.NewMockFiscalPeriodRepository(t),
		validator: invoicevalidation.NewValidator(
			invoicevalidation.ValidatorParams{Logger: zap.NewNop()},
		),
		rec: invoiceaction.NewRecorder(
			invoiceaction.RecorderParams{Logger: zap.NewNop()},
		),
	}

	err := svc.createInvoiceJournalPosting(
		t.Context(),
		&invoice.Invoice{
			ID:               pulid.MustNew("inv_"),
			OrganizationID:   orgID,
			BusinessUnitID:   buID,
			BillType:         billingqueue.BillTypeInvoice,
			TotalAmountMinor: 10000,
			PostedAt:         &now,
		},
		testutil.NewSessionActor(userID, orgID, buID),
	)

	require.NoError(t, err)
}

func TestCreateInvoiceJournalPostingSkipsWhenAutoPostDisabled(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	now := int64(1_700_000_000)
	accountingRepo := tenantportsmock.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().
		GetByOrgID(mock.Anything, orgID).
		Return(&accountingcontrol.AccountingControl{
			JournalPostingMode: accountingcontrol.JournalPostingModeAutomatic,
			AutoPostSourceEvents: []accountingcontrol.JournalSourceEventType{
				accountingcontrol.JournalSourceEventCustomerPaymentPosted,
			},
		}, nil)
	journalRepo := journalentryportsmock.NewMockJournalPostingRepository(t)
	svc := &Service{
		l:                 zap.NewNop(),
		accountingRepo:    accountingRepo,
		journalRepo:       journalRepo,
		sequenceGenerator: testutil.TestSequenceGenerator{SingleValue: "SEQ-1"},
		fiscalPeriodRepo:  fiscalperiodportsmock.NewMockFiscalPeriodRepository(t),
		validator: invoicevalidation.NewValidator(
			invoicevalidation.ValidatorParams{Logger: zap.NewNop()},
		),
		rec: invoiceaction.NewRecorder(
			invoiceaction.RecorderParams{Logger: zap.NewNop()},
		),
	}

	err := svc.createInvoiceJournalPosting(
		t.Context(),
		&invoice.Invoice{
			ID:               pulid.MustNew("inv_"),
			OrganizationID:   orgID,
			BusinessUnitID:   buID,
			BillType:         billingqueue.BillTypeInvoice,
			TotalAmountMinor: 10000,
			PostedAt:         &now,
		},
		testutil.NewSessionActor(pulid.MustNew("usr_"), orgID, buID),
	)

	require.NoError(t, err)
}

func TestResolveInvoicePostingPeriodUsesNextOpenPeriod(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	fyID := pulid.MustNew("fy_")
	now := int64(1_700_000_000)
	fiscalRepo := fiscalperiodportsmock.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().
		GetPeriodByDate(mock.Anything, fiscalperiodports.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: now}).
		Return(&fiscalperiod.FiscalPeriod{FiscalYearID: fyID, PeriodNumber: 1, Status: fiscalperiod.StatusClosed}, nil)
	fiscalRepo.EXPECT().
		ListByFiscalYearID(mock.Anything, fiscalperiodports.ListByFiscalYearIDRequest{FiscalYearID: fyID, OrgID: orgID, BuID: buID}).
		Return([]*fiscalperiod.FiscalPeriod{{FiscalYearID: fyID, PeriodNumber: 1, Status: fiscalperiod.StatusClosed}, {ID: pulid.MustNew("fp_"), FiscalYearID: fyID, PeriodNumber: 2, Status: fiscalperiod.StatusOpen, StartDate: 1_700_001_000}}, nil)
	resolved, err := journalposting.ResolvePeriod(
		t.Context(),
		fiscalRepo,
		&journalposting.ResolvePeriodRequest{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Date:           now,
			Policy:         accountingcontrol.ClosedPeriodPostingPolicyPostToNextOpen,
			Subject:        "invoice",
		},
	)

	require.NoError(t, err)
	require.NotNil(t, resolved.Period)
	assert.Equal(t, int64(1_700_001_000), resolved.AccountingDate)
}
