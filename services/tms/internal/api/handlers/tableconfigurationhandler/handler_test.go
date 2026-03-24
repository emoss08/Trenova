package tableconfigurationhandler_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/tableconfigurationhandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/tableconfiguration"
	"github.com/emoss08/trenova/internal/core/services/tableconfigurationservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var errNotFound = errors.New("table configuration not found")

func setupTableConfigurationHandler(
	t *testing.T,
	repo *mocks.MockTableConfigurationRepository,
) *tableconfigurationhandler.Handler {
	t.Helper()

	logger := zap.NewNop()

	service := tableconfigurationservice.New(tableconfigurationservice.Params{
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

	return tableconfigurationhandler.New(tableconfigurationhandler.Params{
		Service:      service,
		ErrorHandler: errorHandler,
	})
}

func TestTableConfigurationHandler_List_Success(t *testing.T) {
	t.Parallel()

	tcID := pulid.MustNew("tc_")
	repo := mocks.NewMockTableConfigurationRepository(t)
	repo.On("List", mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*tableconfiguration.TableConfiguration]{
			Items: []*tableconfiguration.TableConfiguration{
				{
					ID:             tcID,
					OrganizationID: testutil.TestOrgID,
					BusinessUnitID: testutil.TestBuID,
					UserID:         testutil.TestUserID,
					Name:           "My Config",
					Resource:       "fleet_codes",
					Visibility:     tableconfiguration.VisibilityPrivate,
					TableConfig:    &tableconfiguration.TableConfig{PageSize: 25},
				},
			},
			Total: 1,
		}, nil)

	handler := setupTableConfigurationHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/table-configurations/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp pagination.Response[[]map[string]any]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, 1, resp.Count)
	assert.Len(t, resp.Results, 1)
}

func TestTableConfigurationHandler_List_WithPagination(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockTableConfigurationRepository(t)
	repo.On("List", mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*tableconfiguration.TableConfiguration]{
			Items: []*tableconfiguration.TableConfiguration{},
			Total: 30,
		}, nil)

	handler := setupTableConfigurationHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/table-configurations/").
		WithQuery(map[string]string{"limit": "10", "offset": "0"}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp pagination.Response[[]map[string]any]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, 30, resp.Count)
}

func TestTableConfigurationHandler_List_Error(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockTableConfigurationRepository(t)
	repo.On("List", mock.Anything, mock.Anything).Return(nil, errors.New("database error"))

	handler := setupTableConfigurationHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/table-configurations/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestTableConfigurationHandler_Get_Success(t *testing.T) {
	t.Parallel()

	tcID := pulid.MustNew("tc_")
	repo := mocks.NewMockTableConfigurationRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&tableconfiguration.TableConfiguration{
		ID:             tcID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		UserID:         testutil.TestUserID,
		Name:           "My Config",
		Resource:       "fleet_codes",
		Visibility:     tableconfiguration.VisibilityPrivate,
		TableConfig:    &tableconfiguration.TableConfig{PageSize: 25},
	}, nil)

	handler := setupTableConfigurationHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/table-configurations/" + tcID.String()).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "My Config", resp["name"])
}

func TestTableConfigurationHandler_Get_NotFound(t *testing.T) {
	t.Parallel()

	tcID := pulid.MustNew("tc_")
	repo := mocks.NewMockTableConfigurationRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errNotFound)

	handler := setupTableConfigurationHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/table-configurations/" + tcID.String()).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestTableConfigurationHandler_Get_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockTableConfigurationRepository(t)
	handler := setupTableConfigurationHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/table-configurations/invalid-id").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestTableConfigurationHandler_GetDefault_Success(t *testing.T) {
	t.Parallel()

	tcID := pulid.MustNew("tc_")
	repo := mocks.NewMockTableConfigurationRepository(t)
	repo.On("GetDefaultForResource", mock.Anything, mock.Anything).
		Return(&tableconfiguration.TableConfiguration{
			ID:             tcID,
			OrganizationID: testutil.TestOrgID,
			BusinessUnitID: testutil.TestBuID,
			UserID:         testutil.TestUserID,
			Name:           "Default Config",
			Resource:       "fleet_codes",
			Visibility:     tableconfiguration.VisibilityPrivate,
			IsDefault:      true,
			TableConfig:    &tableconfiguration.TableConfig{PageSize: 25},
		}, nil)

	handler := setupTableConfigurationHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/table-configurations/default").
		WithQuery(map[string]string{"resource": "fleet_codes"}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "Default Config", resp["name"])
	assert.Equal(t, true, resp["isDefault"])
}

func TestTableConfigurationHandler_GetDefault_MissingResource(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockTableConfigurationRepository(t)
	handler := setupTableConfigurationHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/table-configurations/default").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestTableConfigurationHandler_GetDefault_Error(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockTableConfigurationRepository(t)
	repo.On("GetDefaultForResource", mock.Anything, mock.Anything).
		Return(nil, errors.New("database error"))

	handler := setupTableConfigurationHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/table-configurations/default").
		WithQuery(map[string]string{"resource": "fleet_codes"}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestTableConfigurationHandler_GetDefault_NotFoundReturnsNull(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockTableConfigurationRepository(t)
	repo.On("GetDefaultForResource", mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("TableConfiguration not found within your organization"))

	handler := setupTableConfigurationHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/table-configurations/default").
		WithQuery(map[string]string{"resource": "fleet_codes"}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	assert.Equal(t, "null", ginCtx.Recorder.Body.String())
}

func TestTableConfigurationHandler_Create_Success(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockTableConfigurationRepository(t)
	repo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *tableconfiguration.TableConfiguration) (*tableconfiguration.TableConfiguration, error) {
			entity.ID = pulid.MustNew("tc_")
			return entity, nil
		})

	handler := setupTableConfigurationHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/table-configurations/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":       "New Config",
			"resource":   "fleet_codes",
			"visibility": "Private",
			"tableConfig": map[string]any{
				"pageSize":         25,
				"columnVisibility": map[string]bool{},
				"columnOrder":      []string{},
				"sort":             []any{},
				"filterGroups":     []any{},
				"fieldFilters":     []any{},
				"joinOperator":     "AND",
			},
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusCreated, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "New Config", resp["name"])
}

func TestTableConfigurationHandler_Create_BadJSON(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockTableConfigurationRepository(t)
	handler := setupTableConfigurationHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/table-configurations/").
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestTableConfigurationHandler_Create_ServiceError(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockTableConfigurationRepository(t)
	repo.On("Create", mock.Anything, mock.Anything).Return(nil, errors.New("create failed"))

	handler := setupTableConfigurationHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/table-configurations/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":       "New Config",
			"resource":   "fleet_codes",
			"visibility": "Private",
			"tableConfig": map[string]any{
				"pageSize":         25,
				"columnVisibility": map[string]bool{},
				"columnOrder":      []string{},
				"sort":             []any{},
				"filterGroups":     []any{},
				"fieldFilters":     []any{},
				"joinOperator":     "AND",
			},
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestTableConfigurationHandler_Update_Success(t *testing.T) {
	t.Parallel()

	tcID := pulid.MustNew("tc_")
	repo := mocks.NewMockTableConfigurationRepository(t)
	repo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *tableconfiguration.TableConfiguration) (*tableconfiguration.TableConfiguration, error) {
			return entity, nil
		})

	handler := setupTableConfigurationHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/table-configurations/" + tcID.String()).
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":       "Updated Config",
			"resource":   "fleet_codes",
			"visibility": "Private",
			"tableConfig": map[string]any{
				"pageSize":         50,
				"columnVisibility": map[string]bool{},
				"columnOrder":      []string{},
				"sort":             []any{},
				"filterGroups":     []any{},
				"fieldFilters":     []any{},
				"joinOperator":     "AND",
			},
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "Updated Config", resp["name"])
}

func TestTableConfigurationHandler_Update_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockTableConfigurationRepository(t)
	handler := setupTableConfigurationHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/table-configurations/invalid-id").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":     "Updated Config",
			"resource": "fleet_codes",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestTableConfigurationHandler_Update_BadJSON(t *testing.T) {
	t.Parallel()

	tcID := pulid.MustNew("tc_")
	repo := mocks.NewMockTableConfigurationRepository(t)
	handler := setupTableConfigurationHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/table-configurations/" + tcID.String()).
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestTableConfigurationHandler_Update_ServiceError(t *testing.T) {
	t.Parallel()

	tcID := pulid.MustNew("tc_")
	repo := mocks.NewMockTableConfigurationRepository(t)
	repo.On("Update", mock.Anything, mock.Anything).Return(nil, errors.New("update failed"))

	handler := setupTableConfigurationHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/table-configurations/" + tcID.String()).
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":       "Updated Config",
			"resource":   "fleet_codes",
			"visibility": "Private",
			"tableConfig": map[string]any{
				"pageSize":         50,
				"columnVisibility": map[string]bool{},
				"columnOrder":      []string{},
				"sort":             []any{},
				"filterGroups":     []any{},
				"fieldFilters":     []any{},
				"joinOperator":     "AND",
			},
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestTableConfigurationHandler_Patch_Success(t *testing.T) {
	t.Parallel()

	tcID := pulid.MustNew("tc_")
	repo := mocks.NewMockTableConfigurationRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&tableconfiguration.TableConfiguration{
		ID:             tcID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		UserID:         testutil.TestUserID,
		Name:           "Original Config",
		Resource:       "fleet_codes",
		Visibility:     tableconfiguration.VisibilityPrivate,
		TableConfig:    &tableconfiguration.TableConfig{PageSize: 25},
	}, nil)
	repo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *tableconfiguration.TableConfiguration) (*tableconfiguration.TableConfiguration, error) {
			return entity, nil
		})

	handler := setupTableConfigurationHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/table-configurations/" + tcID.String()).
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":       "Patched Config",
			"resource":   "fleet_codes",
			"visibility": "Private",
			"tableConfig": map[string]any{
				"pageSize":         50,
				"columnVisibility": map[string]bool{},
				"columnOrder":      []string{},
				"sort":             []any{},
				"filterGroups":     []any{},
				"fieldFilters":     []any{},
				"joinOperator":     "AND",
			},
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "Patched Config", resp["name"])
}

func TestTableConfigurationHandler_Patch_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockTableConfigurationRepository(t)
	handler := setupTableConfigurationHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/table-configurations/invalid-id").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name": "Patched Config",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestTableConfigurationHandler_Patch_GetByIDError(t *testing.T) {
	t.Parallel()

	tcID := pulid.MustNew("tc_")
	repo := mocks.NewMockTableConfigurationRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errNotFound)

	handler := setupTableConfigurationHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/table-configurations/" + tcID.String()).
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name": "Patched Config",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestTableConfigurationHandler_Patch_BadJSON(t *testing.T) {
	t.Parallel()

	tcID := pulid.MustNew("tc_")
	repo := mocks.NewMockTableConfigurationRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&tableconfiguration.TableConfiguration{
		ID:             tcID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		UserID:         testutil.TestUserID,
		Name:           "Original Config",
		Resource:       "fleet_codes",
		Visibility:     tableconfiguration.VisibilityPrivate,
		TableConfig:    &tableconfiguration.TableConfig{PageSize: 25},
	}, nil)

	handler := setupTableConfigurationHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/table-configurations/" + tcID.String()).
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestTableConfigurationHandler_Patch_UpdateError(t *testing.T) {
	t.Parallel()

	tcID := pulid.MustNew("tc_")
	repo := mocks.NewMockTableConfigurationRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&tableconfiguration.TableConfiguration{
		ID:             tcID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		UserID:         testutil.TestUserID,
		Name:           "Original Config",
		Resource:       "fleet_codes",
		Visibility:     tableconfiguration.VisibilityPrivate,
		TableConfig:    &tableconfiguration.TableConfig{PageSize: 25},
	}, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return(nil, errors.New("update failed"))

	handler := setupTableConfigurationHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/table-configurations/" + tcID.String()).
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":       "Patched Config",
			"resource":   "fleet_codes",
			"visibility": "Private",
			"tableConfig": map[string]any{
				"pageSize":         50,
				"columnVisibility": map[string]bool{},
				"columnOrder":      []string{},
				"sort":             []any{},
				"filterGroups":     []any{},
				"fieldFilters":     []any{},
				"joinOperator":     "AND",
			},
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestTableConfigurationHandler_Delete_Success(t *testing.T) {
	t.Parallel()

	tcID := pulid.MustNew("tc_")
	repo := mocks.NewMockTableConfigurationRepository(t)
	repo.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	handler := setupTableConfigurationHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodDelete).
		WithPath("/api/v1/table-configurations/" + tcID.String()).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusNoContent, ginCtx.ResponseCode())
}

func TestTableConfigurationHandler_Delete_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockTableConfigurationRepository(t)
	handler := setupTableConfigurationHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodDelete).
		WithPath("/api/v1/table-configurations/invalid-id").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestTableConfigurationHandler_Delete_Error(t *testing.T) {
	t.Parallel()

	tcID := pulid.MustNew("tc_")
	repo := mocks.NewMockTableConfigurationRepository(t)
	repo.On("Delete", mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("delete failed"))

	handler := setupTableConfigurationHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodDelete).
		WithPath("/api/v1/table-configurations/" + tcID.String()).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestTableConfigurationHandler_SetDefault_Success(t *testing.T) {
	t.Parallel()

	tcID := pulid.MustNew("tc_")
	repo := mocks.NewMockTableConfigurationRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&tableconfiguration.TableConfiguration{
		ID:             tcID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		UserID:         testutil.TestUserID,
		Name:           "My Config",
		Resource:       "fleet_codes",
		Visibility:     tableconfiguration.VisibilityPrivate,
		TableConfig:    &tableconfiguration.TableConfig{PageSize: 25},
	}, nil)
	repo.On("ClearDefaultForResource", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil)
	repo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *tableconfiguration.TableConfiguration) (*tableconfiguration.TableConfiguration, error) {
			return entity, nil
		})

	handler := setupTableConfigurationHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/table-configurations/" + tcID.String() + "/set-default").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, true, resp["isDefault"])
}

func TestTableConfigurationHandler_SetDefault_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockTableConfigurationRepository(t)
	handler := setupTableConfigurationHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/table-configurations/invalid-id/set-default").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestTableConfigurationHandler_SetDefault_GetByIDError(t *testing.T) {
	t.Parallel()

	tcID := pulid.MustNew("tc_")
	repo := mocks.NewMockTableConfigurationRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errNotFound)

	handler := setupTableConfigurationHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/table-configurations/" + tcID.String() + "/set-default").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestTableConfigurationHandler_SetDefault_ClearDefaultError(t *testing.T) {
	t.Parallel()

	tcID := pulid.MustNew("tc_")
	repo := mocks.NewMockTableConfigurationRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&tableconfiguration.TableConfiguration{
		ID:             tcID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		UserID:         testutil.TestUserID,
		Name:           "My Config",
		Resource:       "fleet_codes",
		Visibility:     tableconfiguration.VisibilityPrivate,
		TableConfig:    &tableconfiguration.TableConfig{PageSize: 25},
	}, nil)
	repo.On("ClearDefaultForResource", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("clear default failed"))

	handler := setupTableConfigurationHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/table-configurations/" + tcID.String() + "/set-default").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestTableConfigurationHandler_SetDefault_UpdateError(t *testing.T) {
	t.Parallel()

	tcID := pulid.MustNew("tc_")
	repo := mocks.NewMockTableConfigurationRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&tableconfiguration.TableConfiguration{
		ID:             tcID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		UserID:         testutil.TestUserID,
		Name:           "My Config",
		Resource:       "fleet_codes",
		Visibility:     tableconfiguration.VisibilityPrivate,
		TableConfig:    &tableconfiguration.TableConfig{PageSize: 25},
	}, nil)
	repo.On("ClearDefaultForResource", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil)
	repo.On("Update", mock.Anything, mock.Anything).Return(nil, errors.New("update failed"))

	handler := setupTableConfigurationHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/table-configurations/" + tcID.String() + "/set-default").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}
