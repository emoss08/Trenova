package trailerhandler_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/trailerhandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/customfield"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/services/customfieldservice"
	"github.com/emoss08/trenova/internal/core/services/trailerservice"
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

var errNotFound = errors.New("trailer not found")
var errService = errors.New("service error")

func setupTrailerHandler(t *testing.T, repo *mocks.MockTrailerRepository) *trailerhandler.Handler {
	t.Helper()

	logger := zap.NewNop()

	cfDefRepo := mocks.NewMockCustomFieldDefinitionRepository(t)
	cfDefRepo.On("GetActiveByResourceType", mock.Anything, mock.Anything).
		Maybe().
		Return([]*customfield.CustomFieldDefinition{}, nil)
	cfValRepo := mocks.NewMockCustomFieldValueRepository(t)
	cfValRepo.On("GetByResource", mock.Anything, mock.Anything).
		Maybe().
		Return([]*customfield.CustomFieldValue{}, nil)
	cfValRepo.On("GetByResources", mock.Anything, mock.Anything).
		Maybe().
		Return(map[string][]*customfield.CustomFieldValue{}, nil)
	cfValRepo.On("Upsert", mock.Anything, mock.Anything).Maybe().Return(nil)

	cfValuesService := customfieldservice.NewValuesService(customfieldservice.ValuesServiceParams{
		Logger:         logger,
		ValueRepo:      cfValRepo,
		DefinitionRepo: cfDefRepo,
		Validator: customfieldservice.NewValuesValidator(customfieldservice.ValuesValidatorParams{
			Logger: logger,
			Repo:   cfDefRepo,
		}),
	})

	service := trailerservice.New(trailerservice.Params{
		Logger:                    logger,
		Repo:                      repo,
		Validator:                 trailerservice.NewTestValidator(),
		AuditService:              &mocks.NoopAuditService{},
		Realtime:                  &mocks.NoopRealtimeService{},
		CustomFieldsValuesService: cfValuesService,
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

	return trailerhandler.New(trailerhandler.Params{
		Service:              service,
		ErrorHandler:         errorHandler,
		PermissionMiddleware: pm,
	})
}

func TestTrailerHandler_List_Success(t *testing.T) {
	t.Parallel()

	trID := pulid.MustNew("tr_")
	repo := mocks.NewMockTrailerRepository(t)
	repo.On("List", mock.Anything, mock.Anything).Return(&pagination.ListResult[*trailer.Trailer]{
		Items: []*trailer.Trailer{
			{
				ID:             trID,
				OrganizationID: testutil.TestOrgID,
				BusinessUnitID: testutil.TestBuID,
				Code:           "TR001",
				Status:         domaintypes.EquipmentStatusAvailable,
			},
		},
		Total: 1,
	}, nil)

	handler := setupTrailerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/trailers/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp pagination.Response[[]map[string]any]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, 1, resp.Count)
	assert.Len(t, resp.Results, 1)
}

func TestTrailerHandler_List_WithPagination(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockTrailerRepository(t)
	repo.On("List", mock.Anything, mock.Anything).Return(&pagination.ListResult[*trailer.Trailer]{
		Items: []*trailer.Trailer{},
		Total: 50,
	}, nil)

	handler := setupTrailerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/trailers/").
		WithQuery(map[string]string{"limit": "10", "offset": "20"}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp pagination.Response[[]map[string]any]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, 50, resp.Count)
}

func TestTrailerHandler_List_Error(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockTrailerRepository(t)
	repo.On("List", mock.Anything, mock.Anything).Return(nil, errService)

	handler := setupTrailerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/trailers/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestTrailerHandler_List_WithQueryParams(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockTrailerRepository(t)
	repo.On("List", mock.Anything, mock.Anything).Return(&pagination.ListResult[*trailer.Trailer]{
		Items: []*trailer.Trailer{},
		Total: 0,
	}, nil)

	handler := setupTrailerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/trailers/").
		WithQuery(map[string]string{
			"includeEquipmentDetails": "true",
			"includeFleetDetails":     "true",
			"status":                  "Available",
		}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestTrailerHandler_Get_Success(t *testing.T) {
	t.Parallel()

	trID := pulid.MustNew("tr_")
	repo := mocks.NewMockTrailerRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&trailer.Trailer{
		ID:             trID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		Code:           "TR001",
		Status:         domaintypes.EquipmentStatusAvailable,
	}, nil)

	handler := setupTrailerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/trailers/" + trID.String() + "/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "TR001", resp["code"])
}

func TestTrailerHandler_Get_NotFound(t *testing.T) {
	t.Parallel()

	trID := pulid.MustNew("tr_")
	repo := mocks.NewMockTrailerRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errNotFound)

	handler := setupTrailerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/trailers/" + trID.String() + "/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestTrailerHandler_Get_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockTrailerRepository(t)
	handler := setupTrailerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/trailers/invalid-id/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestTrailerHandler_Create_Success(t *testing.T) {
	t.Parallel()

	eqTypeID := pulid.MustNew("eqt_")
	eqMfgID := pulid.MustNew("eqm_")
	repo := mocks.NewMockTrailerRepository(t)
	repo.On("Create", mock.Anything, mock.Anything).
		Return(func(_ context.Context, entity *trailer.Trailer) *trailer.Trailer {
			entity.ID = pulid.MustNew("tr_")
			return entity
		}, nil)

	handler := setupTrailerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/trailers/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"code":                    "TR001",
			"status":                  "Available",
			"equipmentTypeId":         eqTypeID.String(),
			"equipmentManufacturerId": eqMfgID.String(),
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusCreated, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "TR001", resp["code"])
}

func TestTrailerHandler_Create_BadJSON(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockTrailerRepository(t)
	handler := setupTrailerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/trailers/").
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestTrailerHandler_Create_ServiceError(t *testing.T) {
	t.Parallel()

	eqTypeID := pulid.MustNew("eqt_")
	eqMfgID := pulid.MustNew("eqm_")
	repo := mocks.NewMockTrailerRepository(t)
	repo.On("Create", mock.Anything, mock.Anything).Return(nil, errService)

	handler := setupTrailerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/trailers/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"code":                    "TR001",
			"status":                  "Available",
			"equipmentTypeId":         eqTypeID.String(),
			"equipmentManufacturerId": eqMfgID.String(),
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestTrailerHandler_Update_Success(t *testing.T) {
	t.Parallel()

	trID := pulid.MustNew("tr_")
	eqTypeID := pulid.MustNew("eqt_")
	eqMfgID := pulid.MustNew("eqm_")
	repo := mocks.NewMockTrailerRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&trailer.Trailer{
		ID:                      trID,
		OrganizationID:          testutil.TestOrgID,
		BusinessUnitID:          testutil.TestBuID,
		Code:                    "TR001",
		EquipmentTypeID:         eqTypeID,
		EquipmentManufacturerID: eqMfgID,
		Status:                  domaintypes.EquipmentStatusAvailable,
	}, nil)
	repo.On("Update", mock.Anything, mock.Anything).
		Return(func(_ context.Context, entity *trailer.Trailer) *trailer.Trailer {
			return entity
		}, nil)

	handler := setupTrailerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/trailers/" + trID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"code":                    "TR002",
			"status":                  "Available",
			"equipmentTypeId":         eqTypeID.String(),
			"equipmentManufacturerId": eqMfgID.String(),
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "TR002", resp["code"])
}

func TestTrailerHandler_Update_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockTrailerRepository(t)
	handler := setupTrailerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/trailers/invalid-id/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"code": "TR002",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestTrailerHandler_Update_BadJSON(t *testing.T) {
	t.Parallel()

	trID := pulid.MustNew("tr_")
	repo := mocks.NewMockTrailerRepository(t)
	handler := setupTrailerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/trailers/" + trID.String() + "/").
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestTrailerHandler_Update_ServiceError(t *testing.T) {
	t.Parallel()

	trID := pulid.MustNew("tr_")
	eqTypeID := pulid.MustNew("eqt_")
	eqMfgID := pulid.MustNew("eqm_")
	repo := mocks.NewMockTrailerRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&trailer.Trailer{
		ID:                      trID,
		OrganizationID:          testutil.TestOrgID,
		BusinessUnitID:          testutil.TestBuID,
		Code:                    "TR001",
		EquipmentTypeID:         eqTypeID,
		EquipmentManufacturerID: eqMfgID,
		Status:                  domaintypes.EquipmentStatusAvailable,
	}, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return(nil, errService)

	handler := setupTrailerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/trailers/" + trID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"code":                    "TR002",
			"status":                  "Available",
			"equipmentTypeId":         eqTypeID.String(),
			"equipmentManufacturerId": eqMfgID.String(),
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestTrailerHandler_Patch_Success(t *testing.T) {
	t.Parallel()

	trID := pulid.MustNew("tr_")
	eqTypeID := pulid.MustNew("eqt_")
	eqMfgID := pulid.MustNew("eqm_")
	repo := mocks.NewMockTrailerRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&trailer.Trailer{
		ID:                      trID,
		OrganizationID:          testutil.TestOrgID,
		BusinessUnitID:          testutil.TestBuID,
		Code:                    "TR001",
		EquipmentTypeID:         eqTypeID,
		EquipmentManufacturerID: eqMfgID,
		Status:                  domaintypes.EquipmentStatusAvailable,
	}, nil)
	repo.On("Update", mock.Anything, mock.Anything).
		Return(func(_ context.Context, entity *trailer.Trailer) *trailer.Trailer {
			return entity
		}, nil)

	handler := setupTrailerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/trailers/" + trID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"code": "TR003",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "TR003", resp["code"])
}

func TestTrailerHandler_Patch_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockTrailerRepository(t)
	handler := setupTrailerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/trailers/invalid-id/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"code": "TR003",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestTrailerHandler_Patch_GetError(t *testing.T) {
	t.Parallel()

	trID := pulid.MustNew("tr_")
	repo := mocks.NewMockTrailerRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errNotFound)

	handler := setupTrailerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/trailers/" + trID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"code": "TR003",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestTrailerHandler_Patch_BadJSON(t *testing.T) {
	t.Parallel()

	trID := pulid.MustNew("tr_")
	eqTypeID := pulid.MustNew("eqt_")
	eqMfgID := pulid.MustNew("eqm_")
	repo := mocks.NewMockTrailerRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&trailer.Trailer{
		ID:                      trID,
		OrganizationID:          testutil.TestOrgID,
		BusinessUnitID:          testutil.TestBuID,
		Code:                    "TR001",
		EquipmentTypeID:         eqTypeID,
		EquipmentManufacturerID: eqMfgID,
		Status:                  domaintypes.EquipmentStatusAvailable,
	}, nil)

	handler := setupTrailerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/trailers/" + trID.String() + "/").
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestTrailerHandler_Patch_UpdateError(t *testing.T) {
	t.Parallel()

	trID := pulid.MustNew("tr_")
	eqTypeID := pulid.MustNew("eqt_")
	eqMfgID := pulid.MustNew("eqm_")
	repo := mocks.NewMockTrailerRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&trailer.Trailer{
		ID:                      trID,
		OrganizationID:          testutil.TestOrgID,
		BusinessUnitID:          testutil.TestBuID,
		Code:                    "TR001",
		EquipmentTypeID:         eqTypeID,
		EquipmentManufacturerID: eqMfgID,
		Status:                  domaintypes.EquipmentStatusAvailable,
	}, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return(nil, errService)

	handler := setupTrailerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/trailers/" + trID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"code": "TR003",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestTrailerHandler_BulkUpdateStatus_Success(t *testing.T) {
	t.Parallel()

	trID1 := pulid.MustNew("tr_")
	trID2 := pulid.MustNew("tr_")
	repo := mocks.NewMockTrailerRepository(t)
	repo.On("GetByIDs", mock.Anything, mock.Anything).Return([]*trailer.Trailer{
		{
			ID:             trID1,
			OrganizationID: testutil.TestOrgID,
			BusinessUnitID: testutil.TestBuID,
			Code:           "TR001",
			Status:         domaintypes.EquipmentStatusAvailable,
		},
		{
			ID:             trID2,
			OrganizationID: testutil.TestOrgID,
			BusinessUnitID: testutil.TestBuID,
			Code:           "TR002",
			Status:         domaintypes.EquipmentStatusAvailable,
		},
	}, nil)
	repo.On("BulkUpdateStatus", mock.Anything, mock.Anything).Return([]*trailer.Trailer{
		{
			ID:             trID1,
			OrganizationID: testutil.TestOrgID,
			BusinessUnitID: testutil.TestBuID,
			Code:           "TR001",
			Status:         domaintypes.EquipmentStatusOOS,
		},
		{
			ID:             trID2,
			OrganizationID: testutil.TestOrgID,
			BusinessUnitID: testutil.TestBuID,
			Code:           "TR002",
			Status:         domaintypes.EquipmentStatusOOS,
		},
	}, nil)

	handler := setupTrailerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/trailers/bulk-update-status/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"trailerIds": []string{trID1.String(), trID2.String()},
			"status":     "OutOfService",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp []map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Len(t, resp, 2)
}

func TestTrailerHandler_BulkUpdateStatus_BadJSON(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockTrailerRepository(t)
	handler := setupTrailerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/trailers/bulk-update-status/").
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestTrailerHandler_BulkUpdateStatus_ServiceError(t *testing.T) {
	t.Parallel()

	trID1 := pulid.MustNew("tr_")
	repo := mocks.NewMockTrailerRepository(t)
	repo.On("GetByIDs", mock.Anything, mock.Anything).Return(nil, errService)

	handler := setupTrailerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/trailers/bulk-update-status/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"trailerIds": []string{trID1.String()},
			"status":     "OutOfService",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}
