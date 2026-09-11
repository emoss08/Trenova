package fiscalyearhandler_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/fiscalyearhandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/fiscalclose"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/domain/journalsource"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/fiscalcloseservice"
	"github.com/emoss08/trenova/internal/core/services/fiscalyearservice"
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

	fyRepo := mocks.NewMockFiscalYearRepository(t)
	fpRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalYearID := pulid.MustNew("fy_")
	fiscalYear := &fiscalyear.FiscalYear{ID: fiscalYearID, OrganizationID: sharedtestutil.TestOrgID, BusinessUnitID: sharedtestutil.TestBuID, Status: fiscalyear.StatusOpen}
	fyRepo.EXPECT().GetByID(mock.Anything, repositories.GetFiscalYearByIDRequest{ID: fiscalYearID, TenantInfo: pagination.TenantInfo{OrgID: sharedtestutil.TestOrgID, BuID: sharedtestutil.TestBuID}}).Return(fiscalYear, nil).Once()
	fpRepo.EXPECT().GetOpenPeriodsCountByFiscalYear(mock.Anything, repositories.GetOpenPeriodsCountByFiscalYearRequest{FiscalYearID: fiscalYearID, OrgID: sharedtestutil.TestOrgID, BuID: sharedtestutil.TestBuID}).Return(2, nil).Once()
	fpRepo.EXPECT().ListByFiscalYearID(mock.Anything, mock.Anything).Return([]*fiscalperiod.FiscalPeriod{}, nil).Once()
	fyRepo.EXPECT().GetNextFiscalYear(mock.Anything, mock.Anything).Return(nil, errortypes.NewNotFoundError("FiscalYear not found")).Once()

	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().GetByOrgID(mock.Anything, mock.Anything).Return(nil, errortypes.NewNotFoundError("AccountingControl not found")).Once()
	balanceRepo := mocks.NewMockGLBalanceRepository(t)
	balanceRepo.EXPECT().ListYearToDateBalances(mock.Anything, mock.Anything).Return([]*repositories.GLPeriodAccountBalance{}, nil).Once()
	sourceRepo := mocks.NewMockJournalSourceRepository(t)
	sourceRepo.EXPECT().ListByObject(mock.Anything, mock.Anything).Return([]*journalsource.Source{}, nil).Once()

	closeService := fiscalcloseservice.New(fiscalcloseservice.Params{
		Logger:           zap.NewNop(),
		FiscalYearRepo:   fyRepo,
		FiscalPeriodRepo: fpRepo,
		GLBalanceRepo:    balanceRepo,
		AccountingRepo:   accountingRepo,
		SourceRepo:       sourceRepo,
	})

	handler := newFiscalYearHandler(t, fyRepo, fpRepo, closeService)
	ginCtx := sharedtestutil.NewGinTestContext().WithMethod(http.MethodGet).WithPath("/api/v1/fiscal-years/" + fiscalYearID.String() + "/close-blockers/").WithDefaultAuthContext()
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	var resp fiscalclose.Result
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.False(t, resp.CanClose)
	require.NotEmpty(t, resp.Blockers)
	assert.Equal(t, errortypes.ErrInvalid, resp.Blockers[0].Code)
	assert.Contains(t, resp.Blockers[0].Message, "period(s) are still open")
	assert.Contains(t, blockerMessages(resp.Blockers), "accounting control")
}

func blockerMessages(blockers []*fiscalclose.Blocker) string {
	joined := make([]string, 0, len(blockers))
	for _, blocker := range blockers {
		joined = append(joined, blocker.Message)
	}

	return strings.Join(joined, " | ")
}

func newFiscalYearHandler(
	t *testing.T,
	fyRepo *mocks.MockFiscalYearRepository,
	fpRepo *mocks.MockFiscalPeriodRepository,
	closeService *fiscalcloseservice.Service,
) *fiscalyearhandler.Handler {
	t.Helper()

	logger := zap.NewNop()
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{Logger: logger, Config: &config.Config{App: config.AppConfig{Debug: true}}})
	pm := middleware.NewPermissionMiddleware(middleware.PermissionMiddlewareParams{PermissionEngine: &mocks.AllowAllPermissionEngine{}, ErrorHandler: errorHandler})
	service := fiscalyearservice.New(fiscalyearservice.Params{Logger: logger, Repo: fyRepo, FiscalPeriodRepo: fpRepo, AuditService: &mocks.NoopAuditService{}, CloseService: closeService})

	return fiscalyearhandler.New(fiscalyearhandler.Params{Service: service, ErrorHandler: errorHandler, PermissionMiddleware: pm})
}
