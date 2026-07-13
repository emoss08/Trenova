package ratetablehandler_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/ratetablehandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/ratetable"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/ratetableservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

//go:fix inline
func strPtr(s string) *string {
	return new(s)
}

func newTestEntity() *ratetable.RateTable {
	return &ratetable.RateTable{
		ID:             pulid.MustNew("rt_"),
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		Name:           "Lane Rates",
		Key:            "lane_rate",
		LookupType:     ratetable.LookupTypeExact,
		Active:         true,
		Version:        1,
		Entries: []*ratetable.RateTableEntry{
			{MatchKey: new("ATL-MIA"), Value: decimal.RequireFromString("1450")},
		},
	}
}

func setupHandler(t *testing.T, repo *mocks.MockRateTableRepository) *ratetablehandler.Handler {
	t.Helper()

	return setupHandlerWithPermissions(t, repo, &mocks.AllowAllPermissionEngine{})
}

func setupHandlerWithPermissions(
	t *testing.T,
	repo *mocks.MockRateTableRepository,
	permEngine serviceports.PermissionEngine,
) *ratetablehandler.Handler {
	t.Helper()

	logger := zap.NewNop()

	service := ratetableservice.New(ratetableservice.Params{
		Logger:       logger,
		Repo:         repo,
		Validator:    ratetableservice.NewValidator(ratetableservice.ValidatorParams{Repo: repo}),
		AuditService: &mocks.NoopAuditService{},
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

	pm := middleware.NewPermissionMiddleware(middleware.PermissionMiddlewareParams{
		PermissionEngine: permEngine,
		ErrorHandler:     errorHandler,
	})

	return ratetablehandler.New(ratetablehandler.Params{
		Service:              service,
		ErrorHandler:         errorHandler,
		PermissionMiddleware: pm,
	})
}

func TestRateTableHandler_List_Success(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockRateTableRepository(t)
	repo.On("List", mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*ratetable.RateTable]{
			Items: []*ratetable.RateTable{newTestEntity()},
			Total: 1,
		}, nil)

	handler := setupHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/rate-tables/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp pagination.Response[[]map[string]any]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, 1, resp.Count)
	assert.Len(t, resp.Results, 1)
}

func TestRateTableHandler_List_Filters(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockRateTableRepository(t)
	repo.On("List", mock.Anything, mock.MatchedBy(func(req *repositories.ListRateTablesRequest) bool {
		return req.LookupType == "Exact" && req.Active != nil && *req.Active
	})).
		Return(&pagination.ListResult[*ratetable.RateTable]{
			Items: []*ratetable.RateTable{},
			Total: 0,
		}, nil)

	handler := setupHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/rate-tables/").
		WithQuery(map[string]string{"lookupType": "Exact", "active": "true"}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	repo.AssertExpectations(t)
}

func TestRateTableHandler_List_Error(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockRateTableRepository(t)
	repo.On("List", mock.Anything, mock.Anything).Return(nil, errors.New("database error"))

	handler := setupHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/rate-tables/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestRateTableHandler_Get_Success(t *testing.T) {
	t.Parallel()

	entity := newTestEntity()
	repo := mocks.NewMockRateTableRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(entity, nil)

	handler := setupHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/rate-tables/" + entity.ID.String() + "/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "Lane Rates", resp["name"])
	assert.Len(t, resp["entries"], 1)
}

func TestRateTableHandler_Get_InvalidID(t *testing.T) {
	t.Parallel()

	handler := setupHandler(t, mocks.NewMockRateTableRepository(t))

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/rate-tables/invalid-id/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestRateTableHandler_Create_Success(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockRateTableRepository(t)
	repo.On("GetByKeys", mock.Anything, mock.Anything).Return([]*ratetable.RateTable{}, nil)
	repo.On("Create", mock.Anything, mock.Anything).
		Return(func(_ context.Context, entity *ratetable.RateTable) (*ratetable.RateTable, error) {
			entity.ID = pulid.MustNew("rt_")
			return entity, nil
		})

	handler := setupHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/rate-tables/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":       "Fuel Surcharge",
			"key":        "fuel_surcharge",
			"lookupType": "Range",
			"active":     true,
			"entries": []map[string]any{
				{"rangeMin": "0", "rangeMax": "3", "value": "0"},
				{"rangeMin": "3", "value": "0.12"},
			},
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusCreated, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "Fuel Surcharge", resp["name"])
}

func TestRateTableHandler_Create_ValidationError(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockRateTableRepository(t)
	repo.On("GetByKeys", mock.Anything, mock.Anything).Return([]*ratetable.RateTable{}, nil)

	handler := setupHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/rate-tables/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":       "Bad Table",
			"key":        "bad_table",
			"lookupType": "Exact",
			"entries": []map[string]any{
				{"value": "1"},
			},
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
	repo.AssertNotCalled(t, "Create")
}

func TestRateTableHandler_Create_BadJSON(t *testing.T) {
	t.Parallel()

	handler := setupHandler(t, mocks.NewMockRateTableRepository(t))

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/rate-tables/").
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestRateTableHandler_Create_PermissionDenied(t *testing.T) {
	t.Parallel()

	permEngine := mocks.NewMockPermissionEngine(t)
	permEngine.EXPECT().
		Check(mock.Anything, mock.Anything).
		Return(&serviceports.PermissionCheckResult{Allowed: false}, nil)

	repo := mocks.NewMockRateTableRepository(t)
	handler := setupHandlerWithPermissions(t, repo, permEngine)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/rate-tables/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":       "Fuel Surcharge",
			"key":        "fuel_surcharge",
			"lookupType": "Range",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusForbidden, ginCtx.ResponseCode())
	repo.AssertNotCalled(t, "Create")
}

func TestRateTableHandler_Update_Success(t *testing.T) {
	t.Parallel()

	entity := newTestEntity()
	repo := mocks.NewMockRateTableRepository(t)
	repo.On("GetByKeys", mock.Anything, mock.Anything).Return([]*ratetable.RateTable{}, nil)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(entity, nil)
	repo.On("Update", mock.Anything, mock.Anything).
		Return(func(_ context.Context, e *ratetable.RateTable) (*ratetable.RateTable, error) {
			return e, nil
		})

	handler := setupHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/rate-tables/" + entity.ID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":       "Updated Lane Rates",
			"key":        "lane_rate",
			"lookupType": "Exact",
			"active":     true,
			"version":    1,
			"entries": []map[string]any{
				{"matchKey": "ATL-MIA", "value": "1500"},
			},
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "Updated Lane Rates", resp["name"])
}

func TestRateTableHandler_Update_InvalidID(t *testing.T) {
	t.Parallel()

	handler := setupHandler(t, mocks.NewMockRateTableRepository(t))

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/rate-tables/invalid-id/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":       "Updated",
			"key":        "updated",
			"lookupType": "Exact",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestRateTableHandler_Delete_Success(t *testing.T) {
	t.Parallel()

	entity := newTestEntity()
	repo := mocks.NewMockRateTableRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(entity, nil)
	repo.On("Delete", mock.Anything, mock.Anything).Return(nil)

	handler := setupHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodDelete).
		WithPath("/api/v1/rate-tables/" + entity.ID.String() + "/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusNoContent, ginCtx.ResponseCode())
	repo.AssertExpectations(t)
}

func TestRateTableHandler_Delete_Error(t *testing.T) {
	t.Parallel()

	entity := newTestEntity()
	repo := mocks.NewMockRateTableRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errors.New("not found"))

	handler := setupHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodDelete).
		WithPath("/api/v1/rate-tables/" + entity.ID.String() + "/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
	repo.AssertNotCalled(t, "Delete")
}

func TestRateTableHandler_SelectOptions_Success(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockRateTableRepository(t)
	repo.On("SelectOptions", mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*ratetable.RateTable]{
			Items: []*ratetable.RateTable{newTestEntity()},
			Total: 1,
		}, nil)

	handler := setupHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/rate-tables/select-options/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestRateTableHandler_GetOption_Success(t *testing.T) {
	t.Parallel()

	entity := newTestEntity()
	repo := mocks.NewMockRateTableRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(entity, nil)

	handler := setupHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/rate-tables/select-options/" + entity.ID.String() + "/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}
