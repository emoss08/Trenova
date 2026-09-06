package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListWorkerPoliciesRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	ActiveOnly bool                  `json:"activeOnly"`
	// AsOf drops policies that have not taken effect yet.
	AsOf  int64 `json:"asOf"`
	Limit int   `json:"limit"`
}

type GetWorkerPolicyByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ListPolicyAcknowledgementsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	PolicyID   pulid.ID              `json:"policyId"`
	WorkerID   pulid.ID              `json:"workerId"`
	// VersionLabel narrows to signatures on one version, which is the one
	// question compliance asks: who has signed what is in force now.
	VersionLabel  string `json:"versionLabel"`
	IncludeWorker bool   `json:"includeWorker"`
	Limit         int    `json:"limit"`
}

// PolicyComplianceRow is who a policy binds and whether they have signed the
// version in force. It is derived on read: the roster and the acknowledgements
// both move, and a stored answer would be wrong by the next hire.
type PolicyComplianceRow struct {
	WorkerID       pulid.ID `bun:"worker_id"`
	FirstName      string   `bun:"first_name"`
	LastName       string   `bun:"last_name"`
	WorkerType     string   `bun:"worker_type"`
	AcknowledgedAt int64    `bun:"acknowledged_at"`
	SignatureName  string   `bun:"signature_name"`
}

type PolicyComplianceRequest struct {
	TenantInfo   pagination.TenantInfo
	PolicyID     pulid.ID
	VersionLabel string
	AppliesTo    worker.PolicyAudience
	Limit        int
}

type ListProfileChangeRequestsRequest struct {
	TenantInfo pagination.TenantInfo        `json:"tenantInfo"`
	WorkerID   pulid.ID                     `json:"workerId"`
	Statuses   []worker.ProfileChangeStatus `json:"statuses"`
	// ManagerIDs narrows the queue to the people a manager answers for.
	ManagerIDs    []pulid.ID `json:"managerIds"`
	IncludeWorker bool       `json:"includeWorker"`
	Limit         int        `json:"limit"`
}

type GetProfileChangeRequestByIDRequest struct {
	ID            pulid.ID              `json:"id"`
	TenantInfo    pagination.TenantInfo `json:"tenantInfo"`
	IncludeWorker bool                  `json:"includeWorker"`
}

type SelfServiceRepository interface {
	ListPolicies(ctx context.Context, req *ListWorkerPoliciesRequest) ([]*worker.WorkerPolicy, error)
	GetPolicyByID(
		ctx context.Context,
		req *GetWorkerPolicyByIDRequest,
	) (*worker.WorkerPolicy, error)
	CreatePolicy(ctx context.Context, entity *worker.WorkerPolicy) (*worker.WorkerPolicy, error)
	UpdatePolicy(ctx context.Context, entity *worker.WorkerPolicy) (*worker.WorkerPolicy, error)

	ListAcknowledgements(
		ctx context.Context,
		req *ListPolicyAcknowledgementsRequest,
	) ([]*worker.WorkerPolicyAcknowledgement, error)
	CreateAcknowledgement(
		ctx context.Context,
		entity *worker.WorkerPolicyAcknowledgement,
	) (*worker.WorkerPolicyAcknowledgement, error)
	// CountAcknowledgements is how many signatures a version carries, which is
	// what decides whether its words may still change.
	CountAcknowledgements(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		policyID pulid.ID,
		versionLabel string,
	) (int, error)
	// PolicyCompliance is every active worker the policy binds, with their
	// signature on the version in force if they have given one.
	PolicyCompliance(
		ctx context.Context,
		req *PolicyComplianceRequest,
	) ([]PolicyComplianceRow, error)

	ListChangeRequests(
		ctx context.Context,
		req *ListProfileChangeRequestsRequest,
	) ([]*worker.WorkerProfileChangeRequest, error)
	GetChangeRequestByID(
		ctx context.Context,
		req *GetProfileChangeRequestByIDRequest,
	) (*worker.WorkerProfileChangeRequest, error)
	CreateChangeRequest(
		ctx context.Context,
		entity *worker.WorkerProfileChangeRequest,
	) (*worker.WorkerProfileChangeRequest, error)
	UpdateChangeRequest(
		ctx context.Context,
		entity *worker.WorkerProfileChangeRequest,
	) (*worker.WorkerProfileChangeRequest, error)
}
