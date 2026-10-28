package pto

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*WorkerPTOPolicyAssignment)(nil)
)

type WorkerPTOPolicyAssignment struct {
	keel.Table `keel:"worker_pto_policy_assignments"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:wppa_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	WorkerID       pulid.ID `json:"workerId"       keel:"worker_id"`
	PTOPolicyID    pulid.ID `json:"ptoPolicyId"    keel:"pto_policy_id"`
	EffectiveFrom  int64    `json:"effectiveFrom"  keel:"effective_from"`
	EffectiveTo    *int64   `json:"effectiveTo"    keel:"effective_to"`
	AssignedByID   pulid.ID `json:"assignedById"   keel:"assigned_by_id,nullzero"`
	Note           string   `json:"note"           keel:"note,nullzero"`
	Version        int64    `json:"version"        keel:"version,version"`
	CreatedAt      int64    `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt"      keel:"updated_at,autoupdate"`

	Policy *PTOPolicy     `json:"policy,omitempty" keel:"belongs:pto_policy_id"`
	Worker *worker.Worker `json:"worker,omitempty" keel:"belongs:worker_id"`
}

func (a *WorkerPTOPolicyAssignment) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(a,
		validation.Field(&a.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&a.PTOPolicyID, validation.Required.Error("Policy is required")),
		validation.Field(&a.EffectiveFrom,
			validation.Required.Error("Effective date is required"),
			validation.Min(int64(1)).Error("Effective date must be a positive value"),
		),
	))

	if a.EffectiveTo != nil && *a.EffectiveTo <= a.EffectiveFrom {
		multiErr.Add(
			"effectiveTo",
			errortypes.ErrInvalid,
			"End date must be after the effective date",
		)
	}
}

func (a *WorkerPTOPolicyAssignment) IsOpen() bool {
	return a.EffectiveTo == nil
}

func (a *WorkerPTOPolicyAssignment) CoversDate(unix int64) bool {
	if unix < a.EffectiveFrom {
		return false
	}
	return a.EffectiveTo == nil || unix < *a.EffectiveTo
}

func (a *WorkerPTOPolicyAssignment) GetID() pulid.ID { return a.ID }

func (a *WorkerPTOPolicyAssignment) GetCreatedAt() int64 { return a.CreatedAt }

func (a *WorkerPTOPolicyAssignment) GetOrganizationID() pulid.ID { return a.OrganizationID }

func (a *WorkerPTOPolicyAssignment) GetBusinessUnitID() pulid.ID { return a.BusinessUnitID }

func (a *WorkerPTOPolicyAssignment) GetTableName() string { return "worker_pto_policy_assignments" }

func (a *WorkerPTOPolicyAssignment) GetResourceType() string { return "worker_pto_policy_assignment" }

func (a *WorkerPTOPolicyAssignment) GetResourceID() string { return a.ID.String() }

func (WorkerPTOPolicyAssignment) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_worker_pto_policy_assignments_open",
			Message: "Worker already has an open PTO policy assignment",
		},
		{
			Name:    "fk_worker_pto_policy_assignments_worker",
			Message: "Worker does not exist or belongs to a different organization",
		},
		{
			Name:    "fk_worker_pto_policy_assignments_policy",
			Message: "PTO policy does not exist or belongs to a different organization",
		},
	}
}
