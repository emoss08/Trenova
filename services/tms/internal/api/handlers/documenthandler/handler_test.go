package documenthandler_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/api/handlers/documenthandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/internal/core/services/documentservice"
	"github.com/emoss08/trenova/internal/core/services/thumbnailservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
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

var errService = errors.New("service error")

type mockStorageClient struct {
	uploadFunc             func(ctx context.Context, params *storage.UploadParams) (*storage.FileInfo, error)
	downloadFunc           func(ctx context.Context, key string) (*storage.DownloadResult, error)
	deleteFunc             func(ctx context.Context, key string) error
	getPresignedFunc       func(ctx context.Context, params *storage.PresignedURLParams) (string, error)
	getPresignedUploadFunc func(
		ctx context.Context,
		params *storage.PresignedUploadURLParams,
	) (string, error)
	initiateMultipartFunc func(
		ctx context.Context,
		params *storage.MultipartUploadParams,
	) (string, error)
	getMultipartPartFunc func(
		ctx context.Context,
		params *storage.MultipartUploadPartURLParams,
	) (string, error)
	completeMultipartFunc func(
		ctx context.Context,
		params *storage.CompleteMultipartUploadParams,
	) error
	abortMultipartFunc func(
		ctx context.Context,
		params *storage.AbortMultipartUploadParams,
	) error
	listMultipartPartsFunc func(
		ctx context.Context,
		params *storage.ListMultipartUploadPartsParams,
	) ([]storage.UploadedPart, error)
	existsFunc      func(ctx context.Context, key string) (bool, error)
	getFileInfoFunc func(ctx context.Context, key string) (*storage.FileInfo, error)
}

func (m *mockStorageClient) Upload(
	ctx context.Context,
	params *storage.UploadParams,
) (*storage.FileInfo, error) {
	if m.uploadFunc != nil {
		return m.uploadFunc(ctx, params)
	}
	return &storage.FileInfo{
		Key:         params.Key,
		Size:        params.Size,
		ContentType: params.ContentType,
	}, nil
}

func (m *mockStorageClient) Download(
	ctx context.Context,
	key string,
) (*storage.DownloadResult, error) {
	if m.downloadFunc != nil {
		return m.downloadFunc(ctx, key)
	}
	return nil, nil
}

func (m *mockStorageClient) Delete(ctx context.Context, key string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, key)
	}
	return nil
}

func (m *mockStorageClient) GetPresignedURL(
	ctx context.Context,
	params *storage.PresignedURLParams,
) (string, error) {
	if m.getPresignedFunc != nil {
		return m.getPresignedFunc(ctx, params)
	}
	return "https://storage.example.com/presigned?token=abc123", nil
}

func (m *mockStorageClient) Exists(ctx context.Context, key string) (bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, key)
	}
	return true, nil
}

func (m *mockStorageClient) GetPresignedUploadURL(
	ctx context.Context,
	params *storage.PresignedUploadURLParams,
) (string, error) {
	if m.getPresignedUploadFunc != nil {
		return m.getPresignedUploadFunc(ctx, params)
	}
	return "https://storage.example.com/upload?token=abc123", nil
}

func (m *mockStorageClient) InitiateMultipartUpload(
	ctx context.Context,
	params *storage.MultipartUploadParams,
) (string, error) {
	if m.initiateMultipartFunc != nil {
		return m.initiateMultipartFunc(ctx, params)
	}
	return "upload-id", nil
}

func (m *mockStorageClient) GetMultipartUploadPartURL(
	ctx context.Context,
	params *storage.MultipartUploadPartURLParams,
) (string, error) {
	if m.getMultipartPartFunc != nil {
		return m.getMultipartPartFunc(ctx, params)
	}
	return "https://storage.example.com/part?token=abc123", nil
}

func (m *mockStorageClient) CompleteMultipartUpload(
	ctx context.Context,
	params *storage.CompleteMultipartUploadParams,
) error {
	if m.completeMultipartFunc != nil {
		return m.completeMultipartFunc(ctx, params)
	}
	return nil
}

func (m *mockStorageClient) AbortMultipartUpload(
	ctx context.Context,
	params *storage.AbortMultipartUploadParams,
) error {
	if m.abortMultipartFunc != nil {
		return m.abortMultipartFunc(ctx, params)
	}
	return nil
}

func (m *mockStorageClient) ListMultipartUploadParts(
	ctx context.Context,
	params *storage.ListMultipartUploadPartsParams,
) ([]storage.UploadedPart, error) {
	if m.listMultipartPartsFunc != nil {
		return m.listMultipartPartsFunc(ctx, params)
	}
	return nil, nil
}

func (m *mockStorageClient) GetFileInfo(
	ctx context.Context,
	key string,
) (*storage.FileInfo, error) {
	if m.getFileInfoFunc != nil {
		return m.getFileInfoFunc(ctx, key)
	}
	return &storage.FileInfo{Key: key}, nil
}

func setupHandler(
	t *testing.T,
	repo *mocks.MockDocumentRepository,
	storageClient *mockStorageClient,
) *documenthandler.Handler {
	t.Helper()

	logger := zap.NewNop()

	cfg := &config.Config{
		App: config.AppConfig{Debug: true},
		Storage: config.StorageConfig{
			MaxFileSize: 50 * 1024 * 1024,
			AllowedMIMETypes: []string{
				"application/pdf",
				"image/png",
				"application/octet-stream",
			},
			PresignedURLExpiry: 15 * time.Minute,
		},
	}

	validator := documentservice.NewValidator(documentservice.ValidatorParams{Config: cfg})
	thumbnailGen := thumbnailservice.NewGenerator()
	cacheRepo := mocks.NewMockDocumentCacheRepository(t)
	cacheRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(nil, repositories.ErrCacheMiss).
		Maybe()
	sessionRepo := mocks.NewMockDocumentUploadSessionRepository(t)
	sessionRepo.On("ClearDocumentReference", mock.Anything, mock.Anything, mock.Anything).Maybe().Return(nil)
	sessionRepo.On("ClearDocumentReferences", mock.Anything, mock.Anything, mock.Anything).Maybe().Return(nil)

	service := documentservice.NewTestService(
		logger,
		repo,
		cacheRepo,
		sessionRepo,
		storageClient,
		validator,
		&mocks.NoopAuditService{},
		cfg.GetStorageConfig(),
		thumbnailGen,
		nil,
	)

	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: logger,
		Config: cfg,
	})

	pm := middleware.NewPermissionMiddleware(middleware.PermissionMiddlewareParams{
		PermissionEngine: &mocks.AllowAllPermissionEngine{},
		ErrorHandler:     errorHandler,
	})

	return documenthandler.NewTestHandler(service, errorHandler, pm)
}

func TestDocumentHandler_List_Success(t *testing.T) {
	t.Parallel()

	docID := pulid.MustNew("doc_")

	repo := mocks.NewMockDocumentRepository(t)
	repo.On("List", mock.Anything, mock.Anything).Return(&pagination.ListResult[*document.Document]{
		Items: []*document.Document{
			{
				ID:             docID,
				OrganizationID: testutil.TestOrgID,
				BusinessUnitID: testutil.TestBuID,
				FileName:       "test.pdf",
				OriginalName:   "test.pdf",
				FileSize:       1024,
				FileType:       "application/pdf",
				StoragePath:    "org/trailer/test.pdf",
				Status:         document.StatusActive,
				ResourceID:     "res123",
				ResourceType:   "trailer",
			},
		},
		Total: 1,
	}, nil)

	handler := setupHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/documents/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp pagination.Response[[]map[string]any]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, 1, resp.Count)
	assert.Len(t, resp.Results, 1)
}

func TestDocumentHandler_List_WithFilters(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockDocumentRepository(t)
	repo.On("List", mock.Anything, mock.Anything).Return(&pagination.ListResult[*document.Document]{
		Items: []*document.Document{},
		Total: 0,
	}, nil)

	handler := setupHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/documents/").
		WithQuery(map[string]string{
			"resourceType": "trailer",
			"status":       "Active",
			"limit":        "10",
			"offset":       "0",
		}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestDocumentHandler_List_ServiceError(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockDocumentRepository(t)
	repo.On("List", mock.Anything, mock.Anything).Return(nil, errService)

	handler := setupHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/documents/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestDocumentHandler_Get_Success(t *testing.T) {
	t.Parallel()

	docID := pulid.MustNew("doc_")

	repo := mocks.NewMockDocumentRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req repositories.GetDocumentByIDRequest) (*document.Document, error) {
			return &document.Document{
				ID:             req.ID,
				OrganizationID: testutil.TestOrgID,
				BusinessUnitID: testutil.TestBuID,
				FileName:       "test.pdf",
				OriginalName:   "test.pdf",
				FileSize:       2048,
				FileType:       "application/pdf",
				StoragePath:    "org/trailer/test.pdf",
				Status:         document.StatusActive,
				ResourceID:     "res123",
				ResourceType:   "trailer",
			}, nil
		})

	handler := setupHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/documents/" + docID.String() + "/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "test.pdf", resp["originalName"])
}

func TestDocumentHandler_Get_NotFound(t *testing.T) {
	t.Parallel()

	docID := pulid.MustNew("doc_")

	repo := mocks.NewMockDocumentRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("Document not found"))

	handler := setupHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/documents/" + docID.String() + "/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusNotFound, ginCtx.ResponseCode())
}

func TestDocumentHandler_Get_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockDocumentRepository(t)
	handler := setupHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/documents/invalid-id/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestDocumentHandler_Download_Success(t *testing.T) {
	t.Parallel()

	docID := pulid.MustNew("doc_")

	repo := mocks.NewMockDocumentRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req repositories.GetDocumentByIDRequest) (*document.Document, error) {
			return &document.Document{
				ID:             req.ID,
				OrganizationID: testutil.TestOrgID,
				BusinessUnitID: testutil.TestBuID,
				FileName:       "download.pdf",
				OriginalName:   "download.pdf",
				StoragePath:    "org/trailer/download.pdf",
				Status:         document.StatusActive,
			}, nil
		})

	storageClient := &mockStorageClient{
		getPresignedFunc: func(_ context.Context, params *storage.PresignedURLParams) (string, error) {
			assert.Contains(t, params.ContentDisposition, "attachment")
			return "https://storage.example.com/download?token=xyz", nil
		},
	}

	handler := setupHandler(t, repo, storageClient)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/documents/" + docID.String() + "/download/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "https://storage.example.com/download?token=xyz", resp["url"])
}

func TestDocumentHandler_Download_NotFound(t *testing.T) {
	t.Parallel()

	docID := pulid.MustNew("doc_")

	repo := mocks.NewMockDocumentRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("Document not found"))

	handler := setupHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/documents/" + docID.String() + "/download/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusNotFound, ginCtx.ResponseCode())
}

func TestDocumentHandler_Download_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockDocumentRepository(t)
	handler := setupHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/documents/bad-id/download/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestDocumentHandler_Download_StorageError(t *testing.T) {
	t.Parallel()

	docID := pulid.MustNew("doc_")

	repo := mocks.NewMockDocumentRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req repositories.GetDocumentByIDRequest) (*document.Document, error) {
			return &document.Document{
				ID:          req.ID,
				StoragePath: "org/trailer/file.pdf",
			}, nil
		})

	storageClient := &mockStorageClient{
		getPresignedFunc: func(_ context.Context, _ *storage.PresignedURLParams) (string, error) {
			return "", errService
		},
	}

	handler := setupHandler(t, repo, storageClient)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/documents/" + docID.String() + "/download/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestDocumentHandler_View_Success(t *testing.T) {
	t.Parallel()

	docID := pulid.MustNew("doc_")

	repo := mocks.NewMockDocumentRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req repositories.GetDocumentByIDRequest) (*document.Document, error) {
			return &document.Document{
				ID:           req.ID,
				FileName:     "view.pdf",
				OriginalName: "view.pdf",
				StoragePath:  "org/trailer/view.pdf",
				Status:       document.StatusActive,
			}, nil
		})

	storageClient := &mockStorageClient{
		getPresignedFunc: func(_ context.Context, params *storage.PresignedURLParams) (string, error) {
			assert.Contains(t, params.ContentDisposition, "inline")
			return "https://storage.example.com/view?token=abc", nil
		},
	}

	handler := setupHandler(t, repo, storageClient)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/documents/" + docID.String() + "/view/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "https://storage.example.com/view?token=abc", resp["url"])
}

func TestDocumentHandler_View_NotFound(t *testing.T) {
	t.Parallel()

	docID := pulid.MustNew("doc_")

	repo := mocks.NewMockDocumentRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("Document not found"))

	handler := setupHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/documents/" + docID.String() + "/view/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusNotFound, ginCtx.ResponseCode())
}

func TestDocumentHandler_View_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockDocumentRepository(t)
	handler := setupHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/documents/bad-id/view/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestDocumentHandler_View_StorageError(t *testing.T) {
	t.Parallel()

	docID := pulid.MustNew("doc_")

	repo := mocks.NewMockDocumentRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req repositories.GetDocumentByIDRequest) (*document.Document, error) {
			return &document.Document{
				ID:          req.ID,
				StoragePath: "org/trailer/file.pdf",
			}, nil
		})

	storageClient := &mockStorageClient{
		getPresignedFunc: func(_ context.Context, _ *storage.PresignedURLParams) (string, error) {
			return "", errService
		},
	}

	handler := setupHandler(t, repo, storageClient)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/documents/" + docID.String() + "/view/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestDocumentHandler_Preview_Success(t *testing.T) {
	t.Parallel()

	docID := pulid.MustNew("doc_")

	repo := mocks.NewMockDocumentRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req repositories.GetDocumentByIDRequest) (*document.Document, error) {
			return &document.Document{
				ID:                 req.ID,
				FileName:           "preview.pdf",
				OriginalName:       "preview.pdf",
				StoragePath:        "org/trailer/preview.pdf",
				PreviewStoragePath: "org/trailer/preview-thumb.webp",
				PreviewStatus:      document.PreviewStatusReady,
				Status:             document.StatusActive,
			}, nil
		})

	storageClient := &mockStorageClient{
		getPresignedFunc: func(_ context.Context, params *storage.PresignedURLParams) (string, error) {
			assert.Equal(t, "org/trailer/preview-thumb.webp", params.Key)
			return "https://storage.example.com/preview?token=prev", nil
		},
	}

	handler := setupHandler(t, repo, storageClient)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/documents/" + docID.String() + "/preview/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "https://storage.example.com/preview?token=prev", resp["url"])
}

func TestDocumentHandler_Preview_NoPreviewAvailable(t *testing.T) {
	t.Parallel()

	docID := pulid.MustNew("doc_")

	repo := mocks.NewMockDocumentRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req repositories.GetDocumentByIDRequest) (*document.Document, error) {
			return &document.Document{
				ID:                 req.ID,
				StoragePath:        "org/trailer/file.pdf",
				PreviewStoragePath: "",
			}, nil
		})

	handler := setupHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/documents/" + docID.String() + "/preview/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusNotFound, ginCtx.ResponseCode())
}

func TestDocumentHandler_Preview_NotFound(t *testing.T) {
	t.Parallel()

	docID := pulid.MustNew("doc_")

	repo := mocks.NewMockDocumentRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("Document not found"))

	handler := setupHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/documents/" + docID.String() + "/preview/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusNotFound, ginCtx.ResponseCode())
}

func TestDocumentHandler_Preview_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockDocumentRepository(t)
	handler := setupHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/documents/bad-id/preview/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestDocumentHandler_Upload_Success(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockDocumentRepository(t)
	repo.EXPECT().Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *document.Document) (*document.Document, error) {
			entity.ID = pulid.MustNew("doc_")
			return entity, nil
		})

	handler := setupHandler(t, repo, &mockStorageClient{})

	resourceID := pulid.MustNew("tr_").String()
	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/documents/upload/").
		WithDefaultAuthContext().
		WithMultipartForm(
			map[string]string{
				"resourceId":   resourceID,
				"resourceType": "trailer",
				"description":  "Test upload",
			},
			testutil.MultipartFile{
				FieldName:   "file",
				Filename:    "upload-test.pdf",
				ContentType: "application/pdf",
				Data:        []byte("PDF content here"),
			},
		)

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusCreated, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "upload-test.pdf", resp["originalName"])
	assert.Equal(t, resourceID, resp["resourceId"])
	assert.Equal(t, "trailer", resp["resourceType"])
}

func TestDocumentHandler_Upload_MissingRequiredFields(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockDocumentRepository(t)
	handler := setupHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/documents/upload/").
		WithDefaultAuthContext().
		WithMultipartForm(
			map[string]string{},
			testutil.MultipartFile{
				FieldName:   "file",
				Filename:    "test.pdf",
				ContentType: "application/pdf",
				Data:        []byte("content"),
			},
		)

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestDocumentHandler_Upload_MissingFile(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockDocumentRepository(t)
	handler := setupHandler(t, repo, &mockStorageClient{})

	resourceID := pulid.MustNew("tr_").String()
	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/documents/upload/").
		WithDefaultAuthContext().
		WithMultipartForm(
			map[string]string{
				"resourceId":   resourceID,
				"resourceType": "trailer",
			},
		)

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestDocumentHandler_Upload_StorageError(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockDocumentRepository(t)
	storageClient := &mockStorageClient{
		uploadFunc: func(_ context.Context, _ *storage.UploadParams) (*storage.FileInfo, error) {
			return nil, errService
		},
	}

	handler := setupHandler(t, repo, storageClient)

	resourceID := pulid.MustNew("tr_").String()
	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/documents/upload/").
		WithDefaultAuthContext().
		WithMultipartForm(
			map[string]string{
				"resourceId":   resourceID,
				"resourceType": "trailer",
			},
			testutil.MultipartFile{
				FieldName:   "file",
				Filename:    "test.pdf",
				ContentType: "application/pdf",
				Data:        []byte("content"),
			},
		)

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestDocumentHandler_Upload_CreateError(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockDocumentRepository(t)
	repo.On("Create", mock.Anything, mock.Anything).Return(nil, errService)

	handler := setupHandler(t, repo, &mockStorageClient{})

	resourceID := pulid.MustNew("tr_").String()
	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/documents/upload/").
		WithDefaultAuthContext().
		WithMultipartForm(
			map[string]string{
				"resourceId":   resourceID,
				"resourceType": "trailer",
			},
			testutil.MultipartFile{
				FieldName:   "file",
				Filename:    "test.pdf",
				ContentType: "application/pdf",
				Data:        []byte("content"),
			},
		)

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestDocumentHandler_Upload_InvalidMIMEType(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockDocumentRepository(t)
	handler := setupHandler(t, repo, &mockStorageClient{})

	resourceID := pulid.MustNew("tr_").String()
	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/documents/upload/").
		WithDefaultAuthContext().
		WithMultipartForm(
			map[string]string{
				"resourceId":   resourceID,
				"resourceType": "trailer",
			},
			testutil.MultipartFile{
				FieldName:   "file",
				Filename:    "malware.exe",
				ContentType: "application/x-executable",
				Data:        []byte("bad content"),
			},
		)

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestDocumentHandler_UploadBulk_Success(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockDocumentRepository(t)
	repo.EXPECT().Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *document.Document) (*document.Document, error) {
			entity.ID = pulid.MustNew("doc_")
			return entity, nil
		})

	handler := setupHandler(t, repo, &mockStorageClient{})

	resourceID := pulid.MustNew("tr_").String()
	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/documents/upload-bulk/").
		WithDefaultAuthContext().
		WithMultipartFormFiles(
			map[string]string{
				"resourceId":   resourceID,
				"resourceType": "trailer",
			},
			"files",
			[]testutil.MultipartFile{
				{Filename: "bulk1.pdf", ContentType: "application/pdf", Data: []byte("content1")},
				{Filename: "bulk2.pdf", ContentType: "application/pdf", Data: []byte("content2")},
			},
		)

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusCreated, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, float64(2), resp["successCount"])
	assert.Equal(t, float64(0), resp["errorCount"])
}

func TestDocumentHandler_UploadBulk_MissingFields(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockDocumentRepository(t)
	handler := setupHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/documents/upload-bulk/").
		WithDefaultAuthContext().
		WithMultipartFormFiles(
			map[string]string{},
			"files",
			[]testutil.MultipartFile{
				{Filename: "bulk1.pdf", ContentType: "application/pdf", Data: []byte("content1")},
			},
		)

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestDocumentHandler_UploadBulk_NoFiles(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockDocumentRepository(t)
	handler := setupHandler(t, repo, &mockStorageClient{})

	resourceID := pulid.MustNew("tr_").String()
	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/documents/upload-bulk/").
		WithDefaultAuthContext().
		WithMultipartFormFiles(
			map[string]string{
				"resourceId":   resourceID,
				"resourceType": "trailer",
			},
			"files",
			[]testutil.MultipartFile{},
		)

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestDocumentHandler_Delete_Success(t *testing.T) {
	t.Parallel()

	docID := pulid.MustNew("doc_")

	repo := mocks.NewMockDocumentRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req repositories.GetDocumentByIDRequest) (*document.Document, error) {
			return &document.Document{
				ID:             req.ID,
				OrganizationID: testutil.TestOrgID,
				BusinessUnitID: testutil.TestBuID,
				StoragePath:    "org/trailer/test.pdf",
			}, nil
		})
	repo.On("Delete", mock.Anything, mock.Anything).Return(nil)

	handler := setupHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodDelete).
		WithPath("/api/v1/documents/" + docID.String() + "/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusNoContent, ginCtx.ResponseCode())
}

func TestDocumentHandler_Delete_NotFound(t *testing.T) {
	t.Parallel()

	docID := pulid.MustNew("doc_")

	repo := mocks.NewMockDocumentRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("Document not found"))

	handler := setupHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodDelete).
		WithPath("/api/v1/documents/" + docID.String() + "/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusNotFound, ginCtx.ResponseCode())
}

func TestDocumentHandler_Delete_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockDocumentRepository(t)
	handler := setupHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodDelete).
		WithPath("/api/v1/documents/invalid-id/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestDocumentHandler_Delete_RepoError(t *testing.T) {
	t.Parallel()

	docID := pulid.MustNew("doc_")

	repo := mocks.NewMockDocumentRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req repositories.GetDocumentByIDRequest) (*document.Document, error) {
			return &document.Document{
				ID:          req.ID,
				StoragePath: "org/trailer/test.pdf",
			}, nil
		})
	repo.On("Delete", mock.Anything, mock.Anything).Return(errService)

	handler := setupHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodDelete).
		WithPath("/api/v1/documents/" + docID.String() + "/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestDocumentHandler_BulkDelete_Success(t *testing.T) {
	t.Parallel()

	docID1 := pulid.MustNew("doc_")
	docID2 := pulid.MustNew("doc_")

	repo := mocks.NewMockDocumentRepository(t)
	repo.EXPECT().GetByIDs(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req repositories.BulkDeleteDocumentRequest) ([]*document.Document, error) {
			docs := make([]*document.Document, 0, len(req.IDs))
			for _, id := range req.IDs {
				docs = append(docs, &document.Document{
					ID:             id,
					OrganizationID: testutil.TestOrgID,
					BusinessUnitID: testutil.TestBuID,
					StoragePath:    "org/trailer/" + id.String() + ".pdf",
				})
			}
			return docs, nil
		})
	repo.On("BulkDelete", mock.Anything, mock.Anything).Return(nil)

	handler := setupHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/documents/bulk-delete/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"ids": []string{docID1.String(), docID2.String()},
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, float64(2), resp["deletedCount"])
	assert.Equal(t, float64(0), resp["errorCount"])
}

func TestDocumentHandler_BulkDelete_InvalidJSON(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockDocumentRepository(t)
	handler := setupHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/documents/bulk-delete/").
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestDocumentHandler_BulkDelete_EmptyIDs(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockDocumentRepository(t)
	handler := setupHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/documents/bulk-delete/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"ids": []string{},
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestDocumentHandler_BulkDelete_InvalidIDs(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockDocumentRepository(t)
	handler := setupHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/documents/bulk-delete/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"ids": []string{"invalid-id-format"},
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestDocumentHandler_BulkDelete_ServiceError(t *testing.T) {
	t.Parallel()

	docID := pulid.MustNew("doc_")

	repo := mocks.NewMockDocumentRepository(t)
	repo.On("GetByIDs", mock.Anything, mock.Anything).Return(nil, errService)

	handler := setupHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/documents/bulk-delete/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"ids": []string{docID.String()},
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestDocumentHandler_GetByResource_Success(t *testing.T) {
	t.Parallel()

	docID1 := pulid.MustNew("doc_")
	docID2 := pulid.MustNew("doc_")
	resourceID := pulid.MustNew("tr_").String()

	repo := mocks.NewMockDocumentRepository(t)
	repo.On("GetByResourceID", mock.Anything, mock.Anything).Return([]*document.Document{
		{
			ID:           docID1,
			FileName:     "doc1.pdf",
			OriginalName: "doc1.pdf",
			ResourceID:   resourceID,
			ResourceType: "trailer",
		},
		{
			ID:           docID2,
			FileName:     "doc2.pdf",
			OriginalName: "doc2.pdf",
			ResourceID:   resourceID,
			ResourceType: "trailer",
		},
	}, nil)

	handler := setupHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/documents/resource/trailer/" + resourceID + "/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp []map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Len(t, resp, 2)
}

func TestDocumentHandler_GetByResource_Empty(t *testing.T) {
	t.Parallel()

	resourceID := pulid.MustNew("tr_").String()

	repo := mocks.NewMockDocumentRepository(t)
	repo.On("GetByResourceID", mock.Anything, mock.Anything).Return([]*document.Document{}, nil)

	handler := setupHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/documents/resource/trailer/" + resourceID + "/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp []map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Len(t, resp, 0)
}

func TestDocumentHandler_GetByResource_ServiceError(t *testing.T) {
	t.Parallel()

	resourceID := pulid.MustNew("tr_").String()

	repo := mocks.NewMockDocumentRepository(t)
	repo.On("GetByResourceID", mock.Anything, mock.Anything).Return(nil, errService)

	handler := setupHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/documents/resource/trailer/" + resourceID + "/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestDocumentHandler_Upload_WithTags(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockDocumentRepository(t)
	repo.EXPECT().Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *document.Document) (*document.Document, error) {
			entity.ID = pulid.MustNew("doc_")
			return entity, nil
		})

	handler := setupHandler(t, repo, &mockStorageClient{})

	resourceID := pulid.MustNew("tr_").String()
	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/documents/upload/").
		WithDefaultAuthContext().
		WithMultipartForm(
			map[string]string{
				"resourceId":   resourceID,
				"resourceType": "trailer",
				"description":  "Tagged document",
				"tags":         "important",
			},
			testutil.MultipartFile{
				FieldName:   "file",
				Filename:    "tagged.pdf",
				ContentType: "application/pdf",
				Data:        []byte("tagged content"),
			},
		)

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusCreated, ginCtx.ResponseCode())
}

func TestDocumentHandler_Delete_WithPreviewPath(t *testing.T) {
	t.Parallel()

	docID := pulid.MustNew("doc_")
	deletedPaths := make([]string, 0)

	repo := mocks.NewMockDocumentRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req repositories.GetDocumentByIDRequest) (*document.Document, error) {
			return &document.Document{
				ID:                 req.ID,
				OrganizationID:     testutil.TestOrgID,
				BusinessUnitID:     testutil.TestBuID,
				StoragePath:        "org/trailer/test.pdf",
				PreviewStoragePath: "org/trailer/test-thumb.webp",
			}, nil
		})
	repo.On("Delete", mock.Anything, mock.Anything).Return(nil)

	storageClient := &mockStorageClient{
		deleteFunc: func(_ context.Context, key string) error {
			deletedPaths = append(deletedPaths, key)
			return nil
		},
	}

	handler := setupHandler(t, repo, storageClient)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodDelete).
		WithPath("/api/v1/documents/" + docID.String() + "/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusNoContent, ginCtx.ResponseCode())
	assert.Contains(t, deletedPaths, "org/trailer/test.pdf")
	assert.Contains(t, deletedPaths, "org/trailer/test-thumb.webp")
}

func TestDocumentHandler_Preview_StorageError(t *testing.T) {
	t.Parallel()

	docID := pulid.MustNew("doc_")

	repo := mocks.NewMockDocumentRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req repositories.GetDocumentByIDRequest) (*document.Document, error) {
			return &document.Document{
				ID:                 req.ID,
				StoragePath:        "org/trailer/file.pdf",
				PreviewStoragePath: "org/trailer/thumb.webp",
			}, nil
		})

	storageClient := &mockStorageClient{
		getPresignedFunc: func(_ context.Context, _ *storage.PresignedURLParams) (string, error) {
			return "", errService
		},
	}

	handler := setupHandler(t, repo, storageClient)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/documents/" + docID.String() + "/preview/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestDocumentHandler_List_WithResourceIDFilter(t *testing.T) {
	t.Parallel()

	resourceID := pulid.MustNew("tr_").String()

	repo := mocks.NewMockDocumentRepository(t)
	repo.On("List", mock.Anything, mock.Anything).Return(&pagination.ListResult[*document.Document]{
		Items: []*document.Document{},
		Total: 0,
	}, nil)

	handler := setupHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/documents/").
		WithQuery(map[string]string{
			"resourceId": resourceID,
		}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestDocumentHandler_BulkDelete_BulkDeleteRepoError(t *testing.T) {
	t.Parallel()

	docID := pulid.MustNew("doc_")

	repo := mocks.NewMockDocumentRepository(t)
	repo.EXPECT().GetByIDs(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req repositories.BulkDeleteDocumentRequest) ([]*document.Document, error) {
			docs := make([]*document.Document, 0, len(req.IDs))
			for _, id := range req.IDs {
				docs = append(docs, &document.Document{
					ID:          id,
					StoragePath: "org/trailer/" + id.String() + ".pdf",
				})
			}
			return docs, nil
		})
	repo.On("BulkDelete", mock.Anything, mock.Anything).Return(errService)

	handler := setupHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/documents/bulk-delete/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"ids": []string{docID.String()},
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}
