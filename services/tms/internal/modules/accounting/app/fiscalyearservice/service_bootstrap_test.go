package fiscalyearservice

import (
	"context"
	"github.com/emoss08/trenova/internal/testutil/mocks/fiscalperiodportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/fiscalyearportsmock"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalyearports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestEnsureBootstrapCurrentFiscalYear_ReturnsExisting(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	fyRepo := fiscalyearportsmock.NewMockFiscalYearRepository(t)
	fpRepo := fiscalperiodportsmock.NewMockFiscalPeriodRepository(t)
	service := &Service{
		l:                zap.NewNop(),
		repo:             fyRepo,
		fiscalPeriodRepo: fpRepo,
	}

	existing := &fiscalyear.FiscalYear{
		ID:             pulid.MustNew("fyr_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         fiscalyear.StatusOpen,
		IsCurrent:      true,
	}

	fyRepo.EXPECT().
		GetCurrentFiscalYear(ctx, fiscalyearports.GetCurrentFiscalYearRequest{
			OrgID: orgID,
			BuID:  buID,
		}).
		Return(existing, nil).
		Once()

	result, err := service.ensureBootstrapCurrentFiscalYear(ctx, orgID, buID)
	require.NoError(t, err)
	assert.Equal(t, existing, result)
}

func TestEnsureBootstrapCurrentFiscalYear_ReturnsConflictWhenHistoryExists(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	fyRepo := fiscalyearportsmock.NewMockFiscalYearRepository(t)
	fpRepo := fiscalperiodportsmock.NewMockFiscalPeriodRepository(t)
	service := &Service{
		l:                zap.NewNop(),
		repo:             fyRepo,
		fiscalPeriodRepo: fpRepo,
	}

	fyRepo.EXPECT().
		GetCurrentFiscalYear(ctx, fiscalyearports.GetCurrentFiscalYearRequest{
			OrgID: orgID,
			BuID:  buID,
		}).
		Return(nil, errortypes.NewNotFoundError("not found")).
		Once()

	fyRepo.EXPECT().
		CountByTenant(ctx, fiscalyearports.CountFiscalYearsByTenantRequest{
			OrgID: orgID,
			BuID:  buID,
		}).
		Return(2, nil).
		Once()

	result, err := service.ensureBootstrapCurrentFiscalYear(ctx, orgID, buID)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errortypes.IsConflictError(err))
}

func TestEnsureBootstrapCurrentFiscalYear_BootstrapsWhenNoFiscalYearsExist(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	fyRepo := fiscalyearportsmock.NewMockFiscalYearRepository(t)
	fpRepo := fiscalperiodportsmock.NewMockFiscalPeriodRepository(t)
	service := &Service{
		l:                zap.NewNop(),
		repo:             fyRepo,
		fiscalPeriodRepo: fpRepo,
	}

	fyRepo.EXPECT().
		GetCurrentFiscalYear(ctx, fiscalyearports.GetCurrentFiscalYearRequest{
			OrgID: orgID,
			BuID:  buID,
		}).
		Return(nil, errortypes.NewNotFoundError("not found")).
		Once()

	fyRepo.EXPECT().
		CountByTenant(ctx, fiscalyearports.CountFiscalYearsByTenantRequest{
			OrgID: orgID,
			BuID:  buID,
		}).
		Return(0, nil).
		Once()

	fyRepo.EXPECT().
		Create(ctx, mock.MatchedBy(mockFiscalYearForBootstrap(orgID, buID))).
		RunAndReturn(func(_ context.Context, entity *fiscalyear.FiscalYear) (*fiscalyear.FiscalYear, error) {
			entity.ID = pulid.MustNew("fyr_")
			return entity, nil
		}).
		Once()

	fpRepo.EXPECT().
		BulkCreate(ctx, mock.MatchedBy(mockBulkCreateRequest(orgID, buID))).
		Return(nil).
		Once()

	result, err := service.ensureBootstrapCurrentFiscalYear(ctx, orgID, buID)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsCurrent)
	assert.Equal(t, fiscalyear.StatusOpen, result.Status)
	assert.Equal(t, orgID, result.OrganizationID)
	assert.Equal(t, buID, result.BusinessUnitID)
}

func TestEnsureBootstrapCurrentFiscalYear_HandlesCreateRace(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	fyRepo := fiscalyearportsmock.NewMockFiscalYearRepository(t)
	fpRepo := fiscalperiodportsmock.NewMockFiscalPeriodRepository(t)
	service := &Service{
		l:                zap.NewNop(),
		repo:             fyRepo,
		fiscalPeriodRepo: fpRepo,
	}

	currentReq := fiscalyearports.GetCurrentFiscalYearRequest{OrgID: orgID, BuID: buID}

	fyRepo.EXPECT().
		GetCurrentFiscalYear(ctx, currentReq).
		Return(nil, errortypes.NewNotFoundError("not found")).
		Once()

	fyRepo.EXPECT().
		CountByTenant(ctx, fiscalyearports.CountFiscalYearsByTenantRequest{
			OrgID: orgID,
			BuID:  buID,
		}).
		Return(0, nil).
		Once()

	fyRepo.EXPECT().
		Create(ctx, mock.MatchedBy(mockFiscalYearForBootstrap(orgID, buID))).
		Return(nil, &pgconn.PgError{Code: "23505"}).
		Once()

	fallback := &fiscalyear.FiscalYear{
		ID:             pulid.MustNew("fyr_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         fiscalyear.StatusOpen,
		IsCurrent:      true,
	}

	fyRepo.EXPECT().
		GetCurrentFiscalYear(ctx, currentReq).
		Return(fallback, nil).
		Once()

	result, err := service.ensureBootstrapCurrentFiscalYear(ctx, orgID, buID)
	require.NoError(t, err)
	assert.Equal(t, fallback, result)
}

func TestList_EmptyWithExistingHistoryReturnsEmptyWithoutError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	fyRepo := fiscalyearportsmock.NewMockFiscalYearRepository(t)
	fpRepo := fiscalperiodportsmock.NewMockFiscalPeriodRepository(t)
	service := &Service{
		l:                zap.NewNop(),
		repo:             fyRepo,
		fiscalPeriodRepo: fpRepo,
	}

	req := &fiscalyearports.ListFiscalYearsRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: pagination.TenantInfo{
				OrgID: orgID,
				BuID:  buID,
			},
		},
	}

	emptyResult := &pagination.ListResult[*fiscalyear.FiscalYear]{
		Items:      []*fiscalyear.FiscalYear{},
		Total:      0,
		TotalExact: true,
	}

	fyRepo.EXPECT().List(ctx, req).Return(emptyResult, nil).Once()
	fyRepo.EXPECT().
		GetCurrentFiscalYear(ctx, fiscalyearports.GetCurrentFiscalYearRequest{
			OrgID: orgID,
			BuID:  buID,
		}).
		Return(nil, errortypes.NewNotFoundError("not found")).
		Once()
	fyRepo.EXPECT().
		CountByTenant(ctx, fiscalyearports.CountFiscalYearsByTenantRequest{
			OrgID: orgID,
			BuID:  buID,
		}).
		Return(1, nil).
		Once()

	result, err := service.List(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, emptyResult, result)
}

func TestList_EmptyBootstrapsAndReloads(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	fyRepo := fiscalyearportsmock.NewMockFiscalYearRepository(t)
	fpRepo := fiscalperiodportsmock.NewMockFiscalPeriodRepository(t)
	service := &Service{
		l:                zap.NewNop(),
		repo:             fyRepo,
		fiscalPeriodRepo: fpRepo,
	}

	req := &fiscalyearports.ListFiscalYearsRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: pagination.TenantInfo{
				OrgID: orgID,
				BuID:  buID,
			},
		},
	}

	emptyResult := &pagination.ListResult[*fiscalyear.FiscalYear]{
		Items:      []*fiscalyear.FiscalYear{},
		Total:      0,
		TotalExact: true,
	}
	created := &fiscalyear.FiscalYear{
		ID:             pulid.MustNew("fyr_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         fiscalyear.StatusOpen,
		IsCurrent:      true,
	}
	finalResult := &pagination.ListResult[*fiscalyear.FiscalYear]{
		Items:      []*fiscalyear.FiscalYear{created},
		Total:      1,
		TotalExact: true,
	}

	fyRepo.EXPECT().List(ctx, req).Return(emptyResult, nil).Once()
	fyRepo.EXPECT().
		GetCurrentFiscalYear(ctx, fiscalyearports.GetCurrentFiscalYearRequest{
			OrgID: orgID,
			BuID:  buID,
		}).
		Return(nil, errortypes.NewNotFoundError("not found")).
		Once()
	fyRepo.EXPECT().
		CountByTenant(ctx, fiscalyearports.CountFiscalYearsByTenantRequest{
			OrgID: orgID,
			BuID:  buID,
		}).
		Return(0, nil).
		Once()
	fyRepo.EXPECT().
		Create(ctx, mock.MatchedBy(mockFiscalYearForBootstrap(orgID, buID))).
		RunAndReturn(func(_ context.Context, entity *fiscalyear.FiscalYear) (*fiscalyear.FiscalYear, error) {
			entity.ID = created.ID
			return entity, nil
		}).
		Once()
	fpRepo.EXPECT().
		BulkCreate(ctx, mock.MatchedBy(mockBulkCreateRequest(orgID, buID))).
		Return(nil).
		Once()
	fyRepo.EXPECT().List(ctx, req).Return(finalResult, nil).Once()

	result, err := service.List(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, finalResult, result)
}

func mockFiscalYearForBootstrap(orgID, buID pulid.ID) func(*fiscalyear.FiscalYear) bool {
	return func(entity *fiscalyear.FiscalYear) bool {
		nowYear := time.Now().UTC().Year()

		start := time.Date(nowYear, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()
		end := time.Date(nowYear, time.December, 31, 23, 59, 59, 0, time.UTC).Unix()

		return entity.OrganizationID == orgID &&
			entity.BusinessUnitID == buID &&
			entity.Year == nowYear &&
			entity.Status == fiscalyear.StatusOpen &&
			entity.IsCurrent &&
			entity.IsCalendarYear &&
			entity.StartDate == start &&
			entity.EndDate == end
	}
}

func mockBulkCreateRequest(
	orgID, buID pulid.ID,
) func(*fiscalperiodports.BulkCreateFiscalPeriodsRequest) bool {
	return func(req *fiscalperiodports.BulkCreateFiscalPeriodsRequest) bool {
		if req == nil || len(req.Periods) == 0 {
			return false
		}

		if req.TenantInfo.OrgID != orgID || req.TenantInfo.BuID != buID {
			return false
		}

		return true
	}
}
