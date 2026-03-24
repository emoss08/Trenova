package pagefavoritehandler_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/pagefavoritehandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/pagefavorite"
	"github.com/emoss08/trenova/internal/core/services/pagefavoriteservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var errRepo = errors.New("repository error")

func setupPageFavoriteHandler(
	t *testing.T,
	repo *mocks.MockPageFavoriteRepository,
) *pagefavoritehandler.Handler {
	t.Helper()

	logger := zap.NewNop()

	service := pagefavoriteservice.New(pagefavoriteservice.Params{
		Logger: logger,
		Repo:   repo,
	})

	cfg := &config.Config{
		App: config.AppConfig{
			Debug: true,
		},
	}

	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: logger,
		Config: cfg,
	})

	return pagefavoritehandler.New(pagefavoritehandler.Params{
		Service:      service,
		ErrorHandler: errorHandler,
	})
}

func TestPageFavoriteHandler_List_Success(t *testing.T) {
	t.Parallel()

	pfID := pulid.MustNew("pf_")

	repo := mocks.NewMockPageFavoriteRepository(t)
	repo.On("List", mock.Anything, mock.Anything).Return([]*pagefavorite.PageFavorite{
		{
			ID:             pfID,
			OrganizationID: testutil.TestOrgID,
			BusinessUnitID: testutil.TestBuID,
			UserID:         testutil.TestUserID,
			PageURL:        "/dashboard",
			PageTitle:      "Dashboard",
		},
	}, nil)

	handler := setupPageFavoriteHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/page-favorites/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp []map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Len(t, resp, 1)
	assert.Equal(t, "/dashboard", resp[0]["pageUrl"])
}

func TestPageFavoriteHandler_List_Empty(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockPageFavoriteRepository(t)
	repo.On("List", mock.Anything, mock.Anything).Return([]*pagefavorite.PageFavorite{}, nil)

	handler := setupPageFavoriteHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/page-favorites/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp []map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Len(t, resp, 0)
}

func TestPageFavoriteHandler_List_RepoError(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockPageFavoriteRepository(t)
	repo.On("List", mock.Anything, mock.Anything).Return(nil, errRepo)

	handler := setupPageFavoriteHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/page-favorites/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestPageFavoriteHandler_Toggle_CreateFavorite(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockPageFavoriteRepository(t)
	repo.On("GetByURL", mock.Anything, mock.Anything).Return(nil, false, nil)
	repo.EXPECT().Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *pagefavorite.PageFavorite) (*pagefavorite.PageFavorite, error) {
			entity.ID = pulid.MustNew("pf_")
			return entity, nil
		})

	handler := setupPageFavoriteHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/page-favorites/toggle").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"pageUrl":   "/settings",
			"pageTitle": "Settings",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, true, resp["favorited"])
}

func TestPageFavoriteHandler_Toggle_DeleteFavorite(t *testing.T) {
	t.Parallel()

	pfID := pulid.MustNew("pf_")

	repo := mocks.NewMockPageFavoriteRepository(t)
	repo.On("GetByURL", mock.Anything, mock.Anything).Return(&pagefavorite.PageFavorite{
		ID:             pfID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		UserID:         testutil.TestUserID,
		PageURL:        "/settings",
		PageTitle:      "Settings",
	}, true, nil)
	repo.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	handler := setupPageFavoriteHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/page-favorites/toggle").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"pageUrl":   "/settings",
			"pageTitle": "Settings",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, false, resp["favorited"])
}

func TestPageFavoriteHandler_Toggle_BadJSON(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockPageFavoriteRepository(t)
	handler := setupPageFavoriteHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/page-favorites/toggle").
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestPageFavoriteHandler_Check_Favorited(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockPageFavoriteRepository(t)
	repo.On("GetByURL", mock.Anything, mock.Anything).
		Return(&pagefavorite.PageFavorite{}, true, nil)

	handler := setupPageFavoriteHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/page-favorites/check").
		WithQuery(map[string]string{"pageUrl": "/dashboard"}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, true, resp["favorited"])
}

func TestPageFavoriteHandler_Check_NotFavorited(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockPageFavoriteRepository(t)
	repo.On("GetByURL", mock.Anything, mock.Anything).Return(nil, false, nil)

	handler := setupPageFavoriteHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/page-favorites/check").
		WithQuery(map[string]string{"pageUrl": "/dashboard"}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, false, resp["favorited"])
}

func TestPageFavoriteHandler_Check_MissingPageURL(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockPageFavoriteRepository(t)
	handler := setupPageFavoriteHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/page-favorites/check").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}
