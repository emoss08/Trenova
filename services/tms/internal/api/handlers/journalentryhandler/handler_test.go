package journalentryhandler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/journalentryhandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/internal/core/domain/journalsource"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/journalentryservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	sharedtestutil "github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestListEntries(t *testing.T) {
	t.Parallel()

	entryRepo := mocks.NewMockJournalEntryRepository(t)
	sourceRepo := mocks.NewMockJournalSourceRepository(t)
	entry := &journalentry.Entry{ID: pulid.MustNew("je_"), EntryNumber: "JE-1"}
	entryRepo.EXPECT().List(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, req *repositories.ListJournalEntriesRequest) (*pagination.ListResult[*journalentry.Entry], error) {
		assert.Equal(t, sharedtestutil.TestOrgID, req.Filter.TenantInfo.OrgID)
		return &pagination.ListResult[*journalentry.Entry]{Items: []*journalentry.Entry{entry}, Total: 1}, nil
	}).Once()
	handler := newJournalEntryHandler(t, entryRepo, sourceRepo)

	ginCtx := sharedtestutil.NewGinTestContext().WithMethod(http.MethodGet).WithPath("/api/v1/accounting/journal-entries/").WithDefaultAuthContext()
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	var resp pagination.Response[[]*journalentry.Entry]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	require.Len(t, resp.Results, 1)
	assert.Equal(t, entry.ID, resp.Results[0].ID)
}

func TestGetEntry(t *testing.T) {
	t.Parallel()

	entryRepo := mocks.NewMockJournalEntryRepository(t)
	sourceRepo := mocks.NewMockJournalSourceRepository(t)
	entryID := pulid.MustNew("je_")
	entryRepo.EXPECT().GetByID(mock.Anything, repositories.GetJournalEntryByIDRequest{ID: entryID, TenantInfo: pagination.TenantInfo{OrgID: sharedtestutil.TestOrgID, BuID: sharedtestutil.TestBuID, UserID: sharedtestutil.TestUserID}}).Return(&journalentry.Entry{ID: entryID}, nil).Once()
	handler := newJournalEntryHandler(t, entryRepo, sourceRepo)

	ginCtx := sharedtestutil.NewGinTestContext().WithMethod(http.MethodGet).WithPath("/api/v1/accounting/journal-entries/" + entryID.String() + "/").WithDefaultAuthContext()
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestGetSource(t *testing.T) {
	t.Parallel()

	entryRepo := mocks.NewMockJournalEntryRepository(t)
	sourceRepo := mocks.NewMockJournalSourceRepository(t)
	sourceRepo.EXPECT().GetByObject(mock.Anything, repositories.GetJournalSourceByObjectRequest{TenantInfo: pagination.TenantInfo{OrgID: sharedtestutil.TestOrgID, BuID: sharedtestutil.TestBuID, UserID: sharedtestutil.TestUserID}, SourceObjectType: "Invoice", SourceObjectID: "inv_1"}).Return(&journalsource.Source{ID: pulid.MustNew("jsrc_")}, nil).Once()
	handler := newJournalEntryHandler(t, entryRepo, sourceRepo)

	ginCtx := sharedtestutil.NewGinTestContext().WithMethod(http.MethodGet).WithPath("/api/v1/accounting/journal-entries/source/Invoice/inv_1/").WithDefaultAuthContext()
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func newJournalEntryHandler(t *testing.T, entryRepo *mocks.MockJournalEntryRepository, sourceRepo *mocks.MockJournalSourceRepository) *journalentryhandler.Handler {
	t.Helper()

	logger := zap.NewNop()
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{Logger: logger, Config: &config.Config{App: config.AppConfig{Debug: true}}})
	pm := middleware.NewPermissionMiddleware(middleware.PermissionMiddlewareParams{PermissionEngine: &mocks.AllowAllPermissionEngine{}, ErrorHandler: errorHandler})
	service := journalentryservice.New(journalentryservice.Params{Logger: logger, EntryRepo: entryRepo, SourceRepo: sourceRepo})

	return journalentryhandler.New(journalentryhandler.Params{Service: service, ErrorHandler: errorHandler, PermissionMiddleware: pm})
}
