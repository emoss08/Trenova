package orgstructureservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/orgstructureservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// positionRepo holds a few positions and records what was assigned, so a test
// can prove a refusal happened before anything was written.
type positionRepo struct {
	repositories.OrgStructureRepository
	positions map[pulid.ID]*worker.JobPosition
	staff     []repositories.StaffCountRow
	byPos     []repositories.HeadcountRow
	byDept    []repositories.HeadcountRow
	workerSet []repositories.SetWorkerPositionRequest
	userSet   []repositories.SetUserPositionRequest
}

func (r *positionRepo) GetPositionByID(
	_ context.Context,
	req *repositories.GetJobPositionByIDRequest,
) (*worker.JobPosition, error) {
	position, ok := r.positions[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("JobPosition not found")
	}
	return position, nil
}

func (r *positionRepo) SetWorkerPosition(
	_ context.Context,
	req *repositories.SetWorkerPositionRequest,
) error {
	r.workerSet = append(r.workerSet, *req)
	return nil
}

func (r *positionRepo) SetUserPosition(
	_ context.Context,
	req *repositories.SetUserPositionRequest,
) error {
	r.userSet = append(r.userSet, *req)
	return nil
}

func (r *positionRepo) HeadcountByFleet(
	context.Context,
	pagination.TenantInfo,
) ([]repositories.HeadcountRow, error) {
	return nil, nil
}

func (r *positionRepo) HeadcountByPosition(
	context.Context,
	pagination.TenantInfo,
) ([]repositories.HeadcountRow, error) {
	return r.byPos, nil
}

func (r *positionRepo) HeadcountByDepartment(
	context.Context,
	pagination.TenantInfo,
) ([]repositories.HeadcountRow, error) {
	return r.byDept, nil
}

func (r *positionRepo) StaffByPosition(
	context.Context,
	pagination.TenantInfo,
) ([]repositories.StaffCountRow, error) {
	return r.staff, nil
}

const (
	drivingPosition  = pulid.ID("jpos_driving")
	deskPosition     = pulid.ID("jpos_desk")
	archivedPosition = pulid.ID("jpos_archived")
	someWorker       = pulid.ID("wrk_1")
	someUser         = pulid.ID("usr_1")
)

func newPositionRepo() *positionRepo {
	return &positionRepo{
		positions: map[pulid.ID]*worker.JobPosition{
			drivingPosition: {
				ID: drivingPosition, Title: "Driver", Status: "Active", IsDrivingPosition: true,
			},
			deskPosition: {
				ID: deskPosition, Title: "Load Planner", Status: "Active", IsDrivingPosition: false,
			},
			archivedPosition: {
				ID: archivedPosition, Title: "Old", Status: "Inactive", IsDrivingPosition: false,
			},
		},
	}
}

func assign(holder, position pulid.ID) *orgstructureservice.AssignRequest {
	return &orgstructureservice.AssignRequest{
		TenantInfo: scopeTenant(),
		HolderID:   holder,
		PositionID: position,
		UserID:     managerUser,
	}
}

// A driver on a dispatch desk, or a dispatcher on a driving title, is a person
// on the wrong roster; both are refused before anything is written.
func TestAssignWorkerPosition_RefusesAFrontOfficeTitle(t *testing.T) {
	repo := newPositionRepo()
	svc := orgstructureservice.NewWithDeps(orgstructureservice.Deps{Repo: repo})

	err := svc.AssignWorkerPosition(t.Context(), assign(someWorker, deskPosition))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "held by users")
	assert.Empty(t, repo.workerSet)
}

func TestAssignUserPosition_RefusesADrivingTitle(t *testing.T) {
	repo := newPositionRepo()
	svc := orgstructureservice.NewWithDeps(orgstructureservice.Deps{Repo: repo})

	err := svc.AssignUserPosition(t.Context(), assign(someUser, drivingPosition))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "needs a driver")
	assert.Empty(t, repo.userSet)
}

func TestAssignPosition_PutsEachRosterOnItsOwnTitle(t *testing.T) {
	repo := newPositionRepo()
	svc := orgstructureservice.NewWithDeps(orgstructureservice.Deps{Repo: repo})

	require.NoError(t, svc.AssignWorkerPosition(t.Context(), assign(someWorker, drivingPosition)))
	require.NoError(t, svc.AssignUserPosition(t.Context(), assign(someUser, deskPosition)))

	require.Len(t, repo.workerSet, 1)
	assert.Equal(t, drivingPosition, repo.workerSet[0].PositionID)
	require.Len(t, repo.userSet, 1)
	assert.Equal(t, deskPosition, repo.userSet[0].PositionID)
}

func TestAssignPosition_RefusesAnArchivedTitle(t *testing.T) {
	repo := newPositionRepo()
	svc := orgstructureservice.NewWithDeps(orgstructureservice.Deps{Repo: repo})

	err := svc.AssignUserPosition(t.Context(), assign(someUser, archivedPosition))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "archived")
	assert.Empty(t, repo.userSet)
}

// Taking somebody off a position needs no position at all, and must not go
// looking one up.
func TestAssignPosition_ClearsWithoutLookingAnythingUp(t *testing.T) {
	repo := newPositionRepo()
	repo.positions = nil
	svc := orgstructureservice.NewWithDeps(orgstructureservice.Deps{Repo: repo})

	require.NoError(t, svc.AssignWorkerPosition(t.Context(), assign(someWorker, pulid.Nil)))

	require.Len(t, repo.workerSet, 1)
	assert.True(t, repo.workerSet[0].PositionID.IsNil())
}

func TestAssignPosition_NeedsSomebodyToAssign(t *testing.T) {
	repo := newPositionRepo()
	svc := orgstructureservice.NewWithDeps(orgstructureservice.Deps{Repo: repo})

	err := svc.AssignUserPosition(t.Context(), assign(pulid.Nil, deskPosition))

	require.Error(t, err)
	assert.Empty(t, repo.userSet)
}

// The front office holds titles through memberships, so the chart's counts
// have to be read from two places and laid over each other.
func TestHeadcount_LaysStaffOverTheWorkerGrouping(t *testing.T) {
	repo := newPositionRepo()
	repo.byPos = []repositories.HeadcountRow{
		{Key: drivingPosition.String(), Label: "Driver", Workers: 30, Drivers: 30},
	}
	repo.byDept = []repositories.HeadcountRow{
		{Key: "Operations", Label: "Operations", Workers: 30, Drivers: 30},
	}
	repo.staff = []repositories.StaffCountRow{
		{PositionID: deskPosition, Title: "Load Planner", Code: "LP", Department: "Operations", Staff: 3},
		{PositionID: pulid.ID("jpos_ceo"), Title: "Chief Executive", Code: "CEO", Department: "Executive", Staff: 1},
	}
	svc := orgstructureservice.NewWithDeps(orgstructureservice.Deps{Repo: repo})

	headcount, err := svc.Headcount(t.Context(), scopeTenant())
	require.NoError(t, err)

	assert.Equal(t, 4, headcount.StaffTotal)
	require.Len(t, headcount.ByPosition, 3)
	assert.Equal(t, 30, headcount.ByPosition[0].Workers)
	assert.Equal(t, 0, headcount.ByPosition[0].Staff)
	assert.Equal(t, "Load Planner", headcount.ByPosition[1].Label)
	assert.Equal(t, 3, headcount.ByPosition[1].Staff)
	assert.Equal(t, "Chief Executive", headcount.ByPosition[2].Label)

	require.Len(t, headcount.ByDepartment, 2)
	assert.Equal(t, 3, headcount.ByDepartment[0].Staff, "staff join the department they share with workers")
	assert.Equal(t, "Executive", headcount.ByDepartment[1].Key)
	assert.Equal(t, 1, headcount.ByDepartment[1].Staff)
}
