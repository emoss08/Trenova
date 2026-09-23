package fiscalperiodservice

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type transitionFixture struct {
	svc        *Service
	periodRepo *mocks.MockFiscalPeriodRepository
	yearRepo   *mocks.MockFiscalYearRepository
	tenant     pagination.TenantInfo
	userID     pulid.ID
	year       *fiscalyear.FiscalYear
}

func newTransitionFixture(t *testing.T, yearStatus fiscalyear.Status) *transitionFixture {
	t.Helper()

	periodRepo := mocks.NewMockFiscalPeriodRepository(t)
	yearRepo := mocks.NewMockFiscalYearRepository(t)
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	return &transitionFixture{
		svc: &Service{
			l:              zap.NewNop(),
			repo:           periodRepo,
			fiscalYearRepo: yearRepo,
			validator:      NewTestValidator(),
			auditService:   &mocks.NoopAuditService{},
		},
		periodRepo: periodRepo,
		yearRepo:   yearRepo,
		tenant:     tenant,
		userID:     pulid.MustNew("usr_"),
		year: &fiscalyear.FiscalYear{
			ID:             pulid.MustNew("fy_"),
			OrganizationID: tenant.OrgID,
			BusinessUnitID: tenant.BuID,
			Name:           "FY 2026",
			Status:         yearStatus,
		},
	}
}

func (f *transitionFixture) period(
	number int,
	status fiscalperiod.Status,
) *fiscalperiod.FiscalPeriod {
	return &fiscalperiod.FiscalPeriod{
		ID:             pulid.MustNew("fp_"),
		OrganizationID: f.tenant.OrgID,
		BusinessUnitID: f.tenant.BuID,
		FiscalYearID:   f.year.ID,
		PeriodNumber:   number,
		PeriodType:     fiscalperiod.PeriodTypeMonth,
		Status:         status,
		Name:           "Period",
	}
}

func (f *transitionFixture) expectState(
	target *fiscalperiod.FiscalPeriod,
	periods ...*fiscalperiod.FiscalPeriod,
) {
	f.periodRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetFiscalPeriodByIDRequest{ID: target.ID, TenantInfo: f.tenant}).
		Return(target, nil).
		Once()
	f.yearRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetFiscalYearByIDRequest{ID: f.year.ID, TenantInfo: f.tenant}).
		Return(f.year, nil).
		Once()
	f.periodRepo.EXPECT().
		ListByFiscalYearID(mock.Anything, repositories.ListByFiscalYearIDRequest{
			FiscalYearID: f.year.ID,
			OrgID:        f.tenant.OrgID,
			BuID:         f.tenant.BuID,
		}).
		Return(append([]*fiscalperiod.FiscalPeriod{target}, periods...), nil).
		Once()
}

func requireFieldError(t *testing.T, err error, field, fragment string) {
	t.Helper()

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	for _, e := range multiErr.Errors {
		if e.Field == field && strings.Contains(e.Error(), fragment) {
			return
		}
	}
	t.Fatalf("expected %q error containing %q, got %s", field, fragment, multiErr.Error())
}

func TestCloseAcceptsLockedPeriod(t *testing.T) {
	t.Parallel()

	f := newTransitionFixture(t, fiscalyear.StatusOpen)
	target := f.period(2, fiscalperiod.StatusLocked)
	f.expectState(target, f.period(1, fiscalperiod.StatusClosed))

	closed := *target
	closed.Status = fiscalperiod.StatusClosed
	f.periodRepo.EXPECT().
		Close(mock.Anything, mock.MatchedBy(func(req repositories.CloseFiscalPeriodRequest) bool {
			return req.ID == target.ID && req.ClosedByID == f.userID && req.ClosedAt > 0
		})).
		Return(&closed, nil).
		Once()

	result, err := f.svc.Close(t.Context(), repositories.CloseFiscalPeriodRequest{
		ID:         target.ID,
		TenantInfo: f.tenant,
	}, f.userID)

	require.NoError(t, err)
	assert.Equal(t, fiscalperiod.StatusClosed, result.Status)
}

func TestCloseRejectsWhenEarlierPeriodWasNeverOpened(t *testing.T) {
	t.Parallel()

	f := newTransitionFixture(t, fiscalyear.StatusOpen)
	target := f.period(3, fiscalperiod.StatusOpen)
	f.expectState(target, f.period(2, fiscalperiod.StatusInactive))

	_, err := f.svc.Close(t.Context(), repositories.CloseFiscalPeriodRequest{
		ID:         target.ID,
		TenantInfo: f.tenant,
	}, f.userID)

	requireFieldError(t, err, "status", "has never been opened")
}

func TestCloseRejectsInactiveAndClosedPeriods(t *testing.T) {
	t.Parallel()

	for _, status := range []fiscalperiod.Status{
		fiscalperiod.StatusInactive,
		fiscalperiod.StatusClosed,
		fiscalperiod.StatusPermanentlyClosed,
	} {
		t.Run(string(status), func(t *testing.T) {
			t.Parallel()

			f := newTransitionFixture(t, fiscalyear.StatusOpen)
			target := f.period(1, status)
			f.expectState(target)

			_, err := f.svc.Close(t.Context(), repositories.CloseFiscalPeriodRequest{
				ID:         target.ID,
				TenantInfo: f.tenant,
			}, f.userID)

			requireFieldError(t, err, "status", "Only Open or Locked")
		})
	}
}

func TestLockRecordsWhoLockedAndWhen(t *testing.T) {
	t.Parallel()

	f := newTransitionFixture(t, fiscalyear.StatusOpen)
	target := f.period(1, fiscalperiod.StatusOpen)
	f.expectState(target)

	locked := *target
	locked.Status = fiscalperiod.StatusLocked
	f.periodRepo.EXPECT().
		Lock(mock.Anything, mock.MatchedBy(func(req repositories.LockFiscalPeriodRequest) bool {
			return req.ID == target.ID && req.LockedByID == f.userID && req.LockedAt > 0
		})).
		Return(&locked, nil).
		Once()

	result, err := f.svc.Lock(t.Context(), repositories.LockFiscalPeriodRequest{
		ID:         target.ID,
		TenantInfo: f.tenant,
	}, f.userID)

	require.NoError(t, err)
	assert.Equal(t, fiscalperiod.StatusLocked, result.Status)
}

func TestLockRejectsClosedPeriod(t *testing.T) {
	t.Parallel()

	f := newTransitionFixture(t, fiscalyear.StatusOpen)
	target := f.period(1, fiscalperiod.StatusClosed)
	f.expectState(target)

	_, err := f.svc.Lock(t.Context(), repositories.LockFiscalPeriodRequest{
		ID:         target.ID,
		TenantInfo: f.tenant,
	}, f.userID)

	requireFieldError(t, err, "status", "Only Open fiscal periods can be locked")
}

func TestReopenRequiresReason(t *testing.T) {
	t.Parallel()

	f := newTransitionFixture(t, fiscalyear.StatusOpen)
	target := f.period(1, fiscalperiod.StatusClosed)
	f.expectState(target)

	_, err := f.svc.Reopen(t.Context(), repositories.ReopenFiscalPeriodRequest{
		ID:           target.ID,
		TenantInfo:   f.tenant,
		ReopenReason: "   ",
	}, f.userID)

	requireFieldError(t, err, "reopenReason", "reason for reopening is required")
}

func TestReopenRecordsReasonAndUser(t *testing.T) {
	t.Parallel()

	f := newTransitionFixture(t, fiscalyear.StatusOpen)
	target := f.period(1, fiscalperiod.StatusClosed)
	f.expectState(target, f.period(2, fiscalperiod.StatusOpen))

	reopened := *target
	reopened.Status = fiscalperiod.StatusOpen
	f.periodRepo.EXPECT().
		Reopen(mock.Anything, mock.MatchedBy(func(req repositories.ReopenFiscalPeriodRequest) bool {
			return req.ID == target.ID &&
				req.ReopenReason == "Late vendor invoice" &&
				req.ReopenedByID == f.userID &&
				req.ReopenedAt > 0
		})).
		Return(&reopened, nil).
		Once()

	result, err := f.svc.Reopen(t.Context(), repositories.ReopenFiscalPeriodRequest{
		ID:           target.ID,
		TenantInfo:   f.tenant,
		ReopenReason: "  Late vendor invoice ",
	}, f.userID)

	require.NoError(t, err)
	assert.Equal(t, fiscalperiod.StatusOpen, result.Status)
}

func TestReopenBlockedWhenFiscalYearIsClosed(t *testing.T) {
	t.Parallel()

	f := newTransitionFixture(t, fiscalyear.StatusClosed)
	target := f.period(12, fiscalperiod.StatusClosed)
	f.expectState(target)

	_, err := f.svc.Reopen(t.Context(), repositories.ReopenFiscalPeriodRequest{
		ID:           target.ID,
		TenantInfo:   f.tenant,
		ReopenReason: "Audit adjustment",
	}, f.userID)

	requireFieldError(t, err, "status", "Reopen the fiscal year first")
}

func TestUnlockBlockedWhenFiscalYearIsPermanentlyClosed(t *testing.T) {
	t.Parallel()

	f := newTransitionFixture(t, fiscalyear.StatusPermanentlyClosed)
	target := f.period(12, fiscalperiod.StatusLocked)
	f.expectState(target)

	_, err := f.svc.Unlock(t.Context(), repositories.UnlockFiscalPeriodRequest{
		ID:         target.ID,
		TenantInfo: f.tenant,
	}, f.userID)

	requireFieldError(t, err, "status", "which is permanently closed")
}

func TestActivateOpensInactivePeriod(t *testing.T) {
	t.Parallel()

	f := newTransitionFixture(t, fiscalyear.StatusOpen)
	target := f.period(13, fiscalperiod.StatusInactive)
	target.PeriodType = fiscalperiod.PeriodTypeAdjusting
	target.IsAdjusting = true
	f.expectState(target, f.period(12, fiscalperiod.StatusLocked))

	opened := *target
	opened.Status = fiscalperiod.StatusOpen
	f.periodRepo.EXPECT().
		Activate(mock.Anything, repositories.ActivateFiscalPeriodRequest{
			ID:         target.ID,
			TenantInfo: f.tenant,
		}).
		Return(&opened, nil).
		Once()

	result, err := f.svc.Activate(t.Context(), repositories.ActivateFiscalPeriodRequest{
		ID:         target.ID,
		TenantInfo: f.tenant,
	}, f.userID)

	require.NoError(t, err)
	assert.Equal(t, fiscalperiod.StatusOpen, result.Status)
}

func TestActivateRejectsPeriodsThatAreNotInactive(t *testing.T) {
	t.Parallel()

	f := newTransitionFixture(t, fiscalyear.StatusOpen)
	target := f.period(1, fiscalperiod.StatusClosed)
	f.expectState(target)

	_, err := f.svc.Activate(t.Context(), repositories.ActivateFiscalPeriodRequest{
		ID:         target.ID,
		TenantInfo: f.tenant,
	}, f.userID)

	requireFieldError(t, err, "status", "Only Inactive fiscal periods can be opened")
}

func TestActivateRequiresEarlierPeriodsToBeOpened(t *testing.T) {
	t.Parallel()

	f := newTransitionFixture(t, fiscalyear.StatusOpen)
	target := f.period(3, fiscalperiod.StatusInactive)
	f.expectState(target, f.period(2, fiscalperiod.StatusInactive))

	_, err := f.svc.Activate(t.Context(), repositories.ActivateFiscalPeriodRequest{
		ID:         target.ID,
		TenantInfo: f.tenant,
	}, f.userID)

	requireFieldError(t, err, "status", "period 2 has not been opened yet")
}

func TestActivateBlockedWhenFiscalYearIsClosed(t *testing.T) {
	t.Parallel()

	f := newTransitionFixture(t, fiscalyear.StatusClosed)
	target := f.period(13, fiscalperiod.StatusInactive)
	f.expectState(target)

	_, err := f.svc.Activate(t.Context(), repositories.ActivateFiscalPeriodRequest{
		ID:         target.ID,
		TenantInfo: f.tenant,
	}, f.userID)

	requireFieldError(t, err, "status", "Reopen the fiscal year first")
}

func TestUpdateKeepsLifecycleFieldsFromTheStoredPeriod(t *testing.T) {
	t.Parallel()

	f := newTransitionFixture(t, fiscalyear.StatusOpen)
	original := f.period(4, fiscalperiod.StatusOpen)
	original.StartDate = 1_775_001_600
	original.EndDate = 1_777_593_599

	incoming := *original
	incoming.Status = fiscalperiod.StatusPermanentlyClosed
	incoming.FiscalYearID = pulid.MustNew("fy_")
	incoming.Name = "April 2026"
	incoming.AllowAdjustingEntries = false

	f.periodRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetFiscalPeriodByIDRequest{
			ID:         original.ID,
			TenantInfo: pagination.TenantInfo{OrgID: f.tenant.OrgID, BuID: f.tenant.BuID},
		}).
		Return(original, nil).
		Once()
	f.periodRepo.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(entity *fiscalperiod.FiscalPeriod) bool {
			return entity.Status == fiscalperiod.StatusOpen &&
				entity.FiscalYearID == original.FiscalYearID &&
				entity.Name == "April 2026"
		})).
		RunAndReturn(func(_ context.Context, entity *fiscalperiod.FiscalPeriod) (*fiscalperiod.FiscalPeriod, error) {
			return entity, nil
		}).
		Once()

	result, err := f.svc.Update(t.Context(), &incoming, f.userID)

	require.NoError(t, err)
	assert.Equal(t, fiscalperiod.StatusOpen, result.Status)
}

func TestUpdateRejectsStructuralChangesOnceOpened(t *testing.T) {
	t.Parallel()

	f := newTransitionFixture(t, fiscalyear.StatusOpen)
	original := f.period(4, fiscalperiod.StatusOpen)
	original.StartDate = 1_775_001_600
	original.EndDate = 1_777_593_599

	incoming := *original
	incoming.PeriodNumber = 5
	incoming.EndDate = original.EndDate + 86_400

	f.periodRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(original, nil).
		Once()

	_, err := f.svc.Update(t.Context(), &incoming, f.userID)

	requireFieldError(t, err, "periodNumber", "before the period is opened")
	requireFieldError(t, err, "endDate", "before the period is opened")
}

func TestUpdateRejectsPermanentlyClosedPeriods(t *testing.T) {
	t.Parallel()

	f := newTransitionFixture(t, fiscalyear.StatusOpen)
	original := f.period(4, fiscalperiod.StatusPermanentlyClosed)
	incoming := *original
	incoming.Name = "Renamed"

	f.periodRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(original, nil).
		Once()

	_, err := f.svc.Update(t.Context(), &incoming, f.userID)

	requireFieldError(t, err, "status", "Permanently closed fiscal periods cannot be changed")
}
