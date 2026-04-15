package fiscalperiodhandler_test

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/fiscalperiodhandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/fiscalclose"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
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

	handler := newFiscalPeriodHandler(t, repo)
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

func newFiscalPeriodHandler(t *testing.T, repo *mocks.MockFiscalPeriodRepository) *fiscalperiodhandler.Handler {
	t.Helper()

	logger := zap.NewNop()
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{Logger: logger, Config: &config.Config{App: config.AppConfig{Debug: true}}})
	pm := middleware.NewPermissionMiddleware(middleware.PermissionMiddlewareParams{PermissionEngine: &mocks.AllowAllPermissionEngine{}, ErrorHandler: errorHandler})
	service := fiscalperiodservice.New(fiscalperiodservice.Params{Logger: logger, Repo: repo, AuditService: &mocks.NoopAuditService{}})

	return fiscalperiodhandler.New(fiscalperiodhandler.Params{Service: service, ErrorHandler: errorHandler, PermissionMiddleware: pm})
}
