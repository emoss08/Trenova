package roleassignmentservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type testDeps struct {
	repo *mocks.MockRoleAssignmentRepository
	svc  *Service
}

func setupTest(t *testing.T) *testDeps {
	t.Helper()
	repo := mocks.NewMockRoleAssignmentRepository(t)
	svc := &Service{
		l:    zap.NewNop(),
		repo: repo,
	}
	return &testDeps{repo: repo, svc: svc}
}

func newTestEntity() *permission.UserRoleAssignment {
	return &permission.UserRoleAssignment{
		ID:             pulid.MustNew("ura_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		RoleID:         pulid.MustNew("role_"),
		AssignedBy:     pulid.MustNew("usr_"),
		AssignedAt:     1700000000,
	}
}

func TestList_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	expected := &pagination.ListResult[*permission.UserRoleAssignment]{
		Items: []*permission.UserRoleAssignment{newTestEntity()},
		Total: 1,
	}
	req := &repositories.ListRoleAssignmentsRequest{
		Filter:      &pagination.QueryOptions{},
		ExpandRoles: true,
	}

	deps.repo.On("List", mock.Anything, req).Return(expected, nil)

	result, err := deps.svc.List(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Items, 1)
	deps.repo.AssertExpectations(t)
}

func TestList_Error(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	req := &repositories.ListRoleAssignmentsRequest{
		Filter: &pagination.QueryOptions{},
	}

	deps.repo.On("List", mock.Anything, req).Return(nil, errors.New("db error"))

	result, err := deps.svc.List(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.repo.AssertExpectations(t)
}

func TestGetByID_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	entity := newTestEntity()
	req := repositories.GetRoleAssignmentByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
		},
		RoleAssignmentID: entity.ID,
		ExpandRoles:      true,
	}

	deps.repo.On("GetByID", mock.Anything, req).Return(entity, nil)

	result, err := deps.svc.GetByID(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, entity.ID, result.ID)
	assert.Equal(t, entity.RoleID, result.RoleID)
	deps.repo.AssertExpectations(t)
}

func TestGetByID_Error(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	req := repositories.GetRoleAssignmentByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
		},
		RoleAssignmentID: pulid.MustNew("ura_"),
	}

	deps.repo.On("GetByID", mock.Anything, req).Return(nil, errors.New("not found"))

	result, err := deps.svc.GetByID(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.repo.AssertExpectations(t)
}

func TestNew(t *testing.T) {
	t.Parallel()
	repo := mocks.NewMockRoleAssignmentRepository(t)
	logger := zap.NewNop()

	svc := New(Params{
		Logger: logger,
		Repo:   repo,
	})

	require.NotNil(t, svc)
}
