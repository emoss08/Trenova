package journalreviewservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var (
	orgID      = pulid.MustNew("org_")
	buID       = pulid.MustNew("bu_")
	tenantInfo = pagination.TenantInfo{OrgID: orgID, BuID: buID}
	reviewerID = pulid.MustNew("usr_")
)

type fakeReview struct {
	entries  map[pulid.ID]*journalentry.JournalEntry
	approved []*repositories.ApproveJournalEntryParams
	posted   []*repositories.PostJournalEntryParams
}

func (f *fakeReview) LockEntry(
	_ context.Context,
	req repositories.LockJournalEntryRequest,
) (*journalentry.JournalEntry, error) {
	entry, ok := f.entries[req.EntryID]
	if !ok || req.TenantInfo != tenantInfo {
		return nil, errortypes.NewNotFoundError("journal entry not found")
	}
	copied := *entry
	return &copied, nil
}

func (f *fakeReview) ApproveEntry(_ context.Context, params *repositories.ApproveJournalEntryParams) error {
	f.approved = append(f.approved, params)
	return nil
}

func (f *fakeReview) PostEntry(_ context.Context, params *repositories.PostJournalEntryParams) error {
	f.posted = append(f.posted, params)
	return nil
}

func (f *fakeReview) ListConnection(
	context.Context,
	*repositories.ListJournalReviewRequest,
) (*pagination.CursorListResult[*journalentry.JournalEntry], error) {
	return &pagination.CursorListResult[*journalentry.JournalEntry]{}, nil
}

func (f *fakeReview) Summarize(
	context.Context,
	pagination.TenantInfo,
) (*repositories.JournalReviewSummary, error) {
	oldest := int64(1_000)
	return &repositories.JournalReviewSummary{
		AwaitingApproval:     2,
		ReadyToPost:          1,
		OldestAccountingDate: &oldest,
	}, nil
}

func entry(number string, status journalentry.Status, approved bool) *journalentry.JournalEntry {
	debit, credit := pulid.MustNew("gla_"), pulid.MustNew("gla_")
	return &journalentry.JournalEntry{
		ID:               pulid.MustNew("je_"),
		OrganizationID:   orgID,
		BusinessUnitID:   buID,
		BatchID:          pulid.MustNew("jb_"),
		EntryNumber:      number,
		Status:           status,
		RequiresApproval: !approved,
		IsApproved:       approved,
		AccountingDate:   1_000,
		TotalDebit:       500,
		TotalCredit:      500,
		Version:          3,
		Lines: []*journalentry.JournalEntryLine{
			{ID: pulid.MustNew("jel_"), GLAccountID: debit, LineNumber: 1, DebitAmount: 500, NetAmount: 500},
			{ID: pulid.MustNew("jel_"), GLAccountID: credit, LineNumber: 2, CreditAmount: 500, NetAmount: -500},
		},
	}
}

func newService(
	t *testing.T,
	repo *fakeReview,
	control *tenant.AccountingControl,
	periods *mocks.MockFiscalPeriodRepository,
) *Service {
	t.Helper()
	controls := mocks.NewMockAccountingControlRepository(t)
	controls.EXPECT().GetByOrgID(mock.Anything, orgID).Return(control, nil).Maybe()
	if periods == nil {
		periods = mocks.NewMockFiscalPeriodRepository(t)
	}
	return &Service{
		l:        zap.NewNop(),
		db:       dbtest.NopConnection{},
		repo:     repo,
		controls: controls,
		periods:  periods,
		now:      func() int64 { return 9_000 },
	}
}

func request(ids ...pulid.ID) *serviceports.JournalReviewRequest {
	return &serviceports.JournalReviewRequest{TenantInfo: tenantInfo, EntryIDs: ids}
}

func actor() *serviceports.RequestActor {
	return &serviceports.RequestActor{UserID: reviewerID}
}

func openPeriod(t *testing.T, date int64) (*mocks.MockFiscalPeriodRepository, *fiscalperiod.FiscalPeriod) {
	t.Helper()
	period := &fiscalperiod.FiscalPeriod{
		ID:           pulid.MustNew("fp_"),
		FiscalYearID: pulid.MustNew("fy_"),
		PeriodNumber: 3,
		Status:       fiscalperiod.StatusOpen,
	}
	periods := mocks.NewMockFiscalPeriodRepository(t)
	periods.EXPECT().
		GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: date}).
		Return(period, nil).
		Maybe()
	return periods, period
}

func TestApproveApprovesPendingEntriesAndRefusesTheRest(t *testing.T) {
	t.Parallel()

	pending := entry("JE-1", journalentry.StatusPending, false)
	posted := entry("JE-2", journalentry.StatusPosted, true)
	repo := &fakeReview{entries: map[pulid.ID]*journalentry.JournalEntry{
		pending.ID: pending,
		posted.ID:  posted,
	}}
	missing := pulid.MustNew("je_")

	result, err := newService(t, repo, &tenant.AccountingControl{}, nil).
		Approve(t.Context(), request(pending.ID, posted.ID, pending.ID, missing), actor())
	require.NoError(t, err)

	require.Len(t, result.Outcomes, 3)
	assert.Equal(t, 1, result.Changed)
	assert.Equal(t, 2, result.Failed)
	assert.Equal(t, "Approved", result.Outcomes[0].Status)
	assert.True(t, result.Outcomes[0].Changed)
	assert.Contains(t, result.Outcomes[1].Error, "already approved")
	assert.Equal(t, "Journal entry not found", result.Outcomes[2].Error)

	require.Len(t, repo.approved, 1)
	assert.Equal(t, pending.ID, repo.approved[0].EntryID)
	assert.Equal(t, pending.BatchID, repo.approved[0].BatchID)
	assert.Equal(t, int64(3), repo.approved[0].Version)
	assert.Equal(t, reviewerID, repo.approved[0].ApprovedByID)
	assert.Equal(t, int64(9_000), repo.approved[0].ApprovedAt)
}

func TestPostWritesAnApprovedEntryInItsOwnOpenPeriod(t *testing.T) {
	t.Parallel()

	approved := entry("JE-7", journalentry.StatusApproved, true)
	repo := &fakeReview{entries: map[pulid.ID]*journalentry.JournalEntry{approved.ID: approved}}
	periods, period := openPeriod(t, approved.AccountingDate)

	result, err := newService(t, repo, &tenant.AccountingControl{}, periods).
		Post(t.Context(), request(approved.ID), actor())
	require.NoError(t, err)

	assert.Equal(t, 1, result.Changed)
	assert.Equal(t, "Posted", result.Outcomes[0].Status)
	require.Len(t, repo.posted, 1)
	posted := repo.posted[0]
	assert.Equal(t, period.ID, posted.FiscalPeriodID)
	assert.Equal(t, period.FiscalYearID, posted.FiscalYearID)
	assert.Equal(t, approved.AccountingDate, posted.AccountingDate)
	assert.Equal(t, reviewerID, posted.PostedByID)
	assert.Equal(t, int64(9_000), posted.PostedAt)
	assert.Equal(t, int64(3), posted.Version)
	require.Len(t, posted.Lines, 2)
	assert.Equal(t, int64(500), posted.Lines[0].DebitAmount)
	assert.Equal(t, int64(-500), posted.Lines[1].NetAmount)
}

func TestPostMovesAnEntryOutOfAPeriodClosedSinceItWasWritten(t *testing.T) {
	t.Parallel()

	approved := entry("JE-8", journalentry.StatusApproved, true)
	repo := &fakeReview{entries: map[pulid.ID]*journalentry.JournalEntry{approved.ID: approved}}
	closed := &fiscalperiod.FiscalPeriod{
		ID:           pulid.MustNew("fp_"),
		FiscalYearID: pulid.MustNew("fy_"),
		PeriodNumber: 3,
		Status:       fiscalperiod.StatusClosed,
	}
	next := &fiscalperiod.FiscalPeriod{
		ID:           pulid.MustNew("fp_"),
		FiscalYearID: closed.FiscalYearID,
		PeriodNumber: 4,
		Status:       fiscalperiod.StatusOpen,
		StartDate:    4_000,
	}
	periods := mocks.NewMockFiscalPeriodRepository(t)
	periods.EXPECT().GetPeriodByDate(mock.Anything, mock.Anything).Return(closed, nil)
	periods.EXPECT().
		ListByFiscalYearID(mock.Anything, mock.Anything).
		Return([]*fiscalperiod.FiscalPeriod{closed, next}, nil).
		Once()

	moving := &tenant.AccountingControl{ClosedPeriodPostingPolicy: tenant.ClosedPeriodPostingPolicyPostToNextOpen}
	result, err := newService(t, repo, moving, periods).Post(t.Context(), request(approved.ID), actor())
	require.NoError(t, err)
	require.Len(t, repo.posted, 1)
	assert.Equal(t, next.ID, repo.posted[0].FiscalPeriodID)
	assert.Equal(t, int64(4_000), repo.posted[0].AccountingDate)
	assert.Equal(t, 1, result.Changed)

	strict := &tenant.AccountingControl{ClosedPeriodPostingPolicy: tenant.ClosedPeriodPostingPolicyRequireReopen}
	refused := &fakeReview{entries: map[pulid.ID]*journalentry.JournalEntry{approved.ID: approved}}
	result, err = newService(t, refused, strict, periods).Post(t.Context(), request(approved.ID), actor())
	require.NoError(t, err)
	assert.Empty(t, refused.posted)
	assert.Equal(t, 1, result.Failed)
	assert.Contains(t, result.Outcomes[0].Error, "closed fiscal period")
}

func TestPostRefusesEntriesThatAreNotReadyOrDoNotBalance(t *testing.T) {
	t.Parallel()

	needsApproval := entry("JE-3", journalentry.StatusPending, false)
	unbalanced := entry("JE-4", journalentry.StatusApproved, true)
	unbalanced.Lines[1].CreditAmount = 499
	repo := &fakeReview{entries: map[pulid.ID]*journalentry.JournalEntry{
		needsApproval.ID: needsApproval,
		unbalanced.ID:    unbalanced,
	}}

	result, err := newService(t, repo, &tenant.AccountingControl{}, nil).
		Post(t.Context(), request(needsApproval.ID, unbalanced.ID), actor())
	require.NoError(t, err)

	assert.Empty(t, repo.posted)
	assert.Equal(t, 2, result.Failed)
	assert.Contains(t, result.Outcomes[0].Error, "needs approval")
	assert.Contains(t, result.Outcomes[1].Error, "does not balance")
}

func TestReviewValidatesTheRequest(t *testing.T) {
	t.Parallel()

	svc := newService(t, &fakeReview{}, &tenant.AccountingControl{}, nil)

	_, err := svc.Approve(t.Context(), request(), actor())
	require.Error(t, err)

	tooMany := make([]pulid.ID, 0, serviceports.MaxJournalReviewEntries+1)
	for range serviceports.MaxJournalReviewEntries + 1 {
		tooMany = append(tooMany, pulid.MustNew("je_"))
	}
	_, err = svc.Post(t.Context(), request(tooMany...), actor())
	require.Error(t, err)

	_, err = svc.Post(t.Context(), request(pulid.MustNew("je_")), nil)
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
}

func TestSummaryReportsTheQueueAndThePostingMode(t *testing.T) {
	t.Parallel()

	svc := newService(t, &fakeReview{}, &tenant.AccountingControl{
		JournalPostingMode:      tenant.JournalPostingModeManual,
		RequireManualJEApproval: true,
	}, nil)

	summary, err := svc.Summary(t.Context(), tenantInfo)
	require.NoError(t, err)
	assert.Equal(t, 2, summary.AwaitingApproval)
	assert.Equal(t, 1, summary.ReadyToPost)
	require.NotNil(t, summary.OldestAccountingDate)
	assert.Equal(t, tenant.JournalPostingModeManual, summary.PostingMode)
	assert.True(t, summary.RequiresApproval)
}
