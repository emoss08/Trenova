package equipmentmanufacturerhandler_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/equipmentmanufacturerhandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/core/services/equipmentmanufacturerservice"
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

var errNotFound = errors.New("equipment manufacturer not found")

func setupEquipManufacturerHandler(
	t *testing.T,
	repo *mocks.MockEquipmentManufacturerRepository,
) *equipmentmanufacturerhandler.Handler {
	t.Helper()

	logger := zap.NewNop()

	service := equipmentmanufacturerservice.New(equipmentmanufacturerservice.Params{
		Logger:       logger,
		Repo:         repo,
		Validator:    equipmentmanufacturerservice.NewTestValidator(),
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
		PermissionEngine: &mocks.AllowAllPermissionEngine{},
		ErrorHandler:     errorHandler,
	})

	return equipmentmanufacturerhandler.New(equipmentmanufacturerhandler.Params{
		Service:              service,
		ErrorHandler:         errorHandler,
		PermissionMiddleware: pm,
	})
}

func TestEquipmentManufacturerHandler_List_Success(t *testing.T) {
	t.Parallel()

	emID := pulid.MustNew("em_")
	repo := mocks.NewMockEquipmentManufacturerRepository(t)
	repo.On("List", mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*equipmentmanufacturer.EquipmentManufacturer]{
			Items: []*equipmentmanufacturer.EquipmentManufacturer{
				{
					ID:             emID,
					OrganizationID: testutil.TestOrgID,
					BusinessUnitID: testutil.TestBuID,
					Name:           "Freightliner",
					Status:         domaintypes.StatusActive,
				},
			},
			Total: 1,
		}, nil)

	handler := setupEquipManufacturerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/equipment-manufacturers/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp pagination.Response[[]map[string]any]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, 1, resp.Count)
	assert.Len(t, resp.Results, 1)
}

func TestEquipmentManufacturerHandler_List_WithPagination(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockEquipmentManufacturerRepository(t)
	repo.On("List", mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*equipmentmanufacturer.EquipmentManufacturer]{
			Items: []*equipmentmanufacturer.EquipmentManufacturer{},
			Total: 75,
		}, nil)

	handler := setupEquipManufacturerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/equipment-manufacturers/").
		WithQuery(map[string]string{"limit": "25", "offset": "50"}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp pagination.Response[[]map[string]any]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, 75, resp.Count)
}

func TestEquipmentManufacturerHandler_Get_Success(t *testing.T) {
	t.Parallel()

	emID := pulid.MustNew("em_")
	repo := mocks.NewMockEquipmentManufacturerRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).
		Return(&equipmentmanufacturer.EquipmentManufacturer{
			ID:             emID,
			OrganizationID: testutil.TestOrgID,
			BusinessUnitID: testutil.TestBuID,
			Name:           "Kenworth",
			Status:         domaintypes.StatusActive,
		}, nil)

	handler := setupEquipManufacturerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/equipment-manufacturers/" + emID.String() + "/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "Kenworth", resp["name"])
}

func TestEquipmentManufacturerHandler_Get_NotFound(t *testing.T) {
	t.Parallel()

	emID := pulid.MustNew("em_")
	repo := mocks.NewMockEquipmentManufacturerRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errNotFound)

	handler := setupEquipManufacturerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/equipment-manufacturers/" + emID.String() + "/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestEquipmentManufacturerHandler_Get_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockEquipmentManufacturerRepository(t)
	handler := setupEquipManufacturerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/equipment-manufacturers/invalid-id/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestEquipmentManufacturerHandler_Create_Success(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockEquipmentManufacturerRepository(t)
	repo.On("Create", mock.Anything, mock.Anything).
		Return(func(_ context.Context, entity *equipmentmanufacturer.EquipmentManufacturer) *equipmentmanufacturer.EquipmentManufacturer {
			entity.ID = pulid.MustNew("em_")
			return entity
		}, nil)

	handler := setupEquipManufacturerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/equipment-manufacturers/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":   "Peterbilt",
			"status": "Active",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusCreated, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "Peterbilt", resp["name"])
}

func TestEquipmentManufacturerHandler_Create_BadJSON(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockEquipmentManufacturerRepository(t)
	handler := setupEquipManufacturerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/equipment-manufacturers/").
		WithDefaultAuthContext().
		WithBody("{not valid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestEquipmentManufacturerHandler_Update_Success(t *testing.T) {
	t.Parallel()

	emID := pulid.MustNew("em_")
	repo := mocks.NewMockEquipmentManufacturerRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).
		Return(&equipmentmanufacturer.EquipmentManufacturer{
			ID:             emID,
			OrganizationID: testutil.TestOrgID,
			BusinessUnitID: testutil.TestBuID,
			Name:           "Freightliner",
			Status:         domaintypes.StatusActive,
		}, nil)
	repo.On("Update", mock.Anything, mock.Anything).
		Return(func(_ context.Context, entity *equipmentmanufacturer.EquipmentManufacturer) *equipmentmanufacturer.EquipmentManufacturer {
			return entity
		}, nil)

	handler := setupEquipManufacturerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/equipment-manufacturers/" + emID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":   "Kenworth",
			"status": "Active",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "Kenworth", resp["name"])
}

func TestEquipmentManufacturerHandler_Update_NotFound(t *testing.T) {
	t.Parallel()

	emID := pulid.MustNew("em_")
	repo := mocks.NewMockEquipmentManufacturerRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errNotFound)

	handler := setupEquipManufacturerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/equipment-manufacturers/" + emID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":   "Kenworth",
			"status": "Active",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestEquipmentManufacturerHandler_Patch_Success(t *testing.T) {
	t.Parallel()

	emID := pulid.MustNew("em_")
	repo := mocks.NewMockEquipmentManufacturerRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).
		Return(&equipmentmanufacturer.EquipmentManufacturer{
			ID:             emID,
			OrganizationID: testutil.TestOrgID,
			BusinessUnitID: testutil.TestBuID,
			Name:           "Freightliner",
			Status:         domaintypes.StatusActive,
		}, nil)
	repo.On("Update", mock.Anything, mock.Anything).
		Return(func(_ context.Context, entity *equipmentmanufacturer.EquipmentManufacturer) *equipmentmanufacturer.EquipmentManufacturer {
			return entity
		}, nil)

	handler := setupEquipManufacturerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/equipment-manufacturers/" + emID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":   "Kenworth",
			"status": "Active",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestEquipmentManufacturerHandler_Patch_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockEquipmentManufacturerRepository(t)
	handler := setupEquipManufacturerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/equipment-manufacturers/invalid-id/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{"name": "Test"})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestEquipmentManufacturerHandler_SelectOptions_Success(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockEquipmentManufacturerRepository(t)
	repo.On("SelectOptions", mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*equipmentmanufacturer.EquipmentManufacturer]{
			Items: []*equipmentmanufacturer.EquipmentManufacturer{},
			Total: 0,
		}, nil)

	handler := setupEquipManufacturerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/equipment-manufacturers/select-options/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestEquipmentManufacturerHandler_GetOption_Success(t *testing.T) {
	t.Parallel()

	emID := pulid.MustNew("em_")
	repo := mocks.NewMockEquipmentManufacturerRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).
		Return(&equipmentmanufacturer.EquipmentManufacturer{
			ID:             emID,
			OrganizationID: testutil.TestOrgID,
			BusinessUnitID: testutil.TestBuID,
			Name:           "Peterbilt",
			Status:         domaintypes.StatusActive,
		}, nil)

	handler := setupEquipManufacturerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/equipment-manufacturers/select-options/" + emID.String()).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestEquipmentManufacturerHandler_GetOption_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockEquipmentManufacturerRepository(t)
	handler := setupEquipManufacturerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/equipment-manufacturers/select-options/invalid-id").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestEquipmentManufacturerHandler_BulkUpdateStatus_Success(t *testing.T) {
	t.Parallel()

	emID := pulid.MustNew("em_")
	repo := mocks.NewMockEquipmentManufacturerRepository(t)
	repo.On("GetByIDs", mock.Anything, mock.Anything).
		Return([]*equipmentmanufacturer.EquipmentManufacturer{
			{ID: emID, Name: "Peterbilt", Status: domaintypes.StatusActive},
		}, nil)
	repo.On("BulkUpdateStatus", mock.Anything, mock.Anything).
		Return([]*equipmentmanufacturer.EquipmentManufacturer{
			{ID: emID, Name: "Peterbilt", Status: domaintypes.StatusInactive},
		}, nil)

	handler := setupEquipManufacturerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/equipment-manufacturers/bulk-update-status/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"ids":    []string{emID.String()},
			"status": "Inactive",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestEquipmentManufacturerHandler_GetOption_NotFound(t *testing.T) {
	t.Parallel()

	emID := pulid.MustNew("em_")
	repo := mocks.NewMockEquipmentManufacturerRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errNotFound)

	handler := setupEquipManufacturerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/equipment-manufacturers/select-options/" + emID.String()).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestEquipmentManufacturerHandler_BulkUpdateStatus_ServiceError(t *testing.T) {
	t.Parallel()

	emID := pulid.MustNew("em_")
	repo := mocks.NewMockEquipmentManufacturerRepository(t)
	repo.On("GetByIDs", mock.Anything, mock.Anything).
		Return([]*equipmentmanufacturer.EquipmentManufacturer{
			{ID: emID, Name: "Peterbilt", Status: domaintypes.StatusActive},
		}, nil)
	repo.On("BulkUpdateStatus", mock.Anything, mock.Anything).
		Return(nil, errors.New("service error"))

	handler := setupEquipManufacturerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/equipment-manufacturers/bulk-update-status/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"ids":    []string{emID.String()},
			"status": "Inactive",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestEquipmentManufacturerHandler_Create_ServiceError(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockEquipmentManufacturerRepository(t)
	repo.On("Create", mock.Anything, mock.Anything).Return(nil, errors.New("service error"))

	handler := setupEquipManufacturerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/equipment-manufacturers/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":   "Peterbilt",
			"status": "Active",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestEquipmentManufacturerHandler_Update_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockEquipmentManufacturerRepository(t)
	handler := setupEquipManufacturerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/equipment-manufacturers/invalid-id/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":   "Kenworth",
			"status": "Active",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestEquipmentManufacturerHandler_BulkUpdateStatus_BadJSON(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockEquipmentManufacturerRepository(t)
	handler := setupEquipManufacturerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/equipment-manufacturers/bulk-update-status/").
		WithDefaultAuthContext().
		WithBody("{bad json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}
