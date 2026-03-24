package fleetcodehandler_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/fleetcodehandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/fleetcode"
	"github.com/emoss08/trenova/internal/core/services/fleetcodeservice"
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

var errNotFound = errors.New("fleet code not found")

func setupFleetCodeHandler(
	t *testing.T,
	repo *mocks.MockFleetCodeRepository,
) *fleetcodehandler.Handler {
	t.Helper()

	logger := zap.NewNop()

	service := fleetcodeservice.New(fleetcodeservice.Params{
		Logger:       logger,
		Repo:         repo,
		Validator:    fleetcodeservice.NewTestValidator(),
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

	return fleetcodehandler.New(fleetcodehandler.Params{
		Service:              service,
		ErrorHandler:         errorHandler,
		PermissionMiddleware: pm,
	})
}

func TestFleetCodeHandler_List_Success(t *testing.T) {
	t.Parallel()

	fcID := pulid.MustNew("fc_")
	repo := mocks.NewMockFleetCodeRepository(t)
	repo.On("List", mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*fleetcode.FleetCode]{
			Items: []*fleetcode.FleetCode{
				{
					ID:             fcID,
					OrganizationID: testutil.TestOrgID,
					BusinessUnitID: testutil.TestBuID,
					Code:           "FC01",
					Status:         domaintypes.StatusActive,
				},
			},
			Total: 1,
		}, nil)

	handler := setupFleetCodeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/fleet-codes/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp pagination.Response[[]map[string]any]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, 1, resp.Count)
	assert.Len(t, resp.Results, 1)
}

func TestFleetCodeHandler_List_WithPagination(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockFleetCodeRepository(t)
	repo.On("List", mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*fleetcode.FleetCode]{
			Items: []*fleetcode.FleetCode{},
			Total: 50,
		}, nil)

	handler := setupFleetCodeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/fleet-codes/").
		WithQuery(map[string]string{"limit": "10", "offset": "20"}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp pagination.Response[[]map[string]any]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, 50, resp.Count)
}

func TestFleetCodeHandler_Get_Success(t *testing.T) {
	t.Parallel()

	fcID := pulid.MustNew("fc_")
	repo := mocks.NewMockFleetCodeRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&fleetcode.FleetCode{
		ID:             fcID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		Code:           "FC01",
		Status:         domaintypes.StatusActive,
	}, nil)

	handler := setupFleetCodeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/fleet-codes/" + fcID.String()).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "FC01", resp["code"])
}

func TestFleetCodeHandler_Get_NotFound(t *testing.T) {
	t.Parallel()

	fcID := pulid.MustNew("fc_")
	repo := mocks.NewMockFleetCodeRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errNotFound)

	handler := setupFleetCodeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/fleet-codes/" + fcID.String()).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestFleetCodeHandler_Get_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockFleetCodeRepository(t)
	handler := setupFleetCodeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/fleet-codes/invalid-id").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestFleetCodeHandler_Create_Success(t *testing.T) {
	t.Parallel()

	managerID := pulid.MustNew("usr_")
	repo := mocks.NewMockFleetCodeRepository(t)
	repo.On("Create", mock.Anything, mock.Anything).
		Return(func(_ context.Context, entity *fleetcode.FleetCode) *fleetcode.FleetCode {
			entity.ID = pulid.MustNew("fc_")
			return entity
		}, nil)

	handler := setupFleetCodeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/fleet-codes/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"code":      "FC01",
			"status":    "Active",
			"managerId": managerID.String(),
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusCreated, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "FC01", resp["code"])
}

func TestFleetCodeHandler_Create_BadJSON(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockFleetCodeRepository(t)
	handler := setupFleetCodeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/fleet-codes/").
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestFleetCodeHandler_Update_Success(t *testing.T) {
	t.Parallel()

	fcID := pulid.MustNew("fc_")
	managerID := pulid.MustNew("usr_")
	repo := mocks.NewMockFleetCodeRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&fleetcode.FleetCode{
		ID:             fcID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		Code:           "FC01",
		ManagerID:      managerID,
		Status:         domaintypes.StatusActive,
	}, nil)
	repo.On("Update", mock.Anything, mock.Anything).
		Return(func(_ context.Context, entity *fleetcode.FleetCode) *fleetcode.FleetCode {
			return entity
		}, nil)

	handler := setupFleetCodeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/fleet-codes/" + fcID.String()).
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"code":      "FC02",
			"status":    "Active",
			"managerId": managerID.String(),
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "FC02", resp["code"])
}

func TestFleetCodeHandler_Update_NotFound(t *testing.T) {
	t.Parallel()

	fcID := pulid.MustNew("fc_")
	managerID := pulid.MustNew("usr_")
	repo := mocks.NewMockFleetCodeRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errNotFound)

	handler := setupFleetCodeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/fleet-codes/" + fcID.String()).
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"code":      "FC02",
			"status":    "Active",
			"managerId": managerID.String(),
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestFleetCodeHandler_Patch_Success(t *testing.T) {
	t.Parallel()

	fcID := pulid.MustNew("fc_")
	managerID := pulid.MustNew("usr_")
	repo := mocks.NewMockFleetCodeRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&fleetcode.FleetCode{
		ID:             fcID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		Code:           "FC01",
		Status:         domaintypes.StatusActive,
	}, nil)
	repo.On("Update", mock.Anything, mock.Anything).
		Return(func(_ context.Context, entity *fleetcode.FleetCode) *fleetcode.FleetCode {
			return entity
		}, nil)

	handler := setupFleetCodeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/fleet-codes/" + fcID.String()).
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"code":      "FC03",
			"status":    "Active",
			"managerId": managerID.String(),
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestFleetCodeHandler_Patch_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockFleetCodeRepository(t)
	handler := setupFleetCodeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/fleet-codes/invalid-id").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"code": "FC03",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestFleetCodeHandler_SelectOptions_Success(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockFleetCodeRepository(t)
	repo.On("SelectOptions", mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*fleetcode.FleetCode]{
			Items: []*fleetcode.FleetCode{},
			Total: 0,
		}, nil)

	handler := setupFleetCodeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/fleet-codes/select-options/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestFleetCodeHandler_GetOption_Success(t *testing.T) {
	t.Parallel()

	fcID := pulid.MustNew("fc_")
	repo := mocks.NewMockFleetCodeRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&fleetcode.FleetCode{
		ID:             fcID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		Code:           "FC01",
		Status:         domaintypes.StatusActive,
	}, nil)

	handler := setupFleetCodeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/fleet-codes/select-options/" + fcID.String()).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestFleetCodeHandler_GetOption_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockFleetCodeRepository(t)
	handler := setupFleetCodeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/fleet-codes/select-options/invalid-id").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestFleetCodeHandler_Create_ServiceError(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockFleetCodeRepository(t)
	repo.On("Create", mock.Anything, mock.Anything).Return(nil, errors.New("service error"))

	handler := setupFleetCodeHandler(t, repo)

	managerID := pulid.MustNew("usr_")
	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/fleet-codes/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"code":      "FC01",
			"status":    "Active",
			"managerId": managerID.String(),
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestFleetCodeHandler_Update_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockFleetCodeRepository(t)
	handler := setupFleetCodeHandler(t, repo)

	managerID := pulid.MustNew("usr_")
	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/fleet-codes/invalid-id").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"code":      "FC02",
			"status":    "Active",
			"managerId": managerID.String(),
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestFleetCodeHandler_Update_ServiceError(t *testing.T) {
	t.Parallel()

	fcID := pulid.MustNew("fc_")
	managerID := pulid.MustNew("usr_")
	repo := mocks.NewMockFleetCodeRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&fleetcode.FleetCode{
		ID:             fcID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		Code:           "FC01",
		ManagerID:      managerID,
		Status:         domaintypes.StatusActive,
	}, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return(nil, errors.New("service error"))

	handler := setupFleetCodeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/fleet-codes/" + fcID.String()).
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"code":      "FC02",
			"status":    "Active",
			"managerId": managerID.String(),
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestFleetCodeHandler_GetOption_NotFound(t *testing.T) {
	t.Parallel()

	fcID := pulid.MustNew("fc_")
	repo := mocks.NewMockFleetCodeRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errNotFound)

	handler := setupFleetCodeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/fleet-codes/select-options/" + fcID.String()).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}
