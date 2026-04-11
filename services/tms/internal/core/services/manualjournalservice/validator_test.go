package manualjournalservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/core/domain/manualjournal"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestValidateDraftUpsertDefaultsCurrencyAndAssignsPeriod(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	periodID := pulid.MustNew("fp_")
	fyID := pulid.MustNew("fy_")
	accountID1 := pulid.MustNew("gla_")
	accountID2 := pulid.MustNew("gla_")
	entity := &manualjournal.Request{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Description:    "Accrual",
		AccountingDate: 1_700_000_000,
		Lines: []*manualjournal.Line{
			{GLAccountID: accountID1, Description: "Debit", DebitAmount: 100},
			{GLAccountID: accountID2, Description: "Credit", CreditAmount: 100},
		},
	}

	fiscalRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: entity.AccountingDate}).Return(&fiscalperiod.FiscalPeriod{
		ID:           periodID,
		FiscalYearID: fyID,
		PeriodType:   fiscalperiod.PeriodTypeAdjusting,
	}, nil)
	glRepo := mocks.NewMockGLAccountRepository(t)
	glRepo.EXPECT().GetByIDs(mock.Anything, repositories.GetGLAccountsByIDsRequest{TenantInfo: repositoriesTenantInfo(orgID, buID), GLAccountIDs: []pulid.ID{accountID1, accountID2}}).Return([]*glaccount.GLAccount{
		{ID: accountID1, Status: domaintypes.StatusActive, AllowManualJE: true},
		{ID: accountID2, Status: domaintypes.StatusActive, AllowManualJE: true},
	}, nil)

	v := &Validator{fiscalRepo: fiscalRepo, glAccountRepo: glRepo}
	err := v.ValidateDraftUpsert(t.Context(), entity, &tenant.AccountingControl{
		CurrencyMode:             tenant.CurrencyModeSingleCurrency,
		FunctionalCurrencyCode:   "USD",
		ManualJournalEntryPolicy: tenant.ManualJournalEntryPolicyAdjustmentOnly,
	})

	require.Nil(t, err)
	assert.Equal(t, "USD", entity.CurrencyCode)
	assert.Equal(t, fyID, entity.RequestedFiscalYearID)
	assert.Equal(t, periodID, entity.RequestedFiscalPeriodID)
}

func TestValidateDraftUpsertBlocksNonAdjustingPeriodWhenPolicyRequiresIt(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	accountID1 := pulid.MustNew("gla_")
	accountID2 := pulid.MustNew("gla_")
	entity := &manualjournal.Request{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Description:    "Accrual",
		AccountingDate: 1_700_000_000,
		CurrencyCode:   "USD",
		Lines: []*manualjournal.Line{
			{GLAccountID: accountID1, Description: "Debit", DebitAmount: 100},
			{GLAccountID: accountID2, Description: "Credit", CreditAmount: 100},
		},
	}

	fiscalRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().GetPeriodByDate(mock.Anything, mock.Anything).Return(&fiscalperiod.FiscalPeriod{
		ID:           pulid.MustNew("fp_"),
		FiscalYearID: pulid.MustNew("fy_"),
		PeriodType:   fiscalperiod.PeriodTypeMonth,
	}, nil)
	glRepo := mocks.NewMockGLAccountRepository(t)
	glRepo.EXPECT().GetByIDs(mock.Anything, mock.Anything).Return([]*glaccount.GLAccount{
		{ID: accountID1, Status: domaintypes.StatusActive, AllowManualJE: true},
		{ID: accountID2, Status: domaintypes.StatusActive, AllowManualJE: true},
	}, nil)

	v := &Validator{fiscalRepo: fiscalRepo, glAccountRepo: glRepo}
	err := v.ValidateDraftUpsert(t.Context(), entity, &tenant.AccountingControl{
		ManualJournalEntryPolicy: tenant.ManualJournalEntryPolicyAdjustmentOnly,
	})

	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "adjusting periods")
}

func TestValidateSubmitRejectsUnbalancedRequest(t *testing.T) {
	t.Parallel()

	v := &Validator{}
	err := v.ValidateSubmit(&manualjournal.Request{
		Status:      manualjournal.StatusDraft,
		TotalDebit:  100,
		TotalCredit: 50,
		Lines: []*manualjournal.Line{
			{DebitAmount: 100},
			{CreditAmount: 50},
		},
	}, &tenant.AccountingControl{ManualJournalEntryPolicy: tenant.ManualJournalEntryPolicyAllowAll})

	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "balanced")
}

func TestValidateRejectRequiresReason(t *testing.T) {
	t.Parallel()

	v := &Validator{}
	err := v.ValidateReject("", &manualjournal.Request{Status: manualjournal.StatusPendingApproval})

	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "Rejection reason")
}

func TestValidateCancelRequiresAllowedStatus(t *testing.T) {
	t.Parallel()

	v := &Validator{}
	err := v.ValidateCancel(&manualjournal.Request{Status: manualjournal.StatusPosted}, "cancel")

	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "cannot be cancelled")
}

func TestValidatePostRequiresApprovedStatus(t *testing.T) {
	t.Parallel()

	v := &Validator{}
	err := v.ValidatePost(&manualjournal.Request{Status: manualjournal.StatusPendingApproval})

	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "approved")
}

func repositoriesTenantInfo(orgID, buID pulid.ID) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: orgID, BuID: buID}
}
