package roleassignmenthandler_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/roleassignmenthandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/modules/identity/app/roleassignmentservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks/permissionportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var errNotFound = errors.New("role assignment not found")

func setupRoleAssignmentHandler(
	t *testing.T,
	repo *permissionportsmock.MockRoleAssignmentRepository,
) *roleassignmenthandler.Handler {
	t.Helper()

	logger := zap.NewNop()

	service := roleassignmentservice.New(roleassignmentservice.Params{
		Logger: logger,
		Repo:   repo,
	})

	cfg := &config.Config{
		App: appconfig.Config{
			Debug: true,
		},
	}

	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger:    logger,
		AppConfig: &cfg.App,
	})

	return roleassignmenthandler.New(roleassignmenthandler.Params{
		Service:      service,
		ErrorHandler: errorHandler,
	})
}

func TestRoleAssignmentHandler_List_Success(t *testing.T) {
	t.Parallel()

	assignmentID := pulid.MustNew("ura_")
	repo := permissionportsmock.NewMockRoleAssignmentRepository(t)
	repo.On("List", mock.Anything, mock.MatchedBy(func(req *permissionports.ListRoleAssignmentsRequest) bool {
		return req.Filter.TenantInfo.OrgID == testutil.TestOrgID
	})).
		Return(&pagination.ListResult[*rbac.UserRoleAssignment]{
			Items: []*rbac.UserRoleAssignment{
				{
					ID:             assignmentID,
					UserID:         pulid.MustNew("usr_"),
					OrganizationID: testutil.TestOrgID,
					RoleID:         pulid.MustNew("rol_"),
				},
			},
			Total:      1,
			TotalExact: true,
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

	const requestedLimit = 10

	items := make([]*rbac.UserRoleAssignment, 0, requestedLimit+1)
	for range requestedLimit + 1 {
		items = append(items, &rbac.UserRoleAssignment{
			ID:             pulid.MustNew("ura_"),
			OrganizationID: testutil.TestOrgID,
			UserID:         pulid.MustNew("usr_"),
			RoleID:         pulid.MustNew("rol_"),
			AssignedAt:     timeutils.NowUnix(),
		})
	}

	repo := permissionportsmock.NewMockRoleAssignmentRepository(t)
	repo.On("List", mock.Anything, mock.MatchedBy(func(req *permissionports.ListRoleAssignmentsRequest) bool {
		return req.Filter.TenantInfo.OrgID == testutil.TestOrgID &&
			req.Filter.Pagination.Limit == requestedLimit+1
	})).
		Return(&pagination.ListResult[*rbac.UserRoleAssignment]{
			Items:      items,
			Total:      30,
			TotalExact: true,
		}, nil)

	handler := setupRoleAssignmentHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/role-assignments/").
		WithQuery(map[string]string{"limit": "10"}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp pagination.CursorResponse[[]map[string]any]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, 30, resp.Count)
	assert.Len(t, resp.Results, requestedLimit)
	assert.True(t, resp.HasNextPage)
	assert.NotEmpty(t, resp.EndCursor)
}

func TestRoleAssignmentHandler_List_RejectsOffset(t *testing.T) {
	t.Parallel()

	handler := setupRoleAssignmentHandler(t, permissionportsmock.NewMockRoleAssignmentRepository(t))

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/role-assignments/").
		WithQuery(map[string]string{"limit": "10", "offset": "0"}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestRoleAssignmentHandler_Get_Success(t *testing.T) {
	t.Parallel()

	assignmentID := pulid.MustNew("ura_")
	repo := permissionportsmock.NewMockRoleAssignmentRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&rbac.UserRoleAssignment{
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
	repo := permissionportsmock.NewMockRoleAssignmentRepository(t)
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

	repo := permissionportsmock.NewMockRoleAssignmentRepository(t)
	handler := setupRoleAssignmentHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/role-assignments/invalid-id").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}
