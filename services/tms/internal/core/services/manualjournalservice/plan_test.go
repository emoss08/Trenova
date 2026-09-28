package manualjournalservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/core/domain/manualjournal"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPlanCreateDraftValidatesAndTotalsWithoutIssuingANumber(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	debitID := pulid.MustNew("gla_")
	creditID := pulid.MustNew("gla_")
	periodID := pulid.MustNew("fp_")

	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(&tenant.AccountingControl{
		CurrencyMode:             tenant.CurrencyModeSingleCurrency,
		FunctionalCurrencyCode:   "USD",
		ManualJournalEntryPolicy: tenant.ManualJournalEntryPolicyAllowAll,
	}, nil)
	fiscalRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().
		GetPeriodByDate(mock.Anything, mock.Anything).
		Return(&fiscalperiod.FiscalPeriod{
			ID:           periodID,
			FiscalYearID: pulid.MustNew("fy_"),
			PeriodType:   fiscalperiod.PeriodTypeMonth,
		}, nil)
	glRepo := mocks.NewMockGLAccountRepository(t)
	glRepo.EXPECT().GetByIDs(mock.Anything, mock.Anything).Return([]*glaccount.GLAccount{
		{ID: debitID, Status: domaintypes.StatusActive, AllowManualJE: true},
		{ID: creditID, Status: domaintypes.StatusActive, AllowManualJE: true},
	}, nil)

	repo := newManualJournalRepositoryFake(nil)
	svc := &Service{
		repo:           repo,
		accountingRepo: accountingRepo,
		validator:      &Validator{fiscalRepo: fiscalRepo, glAccountRepo: glRepo},
	}

	planned, err := svc.PlanCreateDraft(t.Context(), &serviceports.CreateManualJournalRequest{
		Description:    "Accrue fuel",
		AccountingDate: 1_700_000_000,
		Lines: []*serviceports.ManualJournalLineInput{
			{GLAccountID: debitID, Description: "Fuel expense", DebitAmount: 2500},
			{GLAccountID: creditID, Description: "Accrued liabilities", CreditAmount: 2500},
		},
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
	}, testutil.NewSessionActor(userID, orgID, buID))

	require.NoError(t, err)
	assert.Empty(t, planned.RequestNumber)
	assert.Equal(t, manualjournal.StatusDraft, planned.Status)
	assert.Equal(t, "USD", planned.CurrencyCode)
	assert.Equal(t, periodID, planned.RequestedFiscalPeriodID)
	assert.Equal(t, int64(2500), planned.TotalDebit)
	assert.Equal(t, int64(2500), planned.TotalCredit)
	assert.Nil(t, repo.current)
}

func TestPlanSubmitSaysWhetherSubmittingApprovesIt(t *testing.T) {
	t.Parallel()

	for name, requireApproval := range map[string]bool{
		"approval required": true,
		"approval off":      false,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			orgID := pulid.MustNew("org_")
			buID := pulid.MustNew("bu_")
			userID := pulid.MustNew("usr_")
			request := approvedRequest(orgID, buID, userID)
			request.Status = manualjournal.StatusDraft
			request.ApprovedAt = nil
			request.ApprovedByID = pulid.Nil

			accountingRepo := mocks.NewMockAccountingControlRepository(t)
			accountingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(
				&tenant.AccountingControl{
					ManualJournalEntryPolicy: tenant.ManualJournalEntryPolicyAllowAll,
					RequireManualJEApproval:  requireApproval,
				}, nil)
			repo := newManualJournalRepositoryFake(request)
			svc := &Service{repo: repo, accountingRepo: accountingRepo, validator: &Validator{}}

			change, err := svc.PlanSubmit(t.Context(), &serviceports.GetManualJournalRequest{
				RequestID:  request.ID,
				TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
			}, testutil.NewSessionActor(userID, orgID, buID))

			require.NoError(t, err)
			assert.Equal(t, manualjournal.StatusDraft, change.Before.Status)
			if requireApproval {
				assert.Equal(t, manualjournal.StatusPendingApproval, change.After.Status)
				assert.True(t, change.After.ApprovedByID.IsNil())
			} else {
				assert.Equal(t, manualjournal.StatusApproved, change.After.Status)
				assert.Equal(t, userID, change.After.ApprovedByID)
			}
			assert.Nil(t, repo.updated)
		})
	}
}

func TestPlanPostShowsTheEntryItWouldBookWithoutBookingIt(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	periodID := pulid.MustNew("fp_")
	request := approvedRequest(orgID, buID, userID)

	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(&tenant.AccountingControl{
		ClosedPeriodPostingPolicy: tenant.ClosedPeriodPostingPolicyRequireReopen,
	}, nil)
	fiscalRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().
		GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{
			OrgID: orgID,
			BuID:  buID,
			Date:  request.AccountingDate,
		}).
		Return(&fiscalperiod.FiscalPeriod{
			ID:           periodID,
			FiscalYearID: pulid.MustNew("fy_"),
			PeriodType:   fiscalperiod.PeriodTypeMonth,
			Status:       fiscalperiod.StatusOpen,
		}, nil)
	repo := newManualJournalRepositoryFake(request)
	journalRepo := newJournalPostingRepositoryFake()
	svc := &Service{
		repo:           repo,
		journalRepo:    journalRepo,
		accountingRepo: accountingRepo,
		validator:      &Validator{fiscalRepo: fiscalRepo},
	}

	change, err := svc.PlanPost(t.Context(), &serviceports.GetManualJournalRequest{
		RequestID:  request.ID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
	}, testutil.NewSessionActor(userID, orgID, buID))

	require.NoError(t, err)
	assert.Equal(t, manualjournal.StatusApproved, change.Before.Status)
	assert.Equal(t, manualjournal.StatusPosted, change.After.Status)
	require.NotNil(t, change.Journal)
	assert.Equal(t, periodID, change.Journal.FiscalPeriodID)
	assert.Equal(t, request.AccountingDate, change.Journal.AccountingDate)
	require.Len(t, change.Journal.Lines, 2)
	assert.Equal(t, int64(1000), change.Journal.Lines[0].DebitMinor)
	assert.Equal(t, int64(1000), change.Journal.Lines[1].CreditMinor)
	assert.Nil(t, journalRepo.last)
	assert.Nil(t, repo.updated)
}

func TestPlanCancelRefusesWithoutAReason(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	request := approvedRequest(orgID, buID, userID)
	svc := &Service{repo: newManualJournalRepositoryFake(request), validator: &Validator{}}

	_, err := svc.PlanCancel(t.Context(), &serviceports.CancelManualJournalRequest{
		RequestID:  request.ID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
	}, testutil.NewSessionActor(userID, orgID, buID))

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
}
