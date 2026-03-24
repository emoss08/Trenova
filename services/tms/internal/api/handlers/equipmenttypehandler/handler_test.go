package equipmenttypehandler_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/equipmenttypehandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/services/equipmenttypeservice"
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

var errNotFound = errors.New("equipment type not found")

func setupEquipTypeHandler(
	t *testing.T,
	repo *mocks.MockEquipmentTypeRepository,
) *equipmenttypehandler.Handler {
	t.Helper()

	logger := zap.NewNop()

	service := equipmenttypeservice.New(equipmenttypeservice.Params{
		Logger:       logger,
		Repo:         repo,
		Validator:    equipmenttypeservice.NewTestValidator(),
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

	return equipmenttypehandler.New(equipmenttypehandler.Params{
		Service:              service,
		ErrorHandler:         errorHandler,
		PermissionMiddleware: pm,
	})
}

func TestEquipmentTypeHandler_List_Success(t *testing.T) {
	t.Parallel()

	etID := pulid.MustNew("et_")
	repo := mocks.NewMockEquipmentTypeRepository(t)
	repo.On("List", mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*equipmenttype.EquipmentType]{
			Items: []*equipmenttype.EquipmentType{
				{
					ID:             etID,
					OrganizationID: testutil.TestOrgID,
					BusinessUnitID: testutil.TestBuID,
					Code:           "TRK",
					Class:          equipmenttype.ClassTractor,
					Status:         domaintypes.StatusActive,
				},
			},
			Total: 1,
		}, nil)

	handler := setupEquipTypeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/equipment-types/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp pagination.Response[[]map[string]any]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, 1, resp.Count)
	assert.Len(t, resp.Results, 1)
}

func TestEquipmentTypeHandler_List_WithPagination(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockEquipmentTypeRepository(t)
	repo.On("List", mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*equipmenttype.EquipmentType]{
			Items: []*equipmenttype.EquipmentType{},
			Total: 100,
		}, nil)

	handler := setupEquipTypeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/equipment-types/").
		WithQuery(map[string]string{"limit": "20", "offset": "40"}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp pagination.Response[[]map[string]any]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, 100, resp.Count)
}

func TestEquipmentTypeHandler_Get_Success(t *testing.T) {
	t.Parallel()

	etID := pulid.MustNew("et_")
	repo := mocks.NewMockEquipmentTypeRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&equipmenttype.EquipmentType{
		ID:             etID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		Code:           "TRL",
		Class:          equipmenttype.ClassTrailer,
		Status:         domaintypes.StatusActive,
	}, nil)

	handler := setupEquipTypeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/equipment-types/" + etID.String() + "/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "TRL", resp["code"])
}

func TestEquipmentTypeHandler_Get_NotFound(t *testing.T) {
	t.Parallel()

	etID := pulid.MustNew("et_")
	repo := mocks.NewMockEquipmentTypeRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errNotFound)

	handler := setupEquipTypeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/equipment-types/" + etID.String() + "/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestEquipmentTypeHandler_Get_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockEquipmentTypeRepository(t)
	handler := setupEquipTypeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/equipment-types/invalid-id/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestEquipmentTypeHandler_Create_Success(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockEquipmentTypeRepository(t)
	repo.On("Create", mock.Anything, mock.Anything).
		Return(func(_ context.Context, entity *equipmenttype.EquipmentType) *equipmenttype.EquipmentType {
			entity.ID = pulid.MustNew("et_")
			return entity
		}, nil)

	handler := setupEquipTypeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/equipment-types/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"code":   "TRK",
			"class":  "Tractor",
			"status": "Active",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusCreated, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "TRK", resp["code"])
	assert.Equal(t, "Tractor", resp["class"])
}

func TestEquipmentTypeHandler_Create_BadJSON(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockEquipmentTypeRepository(t)
	handler := setupEquipTypeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/equipment-types/").
		WithDefaultAuthContext().
		WithBody("{broken json!!!")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestEquipmentTypeHandler_Update_Success(t *testing.T) {
	t.Parallel()

	etID := pulid.MustNew("et_")
	repo := mocks.NewMockEquipmentTypeRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&equipmenttype.EquipmentType{
		ID:             etID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		Code:           "TRK",
		Class:          equipmenttype.ClassTractor,
		Status:         domaintypes.StatusActive,
	}, nil)
	repo.On("Update", mock.Anything, mock.Anything).
		Return(func(_ context.Context, entity *equipmenttype.EquipmentType) *equipmenttype.EquipmentType {
			return entity
		}, nil)

	handler := setupEquipTypeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/equipment-types/" + etID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"code":   "TRL",
			"class":  "Trailer",
			"status": "Active",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "TRL", resp["code"])
	assert.Equal(t, "Trailer", resp["class"])
}

func TestEquipmentTypeHandler_Update_NotFound(t *testing.T) {
	t.Parallel()

	etID := pulid.MustNew("et_")
	repo := mocks.NewMockEquipmentTypeRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errNotFound)

	handler := setupEquipTypeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/equipment-types/" + etID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"code":   "TRL",
			"class":  "Trailer",
			"status": "Active",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestEquipmentTypeHandler_Patch_Success(t *testing.T) {
	t.Parallel()

	etID := pulid.MustNew("et_")
	repo := mocks.NewMockEquipmentTypeRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&equipmenttype.EquipmentType{
		ID:             etID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		Code:           "TRK",
		Class:          equipmenttype.ClassTractor,
		Status:         domaintypes.StatusActive,
	}, nil)
	repo.On("Update", mock.Anything, mock.Anything).
		Return(func(_ context.Context, entity *equipmenttype.EquipmentType) *equipmenttype.EquipmentType {
			return entity
		}, nil)

	handler := setupEquipTypeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/equipment-types/" + etID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"code":   "TRL",
			"class":  "Trailer",
			"status": "Active",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestEquipmentTypeHandler_Patch_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockEquipmentTypeRepository(t)
	handler := setupEquipTypeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/equipment-types/invalid-id/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{"code": "TRL"})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestEquipmentTypeHandler_SelectOptions_Success(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockEquipmentTypeRepository(t)
	repo.On("SelectOptions", mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*equipmenttype.EquipmentType]{
			Items: []*equipmenttype.EquipmentType{{ID: pulid.MustNew("et_"), Code: "TRK"}},
			Total: 1,
		}, nil)

	handler := setupEquipTypeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/equipment-types/select-options/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestEquipmentTypeHandler_GetOption_Success(t *testing.T) {
	t.Parallel()

	etID := pulid.MustNew("et_")
	repo := mocks.NewMockEquipmentTypeRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&equipmenttype.EquipmentType{
		ID:             etID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		Code:           "TRK",
		Class:          equipmenttype.ClassTractor,
		Status:         domaintypes.StatusActive,
	}, nil)

	handler := setupEquipTypeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/equipment-types/select-options/" + etID.String()).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestEquipmentTypeHandler_GetOption_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockEquipmentTypeRepository(t)
	handler := setupEquipTypeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/equipment-types/select-options/invalid-id").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestEquipmentTypeHandler_BulkUpdateStatus_Success(t *testing.T) {
	t.Parallel()

	etID := pulid.MustNew("et_")
	repo := mocks.NewMockEquipmentTypeRepository(t)
	repo.On("GetByIDs", mock.Anything, mock.Anything).Return([]*equipmenttype.EquipmentType{
		{
			ID:     etID,
			Code:   "TRK",
			Class:  equipmenttype.ClassTractor,
			Status: domaintypes.StatusActive,
		},
	}, nil)
	repo.On("BulkUpdateStatus", mock.Anything, mock.Anything).Return([]*equipmenttype.EquipmentType{
		{
			ID:     etID,
			Code:   "TRK",
			Class:  equipmenttype.ClassTractor,
			Status: domaintypes.StatusInactive,
		},
	}, nil)

	handler := setupEquipTypeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/equipment-types/bulk-update-status/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"ids":    []string{etID.String()},
			"status": "Inactive",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestEquipmentTypeHandler_SelectOptions_WithClassFilter(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockEquipmentTypeRepository(t)
	repo.On("SelectOptions", mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*equipmenttype.EquipmentType]{
			Items: []*equipmenttype.EquipmentType{
				{ID: pulid.MustNew("et_"), Code: "TRK", Class: equipmenttype.ClassTractor},
			},
			Total: 1,
		}, nil)

	handler := setupEquipTypeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/equipment-types/select-options/").
		WithQuery(map[string]string{"classes": "Tractor"}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestEquipmentTypeHandler_GetOption_NotFound(t *testing.T) {
	t.Parallel()

	etID := pulid.MustNew("et_")
	repo := mocks.NewMockEquipmentTypeRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errNotFound)

	handler := setupEquipTypeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/equipment-types/select-options/" + etID.String()).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestEquipmentTypeHandler_BulkUpdateStatus_ServiceError(t *testing.T) {
	t.Parallel()

	etID := pulid.MustNew("et_")
	repo := mocks.NewMockEquipmentTypeRepository(t)
	repo.On("GetByIDs", mock.Anything, mock.Anything).Return([]*equipmenttype.EquipmentType{
		{
			ID:     etID,
			Code:   "TRK",
			Class:  equipmenttype.ClassTractor,
			Status: domaintypes.StatusActive,
		},
	}, nil)
	repo.On("BulkUpdateStatus", mock.Anything, mock.Anything).
		Return(nil, errors.New("service error"))

	handler := setupEquipTypeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/equipment-types/bulk-update-status/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"ids":    []string{etID.String()},
			"status": "Inactive",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestEquipmentTypeHandler_Create_ServiceError(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockEquipmentTypeRepository(t)
	repo.On("Create", mock.Anything, mock.Anything).Return(nil, errors.New("service error"))

	handler := setupEquipTypeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/equipment-types/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"code":   "TRK",
			"class":  "Tractor",
			"status": "Active",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestEquipmentTypeHandler_Update_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockEquipmentTypeRepository(t)
	handler := setupEquipTypeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/equipment-types/invalid-id/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"code":   "TRL",
			"class":  "Trailer",
			"status": "Active",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestEquipmentTypeHandler_BulkUpdateStatus_BadJSON(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockEquipmentTypeRepository(t)
	handler := setupEquipTypeHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/equipment-types/bulk-update-status/").
		WithDefaultAuthContext().
		WithBody("{bad json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}
