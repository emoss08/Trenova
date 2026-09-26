package journalreversalservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/journalreversal"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPlanCreateShowsTheReversalWithoutRequestingIt(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	entryID := pulid.MustNew("je_")
	periodID := pulid.MustNew("fp_")
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}

	entryRepo := mocks.NewMockJournalEntryRepository(t)
	entryRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetJournalEntryByIDRequest{
			ID:         entryID,
			TenantInfo: tenantInfo,
		}).
		Return(postedEntry(entryID, orgID, buID), nil)
	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(&tenant.AccountingControl{
		JournalReversalPolicy:   tenant.JournalReversalPolicyNextOpenPeriod,
		RequireManualJEApproval: false,
	}, nil)
	fiscalRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().
		GetPeriodByDate(mock.Anything, mock.Anything).
		Return(&fiscalperiod.FiscalPeriod{
			ID:           periodID,
			FiscalYearID: pulid.MustNew("fy_"),
			Status:       fiscalperiod.StatusOpen,
		}, nil)
	reversalRepo := mocks.NewMockJournalReversalRepository(t)

	svc := &Service{
		journalEntryRepo:    entryRepo,
		journalReversalRepo: reversalRepo,
		accountingRepo:      accountingRepo,
		validator:           &Validator{fiscalRepo: fiscalRepo},
	}

	change, err := svc.PlanCreate(t.Context(), &serviceports.CreateJournalReversalRequest{
		OriginalJournalEntryID:  entryID,
		RequestedAccountingDate: 1_700_000_000,
		ReasonCode:              "ERR",
		ReasonText:              "Posted to the wrong customer",
		TenantInfo:              tenantInfo,
	}, testutil.NewSessionActor(userID, orgID, buID))

	require.NoError(t, err)
	assert.Nil(t, change.Before)
	assert.True(t, change.After.ID.IsNil())
	assert.Equal(t, journalreversal.StatusApproved, change.After.Status)
	assert.Equal(t, userID, change.After.ApprovedByID)
	assert.Equal(t, periodID, change.After.ResolvedFiscalPeriodID)
	require.NotNil(t, change.OriginalEntry)
	require.NotNil(t, change.Journal)
	require.Len(t, change.Journal.Lines, 2)
	assert.Equal(t, int64(1000), change.Journal.Lines[0].CreditMinor)
	assert.Equal(t, int64(1000), change.Journal.Lines[1].DebitMinor)
}

func TestPlanPostFlipsTheOriginalLinesWithoutBookingThem(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	entryID := pulid.MustNew("je_")
	reversalID := pulid.MustNew("jrev_")
	periodID := pulid.MustNew("fp_")
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}

	reversalRepo := mocks.NewMockJournalReversalRepository(t)
	reversalRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetJournalReversalByIDRequest{
			ID:         reversalID,
			TenantInfo: tenantInfo,
		}).
		Return(&journalreversal.Reversal{
			ID:                      reversalID,
			OrganizationID:          orgID,
			BusinessUnitID:          buID,
			OriginalJournalEntryID:  entryID,
			Status:                  journalreversal.StatusApproved,
			RequestedAccountingDate: 1_700_000_000,
			ResolvedFiscalPeriodID:  periodID,
		}, nil)
	entryRepo := mocks.NewMockJournalEntryRepository(t)
	entryRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(postedEntry(entryID, orgID, buID), nil)

	svc := &Service{journalEntryRepo: entryRepo, journalReversalRepo: reversalRepo}

	change, err := svc.PlanPost(t.Context(), &serviceports.GetJournalReversalRequest{
		ReversalID: reversalID,
		TenantInfo: tenantInfo,
	}, testutil.NewSessionActor(userID, orgID, buID))

	require.NoError(t, err)
	assert.Equal(t, journalreversal.StatusApproved, change.Before.Status)
	assert.Equal(t, journalreversal.StatusPosted, change.After.Status)
	assert.Equal(t, periodID, change.Journal.FiscalPeriodID)
	require.Len(t, change.Journal.Lines, 2)
	assert.Equal(t, int64(1000), change.Journal.Lines[0].CreditMinor)
}
