package pagefavoriteservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/pagefavorite"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type mockPageFavoriteRepo struct {
	mock.Mock
}

func (m *mockPageFavoriteRepo) List(
	ctx context.Context,
	req *repositories.ListPageFavoritesRequest,
) ([]*pagefavorite.PageFavorite, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*pagefavorite.PageFavorite), args.Error(1)
}

func (m *mockPageFavoriteRepo) GetByURL(
	ctx context.Context,
	req *repositories.GetPageFavoriteByURLRequest,
) (*pagefavorite.PageFavorite, bool, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Bool(1), args.Error(2)
	}
	return args.Get(0).(*pagefavorite.PageFavorite), args.Bool(1), args.Error(2)
}

func (m *mockPageFavoriteRepo) Create(
	ctx context.Context,
	entity *pagefavorite.PageFavorite,
) (*pagefavorite.PageFavorite, error) {
	args := m.Called(ctx, entity)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pagefavorite.PageFavorite), args.Error(1)
}

func (m *mockPageFavoriteRepo) Delete(
	ctx context.Context,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	args := m.Called(ctx, id, tenantInfo)
	return args.Error(0)
}

type testDeps struct {
	repo *mockPageFavoriteRepo
	svc  *Service
}

func setupTest(t *testing.T) *testDeps {
	t.Helper()
	repo := new(mockPageFavoriteRepo)
	svc := &Service{
		l:    zap.NewNop(),
		repo: repo,
	}
	return &testDeps{repo: repo, svc: svc}
}

func newTestEntity() *pagefavorite.PageFavorite {
	return &pagefavorite.PageFavorite{
		ID:             pulid.MustNew("pf_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		UserID:         pulid.MustNew("usr_"),
		PageURL:        "/shipments",
		PageTitle:      "Shipments",
		Version:        1,
	}
}

func TestList_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	expected := []*pagefavorite.PageFavorite{newTestEntity()}
	req := &repositories.ListPageFavoritesRequest{
		UserID: pulid.MustNew("usr_"),
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
	}

	deps.repo.On("List", mock.Anything, req).Return(expected, nil)

	result, err := deps.svc.List(ctx, req)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	deps.repo.AssertExpectations(t)
}

func TestList_Error(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	req := &repositories.ListPageFavoritesRequest{
		UserID: pulid.MustNew("usr_"),
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
	}

	deps.repo.On("List", mock.Anything, req).Return(nil, errors.New("db error"))

	result, err := deps.svc.List(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.repo.AssertExpectations(t)
}

func TestToggle_CreateFavorite(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	userID := pulid.MustNew("usr_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	req := &ToggleRequest{
		PageURL:    "/shipments",
		PageTitle:  "Shipments",
		UserID:     userID,
		TenantInfo: tenantInfo,
	}

	created := &pagefavorite.PageFavorite{
		ID:             pulid.MustNew("pf_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		UserID:         userID,
		PageURL:        "/shipments",
		PageTitle:      "Shipments",
	}

	deps.repo.On("GetByURL", mock.Anything, &repositories.GetPageFavoriteByURLRequest{
		PageURL:    req.PageURL,
		UserID:     req.UserID,
		TenantInfo: req.TenantInfo,
	}).Return(nil, false, nil)
	deps.repo.On("Create", mock.Anything, mock.Anything).Return(created, nil)

	result, err := deps.svc.Toggle(ctx, req)

	require.NoError(t, err)
	assert.True(t, result.Favorited)
	assert.NotNil(t, result.Favorite)
	assert.Equal(t, created.ID, result.Favorite.ID)
	deps.repo.AssertExpectations(t)
}

func TestToggle_RemoveFavorite(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	userID := pulid.MustNew("usr_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	existing := newTestEntity()
	existing.UserID = userID
	existing.OrganizationID = tenantInfo.OrgID
	existing.BusinessUnitID = tenantInfo.BuID

	req := &ToggleRequest{
		PageURL:    existing.PageURL,
		PageTitle:  existing.PageTitle,
		UserID:     userID,
		TenantInfo: tenantInfo,
	}

	deps.repo.On("GetByURL", mock.Anything, &repositories.GetPageFavoriteByURLRequest{
		PageURL:    req.PageURL,
		UserID:     req.UserID,
		TenantInfo: req.TenantInfo,
	}).Return(existing, true, nil)
	deps.repo.On("Delete", mock.Anything, existing.ID, tenantInfo).Return(nil)

	result, err := deps.svc.Toggle(ctx, req)

	require.NoError(t, err)
	assert.False(t, result.Favorited)
	assert.Nil(t, result.Favorite)
	deps.repo.AssertExpectations(t)
}

func TestToggle_GetByURLError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	req := &ToggleRequest{
		PageURL:    "/shipments",
		PageTitle:  "Shipments",
		UserID:     pulid.MustNew("usr_"),
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	}

	deps.repo.On("GetByURL", mock.Anything, mock.Anything).
		Return(nil, false, errors.New("db error"))

	result, err := deps.svc.Toggle(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.repo.AssertExpectations(t)
}

func TestToggle_DeleteError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	userID := pulid.MustNew("usr_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	existing := newTestEntity()
	existing.UserID = userID
	existing.OrganizationID = tenantInfo.OrgID
	existing.BusinessUnitID = tenantInfo.BuID

	req := &ToggleRequest{
		PageURL:    existing.PageURL,
		PageTitle:  existing.PageTitle,
		UserID:     userID,
		TenantInfo: tenantInfo,
	}

	deps.repo.On("GetByURL", mock.Anything, &repositories.GetPageFavoriteByURLRequest{
		PageURL:    req.PageURL,
		UserID:     req.UserID,
		TenantInfo: req.TenantInfo,
	}).Return(existing, true, nil)
	deps.repo.On("Delete", mock.Anything, existing.ID, tenantInfo).
		Return(errors.New("delete failed"))

	result, err := deps.svc.Toggle(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.repo.AssertExpectations(t)
}

func TestToggle_CreateError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	req := &ToggleRequest{
		PageURL:    "/shipments",
		PageTitle:  "Shipments",
		UserID:     pulid.MustNew("usr_"),
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	}

	deps.repo.On("GetByURL", mock.Anything, mock.Anything).Return(nil, false, nil)
	deps.repo.On("Create", mock.Anything, mock.Anything).Return(nil, errors.New("create failed"))

	result, err := deps.svc.Toggle(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.repo.AssertExpectations(t)
}

func TestIsFavorited_True(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	userID := pulid.MustNew("usr_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	pageURL := "/shipments"

	existing := newTestEntity()

	deps.repo.On("GetByURL", mock.Anything, &repositories.GetPageFavoriteByURLRequest{
		PageURL:    pageURL,
		UserID:     userID,
		TenantInfo: tenantInfo,
	}).Return(existing, true, nil)

	result, err := deps.svc.IsFavorited(ctx, pageURL, userID, tenantInfo)

	require.NoError(t, err)
	assert.True(t, result)
	deps.repo.AssertExpectations(t)
}

func TestIsFavorited_False(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	userID := pulid.MustNew("usr_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	pageURL := "/shipments"

	deps.repo.On("GetByURL", mock.Anything, &repositories.GetPageFavoriteByURLRequest{
		PageURL:    pageURL,
		UserID:     userID,
		TenantInfo: tenantInfo,
	}).Return(nil, false, nil)

	result, err := deps.svc.IsFavorited(ctx, pageURL, userID, tenantInfo)

	require.NoError(t, err)
	assert.False(t, result)
	deps.repo.AssertExpectations(t)
}

func TestIsFavorited_Error(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	userID := pulid.MustNew("usr_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	pageURL := "/shipments"

	deps.repo.On("GetByURL", mock.Anything, mock.Anything).
		Return(nil, false, errors.New("db error"))

	result, err := deps.svc.IsFavorited(ctx, pageURL, userID, tenantInfo)

	require.Error(t, err)
	assert.False(t, result)
	deps.repo.AssertExpectations(t)
}

func TestNew(t *testing.T) {
	t.Parallel()

	repo := new(mockPageFavoriteRepo)

	svc := New(Params{
		Logger: zap.NewNop(),
		Repo:   repo,
	})

	require.NotNil(t, svc)
}
