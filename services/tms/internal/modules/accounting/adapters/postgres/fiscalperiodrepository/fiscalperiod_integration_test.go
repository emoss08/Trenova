//go:build integration

package fiscalperiodrepository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/accountingtest"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/fiscalyearrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fiscalPeriodFixture struct {
	db      *accountingtest.Database
	repo    fiscalperiodports.FiscalPeriodRepository
	year    *fiscalyear.FiscalYear
	periods []*fiscalperiod.FiscalPeriod
}

func newFiscalPeriodFixture(t *testing.T) *fiscalPeriodFixture {
	t.Helper()

	db := accountingtest.NewDatabase(t)
	f := &fiscalPeriodFixture{db: db, repo: New(Params{DB: db.Conn, Logger: zap.NewNop()})}
	f.year = createYear(t, db, db.A)
	f.periods = f.createPeriods(t, db.A, f.year)

	return f
}

func createYear(
	t *testing.T,
	db *accountingtest.Database,
	tenant accountingtest.Tenant,
) *fiscalyear.FiscalYear {
	t.Helper()

	years := fiscalyearrepository.New(fiscalyearrepository.Params{DB: db.Conn, Logger: zap.NewNop()})
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	year, err := years.Create(tenant.Ctx, &fiscalyear.FiscalYear{
		Status:         fiscalyear.StatusOpen,
		Year:           2026,
		Name:           "FY 2026",
		StartDate:      start.Unix(),
		EndDate:        start.AddDate(1, 0, 0).Unix() - 1,
		IsCalendarYear: true,
	})
	require.NoError(t, err)

	return year
}

func (f *fiscalPeriodFixture) createPeriods(
	t *testing.T,
	tenant accountingtest.Tenant,
	year *fiscalyear.FiscalYear,
) []*fiscalperiod.FiscalPeriod {
	t.Helper()

	start := time.Unix(year.StartDate, 0).UTC()
	periods := make([]*fiscalperiod.FiscalPeriod, 0, 3)
	for i := range 3 {
		from := start.AddDate(0, i, 0)
		periods = append(periods, &fiscalperiod.FiscalPeriod{
			FiscalYearID: year.ID,
			PeriodNumber: i + 1,
			PeriodType:   fiscalperiod.PeriodTypeMonth,
			Status:       fiscalperiod.StatusOpen,
			Name:         fmt.Sprintf("Period %d", i+1),
			StartDate:    from.Unix(),
			EndDate:      from.AddDate(0, 1, 0).Unix() - 1,
		})
	}
	require.NoError(t, f.repo.BulkCreate(tenant.Ctx, &fiscalperiodports.BulkCreateFiscalPeriodsRequest{
		Periods:    periods,
		TenantInfo: tenant.Info,
	}))

	return periods
}

func (f *fiscalPeriodFixture) get(t *testing.T, period *fiscalperiod.FiscalPeriod) *fiscalperiod.FiscalPeriod {
	t.Helper()

	stored, err := f.repo.GetByID(f.db.A.Ctx, fiscalperiodports.GetFiscalPeriodByIDRequest{
		ID:         period.ID,
		TenantInfo: f.db.A.Info,
	})
	require.NoError(t, err)

	return stored
}

func TestGetByIDsReadsOnlyTheTenantsPeriods(t *testing.T) {
	f := newFiscalPeriodFixture(t)
	foreign := f.createPeriods(t, f.db.B, createYear(t, f.db, f.db.B))[0]

	periods, err := f.repo.GetByIDs(f.db.A.Ctx, fiscalperiodports.GetFiscalPeriodsByIDsRequest{
		TenantInfo: f.db.A.Info,
		FiscalPeriodIDs: []pulid.ID{
			f.periods[2].ID,
			foreign.ID,
			f.periods[0].ID,
			pulid.MustNew("fp_"),
		},
	})
	require.NoError(t, err)

	names := make(map[pulid.ID]string, len(periods))
	for _, period := range periods {
		names[period.ID] = period.Name
	}
	assert.Equal(
		t,
		map[pulid.ID]string{f.periods[0].ID: "Period 1", f.periods[2].ID: "Period 3"},
		names,
	)

	none, err := f.repo.GetByIDs(
		f.db.A.Ctx,
		fiscalperiodports.GetFiscalPeriodsByIDsRequest{TenantInfo: f.db.A.Info},
	)
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestUpdateWritesZeroValuedEditableColumns(t *testing.T) {
	f := newFiscalPeriodFixture(t)
	deadline := int64(1_790_000_000)
	period := f.get(t, f.periods[0])
	period.AdjustmentDeadline = &deadline
	period.AllowAdjustingEntries = true
	period, err := f.repo.Update(f.db.A.Ctx, period)
	require.NoError(t, err)

	period.AdjustmentDeadline = nil
	period.AllowAdjustingEntries = false
	period.Status = fiscalperiod.StatusClosed
	closedAt := int64(1_790_000_100)
	period.ClosedAt = &closedAt
	before := period.Version
	_, err = f.repo.Update(f.db.A.Ctx, period)
	require.NoError(t, err)

	stored := f.get(t, period)
	assert.Nil(t, stored.AdjustmentDeadline)
	assert.False(t, stored.AllowAdjustingEntries)
	assert.Equal(t, fiscalperiod.StatusOpen, stored.Status)
	assert.Nil(t, stored.ClosedAt)
	assert.Equal(t, before+1, stored.Version)
}

func TestLockRecordsLockStamp(t *testing.T) {
	f := newFiscalPeriodFixture(t)
	period := f.periods[0]

	locked, err := f.repo.Lock(f.db.A.Ctx, fiscalperiodports.LockFiscalPeriodRequest{
		ID:         period.ID,
		TenantInfo: f.db.A.Info,
		LockedByID: f.db.A.Info.UserID,
		LockedAt:   1_780_000_000,
	})
	require.NoError(t, err)
	assert.Equal(t, fiscalperiod.StatusLocked, locked.Status)
	require.NotNil(t, locked.LockedAt)
	assert.Equal(t, int64(1_780_000_000), *locked.LockedAt)
	assert.Equal(t, f.db.A.Info.UserID, locked.LockedByID)
	assert.Equal(t, period.Version+1, locked.Version)

	unlocked, err := f.repo.Unlock(f.db.A.Ctx, fiscalperiodports.UnlockFiscalPeriodRequest{
		ID:         period.ID,
		TenantInfo: f.db.A.Info,
	})
	require.NoError(t, err)
	assert.Equal(t, fiscalperiod.StatusOpen, unlocked.Status)
	assert.Nil(t, unlocked.LockedAt)
	assert.True(t, unlocked.LockedByID.IsNil())
}

func TestTransitionOfAMissingPeriodIsAVersionMismatch(t *testing.T) {
	f := newFiscalPeriodFixture(t)

	_, err := f.repo.Activate(f.db.B.Ctx, fiscalperiodports.ActivateFiscalPeriodRequest{
		ID:         f.periods[0].ID,
		TenantInfo: f.db.B.Info,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsVersionMismatchError(err))
}

func TestCountUnclosedPeriodsIncludesLocked(t *testing.T) {
	f := newFiscalPeriodFixture(t)
	_, err := f.repo.Lock(f.db.A.Ctx, fiscalperiodports.LockFiscalPeriodRequest{
		ID:         f.periods[0].ID,
		TenantInfo: f.db.A.Info,
		LockedByID: f.db.A.Info.UserID,
		LockedAt:   1_780_000_000,
	})
	require.NoError(t, err)
	_, err = f.repo.Close(f.db.A.Ctx, fiscalperiodports.CloseFiscalPeriodRequest{
		ID:         f.periods[1].ID,
		TenantInfo: f.db.A.Info,
		ClosedByID: f.db.A.Info.UserID,
		ClosedAt:   1_780_000_000,
	})
	require.NoError(t, err)

	count, err := f.repo.CountUnclosedPeriodsByFiscalYear(
		f.db.A.Ctx,
		fiscalperiodports.CountUnclosedPeriodsByFiscalYearRequest{
			FiscalYearID: f.year.ID,
			OrgID:        f.db.A.Info.OrgID,
			BuID:         f.db.A.Info.BuID,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestCloseAllByFiscalYearClosesOnlyOpenPeriodsAndKeepsTheirVersion(t *testing.T) {
	f := newFiscalPeriodFixture(t)
	_, err := f.repo.Lock(f.db.A.Ctx, fiscalperiodports.LockFiscalPeriodRequest{
		ID:         f.periods[0].ID,
		TenantInfo: f.db.A.Info,
		LockedByID: f.db.A.Info.UserID,
		LockedAt:   1_780_000_000,
	})
	require.NoError(t, err)

	closed, err := f.repo.CloseAllByFiscalYear(f.db.A.Ctx, fiscalperiodports.CloseAllByFiscalYearRequest{
		FiscalYearID: f.year.ID,
		OrgID:        f.db.A.Info.OrgID,
		BuID:         f.db.A.Info.BuID,
		ClosedByID:   f.db.A.Info.UserID,
		ClosedAt:     1_790_000_000,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, closed)

	periods, err := f.repo.ListByFiscalYearID(f.db.A.Ctx, fiscalperiodports.ListByFiscalYearIDRequest{
		FiscalYearID: f.year.ID,
		OrgID:        f.db.A.Info.OrgID,
		BuID:         f.db.A.Info.BuID,
	})
	require.NoError(t, err)
	require.Len(t, periods, 3)
	assert.Equal(t, fiscalperiod.StatusLocked, periods[0].Status)
	for i, period := range periods[1:] {
		assert.Equal(t, fiscalperiod.StatusClosed, period.Status)
		assert.Equal(t, f.db.A.Info.UserID, period.ClosedByID)
		assert.Equal(t, f.periods[i+1].Version, period.Version)
	}
}

func TestGetPeriodByDateFindsTheOperatingPeriod(t *testing.T) {
	f := newFiscalPeriodFixture(t)

	period, err := f.repo.GetPeriodByDate(f.db.A.Ctx, fiscalperiodports.GetPeriodByDateRequest{
		OrgID: f.db.A.Info.OrgID,
		BuID:  f.db.A.Info.BuID,
		Date:  f.periods[1].StartDate + 3600,
	})
	require.NoError(t, err)
	assert.Equal(t, f.periods[1].ID, period.ID)

	_, err = f.repo.GetPeriodByDate(f.db.B.Ctx, fiscalperiodports.GetPeriodByDateRequest{
		OrgID: f.db.B.Info.OrgID,
		BuID:  f.db.B.Info.BuID,
		Date:  f.periods[1].StartDate + 3600,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
}

func TestGetByIDForUpdateReturnsRetryableLockError(t *testing.T) {
	f := newFiscalPeriodFixture(t)
	req := fiscalperiodports.GetFiscalPeriodByIDRequest{ID: f.periods[0].ID, TenantInfo: f.db.A.Info}

	locked := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- f.db.Conn.Transact(f.db.A.Ctx, kernelports.TxOptions{}, func(ctx context.Context) error {
			if _, err := f.repo.GetByIDForUpdate(ctx, req); err != nil {
				return err
			}
			close(locked)
			<-release
			return nil
		})
	}()
	<-locked

	entity, err := f.repo.GetByIDForUpdate(f.db.A.Ctx, req)
	close(release)
	require.NoError(t, <-done)

	require.Nil(t, entity)
	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	assert.Equal(t, "55P03", pgErr.Code)
	assert.True(t, dberror.IsRetryableTransactionError(err))
}
