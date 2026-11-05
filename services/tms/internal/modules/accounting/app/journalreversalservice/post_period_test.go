package journalreversalservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalreversal"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalreversalports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/fiscalperiodportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/journalentryportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/journalreversalportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	reversalDate      = 100
	nextPeriodStarted = 900
)

type reversalWorld struct {
	tenantInfo pagination.TenantInfo
	actor      *permissionports.RequestActor
	reversal   *journalreversal.Reversal
	control    *accountingcontrol.AccountingControl
	period     *fiscalperiod.FiscalPeriod
	later      []*fiscalperiod.FiscalPeriod
	posted     []journalentryports.CreateJournalPostingParams
	marked     []journalentryports.MarkJournalEntryReversedRequest
	saved      []*journalreversal.Reversal
	held       []pulid.ID
	closing    bool
	svc        *Service
}

func newReversalWorld(t *testing.T) *reversalWorld {
	t.Helper()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	entryID := pulid.MustNew("je_")
	w := &reversalWorld{
		tenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
		actor:      testutil.NewSessionActor(userID, orgID, buID),
		control:    &accountingcontrol.AccountingControl{FunctionalCurrencyCode: "USD"},
		period: &fiscalperiod.FiscalPeriod{
			ID:           pulid.MustNew("fp_"),
			FiscalYearID: pulid.MustNew("fy_"),
			PeriodNumber: 1,
			Status:       fiscalperiod.StatusOpen,
		},
	}
	w.reversal = &journalreversal.Reversal{
		ID:                      pulid.MustNew("jrev_"),
		OrganizationID:          orgID,
		BusinessUnitID:          buID,
		OriginalJournalEntryID:  entryID,
		Status:                  journalreversal.StatusApproved,
		RequestedAccountingDate: reversalDate,
		ResolvedFiscalYearID:    w.period.FiscalYearID,
		ResolvedFiscalPeriodID:  w.period.ID,
		ReasonCode:              "ERR",
		ReasonText:              "reverse me",
		RequestedByID:           userID,
	}

	reversalRepo := journalreversalportsmock.NewMockJournalReversalRepository(t)
	reversalRepo.EXPECT().GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(context.Context, journalreversalports.GetJournalReversalByIDRequest) (*journalreversal.Reversal, error) {
			stored := *w.reversal
			return &stored, nil
		}).
		Maybe()
	reversalRepo.EXPECT().Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *journalreversal.Reversal) (*journalreversal.Reversal, error) {
			stored := *entity
			w.saved = append(w.saved, &stored)
			return &stored, nil
		}).
		Maybe()
	entryRepo := journalentryportsmock.NewMockJournalEntryRepository(t)
	entryRepo.EXPECT().GetByID(mock.Anything, mock.Anything).
		Return(postedEntry(entryID, orgID, buID), nil).
		Maybe()
	entryRepo.EXPECT().MarkReversed(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req journalentryports.MarkJournalEntryReversedRequest) error {
			w.marked = append(w.marked, req)
			return nil
		}).
		Maybe()
	postingRepo := journalentryportsmock.NewMockJournalPostingRepository(t)
	postingRepo.EXPECT().CreatePosting(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, params journalentryports.CreateJournalPostingParams) error {
			w.posted = append(w.posted, params)
			return nil
		}).
		Maybe()
	accountingRepo := tenantportsmock.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(w.control, nil).Maybe()
	fiscalRepo := fiscalperiodportsmock.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().GetPeriodByDate(mock.Anything, mock.Anything).Return(w.period, nil).Maybe()
	fiscalRepo.EXPECT().ListByFiscalYearID(mock.Anything, mock.Anything).
		Return(nil, nil).
		Maybe()
	fiscalRepo.EXPECT().GetByIDForUpdate(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			req fiscalperiodports.GetFiscalPeriodByIDRequest,
		) (*fiscalperiod.FiscalPeriod, error) {
			w.held = append(w.held, req.ID)
			held := *w.period
			if w.closing {
				held.Status = fiscalperiod.StatusClosed
			}
			return &held, nil
		}).
		Maybe()

	w.svc = &Service{
		db:                  dbtest.NopConnection{},
		journalEntryRepo:    entryRepo,
		journalReversalRepo: reversalRepo,
		journalPostingRepo:  postingRepo,
		accountingRepo:      accountingRepo,
		validator:           &Validator{fiscalRepo: fiscalRepo},
		sequenceGenerator:   testutil.TestSequenceGenerator{SingleValue: "SEQ-1"},
		auditService:        &mocks.NoopAuditService{},
	}

	return w
}

func (w *reversalWorld) request() *journalreversalports.GetJournalReversalRequest {
	return &journalreversalports.GetJournalReversalRequest{
		ReversalID: w.reversal.ID,
		TenantInfo: w.tenantInfo,
	}
}

func (w *reversalWorld) closeAndAllowNextOpen(
	t *testing.T,
) *fiscalperiod.FiscalPeriod {
	t.Helper()

	next := &fiscalperiod.FiscalPeriod{
		ID:           pulid.MustNew("fp_"),
		FiscalYearID: w.period.FiscalYearID,
		PeriodNumber: 2,
		StartDate:    nextPeriodStarted,
		Status:       fiscalperiod.StatusOpen,
	}
	w.period.Status = fiscalperiod.StatusClosed
	w.control.JournalReversalPolicy = accountingcontrol.JournalReversalPolicyNextOpenPeriod

	fiscalRepo := fiscalperiodportsmock.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().GetPeriodByDate(mock.Anything, mock.Anything).Return(w.period, nil)
	fiscalRepo.EXPECT().ListByFiscalYearID(mock.Anything, mock.Anything).
		Return([]*fiscalperiod.FiscalPeriod{next}, nil)
	fiscalRepo.EXPECT().GetByIDForUpdate(mock.Anything, mock.Anything).Return(next, nil).Maybe()
	w.svc.validator = &Validator{fiscalRepo: fiscalRepo}

	return next
}

func TestPostRefusesAPeriodThatClosedAfterTheReversalWasCreated(t *testing.T) {
	t.Parallel()

	w := newReversalWorld(t)
	w.period.Status = fiscalperiod.StatusClosed

	result, err := w.svc.Post(t.Context(), w.request(), w.actor)

	require.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Closed periods require next-open-period reversal policy")
	assert.Empty(t, w.posted)
	assert.Empty(t, w.marked)
	assert.Empty(t, w.saved)
}

func TestPostRefusesAPeriodThatClosesBeforeItsTransactionHoldsThePeriod(t *testing.T) {
	t.Parallel()

	w := newReversalWorld(t)
	w.closing = true

	result, err := w.svc.Post(t.Context(), w.request(), w.actor)

	require.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "closed while this reversal was posting")
	assert.Empty(t, w.posted)
	assert.Empty(t, w.marked)
	assert.Empty(t, w.saved)
}

func TestPostHoldsThePeriodItPostsTo(t *testing.T) {
	t.Parallel()

	w := newReversalWorld(t)

	_, err := w.svc.Post(t.Context(), w.request(), w.actor)

	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{w.period.ID}, w.held)
	assert.Len(t, w.posted, 1)
}

func TestPostRefusesAPeriodThatWentInactiveAfterTheReversalWasCreated(t *testing.T) {
	t.Parallel()

	w := newReversalWorld(t)
	w.period.Status = fiscalperiod.StatusInactive

	result, err := w.svc.Post(t.Context(), w.request(), w.actor)

	require.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "inactive fiscal period")
	assert.Empty(t, w.posted)
}

func TestPlanPostShowsTheRecheckedPeriodAndDate(t *testing.T) {
	t.Parallel()

	w := newReversalWorld(t)
	next := w.closeAndAllowNextOpen(t)

	change, err := w.svc.PlanPost(t.Context(), w.request(), w.actor)

	require.NoError(t, err)
	assert.Equal(t, next.ID, change.Journal.FiscalPeriodID)
	assert.Equal(t, int64(nextPeriodStarted), change.Journal.AccountingDate)
	assert.Equal(t, next.ID, change.After.ResolvedFiscalPeriodID)
	assert.Equal(t, int64(nextPeriodStarted), change.After.RequestedAccountingDate)
	assert.Equal(t, w.period.ID, change.Before.ResolvedFiscalPeriodID)
	assert.Equal(t, int64(reversalDate), change.Before.RequestedAccountingDate)
}

func TestPostWritesTheRecheckedPeriodAndDateEverywhere(t *testing.T) {
	t.Parallel()

	w := newReversalWorld(t)
	next := w.closeAndAllowNextOpen(t)

	result, err := w.svc.Post(t.Context(), w.request(), w.actor)

	require.NoError(t, err)
	require.Len(t, w.posted, 1)
	assert.Equal(t, next.ID, w.posted[0].FiscalPeriodID)
	assert.Equal(t, next.FiscalYearID, w.posted[0].FiscalYearID)
	assert.Equal(t, int64(nextPeriodStarted), w.posted[0].AccountingDate)
	require.Len(t, w.marked, 1)
	assert.Equal(t, int64(nextPeriodStarted), w.marked[0].ReversalDate)
	assert.Equal(t, next.ID, result.ResolvedFiscalPeriodID)
	assert.Equal(t, int64(nextPeriodStarted), result.RequestedAccountingDate)
}

func TestPostKeepsTheResolvedPeriodWhenItStillAcceptsPostings(t *testing.T) {
	t.Parallel()

	for _, status := range []fiscalperiod.Status{
		fiscalperiod.StatusOpen,
		fiscalperiod.StatusLocked,
	} {
		t.Run(string(status), func(t *testing.T) {
			t.Parallel()

			w := newReversalWorld(t)
			w.period.Status = status

			_, err := w.svc.Post(t.Context(), w.request(), w.actor)

			require.NoError(t, err)
			require.Len(t, w.posted, 1)
			assert.Equal(t, w.period.ID, w.posted[0].FiscalPeriodID)
			assert.Equal(t, int64(reversalDate), w.posted[0].AccountingDate)
		})
	}
}

func TestResolvePostingPeriodRejectsAnInactivePeriod(t *testing.T) {
	t.Parallel()

	w := newReversalWorld(t)
	w.period.Status = fiscalperiod.StatusInactive

	period, date, err := w.svc.validator.ResolvePostingPeriod(
		t.Context(),
		w.tenantInfo.OrgID,
		w.tenantInfo.BuID,
		reversalDate,
		w.control,
	)

	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "inactive fiscal period")
	assert.Nil(t, period)
	assert.Zero(t, date)
}
