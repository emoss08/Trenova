package documentservice_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/internal/core/services/documentservice"
	"github.com/emoss08/trenova/internal/core/services/thumbnailservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	storageutil "github.com/emoss08/trenova/shared/testutil/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type mockDocRepo struct {
	ListFn                             func(ctx context.Context, req *repositories.ListDocumentsRequest) (*pagination.ListResult[*document.Document], error)
	GetByIDFn                          func(ctx context.Context, req repositories.GetDocumentByIDRequest) (*document.Document, error)
	GetByIDsFn                         func(ctx context.Context, req repositories.BulkDeleteDocumentRequest) ([]*document.Document, error)
	GetByResFn                         func(ctx context.Context, req *repositories.GetDocumentsByResourceRequest) ([]*document.Document, error)
	ListPendingPreviewReconciliationFn func(ctx context.Context, olderThan int64, limit int) ([]*document.Document, error)
	CreateFn                           func(ctx context.Context, entity *document.Document) (*document.Document, error)
	UpdateFn                           func(ctx context.Context, entity *document.Document) (*document.Document, error)
	UpdatePreviewFn                    func(ctx context.Context, req *repositories.UpdateDocumentPreviewRequest) error
	UpdateIntelligenceFn               func(ctx context.Context, req *repositories.UpdateDocumentIntelligenceRequest) error
	DeleteFn                           func(ctx context.Context, req repositories.DeleteDocumentRequest) error
	BulkDeleteFn                       func(ctx context.Context, req repositories.BulkDeleteDocumentRequest) error
}

func (m *mockDocRepo) List(
	ctx context.Context,
	req *repositories.ListDocumentsRequest,
) (*pagination.ListResult[*document.Document], error) {
	return m.ListFn(ctx, req)
}

func (m *mockDocRepo) GetByID(
	ctx context.Context,
	req repositories.GetDocumentByIDRequest,
) (*document.Document, error) {
	return m.GetByIDFn(ctx, req)
}

func (m *mockDocRepo) GetByIDs(
	ctx context.Context,
	req repositories.BulkDeleteDocumentRequest,
) ([]*document.Document, error) {
	return m.GetByIDsFn(ctx, req)
}

func (m *mockDocRepo) GetByResourceID(
	ctx context.Context,
	req *repositories.GetDocumentsByResourceRequest,
) ([]*document.Document, error) {
	return m.GetByResFn(ctx, req)
}

func (m *mockDocRepo) ListPendingPreviewReconciliation(
	ctx context.Context,
	olderThan int64,
	limit int,
) ([]*document.Document, error) {
	if m.ListPendingPreviewReconciliationFn == nil {
		return nil, nil
	}
	return m.ListPendingPreviewReconciliationFn(ctx, olderThan, limit)
}

func (m *mockDocRepo) Create(
	ctx context.Context,
	entity *document.Document,
) (*document.Document, error) {
	return m.CreateFn(ctx, entity)
}

func (m *mockDocRepo) Update(
	ctx context.Context,
	entity *document.Document,
) (*document.Document, error) {
	return m.UpdateFn(ctx, entity)
}

func (m *mockDocRepo) UpdatePreview(
	ctx context.Context,
	req *repositories.UpdateDocumentPreviewRequest,
) error {
	if m.UpdatePreviewFn == nil {
		return nil
	}
	return m.UpdatePreviewFn(ctx, req)
}

func (m *mockDocRepo) UpdateIntelligence(
	ctx context.Context,
	req *repositories.UpdateDocumentIntelligenceRequest,
) error {
	if m.UpdateIntelligenceFn == nil {
		return nil
	}
	return m.UpdateIntelligenceFn(ctx, req)
}

func (m *mockDocRepo) Delete(ctx context.Context, req repositories.DeleteDocumentRequest) error {
	return m.DeleteFn(ctx, req)
}

func (m *mockDocRepo) BulkDelete(
	ctx context.Context,
	req repositories.BulkDeleteDocumentRequest,
) error {
	return m.BulkDeleteFn(ctx, req)
}

type mockStorageClient struct {
	UploadFn                func(ctx context.Context, params *storage.UploadParams) (*storage.FileInfo, error)
	DownloadFn              func(ctx context.Context, key string) (*storage.DownloadResult, error)
	DeleteFn                func(ctx context.Context, key string) error
	GetPresignedURLFn       func(ctx context.Context, params *storage.PresignedURLParams) (string, error)
	GetPresignedUploadURLFn func(
		ctx context.Context,
		params *storage.PresignedUploadURLParams,
	) (string, error)
	InitiateMultipartUploadFn func(
		ctx context.Context,
		params *storage.MultipartUploadParams,
	) (string, error)
	GetMultipartUploadPartURLFn func(
		ctx context.Context,
		params *storage.MultipartUploadPartURLParams,
	) (string, error)
	CompleteMultipartUploadFn func(
		ctx context.Context,
		params *storage.CompleteMultipartUploadParams,
	) error
	AbortMultipartUploadFn func(
		ctx context.Context,
		params *storage.AbortMultipartUploadParams,
	) error
	ListMultipartUploadPartsFn func(
		ctx context.Context,
		params *storage.ListMultipartUploadPartsParams,
	) ([]storage.UploadedPart, error)
	ExistsFn      func(ctx context.Context, key string) (bool, error)
	GetFileInfoFn func(ctx context.Context, key string) (*storage.FileInfo, error)
}

func (m *mockStorageClient) Upload(
	ctx context.Context,
	params *storage.UploadParams,
) (*storage.FileInfo, error) {
	if m.UploadFn == nil {
		return &storage.FileInfo{Key: params.Key, Size: params.Size, ContentType: params.ContentType}, nil
	}
	return m.UploadFn(ctx, params)
}

func (m *mockStorageClient) Download(
	ctx context.Context,
	key string,
) (*storage.DownloadResult, error) {
	if m.DownloadFn == nil {
		return nil, nil
	}
	return m.DownloadFn(ctx, key)
}

func (m *mockStorageClient) Delete(ctx context.Context, key string) error {
	if m.DeleteFn == nil {
		return nil
	}
	return m.DeleteFn(ctx, key)
}

func (m *mockStorageClient) GetPresignedURL(
	ctx context.Context,
	params *storage.PresignedURLParams,
) (string, error) {
	if m.GetPresignedURLFn == nil {
		return "https://storage.example.com/presigned?token=abc123", nil
	}
	return m.GetPresignedURLFn(ctx, params)
}

func (m *mockStorageClient) Exists(ctx context.Context, key string) (bool, error) {
	if m.ExistsFn == nil {
		return true, nil
	}
	return m.ExistsFn(ctx, key)
}

func (m *mockStorageClient) GetPresignedUploadURL(
	ctx context.Context,
	params *storage.PresignedUploadURLParams,
) (string, error) {
	if m.GetPresignedUploadURLFn == nil {
		return "https://storage.example.com/upload?token=abc123", nil
	}
	return m.GetPresignedUploadURLFn(ctx, params)
}

func (m *mockStorageClient) InitiateMultipartUpload(
	ctx context.Context,
	params *storage.MultipartUploadParams,
) (string, error) {
	if m.InitiateMultipartUploadFn == nil {
		return "upload-id", nil
	}
	return m.InitiateMultipartUploadFn(ctx, params)
}

func (m *mockStorageClient) GetMultipartUploadPartURL(
	ctx context.Context,
	params *storage.MultipartUploadPartURLParams,
) (string, error) {
	if m.GetMultipartUploadPartURLFn == nil {
		return "https://storage.example.com/part?token=abc123", nil
	}
	return m.GetMultipartUploadPartURLFn(ctx, params)
}

func (m *mockStorageClient) CompleteMultipartUpload(
	ctx context.Context,
	params *storage.CompleteMultipartUploadParams,
) error {
	if m.CompleteMultipartUploadFn == nil {
		return nil
	}
	return m.CompleteMultipartUploadFn(ctx, params)
}

func (m *mockStorageClient) AbortMultipartUpload(
	ctx context.Context,
	params *storage.AbortMultipartUploadParams,
) error {
	if m.AbortMultipartUploadFn == nil {
		return nil
	}
	return m.AbortMultipartUploadFn(ctx, params)
}

func (m *mockStorageClient) ListMultipartUploadParts(
	ctx context.Context,
	params *storage.ListMultipartUploadPartsParams,
) ([]storage.UploadedPart, error) {
	if m.ListMultipartUploadPartsFn == nil {
		return nil, nil
	}
	return m.ListMultipartUploadPartsFn(ctx, params)
}

func (m *mockStorageClient) GetFileInfo(
	ctx context.Context,
	key string,
) (*storage.FileInfo, error) {
	if m.GetFileInfoFn == nil {
		return &storage.FileInfo{Key: key}, nil
	}
	return m.GetFileInfoFn(ctx, key)
}

var (
	_ storage.Client                  = (*mockStorageClient)(nil)
	_ repositories.DocumentRepository = (*mockDocRepo)(nil)
)

// func NewTestService(
// 	logger *zap.Logger,
// 	repo repositories.DocumentRepository,
// 	storageClient storage.Client,
// 	validator *documentservice.Validator,
// 	auditService services.AuditService,
// 	cfg *config.StorageConfig,
// 	thumbnailGenerator *thumbnailservice.Generator,
// 	temporalClient client.Client,
// ) *documentservice.Service {
// 	return &documentservice.Service{
// 		l:                  logger.Named("service.document"),
// 		repo:               repo,
// 		storage:            storageClient,
// 		validator:          validator,
// 		auditService:       auditService,
// 		config:             cfg,
// 		thumbnailGenerator: thumbnailGenerator,
// 		temporalClient:     temporalClient,
// 	}
// }

func newUnitTestService(t *testing.T, repo *mockDocRepo, sc *mockStorageClient) *documentservice.Service {
	storageCfg := &config.StorageConfig{
		AllowedMIMETypes:   []string{"application/pdf", "image/png", "text/plain"},
		MaxFileSize:        50 * 1024 * 1024,
		PresignedURLExpiry: 15 * time.Minute,
	}
	validator := documentservice.NewValidator(documentservice.ValidatorParams{
		Config: &config.Config{Storage: *storageCfg},
	})
	cacheRepo := mocks.NewMockDocumentCacheRepository(t)
	cacheRepo.On("GetByID", mock.Anything, mock.Anything).Maybe().Return(nil, repositories.ErrCacheMiss)
	sessionRepo := mocks.NewMockDocumentUploadSessionRepository(t)
	sessionRepo.On("ClearDocumentReference", mock.Anything, mock.Anything, mock.Anything).Maybe().Return(nil)
	sessionRepo.On("ClearDocumentReferences", mock.Anything, mock.Anything, mock.Anything).Maybe().Return(nil)

	return documentservice.NewTestService(
		zap.NewNop(),
		repo,
		cacheRepo,
		sessionRepo,
		sc,
		validator,
		&mocks.NoopAuditService{},
		storageCfg,
		thumbnailservice.NewGenerator(),
		nil,
	)
}

func TestUnit_List_Success(t *testing.T) {
	t.Parallel()

	expected := &pagination.ListResult[*document.Document]{
		Items: []*document.Document{
			{ID: pulid.MustNew("doc_"), OriginalName: "test.pdf"},
		},
		Total: 1,
	}

	repo := &mockDocRepo{
		ListFn: func(_ context.Context, _ *repositories.ListDocumentsRequest) (*pagination.ListResult[*document.Document], error) {
			return expected, nil
		},
	}
	svc := newUnitTestService(t, repo, &mockStorageClient{})

	result, err := svc.List(t.Context(), &repositories.ListDocumentsRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: pagination.TenantInfo{
				OrgID: pulid.MustNew("org_"),
				BuID:  pulid.MustNew("bu_"),
			},
			Pagination: pagination.Info{Limit: 10},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Items, 1)
}

func TestUnit_List_Error(t *testing.T) {
	t.Parallel()

	repo := &mockDocRepo{
		ListFn: func(_ context.Context, _ *repositories.ListDocumentsRequest) (*pagination.ListResult[*document.Document], error) {
			return nil, errors.New("db error")
		},
	}
	svc := newUnitTestService(t, repo, &mockStorageClient{})

	result, err := svc.List(t.Context(), &repositories.ListDocumentsRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: pagination.TenantInfo{
				OrgID: pulid.MustNew("org_"),
				BuID:  pulid.MustNew("bu_"),
			},
			Pagination: pagination.Info{Limit: 10},
		},
	})

	require.Error(t, err)
	assert.Nil(t, result)
}

func TestUnit_Get_Success(t *testing.T) {
	t.Parallel()

	doc := &document.Document{ID: pulid.MustNew("doc_"), OriginalName: "test.pdf"}
	repo := &mockDocRepo{
		GetByIDFn: func(_ context.Context, _ repositories.GetDocumentByIDRequest) (*document.Document, error) {
			return doc, nil
		},
	}
	svc := newUnitTestService(t, repo, &mockStorageClient{})

	result, err := svc.Get(t.Context(), repositories.GetDocumentByIDRequest{
		ID:         doc.ID,
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	})

	require.NoError(t, err)
	assert.Equal(t, doc.ID, result.ID)
}

func TestUnit_Get_NotFound(t *testing.T) {
	t.Parallel()

	repo := &mockDocRepo{
		GetByIDFn: func(_ context.Context, _ repositories.GetDocumentByIDRequest) (*document.Document, error) {
			return nil, errors.New("not found")
		},
	}
	svc := newUnitTestService(t, repo, &mockStorageClient{})

	result, err := svc.Get(t.Context(), repositories.GetDocumentByIDRequest{
		ID:         pulid.MustNew("doc_"),
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	})

	require.Error(t, err)
	assert.Nil(t, result)
}

func TestUnit_GetByResource_Success(t *testing.T) {
	t.Parallel()

	docs := []*document.Document{
		{ID: pulid.MustNew("doc_"), OriginalName: "a.pdf"},
		{ID: pulid.MustNew("doc_"), OriginalName: "b.pdf"},
	}
	repo := &mockDocRepo{
		GetByResFn: func(_ context.Context, _ *repositories.GetDocumentsByResourceRequest) ([]*document.Document, error) {
			return docs, nil
		},
	}
	svc := newUnitTestService(t, repo, &mockStorageClient{})

	result, err := svc.GetByResource(
		t.Context(),
		&repositories.GetDocumentsByResourceRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: pulid.MustNew("org_"),
				BuID:  pulid.MustNew("bu_"),
			},
			ResourceID:   "res_123",
			ResourceType: "trailer",
		},
	)

	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestUnit_Upload_Success(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")

	repo := &mockDocRepo{
		CreateFn: func(_ context.Context, entity *document.Document) (*document.Document, error) {
			entity.ID = pulid.MustNew("doc_")
			return entity, nil
		},
	}
	sc := &mockStorageClient{
		UploadFn: func(_ context.Context, _ *storage.UploadParams) (*storage.FileInfo, error) {
			return &storage.FileInfo{Key: "test-key"}, nil
		},
	}
	svc := newUnitTestService(t, repo, sc)

	fileData := []byte("pdf content here")
	fileHeader := storageutil.NewMockFileHeader("test.pdf", fileData, "application/pdf")

	result, err := svc.Upload(t.Context(), &documentservice.UploadRequest{
		TenantInfo:   pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
		File:         fileHeader,
		ResourceID:   "res_123",
		ResourceType: "trailer",
		Description:  "A test doc",
		Tags:         []string{"tag1"},
	})

	require.NoError(t, err)
	assert.NotNil(t, result.Document)
	assert.Equal(t, "test.pdf", result.Document.OriginalName)
	assert.Equal(t, document.StatusActive, result.Document.Status)
}

func TestUnit_Upload_ValidationFailure(t *testing.T) {
	t.Parallel()

	svc := newUnitTestService(t, &mockDocRepo{}, &mockStorageClient{})

	fileData := []byte("exe content")
	fileHeader := storageutil.NewMockFileHeader("malware.exe", fileData, "application/x-msdownload")

	result, err := svc.Upload(t.Context(), &documentservice.UploadRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
		File:         fileHeader,
		ResourceID:   "res_123",
		ResourceType: "trailer",
	})

	require.Error(t, err)
	assert.Nil(t, result)
}

func TestUnit_Upload_StorageError(t *testing.T) {
	t.Parallel()

	sc := &mockStorageClient{
		UploadFn: func(_ context.Context, _ *storage.UploadParams) (*storage.FileInfo, error) {
			return nil, errors.New("storage unavailable")
		},
	}
	svc := newUnitTestService(t, &mockDocRepo{}, sc)

	fileData := []byte("pdf content")
	fileHeader := storageutil.NewMockFileHeader("test.pdf", fileData, "application/pdf")

	result, err := svc.Upload(t.Context(), &documentservice.UploadRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
		File:         fileHeader,
		ResourceID:   "res_123",
		ResourceType: "trailer",
	})

	require.Error(t, err)
	assert.Nil(t, result)
}

func TestUnit_Upload_CreateRepoError_CleansUpStorage(t *testing.T) {
	t.Parallel()

	var deletedKey string
	sc := &mockStorageClient{
		UploadFn: func(_ context.Context, _ *storage.UploadParams) (*storage.FileInfo, error) {
			return &storage.FileInfo{Key: "uploaded-key"}, nil
		},
		DeleteFn: func(_ context.Context, key string) error {
			deletedKey = key
			return nil
		},
	}
	repo := &mockDocRepo{
		CreateFn: func(_ context.Context, _ *document.Document) (*document.Document, error) {
			return nil, errors.New("db error")
		},
	}
	svc := newUnitTestService(t, repo, sc)

	fileData := []byte("pdf content")
	fileHeader := storageutil.NewMockFileHeader("test.pdf", fileData, "application/pdf")

	result, err := svc.Upload(t.Context(), &documentservice.UploadRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
		File:         fileHeader,
		ResourceID:   "res_123",
		ResourceType: "trailer",
	})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.NotEmpty(t, deletedKey)
}

func TestUnit_BulkUpload_Success(t *testing.T) {
	t.Parallel()

	callCount := 0
	repo := &mockDocRepo{
		CreateFn: func(_ context.Context, entity *document.Document) (*document.Document, error) {
			callCount++
			entity.ID = pulid.MustNew("doc_")
			return entity, nil
		},
	}
	sc := &mockStorageClient{
		UploadFn: func(_ context.Context, _ *storage.UploadParams) (*storage.FileInfo, error) {
			return &storage.FileInfo{}, nil
		},
	}
	svc := newUnitTestService(t, repo, sc)

	files := storageutil.NewMockFileHeaders([]storageutil.MockFileHeader{
		{Filename: "a.pdf", Data: []byte("a"), ContentType: "application/pdf"},
		{Filename: "b.pdf", Data: []byte("b"), ContentType: "application/pdf"},
	})

	result, err := svc.BulkUpload(t.Context(), &documentservice.BulkUploadRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
		Files:        files,
		ResourceID:   "res_123",
		ResourceType: "trailer",
	})

	require.NoError(t, err)
	assert.Len(t, result.Documents, 2)
	assert.Len(t, result.Errors, 0)
	assert.Equal(t, 2, callCount)
}

func TestUnit_BulkUpload_PartialFailure(t *testing.T) {
	t.Parallel()

	createCount := 0
	repo := &mockDocRepo{
		CreateFn: func(_ context.Context, entity *document.Document) (*document.Document, error) {
			createCount++
			if createCount == 2 {
				return nil, errors.New("db error on second file")
			}
			entity.ID = pulid.MustNew("doc_")
			return entity, nil
		},
	}
	sc := &mockStorageClient{
		UploadFn: func(_ context.Context, _ *storage.UploadParams) (*storage.FileInfo, error) {
			return &storage.FileInfo{}, nil
		},
		DeleteFn: func(_ context.Context, _ string) error {
			return nil
		},
	}
	svc := newUnitTestService(t, repo, sc)

	files := storageutil.NewMockFileHeaders([]storageutil.MockFileHeader{
		{Filename: "a.pdf", Data: []byte("a"), ContentType: "application/pdf"},
		{Filename: "b.pdf", Data: []byte("b"), ContentType: "application/pdf"},
		{Filename: "c.pdf", Data: []byte("c"), ContentType: "application/pdf"},
	})

	result, err := svc.BulkUpload(t.Context(), &documentservice.BulkUploadRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
		Files:        files,
		ResourceID:   "res_123",
		ResourceType: "trailer",
	})

	require.NoError(t, err)
	assert.Len(t, result.Documents, 2)
	assert.Len(t, result.Errors, 1)
}

func TestUnit_GetDownloadURL_Success(t *testing.T) {
	t.Parallel()

	doc := &document.Document{
		ID:           pulid.MustNew("doc_"),
		StoragePath:  "org/trailer/file.pdf",
		OriginalName: "my-file.pdf",
	}
	repo := &mockDocRepo{
		GetByIDFn: func(_ context.Context, _ repositories.GetDocumentByIDRequest) (*document.Document, error) {
			return doc, nil
		},
	}
	sc := &mockStorageClient{
		GetPresignedURLFn: func(_ context.Context, params *storage.PresignedURLParams) (string, error) {
			assert.Equal(t, doc.StoragePath, params.Key)
			assert.Contains(t, params.ContentDisposition, "attachment")
			return "https://storage.example.com/signed-url", nil
		},
	}
	svc := newUnitTestService(t, repo, sc)

	url, err := svc.GetDownloadURL(t.Context(), repositories.GetDocumentByIDRequest{
		ID:         doc.ID,
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	})

	require.NoError(t, err)
	assert.Equal(t, "https://storage.example.com/signed-url", url)
}

func TestUnit_GetDownloadURL_DocNotFound(t *testing.T) {
	t.Parallel()

	repo := &mockDocRepo{
		GetByIDFn: func(_ context.Context, _ repositories.GetDocumentByIDRequest) (*document.Document, error) {
			return nil, errors.New("not found")
		},
	}
	svc := newUnitTestService(t, repo, &mockStorageClient{})

	url, err := svc.GetDownloadURL(t.Context(), repositories.GetDocumentByIDRequest{
		ID:         pulid.MustNew("doc_"),
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	})

	require.Error(t, err)
	assert.Empty(t, url)
}

func TestUnit_GetDownloadURL_PresignError(t *testing.T) {
	t.Parallel()

	doc := &document.Document{
		ID:           pulid.MustNew("doc_"),
		StoragePath:  "path/file.pdf",
		OriginalName: "file.pdf",
	}
	repo := &mockDocRepo{
		GetByIDFn: func(_ context.Context, _ repositories.GetDocumentByIDRequest) (*document.Document, error) {
			return doc, nil
		},
	}
	sc := &mockStorageClient{
		GetPresignedURLFn: func(_ context.Context, _ *storage.PresignedURLParams) (string, error) {
			return "", errors.New("presign failed")
		},
	}
	svc := newUnitTestService(t, repo, sc)

	url, err := svc.GetDownloadURL(t.Context(), repositories.GetDocumentByIDRequest{
		ID:         doc.ID,
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	})

	require.Error(t, err)
	assert.Empty(t, url)
}

func TestUnit_GetViewURL_Success(t *testing.T) {
	t.Parallel()

	doc := &document.Document{
		ID:           pulid.MustNew("doc_"),
		StoragePath:  "org/trailer/file.pdf",
		OriginalName: "view-file.pdf",
	}
	repo := &mockDocRepo{
		GetByIDFn: func(_ context.Context, _ repositories.GetDocumentByIDRequest) (*document.Document, error) {
			return doc, nil
		},
	}
	sc := &mockStorageClient{
		GetPresignedURLFn: func(_ context.Context, params *storage.PresignedURLParams) (string, error) {
			assert.Contains(t, params.ContentDisposition, "inline")
			return "https://storage.example.com/view-url", nil
		},
	}
	svc := newUnitTestService(t, repo, sc)

	url, err := svc.GetViewURL(t.Context(), repositories.GetDocumentByIDRequest{
		ID:         doc.ID,
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	})

	require.NoError(t, err)
	assert.Equal(t, "https://storage.example.com/view-url", url)
}

func TestUnit_GetViewURL_DocNotFound(t *testing.T) {
	t.Parallel()

	repo := &mockDocRepo{
		GetByIDFn: func(_ context.Context, _ repositories.GetDocumentByIDRequest) (*document.Document, error) {
			return nil, errors.New("not found")
		},
	}
	svc := newUnitTestService(t, repo, &mockStorageClient{})

	url, err := svc.GetViewURL(t.Context(), repositories.GetDocumentByIDRequest{
		ID:         pulid.MustNew("doc_"),
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	})

	require.Error(t, err)
	assert.Empty(t, url)
}

func TestUnit_GetViewURL_PresignError(t *testing.T) {
	t.Parallel()

	doc := &document.Document{
		ID:           pulid.MustNew("doc_"),
		StoragePath:  "path/file.pdf",
		OriginalName: "file.pdf",
	}
	repo := &mockDocRepo{
		GetByIDFn: func(_ context.Context, _ repositories.GetDocumentByIDRequest) (*document.Document, error) {
			return doc, nil
		},
	}
	sc := &mockStorageClient{
		GetPresignedURLFn: func(_ context.Context, _ *storage.PresignedURLParams) (string, error) {
			return "", errors.New("presign failed")
		},
	}
	svc := newUnitTestService(t, repo, sc)

	url, err := svc.GetViewURL(t.Context(), repositories.GetDocumentByIDRequest{
		ID:         doc.ID,
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	})

	require.Error(t, err)
	assert.Empty(t, url)
}

func TestUnit_Delete_Success(t *testing.T) {
	t.Parallel()

	doc := &document.Document{
		ID:             pulid.MustNew("doc_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		StoragePath:    "org/trailer/file.pdf",
		OriginalName:   "file.pdf",
	}
	var storageDeletions []string
	repo := &mockDocRepo{
		GetByIDFn: func(_ context.Context, _ repositories.GetDocumentByIDRequest) (*document.Document, error) {
			return doc, nil
		},
		DeleteFn: func(_ context.Context, _ repositories.DeleteDocumentRequest) error {
			return nil
		},
	}
	sc := &mockStorageClient{
		DeleteFn: func(_ context.Context, key string) error {
			storageDeletions = append(storageDeletions, key)
			return nil
		},
	}
	svc := newUnitTestService(t, repo, sc)

	err := svc.Delete(t.Context(), repositories.DeleteDocumentRequest{
		ID:         doc.ID,
		TenantInfo: pagination.TenantInfo{OrgID: doc.OrganizationID, BuID: doc.BusinessUnitID},
	}, pulid.MustNew("usr_"))

	require.NoError(t, err)
	assert.Contains(t, storageDeletions, doc.StoragePath)
}

func TestUnit_Delete_WithPreview(t *testing.T) {
	t.Parallel()

	doc := &document.Document{
		ID:                 pulid.MustNew("doc_"),
		OrganizationID:     pulid.MustNew("org_"),
		BusinessUnitID:     pulid.MustNew("bu_"),
		StoragePath:        "org/trailer/file.pdf",
		PreviewStoragePath: "org/trailer/preview.jpg",
		OriginalName:       "file.pdf",
	}
	var storageDeletions []string
	repo := &mockDocRepo{
		GetByIDFn: func(_ context.Context, _ repositories.GetDocumentByIDRequest) (*document.Document, error) {
			return doc, nil
		},
		DeleteFn: func(_ context.Context, _ repositories.DeleteDocumentRequest) error {
			return nil
		},
	}
	sc := &mockStorageClient{
		DeleteFn: func(_ context.Context, key string) error {
			storageDeletions = append(storageDeletions, key)
			return nil
		},
	}
	svc := newUnitTestService(t, repo, sc)

	err := svc.Delete(t.Context(), repositories.DeleteDocumentRequest{
		ID:         doc.ID,
		TenantInfo: pagination.TenantInfo{OrgID: doc.OrganizationID, BuID: doc.BusinessUnitID},
	}, pulid.MustNew("usr_"))

	require.NoError(t, err)
	assert.Contains(t, storageDeletions, doc.StoragePath)
	assert.Contains(t, storageDeletions, doc.PreviewStoragePath)
}

func TestUnit_Delete_NotFound(t *testing.T) {
	t.Parallel()

	repo := &mockDocRepo{
		GetByIDFn: func(_ context.Context, _ repositories.GetDocumentByIDRequest) (*document.Document, error) {
			return nil, errors.New("not found")
		},
	}
	svc := newUnitTestService(t, repo, &mockStorageClient{})

	err := svc.Delete(t.Context(), repositories.DeleteDocumentRequest{
		ID:         pulid.MustNew("doc_"),
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	}, pulid.MustNew("usr_"))

	require.Error(t, err)
}

func TestUnit_Delete_RepoDeleteError(t *testing.T) {
	t.Parallel()

	doc := &document.Document{
		ID:             pulid.MustNew("doc_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		StoragePath:    "path/file.pdf",
		OriginalName:   "file.pdf",
	}
	repo := &mockDocRepo{
		GetByIDFn: func(_ context.Context, _ repositories.GetDocumentByIDRequest) (*document.Document, error) {
			return doc, nil
		},
		DeleteFn: func(_ context.Context, _ repositories.DeleteDocumentRequest) error {
			return errors.New("db delete failed")
		},
	}
	sc := &mockStorageClient{
		DeleteFn: func(_ context.Context, _ string) error {
			return nil
		},
	}
	svc := newUnitTestService(t, repo, sc)

	err := svc.Delete(t.Context(), repositories.DeleteDocumentRequest{
		ID:         doc.ID,
		TenantInfo: pagination.TenantInfo{OrgID: doc.OrganizationID, BuID: doc.BusinessUnitID},
	}, pulid.MustNew("usr_"))

	require.Error(t, err)
	assert.Equal(t, "db delete failed", err.Error())
}

func TestUnit_BulkDelete_Success(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	id1 := pulid.MustNew("doc_")
	id2 := pulid.MustNew("doc_")
	docs := []*document.Document{
		{
			ID:             id1,
			OrganizationID: orgID,
			BusinessUnitID: buID,
			StoragePath:    "path/a.pdf",
			OriginalName:   "a.pdf",
		},
		{
			ID:             id2,
			OrganizationID: orgID,
			BusinessUnitID: buID,
			StoragePath:    "path/b.pdf",
			OriginalName:   "b.pdf",
		},
	}
	var storageDeletions []string
	repo := &mockDocRepo{
		GetByIDsFn: func(_ context.Context, _ repositories.BulkDeleteDocumentRequest) ([]*document.Document, error) {
			return docs, nil
		},
		BulkDeleteFn: func(_ context.Context, _ repositories.BulkDeleteDocumentRequest) error {
			return nil
		},
	}
	sc := &mockStorageClient{
		DeleteFn: func(_ context.Context, key string) error {
			storageDeletions = append(storageDeletions, key)
			return nil
		},
	}
	svc := newUnitTestService(t, repo, sc)

	result, err := svc.BulkDelete(t.Context(), &documentservice.BulkDeleteRequest{
		IDs:        []pulid.ID{id1, id2},
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		UserID:     pulid.MustNew("usr_"),
	})

	require.NoError(t, err)
	assert.Equal(t, 2, result.DeletedCount)
	assert.Len(t, result.Errors, 0)
	assert.Len(t, storageDeletions, 2)
}

func TestUnit_BulkDelete_EmptyIDs(t *testing.T) {
	t.Parallel()

	svc := newUnitTestService(t, &mockDocRepo{}, &mockStorageClient{})

	result, err := svc.BulkDelete(t.Context(), &documentservice.BulkDeleteRequest{
		IDs:        []pulid.ID{},
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		UserID:     pulid.MustNew("usr_"),
	})

	require.NoError(t, err)
	assert.Equal(t, 0, result.DeletedCount)
}

func TestUnit_BulkDelete_GetByIDsError(t *testing.T) {
	t.Parallel()

	repo := &mockDocRepo{
		GetByIDsFn: func(_ context.Context, _ repositories.BulkDeleteDocumentRequest) ([]*document.Document, error) {
			return nil, errors.New("db error")
		},
	}
	svc := newUnitTestService(t, repo, &mockStorageClient{})

	result, err := svc.BulkDelete(t.Context(), &documentservice.BulkDeleteRequest{
		IDs:        []pulid.ID{pulid.MustNew("doc_")},
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		UserID:     pulid.MustNew("usr_"),
	})

	require.Error(t, err)
	assert.Nil(t, result)
}

func TestUnit_BulkDelete_BulkDeleteRepoError(t *testing.T) {
	t.Parallel()

	docs := []*document.Document{
		{ID: pulid.MustNew("doc_"), StoragePath: "path/a.pdf", OriginalName: "a.pdf"},
	}
	repo := &mockDocRepo{
		GetByIDsFn: func(_ context.Context, _ repositories.BulkDeleteDocumentRequest) ([]*document.Document, error) {
			return docs, nil
		},
		BulkDeleteFn: func(_ context.Context, _ repositories.BulkDeleteDocumentRequest) error {
			return errors.New("bulk delete failed")
		},
	}
	sc := &mockStorageClient{
		DeleteFn: func(_ context.Context, _ string) error {
			return nil
		},
	}
	svc := newUnitTestService(t, repo, sc)

	result, err := svc.BulkDelete(t.Context(), &documentservice.BulkDeleteRequest{
		IDs:        []pulid.ID{pulid.MustNew("doc_")},
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		UserID:     pulid.MustNew("usr_"),
	})

	require.Error(t, err)
	assert.Nil(t, result)
}

func TestUnit_BulkDelete_StorageDeleteErrors(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	docs := []*document.Document{
		{
			ID:             pulid.MustNew("doc_"),
			OrganizationID: orgID,
			BusinessUnitID: buID,
			StoragePath:    "path/a.pdf",
			OriginalName:   "a.pdf",
		},
	}
	repo := &mockDocRepo{
		GetByIDsFn: func(_ context.Context, _ repositories.BulkDeleteDocumentRequest) ([]*document.Document, error) {
			return docs, nil
		},
		BulkDeleteFn: func(_ context.Context, _ repositories.BulkDeleteDocumentRequest) error {
			return nil
		},
	}
	sc := &mockStorageClient{
		DeleteFn: func(_ context.Context, _ string) error {
			return errors.New("storage delete error")
		},
	}
	svc := newUnitTestService(t, repo, sc)

	result, err := svc.BulkDelete(t.Context(), &documentservice.BulkDeleteRequest{
		IDs:        []pulid.ID{pulid.MustNew("doc_")},
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		UserID:     pulid.MustNew("usr_"),
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.DeletedCount)
	assert.Empty(t, result.Errors)
}

func TestUnit_GetPreviewURL_Success(t *testing.T) {
	t.Parallel()

	doc := &document.Document{
		ID:                 pulid.MustNew("doc_"),
		StoragePath:        "org/trailer/file.pdf",
		PreviewStoragePath: "org/trailer/preview.jpg",
		PreviewStatus:      document.PreviewStatusReady,
		OriginalName:       "file.pdf",
	}
	repo := &mockDocRepo{
		GetByIDFn: func(_ context.Context, _ repositories.GetDocumentByIDRequest) (*document.Document, error) {
			return doc, nil
		},
	}
	sc := &mockStorageClient{
		GetPresignedURLFn: func(_ context.Context, params *storage.PresignedURLParams) (string, error) {
			assert.Equal(t, doc.PreviewStoragePath, params.Key)
			return "https://storage.example.com/preview-url", nil
		},
	}
	svc := newUnitTestService(t, repo, sc)

	url, err := svc.GetPreviewURL(t.Context(), repositories.GetDocumentByIDRequest{
		ID:         doc.ID,
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	})

	require.NoError(t, err)
	assert.Equal(t, "https://storage.example.com/preview-url", url)
}

func TestUnit_GetPreviewURL_NoPreview(t *testing.T) {
	t.Parallel()

	doc := &document.Document{
		ID:                 pulid.MustNew("doc_"),
		StoragePath:        "org/trailer/file.pdf",
		PreviewStoragePath: "",
		OriginalName:       "file.pdf",
	}
	repo := &mockDocRepo{
		GetByIDFn: func(_ context.Context, _ repositories.GetDocumentByIDRequest) (*document.Document, error) {
			return doc, nil
		},
	}
	svc := newUnitTestService(t, repo, &mockStorageClient{})

	url, err := svc.GetPreviewURL(t.Context(), repositories.GetDocumentByIDRequest{
		ID:         doc.ID,
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	})

	require.Error(t, err)
	assert.Empty(t, url)
}

func TestUnit_GetPreviewURL_DocNotFound(t *testing.T) {
	t.Parallel()

	repo := &mockDocRepo{
		GetByIDFn: func(_ context.Context, _ repositories.GetDocumentByIDRequest) (*document.Document, error) {
			return nil, errors.New("not found")
		},
	}
	svc := newUnitTestService(t, repo, &mockStorageClient{})

	url, err := svc.GetPreviewURL(t.Context(), repositories.GetDocumentByIDRequest{
		ID:         pulid.MustNew("doc_"),
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	})

	require.Error(t, err)
	assert.Empty(t, url)
}

func TestUnit_GetPreviewURL_PresignError(t *testing.T) {
	t.Parallel()

	doc := &document.Document{
		ID:                 pulid.MustNew("doc_"),
		StoragePath:        "org/trailer/file.pdf",
		PreviewStoragePath: "org/trailer/preview.jpg",
		OriginalName:       "file.pdf",
	}
	repo := &mockDocRepo{
		GetByIDFn: func(_ context.Context, _ repositories.GetDocumentByIDRequest) (*document.Document, error) {
			return doc, nil
		},
	}
	sc := &mockStorageClient{
		GetPresignedURLFn: func(_ context.Context, _ *storage.PresignedURLParams) (string, error) {
			return "", errors.New("presign error")
		},
	}
	svc := newUnitTestService(t, repo, sc)

	url, err := svc.GetPreviewURL(t.Context(), repositories.GetDocumentByIDRequest{
		ID:         doc.ID,
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	})

	require.Error(t, err)
	assert.Empty(t, url)
}

func TestNew(t *testing.T) {
	t.Parallel()

	storageCfg := &config.StorageConfig{
		AllowedMIMETypes:   []string{"application/pdf"},
		MaxFileSize:        50 * 1024 * 1024,
		PresignedURLExpiry: 15 * time.Minute,
	}
	validator := documentservice.NewValidator(documentservice.ValidatorParams{
		Config: &config.Config{Storage: *storageCfg},
	})

	repo := &mockDocRepo{
		ListFn: func(ctx context.Context, req *repositories.ListDocumentsRequest) (*pagination.ListResult[*document.Document], error) {
			return nil, nil
		},
	}
	sc := &mockStorageClient{}
	cacheRepo := mocks.NewMockDocumentCacheRepository(t)
	cacheRepo.On("GetByID", mock.Anything, mock.Anything).Maybe().Return(nil, repositories.ErrCacheMiss)
	sessionRepo := mocks.NewMockDocumentUploadSessionRepository(t)
	sessionRepo.On("ClearDocumentReference", mock.Anything, mock.Anything, mock.Anything).Maybe().Return(nil)
	sessionRepo.On("ClearDocumentReferences", mock.Anything, mock.Anything, mock.Anything).Maybe().Return(nil)

	svc := documentservice.NewTestService(
		zap.NewNop(),
		repo,
		cacheRepo,
		sessionRepo,
		sc,
		validator,
		&mocks.NoopAuditService{},
		storageCfg,
		thumbnailservice.NewGenerator(),
		nil,
	)

	require.NotNil(t, svc)
}

func TestNewConstructor(t *testing.T) {
	t.Parallel()

	storageCfg := &config.StorageConfig{
		AllowedMIMETypes:   []string{"application/pdf"},
		MaxFileSize:        50 * 1024 * 1024,
		PresignedURLExpiry: 15 * time.Minute,
	}
	validator := documentservice.NewValidator(documentservice.ValidatorParams{
		Config: &config.Config{Storage: *storageCfg},
	})

	repo := &mockDocRepo{}
	sc := &mockStorageClient{}
	cacheRepo := mocks.NewMockDocumentCacheRepository(t)

	svc := documentservice.New(documentservice.Params{
		Logger:             zap.NewNop(),
		Repo:               repo,
		CacheRepo:          cacheRepo,
		Storage:            sc,
		Validator:          validator,
		AuditService:       &mocks.NoopAuditService{},
		Config:             &config.Config{Storage: *storageCfg},
		ThumbnailGenerator: thumbnailservice.NewGenerator(),
		TemporalClient:     nil,
	})

	require.NotNil(t, svc)
}

func TestUnit_Delete_WithPreviewPath(t *testing.T) {
	t.Parallel()

	doc := &document.Document{
		ID:                 pulid.MustNew("doc_"),
		OrganizationID:     pulid.MustNew("org_"),
		BusinessUnitID:     pulid.MustNew("bu_"),
		StoragePath:        "org/trailer/file.pdf",
		PreviewStoragePath: "org/trailer/preview.jpg",
		OriginalName:       "file.pdf",
	}
	var deletedPaths []string
	repo := &mockDocRepo{
		GetByIDFn: func(_ context.Context, _ repositories.GetDocumentByIDRequest) (*document.Document, error) {
			return doc, nil
		},
		DeleteFn: func(_ context.Context, _ repositories.DeleteDocumentRequest) error {
			return nil
		},
	}
	sc := &mockStorageClient{
		DeleteFn: func(_ context.Context, key string) error {
			deletedPaths = append(deletedPaths, key)
			return nil
		},
	}
	svc := newUnitTestService(t, repo, sc)

	err := svc.Delete(t.Context(), repositories.DeleteDocumentRequest{
		ID:         doc.ID,
		TenantInfo: pagination.TenantInfo{OrgID: doc.OrganizationID, BuID: doc.BusinessUnitID},
	}, pulid.MustNew("usr_"))

	require.NoError(t, err)
	assert.Contains(t, deletedPaths, doc.StoragePath)
	assert.Contains(t, deletedPaths, doc.PreviewStoragePath)
}

func TestUnit_Delete_StorageDeleteError(t *testing.T) {
	t.Parallel()

	doc := &document.Document{
		ID:                 pulid.MustNew("doc_"),
		OrganizationID:     pulid.MustNew("org_"),
		BusinessUnitID:     pulid.MustNew("bu_"),
		StoragePath:        "org/trailer/file.pdf",
		PreviewStoragePath: "",
		OriginalName:       "file.pdf",
	}
	repo := &mockDocRepo{
		GetByIDFn: func(_ context.Context, _ repositories.GetDocumentByIDRequest) (*document.Document, error) {
			return doc, nil
		},
		DeleteFn: func(_ context.Context, _ repositories.DeleteDocumentRequest) error {
			return nil
		},
	}
	sc := &mockStorageClient{
		DeleteFn: func(_ context.Context, _ string) error {
			return errors.New("storage error")
		},
	}
	svc := newUnitTestService(t, repo, sc)

	err := svc.Delete(t.Context(), repositories.DeleteDocumentRequest{
		ID:         doc.ID,
		TenantInfo: pagination.TenantInfo{OrgID: doc.OrganizationID, BuID: doc.BusinessUnitID},
	}, pulid.MustNew("usr_"))

	require.NoError(t, err)
}

func TestUnit_BulkDelete_WithPreviewPaths(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	docs := []*document.Document{
		{
			ID:                 pulid.MustNew("doc_"),
			OrganizationID:     orgID,
			BusinessUnitID:     buID,
			StoragePath:        "path/a.pdf",
			PreviewStoragePath: "path/a_thumb.jpg",
			OriginalName:       "a.pdf",
		},
	}
	var deletedPaths []string
	repo := &mockDocRepo{
		GetByIDsFn: func(_ context.Context, _ repositories.BulkDeleteDocumentRequest) ([]*document.Document, error) {
			return docs, nil
		},
		BulkDeleteFn: func(_ context.Context, _ repositories.BulkDeleteDocumentRequest) error {
			return nil
		},
	}
	sc := &mockStorageClient{
		DeleteFn: func(_ context.Context, key string) error {
			deletedPaths = append(deletedPaths, key)
			return nil
		},
	}
	svc := newUnitTestService(t, repo, sc)

	result, err := svc.BulkDelete(t.Context(), &documentservice.BulkDeleteRequest{
		IDs:        []pulid.ID{docs[0].ID},
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		UserID:     pulid.MustNew("usr_"),
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.DeletedCount)
	assert.Contains(t, deletedPaths, "path/a.pdf")
	assert.Contains(t, deletedPaths, "path/a_thumb.jpg")
}
