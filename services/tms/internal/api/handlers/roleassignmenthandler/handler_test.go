package roleassignmenthandler_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/roleassignmenthandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/services/roleassignmentservice"
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

var errNotFound = errors.New("role assignment not found")

func setupRoleAssignmentHandler(
	t *testing.T,
	repo *mocks.MockRoleAssignmentRepository,
) *roleassignmenthandler.Handler {
	t.Helper()

	logger := zap.NewNop()

	service := roleassignmentservice.New(roleassignmentservice.Params{
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

	return roleassignmenthandler.New(roleassignmenthandler.Params{
		Service:      service,
		ErrorHandler: errorHandler,
	})
}

func TestRoleAssignmentHandler_List_Success(t *testing.T) {
	t.Parallel()

	assignmentID := pulid.MustNew("ura_")
	repo := mocks.NewMockRoleAssignmentRepository(t)
	repo.On("List", mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*permission.UserRoleAssignment]{
			Items: []*permission.UserRoleAssignment{
				{
					ID:             assignmentID,
					UserID:         pulid.MustNew("usr_"),
					OrganizationID: testutil.TestOrgID,
					RoleID:         pulid.MustNew("rol_"),
				},
			},
			Total: 1,
		}, nil)

	handler := setupRoleAssignmentHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/role-assignments/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp pagination.Response[[]map[string]any]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, 1, resp.Count)
	assert.Len(t, resp.Results, 1)
}

func TestRoleAssignmentHandler_List_WithPagination(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockRoleAssignmentRepository(t)
	repo.On("List", mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*permission.UserRoleAssignment]{
			Items: []*permission.UserRoleAssignment{},
			Total: 30,
		}, nil)

	handler := setupRoleAssignmentHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/role-assignments/").
		WithQuery(map[string]string{"limit": "10", "offset": "0"}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp pagination.Response[[]map[string]any]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, 30, resp.Count)
}

func TestRoleAssignmentHandler_Get_Success(t *testing.T) {
	t.Parallel()

	assignmentID := pulid.MustNew("ura_")
	repo := mocks.NewMockRoleAssignmentRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&permission.UserRoleAssignment{
		ID:             assignmentID,
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: testutil.TestOrgID,
		RoleID:         pulid.MustNew("rol_"),
	}, nil)

	handler := setupRoleAssignmentHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/role-assignments/" + assignmentID.String()).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, assignmentID.String(), resp["id"])
}

func TestRoleAssignmentHandler_Get_NotFound(t *testing.T) {
	t.Parallel()

	assignmentID := pulid.MustNew("ura_")
	repo := mocks.NewMockRoleAssignmentRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errNotFound)

	handler := setupRoleAssignmentHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/role-assignments/" + assignmentID.String()).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestRoleAssignmentHandler_Get_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockRoleAssignmentRepository(t)
	handler := setupRoleAssignmentHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/role-assignments/invalid-id").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}
