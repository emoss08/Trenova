package organizationservice

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type mockOrganizationRepo struct {
	mock.Mock
}

func (m *mockOrganizationRepo) GetByID(
	ctx context.Context,
	req repositories.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tenant.Organization), args.Error(1)
}

func (m *mockOrganizationRepo) Update(
	ctx context.Context,
	entity *tenant.Organization,
) (*tenant.Organization, error) {
	args := m.Called(ctx, entity)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tenant.Organization), args.Error(1)
}

func (m *mockOrganizationRepo) ClearLogoURL(
	ctx context.Context,
	orgID pulid.ID,
	version int64,
) (*tenant.Organization, error) {
	args := m.Called(ctx, orgID, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tenant.Organization), args.Error(1)
}

type testDeps struct {
	repo *mockOrganizationRepo
	svc  *service
}

type noopStorageClient struct{}

func (n *noopStorageClient) Upload(
	_ context.Context,
	params *storage.UploadParams,
) (*storage.FileInfo, error) {
	return &storage.FileInfo{
		Key:         params.Key,
		Size:        params.Size,
		ContentType: params.ContentType,
	}, nil
}

func (n *noopStorageClient) Download(_ context.Context, _ string) (*storage.DownloadResult, error) {
	return &storage.DownloadResult{Body: io.NopCloser(bytes.NewReader(nil))}, nil
}

func (n *noopStorageClient) Delete(_ context.Context, _ string) error {
	return nil
}

func (n *noopStorageClient) GetPresignedURL(
	_ context.Context,
	_ *storage.PresignedURLParams,
) (string, error) {
	return "https://example.test/logo.png", nil
}

func (n *noopStorageClient) Exists(_ context.Context, _ string) (bool, error) {
	return true, nil
}

func (n *noopStorageClient) GetPresignedUploadURL(
	_ context.Context,
	_ *storage.PresignedUploadURLParams,
) (string, error) {
	return "https://example.test/upload", nil
}

func (n *noopStorageClient) InitiateMultipartUpload(
	_ context.Context,
	_ *storage.MultipartUploadParams,
) (string, error) {
	return "upload-id", nil
}

func (n *noopStorageClient) GetMultipartUploadPartURL(
	_ context.Context,
	_ *storage.MultipartUploadPartURLParams,
) (string, error) {
	return "https://example.test/part", nil
}

func (n *noopStorageClient) CompleteMultipartUpload(
	_ context.Context,
	_ *storage.CompleteMultipartUploadParams,
) error {
	return nil
}

func (n *noopStorageClient) AbortMultipartUpload(
	_ context.Context,
	_ *storage.AbortMultipartUploadParams,
) error {
	return nil
}

func (n *noopStorageClient) ListMultipartUploadParts(
	_ context.Context,
	_ *storage.ListMultipartUploadPartsParams,
) ([]storage.UploadedPart, error) {
	return nil, nil
}

func (n *noopStorageClient) GetFileInfo(_ context.Context, key string) (*storage.FileInfo, error) {
	return &storage.FileInfo{Key: key}, nil
}

func setupTest(t *testing.T) *testDeps {
	t.Helper()
	repo := new(mockOrganizationRepo)
	svc := &service{
		l:          zap.NewNop(),
		repo:       repo,
		storage:    &noopStorageClient{},
		storageCfg: &config.StorageConfig{PresignedURLExpiry: 15 * time.Minute},
		v:          NewValidator(ValidatorParams{}),
	}
	return &testDeps{repo: repo, svc: svc}
}

func newTestOrganization() *tenant.Organization {
	return &tenant.Organization{
		ID:             pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		StateID:        pulid.MustNew("st_"),
		Name:           "Test Organization",
		ScacCode:       "TEST",
		DOTNumber:      "1234567",
		BucketName:     "test-bucket",
		AddressLine1:   "123 Main St",
		City:           "Springfield",
		PostalCode:     "12345",
		Timezone:       "America/New_York",
		Version:        1,
	}
}

func TestGetByID_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	org := newTestOrganization()

	req := repositories.GetOrganizationByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: org.ID,
			BuID:  org.BusinessUnitID,
		},
		IncludeState: true,
	}

	deps.repo.On("GetByID", mock.Anything, req).Return(org, nil)

	result, err := deps.svc.GetByID(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, org.ID, result.ID)
	assert.Equal(t, org.Name, result.Name)
	deps.repo.AssertExpectations(t)
}

func TestGetByID_NotFound(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	req := repositories.GetOrganizationByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
	}

	notFoundErr := errors.New("organization not found")
	deps.repo.On("GetByID", mock.Anything, req).Return(nil, notFoundErr)

	result, err := deps.svc.GetByID(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, notFoundErr, err)
	deps.repo.AssertExpectations(t)
}

func TestUpdate_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newTestOrganization()

	updated := newTestOrganization()
	updated.ID = entity.ID
	updated.Name = "Updated Organization"

	deps.repo.On("Update", mock.Anything, entity).Return(updated, nil)

	result, err := deps.svc.Update(ctx, entity)

	require.NoError(t, err)
	assert.Equal(t, updated.ID, result.ID)
	assert.Equal(t, "Updated Organization", result.Name)
	deps.repo.AssertExpectations(t)
}

func TestUpdate_ValidationError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	entity := &tenant.Organization{
		ID:             pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		StateID:        pulid.MustNew("st_"),
		Name:           "",
		ScacCode:       "",
		DOTNumber:      "",
		City:           "",
		Timezone:       "",
	}

	result, err := deps.svc.Update(ctx, entity)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.repo.AssertNotCalled(t, "Update")
}

func TestUpdate_RepoError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newTestOrganization()

	repoErr := errors.New("database error")
	deps.repo.On("Update", mock.Anything, entity).Return(nil, repoErr)

	result, err := deps.svc.Update(ctx, entity)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, repoErr, err)
	deps.repo.AssertExpectations(t)
}

func TestUpdate_RepoUniqueConstraintNameMapped(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		constraint string
		field      string
	}{
		{
			name:       "name constraint",
			constraint: "idx_organizations_name_business_unit",
			field:      "name",
		},
		{
			name:       "scac constraint",
			constraint: "idx_organizations_scac_business_unit",
			field:      "scacCode",
		},
		{
			name:       "dot constraint",
			constraint: "idx_organizations_dot_business_unit",
			field:      "dotNumber",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			deps := setupTest(t)
			ctx := t.Context()
			entity := newTestOrganization()

			repoErr := &pgconn.PgError{Code: "23505", ConstraintName: tt.constraint}
			deps.repo.On("Update", mock.Anything, entity).Return(nil, repoErr)

			result, err := deps.svc.Update(ctx, entity)

			require.Error(t, err)
			assert.Nil(t, result)

			var multiErr *errortypes.MultiError
			require.True(t, errors.As(err, &multiErr))
			require.NotEmpty(t, multiErr.Errors)
			assert.Equal(t, tt.field, multiErr.Errors[0].Field)
			assert.Equal(t, errortypes.ErrDuplicate, multiErr.Errors[0].Code)
			deps.repo.AssertExpectations(t)
		})
	}
}

func TestUpdate_RepoUniqueConstraintUnknownUnchanged(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newTestOrganization()

	repoErr := &pgconn.PgError{Code: "23505", ConstraintName: "idx_unknown"}
	deps.repo.On("Update", mock.Anything, entity).Return(nil, repoErr)

	result, err := deps.svc.Update(ctx, entity)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, repoErr, err)
	deps.repo.AssertExpectations(t)
}

func TestDeleteLogo_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	org := newTestOrganization()
	org.LogoURL = "org/logo/path.webp"

	cleared := newTestOrganization()
	cleared.ID = org.ID
	cleared.Version = org.Version + 1
	cleared.LogoURL = ""

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(org, nil)
	deps.repo.On("ClearLogoURL", mock.Anything, org.ID, org.Version).Return(cleared, nil)

	result, err := deps.svc.DeleteLogo(ctx, services.DeleteLogoRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: org.ID,
			BuID:  org.BusinessUnitID,
		},
		OrganizationID: org.ID,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "", result.LogoURL)
	deps.repo.AssertExpectations(t)
}

func TestDeleteLogo_ExternalURL_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	org := newTestOrganization()
	org.LogoURL = "https://cdn.example.com/logo.webp"

	cleared := newTestOrganization()
	cleared.ID = org.ID
	cleared.Version = org.Version + 1
	cleared.LogoURL = ""

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(org, nil)
	deps.repo.On("ClearLogoURL", mock.Anything, org.ID, org.Version).Return(cleared, nil)

	result, err := deps.svc.DeleteLogo(ctx, services.DeleteLogoRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: org.ID,
			BuID:  org.BusinessUnitID,
		},
		OrganizationID: org.ID,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "", result.LogoURL)
	deps.repo.AssertExpectations(t)
}

func TestNew(t *testing.T) {
	t.Parallel()

	repo := new(mockOrganizationRepo)

	svc := New(Params{
		Logger:    zap.NewNop(),
		Repo:      repo,
		Storage:   &noopStorageClient{},
		Config:    &config.Config{},
		Validator: NewValidator(ValidatorParams{}),
	})

	require.NotNil(t, svc)
}
