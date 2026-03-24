package accessorialchargehandler_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/accessorialchargehandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/services/accessorialchargeservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var errNotFound = errors.New("accessorial charge not found")

func setupAccessorialChargeHandler(
	t *testing.T,
	repo *mocks.MockAccessorialChargeRepository,
) *accessorialchargehandler.Handler {
	t.Helper()

	logger := zap.NewNop()

	service := accessorialchargeservice.New(accessorialchargeservice.Params{
		Logger:       logger,
		Repo:         repo,
		Validator:    accessorialchargeservice.NewTestValidator(),
		AuditService: &mocks.NoopAuditService{},
		Transformer:  &mocks.NoopDataTransformer{},
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
		PermissionEngine: &mocks.AllowAllPermissionEngine{},
		ErrorHandler:     errorHandler,
	})

	return accessorialchargehandler.New(accessorialchargehandler.Params{
		Service:              service,
		ErrorHandler:         errorHandler,
		PermissionMiddleware: pm,
	})
}

func TestAccessorialChargeHandler_List_Success(t *testing.T) {
	t.Parallel()

	accID := pulid.MustNew("acc_")
	repo := mocks.NewMockAccessorialChargeRepository(t)
	repo.On("List", mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*accessorialcharge.AccessorialCharge]{
			Items: []*accessorialcharge.AccessorialCharge{
				{
					ID:             accID,
					OrganizationID: testutil.TestOrgID,
					BusinessUnitID: testutil.TestBuID,
					Code:           "ACC01",
					Description:    "Test charge",
					Method:         accessorialcharge.MethodFlat,
					Status:         domaintypes.StatusActive,
				},
			},
			Total: 1,
		}, nil)

	handler := setupAccessorialChargeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/accessorial-charges/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp pagination.Response[[]map[string]any]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, 1, resp.Count)
	assert.Len(t, resp.Results, 1)
}

func TestAccessorialChargeHandler_List_WithPagination(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockAccessorialChargeRepository(t)
	repo.On("List", mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*accessorialcharge.AccessorialCharge]{
			Items: []*accessorialcharge.AccessorialCharge{},
			Total: 50,
		}, nil)

	handler := setupAccessorialChargeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/accessorial-charges/").
		WithQuery(map[string]string{"limit": "10", "offset": "20"}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp pagination.Response[[]map[string]any]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, 50, resp.Count)
}

func TestAccessorialChargeHandler_Get_Success(t *testing.T) {
	t.Parallel()

	accID := pulid.MustNew("acc_")
	repo := mocks.NewMockAccessorialChargeRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&accessorialcharge.AccessorialCharge{
		ID:             accID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		Code:           "ACC01",
		Description:    "Test charge",
		Method:         accessorialcharge.MethodFlat,
		Status:         domaintypes.StatusActive,
	}, nil)

	handler := setupAccessorialChargeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/accessorial-charges/" + accID.String() + "/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "ACC01", resp["code"])
}

func TestAccessorialChargeHandler_Get_NotFound(t *testing.T) {
	t.Parallel()

	accID := pulid.MustNew("acc_")
	repo := mocks.NewMockAccessorialChargeRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errNotFound)

	handler := setupAccessorialChargeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/accessorial-charges/" + accID.String() + "/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestAccessorialChargeHandler_Get_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockAccessorialChargeRepository(t)
	handler := setupAccessorialChargeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/accessorial-charges/invalid-id/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestAccessorialChargeHandler_Create_Success(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockAccessorialChargeRepository(t)
	repo.On("Create", mock.Anything, mock.Anything).
		Return(func(_ context.Context, entity *accessorialcharge.AccessorialCharge) *accessorialcharge.AccessorialCharge {
			entity.ID = pulid.MustNew("acc_")
			return entity
		}, nil)

	handler := setupAccessorialChargeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/accessorial-charges/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"code":        "ACC01",
			"description": "Test charge",
			"method":      "Flat",
			"status":      "Active",
			"amount":      75.00,
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusCreated, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "ACC01", resp["code"])
}

func TestAccessorialChargeHandler_Create_BadJSON(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockAccessorialChargeRepository(t)
	handler := setupAccessorialChargeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/accessorial-charges/").
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestAccessorialChargeHandler_Update_Success(t *testing.T) {
	t.Parallel()

	accID := pulid.MustNew("acc_")
	repo := mocks.NewMockAccessorialChargeRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&accessorialcharge.AccessorialCharge{
		ID:             accID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		Code:           "ACC01",
		Description:    "Test charge",
		Method:         accessorialcharge.MethodFlat,
		Status:         domaintypes.StatusActive,
	}, nil)
	repo.On("Update", mock.Anything, mock.Anything).
		Return(func(_ context.Context, entity *accessorialcharge.AccessorialCharge) *accessorialcharge.AccessorialCharge {
			return entity
		}, nil)

	handler := setupAccessorialChargeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/accessorial-charges/" + accID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"code":        "ACC02",
			"description": "Updated charge",
			"method":      "Flat",
			"status":      "Active",
			"amount":      75.00,
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "ACC02", resp["code"])
}

func TestAccessorialChargeHandler_Update_NotFound(t *testing.T) {
	t.Parallel()

	accID := pulid.MustNew("acc_")
	repo := mocks.NewMockAccessorialChargeRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errNotFound)

	handler := setupAccessorialChargeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/accessorial-charges/" + accID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"code":        "ACC02",
			"description": "Updated charge",
			"method":      "Flat",
			"status":      "Active",
			"amount":      75.00,
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestAccessorialChargeHandler_Patch_Success(t *testing.T) {
	t.Parallel()

	accID := pulid.MustNew("acc_")
	repo := mocks.NewMockAccessorialChargeRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&accessorialcharge.AccessorialCharge{
		ID:             accID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		Code:           "ACC01",
		Method:         accessorialcharge.MethodFlat,
		Status:         domaintypes.StatusActive,
	}, nil)
	repo.On("Update", mock.Anything, mock.Anything).
		Return(func(_ context.Context, entity *accessorialcharge.AccessorialCharge) *accessorialcharge.AccessorialCharge {
			return entity
		}, nil)

	handler := setupAccessorialChargeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/accessorial-charges/" + accID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"code":        "ACC03",
			"description": "Patched charge",
			"method":      "Flat",
			"status":      "Active",
			"amount":      75.00,
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestAccessorialChargeHandler_Patch_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockAccessorialChargeRepository(t)
	handler := setupAccessorialChargeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/accessorial-charges/invalid-id/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"code": "ACC03",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestAccessorialChargeHandler_SelectOptions_Success(t *testing.T) {
	t.Parallel()

	accID := pulid.MustNew("acc_")
	repo := mocks.NewMockAccessorialChargeRepository(t)
	repo.On("SelectOptions", mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*accessorialcharge.AccessorialCharge]{
			Items: []*accessorialcharge.AccessorialCharge{
				{
					ID:   accID,
					Code: "ACC01",
				},
			},
			Total: 1,
		}, nil)

	handler := setupAccessorialChargeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/accessorial-charges/select-options/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestAccessorialChargeHandler_GetOption_Success(t *testing.T) {
	t.Parallel()

	accID := pulid.MustNew("acc_")
	repo := mocks.NewMockAccessorialChargeRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&accessorialcharge.AccessorialCharge{
		ID:             accID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		Code:           "ACC01",
		Method:         accessorialcharge.MethodFlat,
		Status:         domaintypes.StatusActive,
	}, nil)

	handler := setupAccessorialChargeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/accessorial-charges/select-options/" + accID.String() + "/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestAccessorialChargeHandler_GetOption_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockAccessorialChargeRepository(t)
	handler := setupAccessorialChargeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/accessorial-charges/select-options/invalid-id/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestAccessorialChargeHandler_Create_ServiceError(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockAccessorialChargeRepository(t)
	repo.On("Create", mock.Anything, mock.Anything).Return(nil, errors.New("service error"))

	handler := setupAccessorialChargeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/accessorial-charges/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"code":        "ACC01",
			"description": "Test charge",
			"method":      "Flat",
			"status":      "Active",
			"amount":      75.00,
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestAccessorialChargeHandler_Update_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockAccessorialChargeRepository(t)
	handler := setupAccessorialChargeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/accessorial-charges/invalid-id/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"code":        "ACC02",
			"description": "Updated",
			"method":      "Flat",
			"status":      "Active",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestAccessorialChargeHandler_Patch_ServiceError(t *testing.T) {
	t.Parallel()

	accID := pulid.MustNew("acc_")
	repo := mocks.NewMockAccessorialChargeRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&accessorialcharge.AccessorialCharge{
		ID:             accID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		Code:           "ACC01",
		Method:         accessorialcharge.MethodFlat,
		Status:         domaintypes.StatusActive,
	}, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return(nil, errors.New("service error"))

	handler := setupAccessorialChargeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/accessorial-charges/" + accID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"code":        "ACC03",
			"description": "Patched charge",
			"method":      "Flat",
			"status":      "Active",
			"amount":      75.00,
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestAccessorialChargeHandler_GetOption_NotFound(t *testing.T) {
	t.Parallel()

	accID := pulid.MustNew("acc_")
	repo := mocks.NewMockAccessorialChargeRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errNotFound)

	handler := setupAccessorialChargeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/accessorial-charges/select-options/" + accID.String() + "/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}
