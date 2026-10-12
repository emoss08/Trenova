package documentoperationshandler_test

import (
	"github.com/emoss08/trenova/internal/testutil/mocks/documentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentsearchprojectionportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentuploadportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/kernelportsmock"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/documentoperationshandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentoperationsservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentuploadports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type testDeps struct {
	handler          *documentoperationshandler.Handler
	documentRepo     *documentportsmock.MockDocumentRepository
	sessionRepo      *documentuploadportsmock.MockDocumentUploadSessionRepository
	contentService   *documentportsmock.MockDocumentContentService
	searchProjection *documentsearchprojectionportsmock.MockDocumentSearchProjectionService
	workflowStarter  *kernelportsmock.MockWorkflowStarter
}

func setupHandler(t *testing.T) *testDeps {
	t.Helper()

	logger := zap.NewNop()
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: logger,
		Config: &config.Config{App: config.AppConfig{Debug: true}},
	})

	documentRepo := documentportsmock.NewMockDocumentRepository(t)
	sessionRepo := documentuploadportsmock.NewMockDocumentUploadSessionRepository(t)
	contentService := documentportsmock.NewMockDocumentContentService(t)
	searchProjection := documentsearchprojectionportsmock.NewMockDocumentSearchProjectionService(t)
	workflowStarter := kernelportsmock.NewMockWorkflowStarter(t)

	pm := middleware.NewPermissionMiddleware(middleware.PermissionMiddlewareParams{
		PermissionEngine: &mocks.AllowAllPermissionEngine{},
		ErrorHandler:     errorHandler,
	})

	svc := documentoperationsservice.New(documentoperationsservice.Params{
		Logger:           logger,
		DocumentRepo:     documentRepo,
		SessionRepo:      sessionRepo,
		ContentService:   contentService,
		SearchProjection: searchProjection,
		WorkflowStarter:  workflowStarter,
	})

	return &testDeps{
		handler: documentoperationshandler.New(documentoperationshandler.Params{
			Service:              svc,
			ErrorHandler:         errorHandler,
			PermissionMiddleware: pm,
		}),
		documentRepo:     documentRepo,
		sessionRepo:      sessionRepo,
		contentService:   contentService,
		searchProjection: searchProjection,
		workflowStarter:  workflowStarter,
	}
}

func TestGetDiagnostics(t *testing.T) {
	t.Parallel()

	deps := setupHandler(t)
	documentID := pulid.MustNew("doc_")
	lineageID := pulid.MustNew("doc_")
	tenantInfo := pagination.TenantInfo{OrgID: testutil.TestOrgID, BuID: testutil.TestBuID}

	deps.documentRepo.EXPECT().GetByID(mock.Anything, documentports.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenantInfo,
	}).Return(&document.Document{
		ID:             documentID,
		LineageID:      lineageID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		FileType:       "application/pdf",
	}, nil)
	deps.documentRepo.EXPECT().ListVersions(mock.Anything, documentports.ListDocumentVersionsRequest{
		LineageID:  lineageID,
		TenantInfo: tenantInfo,
	}).Return([]*document.Document{{ID: documentID, LineageID: lineageID}}, nil)
	deps.sessionRepo.EXPECT().
		ListRelated(mock.Anything, &documentuploadports.ListRelatedDocumentUploadSessionsRequest{
			TenantInfo: tenantInfo,
			DocumentID: documentID,
			LineageID:  lineageID,
		}).
		Return(nil, nil)
	deps.contentService.EXPECT().GetContent(mock.Anything, documentID, tenantInfo).
		Return(nil, errortypes.NewNotFoundError("content not found"))
	deps.contentService.EXPECT().GetShipmentDraft(mock.Anything, documentID, tenantInfo).
		Return(nil, errortypes.NewNotFoundError("draft not found"))

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/admin/document-operations/" + documentID.String() + "/").
		WithDefaultAuthContext()

	deps.handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var body map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&body))
	assert.Equal(t, documentID.String(), body["document"].(map[string]any)["id"])
}

func TestResyncSearchAccepted(t *testing.T) {
	t.Parallel()

	deps := setupHandler(t)
	documentID := pulid.MustNew("doc_")
	tenantInfo := pagination.TenantInfo{
		OrgID:  testutil.TestOrgID,
		BuID:   testutil.TestBuID,
		UserID: testutil.TestUserID,
	}
	doc := &document.Document{
		ID:             documentID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
	}

	deps.documentRepo.EXPECT().GetByID(mock.Anything, documentports.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenantInfo,
	}).Return(doc, nil)
	deps.contentService.EXPECT().GetContent(mock.Anything, documentID, tenantInfo).
		Return(nil, errortypes.NewNotFoundError("content not found"))
	deps.searchProjection.EXPECT().Upsert(mock.Anything, doc, "").Return(nil)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/admin/document-operations/" + documentID.String() + "/resync-search/").
		WithDefaultAuthContext()

	deps.handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusAccepted, ginCtx.ResponseCode())

	var body map[string]string
	require.NoError(t, ginCtx.ResponseJSON(&body))
	assert.Equal(t, "accepted", body["status"])
}
