package journalreversalservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/internal/core/domain/journalreversal"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestValidateCreateRejectsInvalidEntryStates(t *testing.T) {
	t.Parallel()

	v := &Validator{}
	err := v.ValidateCreate(t.Context(), &journalentry.Entry{Status: "Draft", IsPosted: false, IsReversal: true, ReversedByID: pulid.MustNew("je_")}, 0, "", "")

	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "Only posted journal entries")
	assert.Contains(t, err.Error(), "cannot be reversed again")
	assert.Contains(t, err.Error(), "already been reversed")
	assert.Contains(t, err.Error(), "Requested accounting date is required")
	assert.Contains(t, err.Error(), "Reason code is required")
	assert.Contains(t, err.Error(), "Reason text is required")
}

func TestResolvePostingPeriodUsesNextOpenPeriod(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	fyID := pulid.MustNew("fy_")
	nextPeriodID := pulid.MustNew("fp_")
	fiscalRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: 100}).Return(&fiscalperiod.FiscalPeriod{FiscalYearID: fyID, PeriodNumber: 1, Status: fiscalperiod.StatusClosed}, nil)
	fiscalRepo.EXPECT().ListByFiscalYearID(mock.Anything, repositories.ListByFiscalYearIDRequest{FiscalYearID: fyID, OrgID: orgID, BuID: buID}).Return([]*fiscalperiod.FiscalPeriod{{FiscalYearID: fyID, PeriodNumber: 1, Status: fiscalperiod.StatusClosed}, {ID: nextPeriodID, FiscalYearID: fyID, PeriodNumber: 2, Status: fiscalperiod.StatusOpen, StartDate: 200}}, nil)
	v := &Validator{fiscalRepo: fiscalRepo}

	period, date, err := v.ResolvePostingPeriod(t.Context(), orgID, buID, 100, &tenant.AccountingControl{JournalReversalPolicy: tenant.JournalReversalPolicyNextOpenPeriod})

	require.Nil(t, err)
	require.NotNil(t, period)
	assert.Equal(t, nextPeriodID, period.ID)
	assert.Equal(t, int64(200), date)
}

func TestResolvePostingPeriodRejectsClosedPeriodWithoutPolicy(t *testing.T) {
	t.Parallel()

	fiscalRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().GetPeriodByDate(mock.Anything, mock.Anything).Return(&fiscalperiod.FiscalPeriod{Status: fiscalperiod.StatusClosed}, nil)
	v := &Validator{fiscalRepo: fiscalRepo}

	period, date, err := v.ResolvePostingPeriod(t.Context(), pulid.MustNew("org_"), pulid.MustNew("bu_"), 100, &tenant.AccountingControl{JournalReversalPolicy: tenant.JournalReversalPolicyDisallow})

	require.Nil(t, period)
	assert.Zero(t, date)
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "Closed periods require next-open-period")
}

func TestValidateActionHelpers(t *testing.T) {
	t.Parallel()

	v := &Validator{}
	assert.Nil(t, v.ValidateApprove(&journalreversal.Reversal{Status: journalreversal.StatusRequested}))
	require.NotNil(t, v.ValidateApprove(&journalreversal.Reversal{Status: journalreversal.StatusPosted}))
	require.NotNil(t, v.ValidateReject(&journalreversal.Reversal{Status: journalreversal.StatusRequested}, ""))
	require.NotNil(t, v.ValidateCancel(&journalreversal.Reversal{Status: journalreversal.StatusPosted}, "ok"))
	assert.Nil(t, v.ValidatePost(&journalreversal.Reversal{Status: journalreversal.StatusApproved}))
	require.NotNil(t, v.ValidatePost(&journalreversal.Reversal{Status: journalreversal.StatusRequested}))
}
