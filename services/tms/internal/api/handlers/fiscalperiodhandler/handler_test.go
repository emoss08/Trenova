package fiscalperiodhandler_test

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/fiscalperiodhandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/fiscalclose"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/fiscalperiodservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	sharedtestutil "github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestHandlerCloseBlockers(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockFiscalPeriodRepository(t)
	periodID := pulid.MustNew("fp_")
	fyID := pulid.MustNew("fy_")
	period := &fiscalperiod.FiscalPeriod{ID: periodID, FiscalYearID: fyID, OrganizationID: sharedtestutil.TestOrgID, BusinessUnitID: sharedtestutil.TestBuID, Status: fiscalperiod.StatusClosed, PeriodNumber: 1}
	repo.EXPECT().GetByID(mock.Anything, repositories.GetFiscalPeriodByIDRequest{ID: periodID, TenantInfo: pagination.TenantInfo{OrgID: sharedtestutil.TestOrgID, BuID: sharedtestutil.TestBuID}}).Return(period, nil).Once()
	repo.EXPECT().ListByFiscalYearID(mock.Anything, repositories.ListByFiscalYearIDRequest{FiscalYearID: fyID, OrgID: sharedtestutil.TestOrgID, BuID: sharedtestutil.TestBuID}).Return([]*fiscalperiod.FiscalPeriod{period}, nil).Once()

	handler := newFiscalPeriodHandler(t, repo, mocks.NewMockFiscalYearRepository(t))
	ginCtx := sharedtestutil.NewGinTestContext().WithMethod(http.MethodGet).WithPath("/api/v1/fiscal-periods/" + periodID.String() + "/close-blockers/").WithDefaultAuthContext()
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	var resp fiscalclose.Result
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.False(t, resp.CanClose)
	require.Len(t, resp.Blockers, 1)
	assert.Equal(t, errortypes.ErrInvalid, resp.Blockers[0].Code)
}

func newFiscalPeriodHandler(
	t *testing.T,
	repo *mocks.MockFiscalPeriodRepository,
	yearRepo *mocks.MockFiscalYearRepository,
) *fiscalperiodhandler.Handler {
	t.Helper()

	logger := zap.NewNop()
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{Logger: logger, Config: &config.Config{App: config.AppConfig{Debug: true}}})
	pm := middleware.NewPermissionMiddleware(middleware.PermissionMiddlewareParams{PermissionEngine: &mocks.AllowAllPermissionEngine{}, ErrorHandler: errorHandler})
	service := fiscalperiodservice.New(fiscalperiodservice.Params{Logger: logger, Repo: repo, FiscalYearRepo: yearRepo, AuditService: &mocks.NoopAuditService{}})

	return fiscalperiodhandler.New(fiscalperiodhandler.Params{Service: service, ErrorHandler: errorHandler, PermissionMiddleware: pm})
}

func expectTransitionState(
	repo *mocks.MockFiscalPeriodRepository,
	yearRepo *mocks.MockFiscalYearRepository,
	period *fiscalperiod.FiscalPeriod,
) {
	tenant := pagination.TenantInfo{OrgID: sharedtestutil.TestOrgID, BuID: sharedtestutil.TestBuID, UserID: sharedtestutil.TestUserID}
	repo.EXPECT().GetByID(mock.Anything, repositories.GetFiscalPeriodByIDRequest{ID: period.ID, TenantInfo: tenant}).Return(period, nil).Once()
	yearRepo.EXPECT().GetByID(mock.Anything, repositories.GetFiscalYearByIDRequest{ID: period.FiscalYearID, TenantInfo: tenant}).Return(&fiscalyear.FiscalYear{ID: period.FiscalYearID, Status: fiscalyear.StatusOpen}, nil).Once()
	repo.EXPECT().ListByFiscalYearID(mock.Anything, mock.Anything).Return([]*fiscalperiod.FiscalPeriod{period}, nil).Once()
}

func TestHandlerActivateOpensInactivePeriod(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockFiscalPeriodRepository(t)
	yearRepo := mocks.NewMockFiscalYearRepository(t)
	period := &fiscalperiod.FiscalPeriod{ID: pulid.MustNew("fp_"), FiscalYearID: pulid.MustNew("fy_"), OrganizationID: sharedtestutil.TestOrgID, BusinessUnitID: sharedtestutil.TestBuID, Status: fiscalperiod.StatusInactive, PeriodNumber: 13, PeriodType: fiscalperiod.PeriodTypeAdjusting, IsAdjusting: true}
	expectTransitionState(repo, yearRepo, period)

	opened := *period
	opened.Status = fiscalperiod.StatusOpen
	repo.EXPECT().Activate(mock.Anything, repositories.ActivateFiscalPeriodRequest{ID: period.ID, TenantInfo: pagination.TenantInfo{OrgID: sharedtestutil.TestOrgID, BuID: sharedtestutil.TestBuID, UserID: sharedtestutil.TestUserID}}).Return(&opened, nil).Once()

	handler := newFiscalPeriodHandler(t, repo, yearRepo)
	ginCtx := sharedtestutil.NewGinTestContext().WithMethod(http.MethodPut).WithPath("/api/v1/fiscal-periods/" + period.ID.String() + "/activate/").WithDefaultAuthContext()
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	var resp fiscalperiod.FiscalPeriod
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, fiscalperiod.StatusOpen, resp.Status)
}

func TestHandlerReopenForwardsReason(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockFiscalPeriodRepository(t)
	yearRepo := mocks.NewMockFiscalYearRepository(t)
	period := &fiscalperiod.FiscalPeriod{ID: pulid.MustNew("fp_"), FiscalYearID: pulid.MustNew("fy_"), OrganizationID: sharedtestutil.TestOrgID, BusinessUnitID: sharedtestutil.TestBuID, Status: fiscalperiod.StatusClosed, PeriodNumber: 1}
	expectTransitionState(repo, yearRepo, period)

	reopened := *period
	reopened.Status = fiscalperiod.StatusOpen
	repo.EXPECT().Reopen(mock.Anything, mock.MatchedBy(func(req repositories.ReopenFiscalPeriodRequest) bool {
		return req.ID == period.ID && req.ReopenReason == "Late vendor invoice"
	})).Return(&reopened, nil).Once()

	handler := newFiscalPeriodHandler(t, repo, yearRepo)
	ginCtx := sharedtestutil.NewGinTestContext().WithMethod(http.MethodPut).WithPath("/api/v1/fiscal-periods/" + period.ID.String() + "/reopen/").WithJSONBody(map[string]string{"reopenReason": "Late vendor invoice"}).WithDefaultAuthContext()
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}
