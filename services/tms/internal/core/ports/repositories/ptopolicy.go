package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListPTOPoliciesRequest struct {
	Filter       *pagination.QueryOptions `json:"filter"`
	Cursor       pagination.CursorInfo    `json:"cursor"`
	Status       string                   `json:"status"`
	IncludeRules bool                     `json:"includeRules"`
}

type GetPTOPolicyByIDRequest struct {
	ID           pulid.ID              `json:"id"`
	TenantInfo   pagination.TenantInfo `json:"tenantInfo"`
	IncludeRules bool                  `json:"includeRules"`
}

type PTOPolicyCodeExistsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Code       string                `json:"code"`
	ExcludeID  pulid.ID              `json:"excludeId"`
}

type ClearDefaultPTOPolicyRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	ExceptID   pulid.ID              `json:"exceptId"`
}

type CountOpenPTOAssignmentsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	PolicyID   pulid.ID              `json:"policyId"`
}

type CountOpenPTOAssignmentsByIDsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	PolicyIDs  []pulid.ID            `json:"policyIds"`
}

type ListPTOAssignmentsRequest struct {
	TenantInfo    pagination.TenantInfo `json:"tenantInfo"`
	WorkerID      pulid.ID              `json:"workerId"`
	IncludePolicy bool                  `json:"includePolicy"`
}

type GetPTOAssignmentRequest struct {
	TenantInfo    pagination.TenantInfo `json:"tenantInfo"`
	WorkerID      pulid.ID              `json:"workerId"`
	AsOf          int64                 `json:"asOf"`
	IncludePolicy bool                  `json:"includePolicy"`
}

type GetPTOAssignmentByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ListOpenPTOAssignmentsRequest struct {
	TenantInfo    pagination.TenantInfo `json:"tenantInfo"`
	PolicyID      pulid.ID              `json:"policyId"`
	WorkerID      pulid.ID              `json:"workerId"`
	AfterID       pulid.ID              `json:"afterId"`
	Limit         int                   `json:"limit"`
	IncludePolicy bool                  `json:"includePolicy"`
}

type PTOPolicyRepository interface {
	List(
		ctx context.Context,
		req *ListPTOPoliciesRequest,
	) (*pagination.CursorListResult[*worker.PTOPolicy], error)
	GetByID(ctx context.Context, req *GetPTOPolicyByIDRequest) (*worker.PTOPolicy, error)
	GetDefault(ctx context.Context, tenantInfo pagination.TenantInfo) (*worker.PTOPolicy, error)
	CodeExists(ctx context.Context, req *PTOPolicyCodeExistsRequest) (bool, error)
	Create(ctx context.Context, entity *worker.PTOPolicy) (*worker.PTOPolicy, error)
	Update(ctx context.Context, entity *worker.PTOPolicy) (*worker.PTOPolicy, error)
	ClearDefault(ctx context.Context, req *ClearDefaultPTOPolicyRequest) error
	CountOpenAssignments(ctx context.Context, req *CountOpenPTOAssignmentsRequest) (int, error)
	CountOpenAssignmentsByIDs(
		ctx context.Context,
		req *CountOpenPTOAssignmentsByIDsRequest,
	) (map[pulid.ID]int, error)

	ListAssignments(
		ctx context.Context,
		req *ListPTOAssignmentsRequest,
	) ([]*worker.WorkerPTOPolicyAssignment, error)
	GetAssignmentByID(
		ctx context.Context,
		req *GetPTOAssignmentByIDRequest,
	) (*worker.WorkerPTOPolicyAssignment, error)
	GetActiveAssignment(
		ctx context.Context,
		req *GetPTOAssignmentRequest,
	) (*worker.WorkerPTOPolicyAssignment, error)
	CreateAssignment(
		ctx context.Context,
		entity *worker.WorkerPTOPolicyAssignment,
	) (*worker.WorkerPTOPolicyAssignment, error)
	UpdateAssignment(
		ctx context.Context,
		entity *worker.WorkerPTOPolicyAssignment,
	) (*worker.WorkerPTOPolicyAssignment, error)
	ListOpenAssignments(
		ctx context.Context,
		req *ListOpenPTOAssignmentsRequest,
	) ([]*worker.WorkerPTOPolicyAssignment, error)
	ListTenantsWithOpenAssignments(ctx context.Context) ([]pagination.TenantInfo, error)
}
