//go:build integration

package fiscalyearrepository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/accountingtest"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalyearports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fiscalYearFixture struct {
	db   *accountingtest.Database
	repo fiscalyearports.FiscalYearRepository
}

func newFiscalYearFixture(t *testing.T) *fiscalYearFixture {
	t.Helper()

	db := accountingtest.NewDatabase(t)
	return &fiscalYearFixture{db: db, repo: New(Params{DB: db.Conn, Logger: zap.NewNop()})}
}

func (f *fiscalYearFixture) create(
	t *testing.T,
	tenant accountingtest.Tenant,
	year int,
	status fiscalyear.Status,
) *fiscalyear.FiscalYear {
	t.Helper()

	start := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	created, err := f.repo.Create(tenant.Ctx, &fiscalyear.FiscalYear{
		Status:                status,
		Year:                  year,
		Name:                  fmt.Sprintf("FY %d", year),
		Description:           "Calendar year",
		StartDate:             start.Unix(),
		EndDate:               start.AddDate(1, 0, 0).Unix() - 1,
		IsCalendarYear:        true,
		AllowAdjustingEntries: true,
	})
	require.NoError(t, err)

	return created
}

func (f *fiscalYearFixture) get(t *testing.T, tenant accountingtest.Tenant, year *fiscalyear.FiscalYear) *fiscalyear.FiscalYear {
	t.Helper()

	stored, err := f.repo.GetByID(tenant.Ctx, fiscalyearports.GetFiscalYearByIDRequest{
		ID:         year.ID,
		TenantInfo: tenant.Info,
	})
	require.NoError(t, err)

	return stored
}

func TestUpdateWritesOnlyEditableColumnsIncludingZeroValues(t *testing.T) {
	f := newFiscalYearFixture(t)
	year := f.create(t, f.db.A, 2026, fiscalyear.StatusOpen)

	edited := *year
	edited.Name = "FY 2026 (renamed)"
	edited.Description = ""
	edited.AllowAdjustingEntries = false
	edited.Status = fiscalyear.StatusPermanentlyClosed
	edited.IsCurrent = true

	updated, err := f.repo.Update(f.db.A.Ctx, &edited)
	require.NoError(t, err)
	assert.Equal(t, year.Version+1, updated.Version)

	stored := f.get(t, f.db.A, year)
	assert.Equal(t, "FY 2026 (renamed)", stored.Name)
	assert.Empty(t, stored.Description)
	assert.False(t, stored.AllowAdjustingEntries)
	assert.Equal(t, fiscalyear.StatusOpen, stored.Status)
	assert.False(t, stored.IsCurrent)
	assert.Equal(t, year.StartDate, stored.StartDate)
	assert.Equal(t, year.Version+1, stored.Version)
}

func TestUpdateRefusesAStaleVersion(t *testing.T) {
	f := newFiscalYearFixture(t)
	year := f.create(t, f.db.A, 2026, fiscalyear.StatusOpen)

	stale := *year
	_, err := f.repo.Update(f.db.A.Ctx, year)
	require.NoError(t, err)

	_, err = f.repo.Update(f.db.A.Ctx, &stale)
	require.Error(t, err)
	assert.True(t, errortypes.IsVersionMismatchError(err), "expected a version mismatch, got %v", err)
}

func TestCloseBumpsVersionSoOpenEditorsRefresh(t *testing.T) {
	f := newFiscalYearFixture(t)
	year := f.create(t, f.db.A, 2026, fiscalyear.StatusOpen)

	closed, err := f.repo.Close(f.db.A.Ctx, fiscalyearports.CloseFiscalYearRequest{
		ID:         year.ID,
		TenantInfo: f.db.A.Info,
		ClosedByID: f.db.A.Info.UserID,
		ClosedAt:   1_798_761_600,
	})
	require.NoError(t, err)
	assert.Equal(t, fiscalyear.StatusClosed, closed.Status)
	assert.Equal(t, year.Version+1, closed.Version)
	require.NotNil(t, closed.ClosedAt)
	assert.Equal(t, int64(1_798_761_600), *closed.ClosedAt)
	assert.Equal(t, f.db.A.Info.UserID, closed.ClosedByID)
}

func TestReopenOnlyReopensAClosedYear(t *testing.T) {
	f := newFiscalYearFixture(t)
	year := f.create(t, f.db.A, 2026, fiscalyear.StatusOpen)
	reopen := fiscalyearports.ReopenFiscalYearRequest{
		ID:           year.ID,
		TenantInfo:   f.db.A.Info,
		ReopenedByID: f.db.A.Info.UserID,
		ReopenedAt:   1_798_800_000,
		ReopenReason: "Late vendor invoice",
	}

	_, err := f.repo.Reopen(f.db.A.Ctx, reopen)
	require.Error(t, err)

	_, err = f.repo.Close(f.db.A.Ctx, fiscalyearports.CloseFiscalYearRequest{
		ID:         year.ID,
		TenantInfo: f.db.A.Info,
		ClosedByID: f.db.A.Info.UserID,
		ClosedAt:   1_798_761_600,
	})
	require.NoError(t, err)

	reopened, err := f.repo.Reopen(f.db.A.Ctx, reopen)
	require.NoError(t, err)
	assert.Equal(t, fiscalyear.StatusOpen, reopened.Status)
	assert.Nil(t, reopened.ClosedAt)
	assert.True(t, reopened.ClosedByID.IsNil())
	assert.Equal(t, "Late vendor invoice", reopened.ReopenReason)
}

func TestActivateMovesTheCurrentYear(t *testing.T) {
	f := newFiscalYearFixture(t)
	first := f.create(t, f.db.A, 2025, fiscalyear.StatusDraft)
	second := f.create(t, f.db.A, 2026, fiscalyear.StatusDraft)
	activate := func(year *fiscalyear.FiscalYear) {
		_, err := f.repo.Activate(f.db.A.Ctx, fiscalyearports.ActivateFiscalYearRequest{
			ID:         year.ID,
			TenantInfo: f.db.A.Info,
		})
		require.NoError(t, err)
	}

	activate(first)
	activate(second)

	current, err := f.repo.GetCurrentFiscalYear(f.db.A.Ctx, fiscalyearports.GetCurrentFiscalYearRequest{
		OrgID: f.db.A.Info.OrgID,
		BuID:  f.db.A.Info.BuID,
	})
	require.NoError(t, err)
	assert.Equal(t, second.ID, current.ID)
	assert.Equal(t, fiscalyear.StatusOpen, current.Status)
	assert.False(t, f.get(t, f.db.A, first).IsCurrent)
}

func TestFiscalYearsStayInsideTheirTenant(t *testing.T) {
	f := newFiscalYearFixture(t)
	year := f.create(t, f.db.A, 2026, fiscalyear.StatusOpen)
	f.create(t, f.db.B, 2026, fiscalyear.StatusOpen)

	_, err := f.repo.GetByID(f.db.B.Ctx, fiscalyearports.GetFiscalYearByIDRequest{
		ID:         year.ID,
		TenantInfo: f.db.B.Info,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))

	count, err := f.repo.CountByTenant(f.db.A.Ctx, fiscalyearports.CountFiscalYearsByTenantRequest{
		OrgID: f.db.A.Info.OrgID,
		BuID:  f.db.A.Info.BuID,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	listed, err := f.repo.List(f.db.A.Ctx, &fiscalyearports.ListFiscalYearsRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: f.db.A.Info,
			Pagination: pagination.Info{Limit: 10},
		},
	})
	require.NoError(t, err)
	require.Len(t, listed.Items, 1)
	assert.Equal(t, year.ID, listed.Items[0].ID)
	assert.Equal(t, 1, listed.Total)
}

func TestGetByIDForUpdateFailsFastWhileTheYearIsLocked(t *testing.T) {
	f := newFiscalYearFixture(t)
	year := f.create(t, f.db.A, 2026, fiscalyear.StatusOpen)
	req := fiscalyearports.GetFiscalYearByIDRequest{ID: year.ID, TenantInfo: f.db.A.Info}

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

	_, err := f.repo.GetByIDForUpdate(f.db.A.Ctx, req)
	close(release)
	require.NoError(t, <-done)

	require.Error(t, err)
	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	assert.Equal(t, "55P03", pgErr.Code)
	assert.True(t, dberror.IsRetryableTransactionError(err))
}
