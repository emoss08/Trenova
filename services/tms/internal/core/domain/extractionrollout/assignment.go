package extractionrollout

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*RolloutAssignment)(nil)

const MaxModelRunes = 255

type RolloutAssignment struct {
	bun.BaseModel `bun:"table:extraction_rollout_assignments,alias:exra" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	DocumentID          pulid.ID  `json:"documentId"          bun:"document_id,type:VARCHAR(100),notnull"`
	ExtractedAt         int64     `json:"extractedAt"         bun:"extracted_at,type:BIGINT,notnull"`
	Arm                 Arm       `json:"arm"                 bun:"arm,type:VARCHAR(20),notnull"`
	CandidateProviderID pulid.ID  `json:"candidateProviderId" bun:"candidate_provider_id,type:VARCHAR(100),notnull"`
	ServedProviderID    *pulid.ID `json:"servedProviderId"    bun:"served_provider_id,type:VARCHAR(100),nullzero"`
	ServedModel         string    `json:"servedModel"         bun:"served_model,type:VARCHAR(255),nullzero"`
	Outcome             Outcome   `json:"outcome"             bun:"outcome,type:VARCHAR(20),notnull,default:'Pending'"`
	SettledAt           *int64    `json:"settledAt"           bun:"settled_at,type:BIGINT,nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
}

func (a *RolloutAssignment) PreferredProviderID() pulid.ID {
	if a == nil || a.Arm != ArmCandidate {
		return pulid.Nil
	}

	return a.CandidateProviderID
}

func (a *RolloutAssignment) ServedByCandidate() bool {
	return a.ServedProviderID != nil && *a.ServedProviderID == a.CandidateProviderID
}

func (a *RolloutAssignment) Settle(
	outcome Outcome,
	servedProviderID pulid.ID,
	model string,
	now int64,
) {
	a.Outcome = outcome
	if servedProviderID.IsNotNil() {
		a.ServedProviderID = &servedProviderID
	}
	if model != "" {
		a.ServedModel = model
	}
	a.SettledAt = &now
}

func (a *RolloutAssignment) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		a,
		validation.Field(&a.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&a.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&a.DocumentID, validation.Required.Error("Document is required")),
		validation.Field(&a.ExtractedAt, validation.Required.Error("Extraction time is required")),
		validation.Field(
			&a.CandidateProviderID,
			validation.Required.Error("Candidate is required"),
		),
		validation.Field(&a.Arm, validation.Required.Error("Arm is required"), validation.By(
			func(any) error {
				if a.Arm.IsValid() {
					return nil
				}
				return validation.NewError("validation_invalid_arm", "Arm is invalid")
			},
		)),
		validation.Field(&a.Outcome, validation.By(func(any) error {
			if a.Outcome.IsValid() {
				return nil
			}
			return validation.NewError("validation_invalid_outcome", "Outcome is invalid")
		})),
	))
}

func (a *RolloutAssignment) GetID() pulid.ID { return a.ID }

func (a *RolloutAssignment) GetTableName() string { return "extraction_rollout_assignments" }

func (a *RolloutAssignment) GetOrganizationID() pulid.ID { return a.OrganizationID }

func (a *RolloutAssignment) GetBusinessUnitID() pulid.ID { return a.BusinessUnitID }

func (a *RolloutAssignment) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if a.ID.IsNil() {
			a.ID = pulid.MustNew("exra_")
		}
		if a.CreatedAt == 0 {
			a.CreatedAt = now
		}
		a.UpdatedAt = now
	case *bun.UpdateQuery:
		a.UpdatedAt = now
	}

	return nil
}
