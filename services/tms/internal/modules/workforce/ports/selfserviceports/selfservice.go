package selfserviceports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/profilechange"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerpolicy"
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
	WorkerID       pulid.ID
	FirstName      string
	LastName       string
	WorkerType     string
	AcknowledgedAt int64
	SignatureName  string
}

type PolicyComplianceRequest struct {
	TenantInfo   pagination.TenantInfo
	PolicyID     pulid.ID
	VersionLabel string
	AppliesTo    workerpolicy.PolicyAudience
	Limit        int
}

type ListProfileChangeRequestsRequest struct {
	TenantInfo pagination.TenantInfo               `json:"tenantInfo"`
	WorkerID   pulid.ID                            `json:"workerId"`
	Statuses   []profilechange.ProfileChangeStatus `json:"statuses"`
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
	ListPolicies(
		ctx context.Context,
		req *ListWorkerPoliciesRequest,
	) ([]*workerpolicy.WorkerPolicy, error)
	GetPolicyByID(
		ctx context.Context,
		req *GetWorkerPolicyByIDRequest,
	) (*workerpolicy.WorkerPolicy, error)
	CreatePolicy(
		ctx context.Context,
		entity *workerpolicy.WorkerPolicy,
	) (*workerpolicy.WorkerPolicy, error)
	UpdatePolicy(
		ctx context.Context,
		entity *workerpolicy.WorkerPolicy,
	) (*workerpolicy.WorkerPolicy, error)

	ListAcknowledgements(
		ctx context.Context,
		req *ListPolicyAcknowledgementsRequest,
	) ([]*workerpolicy.WorkerPolicyAcknowledgement, error)
	CreateAcknowledgement(
		ctx context.Context,
		entity *workerpolicy.WorkerPolicyAcknowledgement,
	) (*workerpolicy.WorkerPolicyAcknowledgement, error)
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
	) ([]*profilechange.WorkerProfileChangeRequest, error)
	GetChangeRequestByID(
		ctx context.Context,
		req *GetProfileChangeRequestByIDRequest,
	) (*profilechange.WorkerProfileChangeRequest, error)
	CreateChangeRequest(
		ctx context.Context,
		entity *profilechange.WorkerProfileChangeRequest,
	) (*profilechange.WorkerProfileChangeRequest, error)
	UpdateChangeRequest(
		ctx context.Context,
		entity *profilechange.WorkerProfileChangeRequest,
	) (*profilechange.WorkerProfileChangeRequest, error)
}
