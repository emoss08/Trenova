//go:build integration

package fiscalperiodservice

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

func TestCloseReturnsConflictWhenFiscalPeriodIsLocked(t *testing.T) {
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
		l:            zap.NewNop(),
		db:           conn,
		repo:         fpRepo,
		auditService: &mocks.NoopAuditService{},
	}

	data := seedtest.SeedFullTestData(t, ctx, db)
	fy := mustCreateIntegrationFiscalYear(t, ctx, fyRepo, &fiscalyear.FiscalYear{
		OrganizationID: data.Organization.ID,
		BusinessUnitID: data.BusinessUnit.ID,
		Status:         fiscalyear.StatusOpen,
		Year:           2026,
		Name:           "FY 2026",
		StartDate:      time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(),
		EndDate:        time.Date(2026, time.December, 31, 23, 59, 59, 0, time.UTC).Unix(),
		IsCurrent:      true,
	})
	period := mustCreateFiscalPeriod(t, ctx, fpRepo, &fiscalperiod.FiscalPeriod{
		OrganizationID: data.Organization.ID,
		BusinessUnitID: data.BusinessUnit.ID,
		FiscalYearID:   fy.ID,
		PeriodNumber:   1,
		PeriodType:     fiscalperiod.PeriodTypeMonth,
		Status:         fiscalperiod.StatusOpen,
		Name:           "Period 1 - January 2026",
		StartDate:      time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(),
		EndDate:        time.Date(2026, time.January, 31, 23, 59, 59, 0, time.UTC).Unix(),
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
			_, err := fpRepo.GetByIDForUpdate(lockCtx, repositories.GetFiscalPeriodByIDRequest{
				ID:         period.ID,
				TenantInfo: tenantInfo,
			})
			return err
		},
	)
	lock.WaitLocked(t)

	closed, err := svc.Close(ctx, repositories.CloseFiscalPeriodRequest{
		ID:         period.ID,
		TenantInfo: tenantInfo,
	}, data.User.ID)

	require.Nil(t, closed)
	require.Error(t, err)
	assert.True(t, errortypes.IsConflictError(err))
	assert.Contains(t, err.Error(), "fiscal period is busy")

	lock.Release()
	lock.Wait(t)

	periodAfter, err := fpRepo.GetByID(ctx, repositories.GetFiscalPeriodByIDRequest{
		ID:         period.ID,
		TenantInfo: tenantInfo,
	})
	require.NoError(t, err)
	assert.Equal(t, fiscalperiod.StatusOpen, periodAfter.Status)
	assert.Nil(t, periodAfter.ClosedAt)
	assert.True(t, periodAfter.ClosedByID.IsNil())
}

func mustCreateIntegrationFiscalYear(
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

func mustCreateFiscalPeriod(
	t *testing.T,
	ctx context.Context,
	repo repositories.FiscalPeriodRepository,
	entity *fiscalperiod.FiscalPeriod,
) *fiscalperiod.FiscalPeriod {
	t.Helper()

	created, err := repo.Create(ctx, entity)
	require.NoError(t, err)
	return created
}
