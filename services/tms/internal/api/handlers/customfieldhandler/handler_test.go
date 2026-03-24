package customfieldhandler_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/customfieldhandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/customfield"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/customfieldservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var errNotFound = errors.New("custom field definition not found")

func setupCustomFieldHandler(
	t *testing.T,
	repo *mocks.MockCustomFieldDefinitionRepository,
	valueRepo *mocks.MockCustomFieldValueRepository,
) *customfieldhandler.Handler {
	t.Helper()

	logger := zap.NewNop()

	service := customfieldservice.New(customfieldservice.Params{
		Logger:       logger,
		Repo:         repo,
		ValueRepo:    valueRepo,
		Validator:    customfieldservice.NewTestValidator(),
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

	return customfieldhandler.New(customfieldhandler.Params{
		Service:              service,
		ErrorHandler:         errorHandler,
		PermissionMiddleware: pm,
	})
}

func TestCustomFieldHandler_List_Success(t *testing.T) {
	t.Parallel()

	cfdID := pulid.MustNew("cfd_")

	repo := mocks.NewMockCustomFieldDefinitionRepository(t)
	repo.On("List", mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*customfield.CustomFieldDefinition]{
			Items: []*customfield.CustomFieldDefinition{
				{
					ID:             cfdID,
					OrganizationID: testutil.TestOrgID,
					BusinessUnitID: testutil.TestBuID,
					Name:           "test_field",
					Label:          "Test Field",
					FieldType:      customfield.FieldTypeText,
					ResourceType:   "trailer",
				},
			},
			Total: 1,
		}, nil)

	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	handler := setupCustomFieldHandler(t, repo, valueRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/custom-fields/definitions/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp pagination.Response[[]map[string]any]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, 1, resp.Count)
	assert.Len(t, resp.Results, 1)
}

func TestCustomFieldHandler_Get_Success(t *testing.T) {
	t.Parallel()

	cfdID := pulid.MustNew("cfd_")

	repo := mocks.NewMockCustomFieldDefinitionRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req repositories.GetCustomFieldDefinitionByIDRequest) (*customfield.CustomFieldDefinition, error) {
			return &customfield.CustomFieldDefinition{
				ID:             req.ID,
				OrganizationID: testutil.TestOrgID,
				BusinessUnitID: testutil.TestBuID,
				Name:           "test_field",
				Label:          "Test Field",
				FieldType:      customfield.FieldTypeText,
				ResourceType:   "trailer",
			}, nil
		})

	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	handler := setupCustomFieldHandler(t, repo, valueRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/custom-fields/definitions/" + cfdID.String() + "/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "test_field", resp["name"])
}

func TestCustomFieldHandler_Get_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockCustomFieldDefinitionRepository(t)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	handler := setupCustomFieldHandler(t, repo, valueRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/custom-fields/definitions/invalid-id/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestCustomFieldHandler_Create_Success(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockCustomFieldDefinitionRepository(t)
	repo.On("CountByResourceType", mock.Anything, mock.Anything).Return(0, nil)
	repo.EXPECT().Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *customfield.CustomFieldDefinition) (*customfield.CustomFieldDefinition, error) {
			entity.ID = pulid.MustNew("cfd_")
			return entity, nil
		})

	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	handler := setupCustomFieldHandler(t, repo, valueRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/custom-fields/definitions/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":         "test_field",
			"label":        "Test Field",
			"fieldType":    "text",
			"resourceType": "trailer",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusCreated, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "test_field", resp["name"])
}

func TestCustomFieldHandler_Create_BadJSON(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockCustomFieldDefinitionRepository(t)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	handler := setupCustomFieldHandler(t, repo, valueRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/custom-fields/definitions/").
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestCustomFieldHandler_Update_Success(t *testing.T) {
	t.Parallel()

	cfdID := pulid.MustNew("cfd_")

	repo := mocks.NewMockCustomFieldDefinitionRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req repositories.GetCustomFieldDefinitionByIDRequest) (*customfield.CustomFieldDefinition, error) {
			return &customfield.CustomFieldDefinition{
				ID:             req.ID,
				OrganizationID: testutil.TestOrgID,
				BusinessUnitID: testutil.TestBuID,
				Name:           "test_field",
				Label:          "Test Field",
				FieldType:      customfield.FieldTypeText,
				ResourceType:   "trailer",
			}, nil
		})
	repo.EXPECT().Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *customfield.CustomFieldDefinition) (*customfield.CustomFieldDefinition, error) {
			return entity, nil
		})

	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	valueRepo.On("CountByDefinition", mock.Anything, mock.Anything).Return(0, nil)
	valueRepo.On("CountResourcesByDefinition", mock.Anything, mock.Anything).Return(0, nil)

	handler := setupCustomFieldHandler(t, repo, valueRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/custom-fields/definitions/" + cfdID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":         "updated_field",
			"label":        "Updated Field",
			"fieldType":    "text",
			"resourceType": "trailer",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "updated_field", resp["name"])
}

func TestCustomFieldHandler_Delete_Success(t *testing.T) {
	t.Parallel()

	cfdID := pulid.MustNew("cfd_")

	repo := mocks.NewMockCustomFieldDefinitionRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req repositories.GetCustomFieldDefinitionByIDRequest) (*customfield.CustomFieldDefinition, error) {
			return &customfield.CustomFieldDefinition{
				ID:             req.ID,
				OrganizationID: testutil.TestOrgID,
				BusinessUnitID: testutil.TestBuID,
				Name:           "test_field",
				Label:          "Test Field",
				FieldType:      customfield.FieldTypeText,
				ResourceType:   "trailer",
			}, nil
		})
	repo.On("Delete", mock.Anything, mock.Anything).Return(nil)

	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	valueRepo.On("CountByDefinition", mock.Anything, mock.Anything).Return(0, nil)
	valueRepo.On("CountResourcesByDefinition", mock.Anything, mock.Anything).Return(0, nil)

	handler := setupCustomFieldHandler(t, repo, valueRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodDelete).
		WithPath("/api/v1/custom-fields/definitions/" + cfdID.String() + "/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusNoContent, ginCtx.ResponseCode())
}

func TestCustomFieldHandler_GetResourceTypes_Success(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockCustomFieldDefinitionRepository(t)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	handler := setupCustomFieldHandler(t, repo, valueRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/custom-fields/resource-types/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	resourceTypes, ok := resp["resourceTypes"].([]any)
	require.True(t, ok)
	assert.NotEmpty(t, resourceTypes)
}

func TestCustomFieldHandler_GetByResourceType_Success(t *testing.T) {
	t.Parallel()

	cfdID := pulid.MustNew("cfd_")

	repo := mocks.NewMockCustomFieldDefinitionRepository(t)
	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).
		Return([]*customfield.CustomFieldDefinition{
			{
				ID:             cfdID,
				OrganizationID: testutil.TestOrgID,
				BusinessUnitID: testutil.TestBuID,
				Name:           "active_field",
				Label:          "Active Field",
				FieldType:      customfield.FieldTypeText,
				ResourceType:   "trailer",
				IsActive:       true,
			},
		}, nil)

	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	handler := setupCustomFieldHandler(t, repo, valueRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/custom-fields/resources/trailer/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp []map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Len(t, resp, 1)
	assert.Equal(t, "active_field", resp[0]["name"])
}

func TestCustomFieldHandler_Patch_Success(t *testing.T) {
	t.Parallel()

	cfdID := pulid.MustNew("cfd_")

	repo := mocks.NewMockCustomFieldDefinitionRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req repositories.GetCustomFieldDefinitionByIDRequest) (*customfield.CustomFieldDefinition, error) {
			return &customfield.CustomFieldDefinition{
				ID:             req.ID,
				OrganizationID: testutil.TestOrgID,
				BusinessUnitID: testutil.TestBuID,
				Name:           "test_field",
				Label:          "Test Field",
				FieldType:      customfield.FieldTypeText,
				ResourceType:   "trailer",
			}, nil
		})
	repo.EXPECT().Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *customfield.CustomFieldDefinition) (*customfield.CustomFieldDefinition, error) {
			return entity, nil
		})

	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	valueRepo.On("CountByDefinition", mock.Anything, mock.Anything).Return(0, nil)
	valueRepo.On("CountResourcesByDefinition", mock.Anything, mock.Anything).Return(0, nil)

	handler := setupCustomFieldHandler(t, repo, valueRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/custom-fields/definitions/" + cfdID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"label": "Patched Label",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "Patched Label", resp["label"])
}

func TestCustomFieldHandler_Patch_NotFound(t *testing.T) {
	t.Parallel()

	cfdID := pulid.MustNew("cfd_")

	repo := mocks.NewMockCustomFieldDefinitionRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errNotFound)

	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	handler := setupCustomFieldHandler(t, repo, valueRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/custom-fields/definitions/" + cfdID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"label": "Patched Label",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestCustomFieldHandler_Patch_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockCustomFieldDefinitionRepository(t)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	handler := setupCustomFieldHandler(t, repo, valueRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/custom-fields/definitions/invalid-id/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"label": "Patched Label",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestCustomFieldHandler_Get_NotFound(t *testing.T) {
	t.Parallel()

	cfdID := pulid.MustNew("cfd_")

	repo := mocks.NewMockCustomFieldDefinitionRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errNotFound)

	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	handler := setupCustomFieldHandler(t, repo, valueRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/custom-fields/definitions/" + cfdID.String() + "/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestCustomFieldHandler_Create_ServiceError(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockCustomFieldDefinitionRepository(t)
	repo.On("CountByResourceType", mock.Anything, mock.Anything).Return(0, nil)
	repo.On("Create", mock.Anything, mock.Anything).Return(nil, errors.New("service error"))

	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	handler := setupCustomFieldHandler(t, repo, valueRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/custom-fields/definitions/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":         "test_field",
			"label":        "Test Field",
			"fieldType":    "text",
			"resourceType": "trailer",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestCustomFieldHandler_Update_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockCustomFieldDefinitionRepository(t)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	handler := setupCustomFieldHandler(t, repo, valueRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/custom-fields/definitions/invalid-id/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":         "test_field",
			"label":        "Test Field",
			"fieldType":    "text",
			"resourceType": "trailer",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestCustomFieldHandler_Delete_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockCustomFieldDefinitionRepository(t)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	handler := setupCustomFieldHandler(t, repo, valueRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodDelete).
		WithPath("/api/v1/custom-fields/definitions/invalid-id/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestCustomFieldHandler_Delete_ServiceError(t *testing.T) {
	t.Parallel()

	cfdID := pulid.MustNew("cfd_")

	repo := mocks.NewMockCustomFieldDefinitionRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req repositories.GetCustomFieldDefinitionByIDRequest) (*customfield.CustomFieldDefinition, error) {
			return &customfield.CustomFieldDefinition{
				ID:             req.ID,
				OrganizationID: testutil.TestOrgID,
				BusinessUnitID: testutil.TestBuID,
				Name:           "test_field",
				Label:          "Test Field",
				FieldType:      customfield.FieldTypeText,
				ResourceType:   "trailer",
			}, nil
		})
	repo.On("Delete", mock.Anything, mock.Anything).Return(errors.New("delete error"))

	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	valueRepo.On("CountByDefinition", mock.Anything, mock.Anything).Return(0, nil)
	valueRepo.On("CountResourcesByDefinition", mock.Anything, mock.Anything).Return(0, nil)

	handler := setupCustomFieldHandler(t, repo, valueRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodDelete).
		WithPath("/api/v1/custom-fields/definitions/" + cfdID.String() + "/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestCustomFieldHandler_GetByResourceType_Error(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockCustomFieldDefinitionRepository(t)
	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).
		Return(nil, errors.New("repo error"))

	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	handler := setupCustomFieldHandler(t, repo, valueRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/custom-fields/resources/trailer/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestCustomFieldHandler_Update_ServiceError(t *testing.T) {
	t.Parallel()

	cfdID := pulid.MustNew("cfd_")

	repo := mocks.NewMockCustomFieldDefinitionRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req repositories.GetCustomFieldDefinitionByIDRequest) (*customfield.CustomFieldDefinition, error) {
			return &customfield.CustomFieldDefinition{
				ID:             req.ID,
				OrganizationID: testutil.TestOrgID,
				BusinessUnitID: testutil.TestBuID,
				Name:           "test_field",
				Label:          "Test Field",
				FieldType:      customfield.FieldTypeText,
				ResourceType:   "trailer",
			}, nil
		})
	repo.On("Update", mock.Anything, mock.Anything).Return(nil, errors.New("update error"))

	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	valueRepo.On("CountByDefinition", mock.Anything, mock.Anything).Return(0, nil)
	valueRepo.On("CountResourcesByDefinition", mock.Anything, mock.Anything).Return(0, nil)

	handler := setupCustomFieldHandler(t, repo, valueRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/custom-fields/definitions/" + cfdID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":         "updated_field",
			"label":        "Updated Field",
			"fieldType":    "text",
			"resourceType": "trailer",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestCustomFieldHandler_GetByResourceType_Unsupported(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockCustomFieldDefinitionRepository(t)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	handler := setupCustomFieldHandler(t, repo, valueRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/custom-fields/resources/invalid_type/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}
