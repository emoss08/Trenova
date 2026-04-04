package documentoperationsservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/internal/core/domain/documentupload"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/documentoperationsservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	"go.uber.org/zap"
)

func TestGetDiagnostics(t *testing.T) {
	t.Parallel()

	documentID := pulid.MustNew("doc_")
	lineageID := pulid.MustNew("doc_")
	sessionID := pulid.MustNew("dus_")
	tenantInfo := pagination.TenantInfo{OrgID: testutil.TestOrgID, BuID: testutil.TestBuID}

	docRepo := mocks.NewMockDocumentRepository(t)
	docRepo.EXPECT().GetByID(mock.Anything, repositories.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenantInfo,
	}).Return(&document.Document{
		ID:             documentID,
		LineageID:      lineageID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		FileType:       "application/pdf",
		ResourceID:     "trailer-1",
		ResourceType:   "trailer",
		ContentError:   "ocr failed once",
	}, nil)
	docRepo.EXPECT().ListVersions(mock.Anything, repositories.ListDocumentVersionsRequest{
		LineageID:  lineageID,
		TenantInfo: tenantInfo,
	}).Return([]*document.Document{
		{ID: documentID, LineageID: lineageID},
	}, nil)

	sessionRepo := mocks.NewMockDocumentUploadSessionRepository(t)
	sessionRepo.EXPECT().ListRelated(mock.Anything, &repositories.ListRelatedDocumentUploadSessionsRequest{
		TenantInfo: tenantInfo,
		DocumentID: documentID,
		LineageID:  lineageID,
	}).Return([]*documentupload.Session{
		{
			ID:             sessionID,
			DocumentID:     &documentID,
			LineageID:      &lineageID,
			FailureCode:    "FAILED",
			FailureMessage: "upload failed",
		},
	}, nil)

	contentService := mocks.NewMockDocumentContentService(t)
	contentService.EXPECT().GetContent(mock.Anything, documentID, tenantInfo).Return(&documentcontent.Content{
		DocumentID:  documentID,
		ContentText: "extracted text",
	}, nil)
	contentService.EXPECT().GetShipmentDraft(mock.Anything, documentID, tenantInfo).Return(&documentshipmentdraft.Draft{
		DocumentID: documentID,
	}, nil)

	searchProjection := mocks.NewMockDocumentSearchProjectionService(t)
	workflowStarter := mocks.NewMockWorkflowStarter(t)

	svc := documentoperationsservice.New(documentoperationsservice.Params{
		Logger:           zap.NewNop(),
		DocumentRepo:     docRepo,
		SessionRepo:      sessionRepo,
		ContentService:   contentService,
		SearchProjection: searchProjection,
		WorkflowStarter:  workflowStarter,
	})

	result, err := svc.GetDiagnostics(t.Context(), documentID, tenantInfo)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, documentID, result.Document.ID)
	assert.Len(t, result.Versions, 1)
	assert.Len(t, result.Sessions, 1)
	assert.NotNil(t, result.Content)
	assert.NotNil(t, result.ShipmentDraft)
	assert.Contains(t, result.LastErrors, "content: ocr failed once")
	assert.Contains(t, result.LastErrors, "upload session "+sessionID.String()+": FAILED: upload failed")
	assert.NotEmpty(t, result.WorkflowRefs)
}

func TestResyncSearchIgnoresMissingContent(t *testing.T) {
	t.Parallel()

	documentID := pulid.MustNew("doc_")
	tenantInfo := pagination.TenantInfo{OrgID: testutil.TestOrgID, BuID: testutil.TestBuID}

	docRepo := mocks.NewMockDocumentRepository(t)
	doc := &document.Document{ID: documentID, OrganizationID: testutil.TestOrgID, BusinessUnitID: testutil.TestBuID}
	docRepo.EXPECT().GetByID(mock.Anything, repositories.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenantInfo,
	}).Return(doc, nil)

	contentService := mocks.NewMockDocumentContentService(t)
	contentService.EXPECT().GetContent(mock.Anything, documentID, tenantInfo).
		Return(nil, errortypes.NewNotFoundError("content not found"))

	searchProjection := mocks.NewMockDocumentSearchProjectionService(t)
	searchProjection.EXPECT().Upsert(mock.Anything, doc, "").Return(nil)

	svc := documentoperationsservice.New(documentoperationsservice.Params{
		Logger:           zap.NewNop(),
		DocumentRepo:     docRepo,
		SessionRepo:      mocks.NewMockDocumentUploadSessionRepository(t),
		ContentService:   contentService,
		SearchProjection: searchProjection,
		WorkflowStarter:  mocks.NewMockWorkflowStarter(t),
	})

	require.NoError(t, svc.ResyncSearch(t.Context(), documentID, tenantInfo))
}

func TestRegeneratePreviewQueuesWorkflow(t *testing.T) {
	t.Parallel()

	documentID := pulid.MustNew("doc_")
	tenantInfo := pagination.TenantInfo{OrgID: testutil.TestOrgID, BuID: testutil.TestBuID}

	docRepo := mocks.NewMockDocumentRepository(t)
	docRepo.EXPECT().GetByID(mock.Anything, repositories.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenantInfo,
	}).Return(&document.Document{
		ID:             documentID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		FileType:       "application/pdf",
		StoragePath:    "org/trailer/file.pdf",
		ResourceType:   "trailer",
	}, nil)
	docRepo.EXPECT().UpdatePreview(mock.Anything, &repositories.UpdateDocumentPreviewRequest{
		ID:                 documentID,
		TenantInfo:         tenantInfo,
		PreviewStatus:      document.PreviewStatusPending,
		PreviewStoragePath: "",
	}).Return(nil)

	workflowStarter := mocks.NewMockWorkflowStarter(t)
	workflowStarter.EXPECT().Enabled().Return(true)
	workflowStarter.EXPECT().
		StartWorkflow(mock.Anything, mock.Anything, "GenerateThumbnailWorkflow", mock.Anything).
		Return(nil, &serviceerror.WorkflowExecutionAlreadyStarted{})

	svc := documentoperationsservice.New(documentoperationsservice.Params{
		Logger:           zap.NewNop(),
		DocumentRepo:     docRepo,
		SessionRepo:      mocks.NewMockDocumentUploadSessionRepository(t),
		ContentService:   mocks.NewMockDocumentContentService(t),
		SearchProjection: mocks.NewMockDocumentSearchProjectionService(t),
		WorkflowStarter:  workflowStarter,
	})

	require.NoError(t, svc.RegeneratePreview(t.Context(), documentID, tenantInfo))
}

func TestRegeneratePreviewRequiresSupportedType(t *testing.T) {
	t.Parallel()

	documentID := pulid.MustNew("doc_")
	tenantInfo := pagination.TenantInfo{OrgID: testutil.TestOrgID, BuID: testutil.TestBuID}

	docRepo := mocks.NewMockDocumentRepository(t)
	docRepo.EXPECT().GetByID(mock.Anything, repositories.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenantInfo,
	}).Return(&document.Document{
		ID:       documentID,
		FileType: "text/plain",
	}, nil)

	workflowStarter := mocks.NewMockWorkflowStarter(t)
	workflowStarter.EXPECT().Enabled().Return(true)

	svc := documentoperationsservice.New(documentoperationsservice.Params{
		Logger:           zap.NewNop(),
		DocumentRepo:     docRepo,
		SessionRepo:      mocks.NewMockDocumentUploadSessionRepository(t),
		ContentService:   mocks.NewMockDocumentContentService(t),
		SearchProjection: mocks.NewMockDocumentSearchProjectionService(t),
		WorkflowStarter:  workflowStarter,
	})

	err := svc.RegeneratePreview(context.Background(), documentID, tenantInfo)
	require.Error(t, err)
	assert.True(t, errortypes.IsConflictError(err))
}
