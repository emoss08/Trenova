package organizationhandler_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/api/handlers/organizationhandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/internal/core/services/organizationservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var errNotFound = errors.New("organization not found")

type mockStorageClient struct {
	uploadFn       func(ctx context.Context, params *storage.UploadParams) (*storage.FileInfo, error)
	downloadFn     func(ctx context.Context, key string) (*storage.DownloadResult, error)
	deleteFn       func(ctx context.Context, key string) error
	getPresignedFn func(ctx context.Context, params *storage.PresignedURLParams) (string, error)
	existsFn       func(ctx context.Context, key string) (bool, error)
	getFileInfoFn  func(ctx context.Context, key string) (*storage.FileInfo, error)
}

func (m *mockStorageClient) Upload(
	ctx context.Context,
	params *storage.UploadParams,
) (*storage.FileInfo, error) {
	if m.uploadFn != nil {
		return m.uploadFn(ctx, params)
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
	if m.downloadFn != nil {
		return m.downloadFn(ctx, key)
	}

	return nil, nil
}

func (m *mockStorageClient) Delete(ctx context.Context, key string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, key)
	}

	return nil
}

func (m *mockStorageClient) GetPresignedURL(
	ctx context.Context,
	params *storage.PresignedURLParams,
) (string, error) {
	if m.getPresignedFn != nil {
		return m.getPresignedFn(ctx, params)
	}

	return "https://example.test/logo.png", nil
}

func (m *mockStorageClient) Exists(ctx context.Context, key string) (bool, error) {
	if m.existsFn != nil {
		return m.existsFn(ctx, key)
	}

	return true, nil
}

func (m *mockStorageClient) GetFileInfo(
	ctx context.Context,
	key string,
) (*storage.FileInfo, error) {
	if m.getFileInfoFn != nil {
		return m.getFileInfoFn(ctx, key)
	}

	return &storage.FileInfo{Key: key}, nil
}

func setupOrganizationHandler(
	t *testing.T,
	repo *mocks.MockOrganizationRepository,
	storageClient storage.Client,
) *organizationhandler.Handler {
	t.Helper()

	logger := zap.NewNop()

	cfg := &config.Config{
		App: config.AppConfig{
			Debug: true,
		},
		Storage: config.StorageConfig{
			MaxFileSize:        5 * 1024 * 1024,
			PresignedURLExpiry: 15 * time.Minute,
		},
	}

	service := organizationservice.New(organizationservice.Params{
		Logger:       logger,
		Repo:         repo,
		Storage:      storageClient,
		Config:       cfg,
		Validator:    organizationservice.NewValidator(organizationservice.ValidatorParams{}),
		AuditService: &mocks.NoopAuditService{},
	})

	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: logger,
		Config: cfg,
	})

	pm := middleware.NewPermissionMiddleware(middleware.PermissionMiddlewareParams{
		PermissionEngine: &mocks.AllowAllPermissionEngine{},
		ErrorHandler:     errorHandler,
	})

	return organizationhandler.New(organizationhandler.Params{
		Service:              service,
		ErrorHandler:         errorHandler,
		PermissionMiddleware: pm,
	})
}

func TestOrganizationHandler_Get_Success(t *testing.T) {
	t.Parallel()

	orgID := testutil.TestOrgID
	stateID := pulid.MustNew("st_")

	repo := mocks.NewMockOrganizationRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&tenant.Organization{
		ID:             orgID,
		BusinessUnitID: testutil.TestBuID,
		StateID:        stateID,
		Name:           "Test Org",
		ScacCode:       "ABCD",
		DOTNumber:      "1234567",
		City:           "New York",
		Timezone:       "America/New_York",
	}, nil)

	handler := setupOrganizationHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/organizations/" + orgID.String()).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "Test Org", resp["name"])
}

func TestOrganizationHandler_Get_NotFound(t *testing.T) {
	t.Parallel()

	orgID := testutil.TestOrgID

	repo := mocks.NewMockOrganizationRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errNotFound)

	handler := setupOrganizationHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/organizations/" + orgID.String()).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestOrganizationHandler_Get_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockOrganizationRepository(t)
	handler := setupOrganizationHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/organizations/invalid-id").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestOrganizationHandler_Update_Success(t *testing.T) {
	t.Parallel()

	orgID := testutil.TestOrgID
	stateID := pulid.MustNew("st_")

	repo := mocks.NewMockOrganizationRepository(t)
	repo.EXPECT().Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *tenant.Organization) (*tenant.Organization, error) {
			return entity, nil
		})

	handler := setupOrganizationHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/organizations/" + orgID.String()).
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":         "Updated Org",
			"scacCode":     "WXYZ",
			"dotNumber":    "7654321",
			"city":         "Los Angeles",
			"timezone":     "America/Los_Angeles",
			"stateId":      stateID.String(),
			"addressLine1": "123 Main St",
			"postalCode":   "90001",
			"bucketName":   "test-bucket",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "Updated Org", resp["name"])
}

func TestOrganizationHandler_Update_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockOrganizationRepository(t)
	handler := setupOrganizationHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/organizations/invalid-id").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name": "Updated Org",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestOrganizationHandler_Update_BadJSON(t *testing.T) {
	t.Parallel()

	orgID := testutil.TestOrgID
	repo := mocks.NewMockOrganizationRepository(t)
	handler := setupOrganizationHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/organizations/" + orgID.String()).
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestOrganizationHandler_UploadLogo_Success(t *testing.T) {
	t.Parallel()

	orgID := testutil.TestOrgID
	stateID := pulid.MustNew("st_")

	repo := mocks.NewMockOrganizationRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&tenant.Organization{
		ID:             orgID,
		BusinessUnitID: testutil.TestBuID,
		StateID:        stateID,
		Name:           "Test Org",
		ScacCode:       "ABCD",
		DOTNumber:      "1234567",
		AddressLine1:   "123 Main St",
		City:           "New York",
		PostalCode:     "10001",
		Timezone:       "America/New_York",
		BucketName:     "test-bucket",
		Version:        1,
	}, nil)
	repo.EXPECT().Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *tenant.Organization) (*tenant.Organization, error) {
			return entity, nil
		})

	handler := setupOrganizationHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/organizations/"+orgID.String()+"/logo").
		WithDefaultAuthContext().
		WithMultipartForm(
			nil,
			testutil.MultipartFile{
				FieldName: "file",
				Filename:  "logo.png",
				Data:      []byte("png-content"),
			},
		)

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestOrganizationHandler_GetLogoURL_Success(t *testing.T) {
	t.Parallel()

	orgID := testutil.TestOrgID
	stateID := pulid.MustNew("st_")

	repo := mocks.NewMockOrganizationRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&tenant.Organization{
		ID:             orgID,
		BusinessUnitID: testutil.TestBuID,
		StateID:        stateID,
		Name:           "Test Org",
		ScacCode:       "ABCD",
		DOTNumber:      "1234567",
		AddressLine1:   "123 Main St",
		City:           "New York",
		PostalCode:     "10001",
		Timezone:       "America/New_York",
		BucketName:     "test-bucket",
		LogoURL:        "org/logo/path.png",
		Version:        1,
	}, nil)

	handler := setupOrganizationHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/organizations/" + orgID.String() + "/logo").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]string
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.NotEmpty(t, resp["url"])
}

func TestOrganizationHandler_DeleteLogo_Success(t *testing.T) {
	t.Parallel()

	orgID := testutil.TestOrgID
	stateID := pulid.MustNew("st_")

	repo := mocks.NewMockOrganizationRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&tenant.Organization{
		ID:             orgID,
		BusinessUnitID: testutil.TestBuID,
		StateID:        stateID,
		Name:           "Test Org",
		ScacCode:       "ABCD",
		DOTNumber:      "1234567",
		AddressLine1:   "123 Main St",
		City:           "New York",
		PostalCode:     "10001",
		Timezone:       "America/New_York",
		BucketName:     "test-bucket",
		LogoURL:        "org/logo/path.webp",
		Version:        1,
	}, nil)
	repo.EXPECT().ClearLogoURL(mock.Anything, orgID, int64(1)).
		RunAndReturn(func(_ context.Context, _ pulid.ID, _ int64) (*tenant.Organization, error) {
			return &tenant.Organization{
				ID:             orgID,
				BusinessUnitID: testutil.TestBuID,
				StateID:        stateID,
				Name:           "Test Org",
				ScacCode:       "ABCD",
				DOTNumber:      "1234567",
				AddressLine1:   "123 Main St",
				City:           "New York",
				PostalCode:     "10001",
				Timezone:       "America/New_York",
				BucketName:     "test-bucket",
				LogoURL:        "",
				Version:        2,
			}, nil
		})

	handler := setupOrganizationHandler(t, repo, &mockStorageClient{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodDelete).
		WithPath("/api/v1/organizations/" + orgID.String() + "/logo").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "", resp["logoUrl"])
}
