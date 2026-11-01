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
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/storageconfig"
	"github.com/emoss08/trenova/internal/modules/identity/app/organizationservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var errNotFound = errors.New("organization not found")

type mockStorageClient struct {
	uploadFn             func(ctx context.Context, params *kernelports.UploadParams) (*kernelports.FileInfo, error)
	downloadFn           func(ctx context.Context, key string) (*kernelports.DownloadResult, error)
	deleteFn             func(ctx context.Context, key string) error
	deleteObjectFn       func(ctx context.Context, params *kernelports.DeleteObjectParams) error
	getPresignedFn       func(ctx context.Context, params *kernelports.PresignedURLParams) (string, error)
	getPresignedUploadFn func(
		ctx context.Context,
		params *kernelports.PresignedUploadURLParams,
	) (string, error)
	initiateMultipartFn func(
		ctx context.Context,
		params *kernelports.MultipartUploadParams,
	) (string, error)
	getMultipartPartFn func(
		ctx context.Context,
		params *kernelports.MultipartUploadPartURLParams,
	) (string, error)
	completeMultipartFn func(
		ctx context.Context,
		params *kernelports.CompleteMultipartUploadParams,
	) error
	abortMultipartFn func(
		ctx context.Context,
		params *kernelports.AbortMultipartUploadParams,
	) error
	listMultipartPartsFn func(
		ctx context.Context,
		params *kernelports.ListMultipartUploadPartsParams,
	) ([]kernelports.UploadedPart, error)
	existsFn      func(ctx context.Context, key string) (bool, error)
	getFileInfoFn func(ctx context.Context, key string) (*kernelports.FileInfo, error)
}

func (m *mockStorageClient) Upload(
	ctx context.Context,
	params *kernelports.UploadParams,
) (*kernelports.FileInfo, error) {
	if m.uploadFn != nil {
		return m.uploadFn(ctx, params)
	}

	return &kernelports.FileInfo{
		Key:         params.Key,
		Size:        params.Size,
		ContentType: params.ContentType,
	}, nil
}

func (m *mockStorageClient) Download(
	ctx context.Context,
	key string,
) (*kernelports.DownloadResult, error) {
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

func (m *mockStorageClient) DeleteObject(
	ctx context.Context,
	params *kernelports.DeleteObjectParams,
) error {
	if m.deleteObjectFn != nil {
		return m.deleteObjectFn(ctx, params)
	}

	return m.Delete(ctx, params.Key)
}

func (m *mockStorageClient) GetPresignedURL(
	ctx context.Context,
	params *kernelports.PresignedURLParams,
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

func (m *mockStorageClient) GetPresignedUploadURL(
	ctx context.Context,
	params *kernelports.PresignedUploadURLParams,
) (string, error) {
	if m.getPresignedUploadFn != nil {
		return m.getPresignedUploadFn(ctx, params)
	}

	return "https://example.test/upload", nil
}

func (m *mockStorageClient) InitiateMultipartUpload(
	ctx context.Context,
	params *kernelports.MultipartUploadParams,
) (string, error) {
	if m.initiateMultipartFn != nil {
		return m.initiateMultipartFn(ctx, params)
	}

	return "upload-id", nil
}

func (m *mockStorageClient) GetMultipartUploadPartURL(
	ctx context.Context,
	params *kernelports.MultipartUploadPartURLParams,
) (string, error) {
	if m.getMultipartPartFn != nil {
		return m.getMultipartPartFn(ctx, params)
	}

	return "https://example.test/part", nil
}

func (m *mockStorageClient) CompleteMultipartUpload(
	ctx context.Context,
	params *kernelports.CompleteMultipartUploadParams,
) error {
	if m.completeMultipartFn != nil {
		return m.completeMultipartFn(ctx, params)
	}

	return nil
}

func (m *mockStorageClient) AbortMultipartUpload(
	ctx context.Context,
	params *kernelports.AbortMultipartUploadParams,
) error {
	if m.abortMultipartFn != nil {
		return m.abortMultipartFn(ctx, params)
	}

	return nil
}

func (m *mockStorageClient) ListMultipartUploadParts(
	ctx context.Context,
	params *kernelports.ListMultipartUploadPartsParams,
) ([]kernelports.UploadedPart, error) {
	if m.listMultipartPartsFn != nil {
		return m.listMultipartPartsFn(ctx, params)
	}

	return nil, nil
}

func (m *mockStorageClient) GetFileInfo(
	ctx context.Context,
	key string,
) (*kernelports.FileInfo, error) {
	if m.getFileInfoFn != nil {
		return m.getFileInfoFn(ctx, key)
	}

	return &kernelports.FileInfo{Key: key}, nil
}

func setupOrganizationHandler(
	t *testing.T,
	repo *tenantportsmock.MockOrganizationRepository,
	storageClient kernelports.Client,
) *organizationhandler.Handler {
	t.Helper()

	logger := zap.NewNop()

	cfg := &config.Config{
		App: appconfig.Config{
			Debug: true,
		},
		Storage: storageconfig.Config{
			MaxFileSize:        5 * 1024 * 1024,
			PresignedURLExpiry: 15 * time.Minute,
		},
	}

	service := organizationservice.New(organizationservice.Params{
		Logger:         logger,
		DB:             dbtest.NopConnection{},
		Repo:           repo,
		Storage:        storageClient,
		SecurityConfig: &cfg.Security,
		StorageConfig:  &cfg.Storage,
		Validator:      organizationservice.NewValidator(organizationservice.ValidatorParams{}),
		AuditService:   &mocks.NoopAuditService{},
	})

	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger:    logger,
		AppConfig: &cfg.App,
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

func TestOrganizationHandler_UploadLogo_Success(t *testing.T) {
	t.Parallel()

	orgID := testutil.TestOrgID
	stateID := pulid.MustNew("st_")

	repo := tenantportsmock.NewMockOrganizationRepository(t)
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

	repo := tenantportsmock.NewMockOrganizationRepository(t)
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

	repo := tenantportsmock.NewMockOrganizationRepository(t)
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

func TestOrganizationHandler_RejectsAnotherOrganizationID(t *testing.T) {
	t.Parallel()

	otherOrgID := pulid.MustNew("org_")

	cases := []struct {
		name   string
		method string
		path   string
		body   map[string]any
	}{
		{name: "get logo", method: http.MethodGet, path: "/logo"},
		{name: "delete logo", method: http.MethodDelete, path: "/logo"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := tenantportsmock.NewMockOrganizationRepository(t)
			handler := setupOrganizationHandler(t, repo, &mockStorageClient{})

			ginCtx := testutil.NewGinTestContext().
				WithMethod(tc.method).
				WithPath("/api/v1/organizations/" + otherOrgID.String() + tc.path).
				WithDefaultAuthContext()
			if tc.body != nil {
				ginCtx = ginCtx.WithJSONBody(tc.body)
			}

			handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
			ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

			assert.Equal(t, http.StatusNotFound, ginCtx.ResponseCode())
		})
	}
}
