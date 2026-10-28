package workerports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/approvaldelegation"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
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
	Key   string
	Label string
	Code  string
	Color string
	// Workers is the active headcount; Drivers is how many of them hold a
	// driving position, which is the number a safety director wants.
	Workers    int
	Drivers    int
	Terminated int
	// Staff is how many users hold the position in this organisation. The
	// front office is made of people who log in rather than workers, and a
	// chart that counted only workers would show every desk empty.
	Staff int
}

// StaffCountRow is the users holding one position, read from their
// memberships in the organisation.
type StaffCountRow struct {
	PositionID pulid.ID
	Title      string
	Code       string
	Department string
	Staff      int
}

// PositionHolderKind says which roster a holder comes from.
type PositionHolderKind string

const PositionHolderWorker PositionHolderKind = "Worker"

const PositionHolderUser PositionHolderKind = "User"

// PositionHolderRow is one person in a position, whichever roster they are
// on: a worker on a driving title, or a user on a front-office one.
type PositionHolderRow struct {
	Kind   PositionHolderKind
	ID     pulid.ID
	Name   string
	Status string
	// Detail is the terminal for a worker and the email for a user.
	Detail string
}

type SetWorkerPositionRequest struct {
	TenantInfo pagination.TenantInfo
	WorkerID   pulid.ID
	PositionID pulid.ID
}

type SetUserPositionRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	PositionID pulid.ID
}

// TeamMemberRow is one person on a manager's team, read from the roster cache
// so "my team" and the roster list cannot disagree.
type TeamMemberRow struct {
	WorkerID      pulid.ID
	FirstName     string
	LastName      string
	Status        string
	FleetCodeID   pulid.ID
	FleetCode     string
	FleetColor    string
	PositionID    pulid.ID
	PositionTitle string
	ManagerID     pulid.ID
	// Direct says the worker names this manager themselves, rather than being
	// reached through a terminal they run. The distinction matters: a terminal
	// manager covering forty drivers is not forty direct reports.
	Direct           bool
	ComplianceStatus string
	TrainingHealth   string
	SafetyRating     string
	HireDate         int64
	TerminationDate  *int64
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
	) ([]*approvaldelegation.ApprovalDelegation, error)
	GetDelegationByID(
		ctx context.Context,
		req *GetApprovalDelegationByIDRequest,
	) (*approvaldelegation.ApprovalDelegation, error)
	CreateDelegation(
		ctx context.Context,
		entity *approvaldelegation.ApprovalDelegation,
	) (*approvaldelegation.ApprovalDelegation, error)
	UpdateDelegation(
		ctx context.Context,
		entity *approvaldelegation.ApprovalDelegation,
	) (*approvaldelegation.ApprovalDelegation, error)

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

	// StaffByPosition counts the users holding each position through their
	// membership in the organisation; positions nobody holds are absent.
	StaffByPosition(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]StaffCountRow, error)
	// ListPositionHolders is everyone in a position from either roster,
	// workers first, each alphabetically.
	ListPositionHolders(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		positionID pulid.ID,
	) ([]PositionHolderRow, error)
	// SetWorkerPosition moves a worker onto a position, or off one with a nil
	// id. It touches the one column so the worker's own validation is not
	// re-run over a record nothing else about has changed.
	SetWorkerPosition(ctx context.Context, req *SetWorkerPositionRequest) error
	// SetUserPosition does the same for a user's membership in the organisation.
	// It reports ErrNotFound when the user is not a member.
	SetUserPosition(ctx context.Context, req *SetUserPositionRequest) error
}
