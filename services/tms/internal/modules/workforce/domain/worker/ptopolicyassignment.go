package worker

import (
	"context"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*WorkerPTOPolicyAssignment)(nil)
	_ validationframework.TenantedEntity = (*WorkerPTOPolicyAssignment)(nil)
)

type WorkerPTOPolicyAssignment struct {
	bun.BaseModel `bun:"table:worker_pto_policy_assignments,alias:wppa" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	WorkerID       pulid.ID `json:"workerId"       bun:"worker_id,type:VARCHAR(100),notnull"`
	PTOPolicyID    pulid.ID `json:"ptoPolicyId"    bun:"pto_policy_id,type:VARCHAR(100),notnull"`
	EffectiveFrom  int64    `json:"effectiveFrom"  bun:"effective_from,type:BIGINT,notnull"`
	EffectiveTo    *int64   `json:"effectiveTo"    bun:"effective_to,type:BIGINT,nullzero"`
	AssignedByID   pulid.ID `json:"assignedById"   bun:"assigned_by_id,type:VARCHAR(100),nullzero"`
	Note           string   `json:"note"           bun:"note,type:TEXT,nullzero"`
	Version        int64    `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Policy *PTOPolicy `json:"policy,omitempty" bun:"rel:belongs-to,join:pto_policy_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Worker *Worker    `json:"worker,omitempty" bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
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

func (a *WorkerPTOPolicyAssignment) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if a.ID.IsNil() {
			a.ID = pulid.MustNew("wppa_")
		}
		a.CreatedAt = now
		a.UpdatedAt = now
	case *bun.UpdateQuery:
		a.UpdatedAt = now
	}

	return nil
}
