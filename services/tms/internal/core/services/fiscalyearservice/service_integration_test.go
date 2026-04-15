//go:build integration

package fiscalyearservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/fiscalperiodrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/fiscalyearrepository"
	inttestutil "github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func TestActivateReturnsConflictWhenCurrentFiscalYearIsLocked(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	conn := postgres.NewTestConnection(db)
	fyRepo := fiscalyearrepository.New(fiscalyearrepository.Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})
	fpRepo := fiscalperiodrepository.New(fiscalperiodrepository.Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})
	svc := &Service{
		l:                zap.NewNop(),
		db:               conn,
		repo:             fyRepo,
		fiscalPeriodRepo: fpRepo,
		auditService:     &mocks.NoopAuditService{},
	}

	data := seedtest.SeedFullTestData(t, ctx, db)

	current := mustCreateFiscalYear(t, ctx, fyRepo, &fiscalyear.FiscalYear{
		OrganizationID: data.Organization.ID,
		BusinessUnitID: data.BusinessUnit.ID,
		Status:         fiscalyear.StatusOpen,
		Year:           2025,
		Name:           "FY 2025",
		StartDate:      time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(),
		EndDate:        time.Date(2025, time.December, 31, 23, 59, 59, 0, time.UTC).Unix(),
		IsCurrent:      true,
	})
	target := mustCreateFiscalYear(t, ctx, fyRepo, &fiscalyear.FiscalYear{
		OrganizationID: data.Organization.ID,
		BusinessUnitID: data.BusinessUnit.ID,
		Status:         fiscalyear.StatusDraft,
		Year:           2026,
		Name:           "FY 2026",
		StartDate:      time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(),
		EndDate:        time.Date(2026, time.December, 31, 23, 59, 59, 0, time.UTC).Unix(),
		IsCurrent:      false,
	})
	tenantInfo := pagination.TenantInfo{
		OrgID: data.Organization.ID,
		BuID:  data.BusinessUnit.ID,
	}

	lock := inttestutil.HoldTxLock(
		t,
		conn,
		ports.TxOptions{},
		func(lockCtx context.Context, _ bun.Tx) error {
			_, err := fyRepo.GetCurrentFiscalYearForUpdate(
				lockCtx,
				repositories.GetCurrentFiscalYearRequest{
					OrgID: data.Organization.ID,
					BuID:  data.BusinessUnit.ID,
				},
			)
			return err
		},
	)
	lock.WaitLocked(t)

	activated, err := svc.Activate(ctx, repositories.ActivateFiscalYearRequest{
		ID:         target.ID,
		TenantInfo: tenantInfo,
	}, data.User.ID)

	require.Nil(t, activated)
	require.Error(t, err)
	assert.True(t, errortypes.IsConflictError(err))
	assert.Contains(t, err.Error(), "fiscal year is busy")

	lock.Release()
	lock.Wait(t)

	currentAfter, err := fyRepo.GetCurrentFiscalYear(ctx, repositories.GetCurrentFiscalYearRequest{
		OrgID: data.Organization.ID,
		BuID:  data.BusinessUnit.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, current.ID, currentAfter.ID)

	count, err := db.NewSelect().
		Model((*fiscalyear.FiscalYear)(nil)).
		Where("organization_id = ?", data.Organization.ID).
		Where("business_unit_id = ?", data.BusinessUnit.ID).
		Where("is_current = ?", true).
		Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestGetCloseBlockersReturnsOpenPeriodBlocker(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	conn := postgres.NewTestConnection(db)
	fyRepo := fiscalyearrepository.New(fiscalyearrepository.Params{DB: conn, Logger: zap.NewNop()})
	fpRepo := fiscalperiodrepository.New(fiscalperiodrepository.Params{DB: conn, Logger: zap.NewNop()})
	svc := &Service{l: zap.NewNop(), db: conn, repo: fyRepo, fiscalPeriodRepo: fpRepo, auditService: &mocks.NoopAuditService{}}

	data := seedtest.SeedFullTestData(t, ctx, db)
	fy := mustCreateFiscalYear(t, ctx, fyRepo, &fiscalyear.FiscalYear{OrganizationID: data.Organization.ID, BusinessUnitID: data.BusinessUnit.ID, Status: fiscalyear.StatusOpen, Year: 2026, Name: "FY 2026", StartDate: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(), EndDate: time.Date(2026, time.December, 31, 23, 59, 59, 0, time.UTC).Unix(), IsCurrent: false})
	_, err := fpRepo.Create(ctx, &fiscalperiod.FiscalPeriod{OrganizationID: data.Organization.ID, BusinessUnitID: data.BusinessUnit.ID, FiscalYearID: fy.ID, PeriodNumber: 1, PeriodType: fiscalperiod.PeriodTypeMonth, Status: fiscalperiod.StatusOpen, Name: "Period 1 - January 2026", StartDate: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(), EndDate: time.Date(2026, time.January, 31, 23, 59, 59, 0, time.UTC).Unix()})
	require.NoError(t, err)

	result, err := svc.GetCloseBlockers(ctx, repositories.GetFiscalYearByIDRequest{ID: fy.ID, TenantInfo: pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.CanClose)
	require.NotEmpty(t, result.Blockers)
	assert.Contains(t, result.Blockers[0].Message, "period(s) are still open")
}

func mustCreateFiscalYear(
	t *testing.T,
	ctx context.Context,
	repo repositories.FiscalYearRepository,
	entity *fiscalyear.FiscalYear,
) *fiscalyear.FiscalYear {
	t.Helper()

	created, err := repo.Create(ctx, entity)
	require.NoError(t, err)
	return created
}
