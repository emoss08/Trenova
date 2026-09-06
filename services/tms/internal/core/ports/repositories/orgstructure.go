package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListJobPositionsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	ActiveOnly bool                  `json:"activeOnly"`
	// DrivingOnly narrows to the positions that need a CDL, which is the line
	// most compliance rules are drawn along.
	DrivingOnly      bool                 `json:"drivingOnly"`
	Department       worker.JobDepartment `json:"department"`
	IncludeReportsTo bool                 `json:"includeReportsTo"`
	Limit            int                  `json:"limit"`
}

type GetJobPositionByIDRequest struct {
	ID               pulid.ID              `json:"id"`
	TenantInfo       pagination.TenantInfo `json:"tenantInfo"`
	IncludeReportsTo bool                  `json:"includeReportsTo"`
}

type ListApprovalDelegationsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	// DelegatorID lists what one person has handed out; DelegateID lists what
	// they have been handed. Both empty lists the organisation's delegations.
	DelegatorID pulid.ID `json:"delegatorId"`
	DelegateID  pulid.ID `json:"delegateId"`
	// ActiveAt limits to delegations answering at that instant; zero means all.
	ActiveAt     int64 `json:"activeAt"`
	IncludeUsers bool  `json:"includeUsers"`
	Limit        int   `json:"limit"`
}

type GetApprovalDelegationByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

// HeadcountRow is one grouping of the active roster. The three groupings the
// office asks for — terminal, position and department — come back in the same
// shape so the page can lay them out identically.
type HeadcountRow struct {
	Key   string `bun:"key"`
	Label string `bun:"label"`
	Code  string `bun:"code"`
	Color string `bun:"color"`
	// Workers is the active headcount; Drivers is how many of them hold a
	// driving position, which is the number a safety director wants.
	Workers    int `bun:"workers"`
	Drivers    int `bun:"drivers"`
	Terminated int `bun:"terminated"`
}

// TeamMemberRow is one person on a manager's team, read from the roster cache
// so "my team" and the roster list cannot disagree.
type TeamMemberRow struct {
	WorkerID      pulid.ID `bun:"worker_id"`
	FirstName     string   `bun:"first_name"`
	LastName      string   `bun:"last_name"`
	Status        string   `bun:"status"`
	FleetCodeID   pulid.ID `bun:"fleet_code_id"`
	FleetCode     string   `bun:"fleet_code"`
	FleetColor    string   `bun:"fleet_color"`
	PositionID    pulid.ID `bun:"position_id"`
	PositionTitle string   `bun:"position_title"`
	ManagerID     pulid.ID `bun:"manager_id"`
	// Direct says the worker names this manager themselves, rather than being
	// reached through a terminal they run. The distinction matters: a terminal
	// manager covering forty drivers is not forty direct reports.
	Direct           bool   `bun:"direct"`
	ComplianceStatus string `bun:"compliance_status"`
	TrainingHealth   string `bun:"training_health"`
	SafetyRating     string `bun:"safety_rating"`
	HireDate         int64  `bun:"hire_date"`
	TerminationDate  *int64 `bun:"termination_date"`
}

// TeamScopeRequest asks which workers a user answers for.
type TeamScopeRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	// ManagerIDs is the user asking plus anybody who has delegated to them, so
	// one query answers for the whole set rather than one round trip each.
	ManagerIDs []pulid.ID `json:"managerIds"`
	// IncludeInactive keeps terminated workers in the list; a manager reading
	// their team today does not want them, and one reading a leaver's file does.
	IncludeInactive bool `json:"includeInactive"`
	Limit           int  `json:"limit"`
}

type OrgStructureRepository interface {
	ListPositions(
		ctx context.Context,
		req *ListJobPositionsRequest,
	) ([]*worker.JobPosition, error)
	GetPositionByID(
		ctx context.Context,
		req *GetJobPositionByIDRequest,
	) (*worker.JobPosition, error)
	CreatePosition(
		ctx context.Context,
		entity *worker.JobPosition,
	) (*worker.JobPosition, error)
	UpdatePosition(
		ctx context.Context,
		entity *worker.JobPosition,
	) (*worker.JobPosition, error)
	// CountPositionHolders answers how many workers sit in a position, which is
	// what stops one being archived out from under them.
	CountPositionHolders(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		positionID pulid.ID,
	) (int, error)

	ListDelegations(
		ctx context.Context,
		req *ListApprovalDelegationsRequest,
	) ([]*worker.ApprovalDelegation, error)
	GetDelegationByID(
		ctx context.Context,
		req *GetApprovalDelegationByIDRequest,
	) (*worker.ApprovalDelegation, error)
	CreateDelegation(
		ctx context.Context,
		entity *worker.ApprovalDelegation,
	) (*worker.ApprovalDelegation, error)
	UpdateDelegation(
		ctx context.Context,
		entity *worker.ApprovalDelegation,
	) (*worker.ApprovalDelegation, error)

	HeadcountByFleet(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]HeadcountRow, error)
	HeadcountByPosition(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]HeadcountRow, error)
	HeadcountByDepartment(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]HeadcountRow, error)

	TeamMembers(ctx context.Context, req *TeamScopeRequest) ([]TeamMemberRow, error)
	// ManagesWorker answers the authorization question directly rather than by
	// listing a team and searching it: a manager of four hundred drivers must
	// not pull four hundred rows to approve one day of leave.
	ManagesWorker(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		managerIDs []pulid.ID,
		workerID pulid.ID,
	) (bool, error)
}
