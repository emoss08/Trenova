package extractionrollout

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const MaxModelRunes = 255

type RolloutAssignment struct {
	keel.Table `keel:"extraction_rollout_assignments"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:exra_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	DocumentID          pulid.ID  `json:"documentId" keel:"document_id"`
	ExtractedAt         int64     `json:"extractedAt" keel:"extracted_at"`
	Arm                 Arm       `json:"arm" keel:"arm"`
	CandidateProviderID pulid.ID  `json:"candidateProviderId" keel:"candidate_provider_id"`
	ServedProviderID    *pulid.ID `json:"servedProviderId" keel:"served_provider_id"`
	ServedModel         string    `json:"servedModel" keel:"served_model,nullzero"`
	Outcome             Outcome   `json:"outcome" keel:"outcome"`
	SettledAt           *int64    `json:"settledAt" keel:"settled_at"`

	Version   int64 `json:"version" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
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

func (a *RolloutAssignment) BeforeCreate(_ context.Context) error {
	if a.Outcome == "" {
		a.Outcome = OutcomePending
	}

	return nil
}
