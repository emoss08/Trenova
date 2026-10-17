package extractionrollout

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*RolloutExclusion)(nil)

type RolloutExclusion struct {
	bun.BaseModel `bun:"table:extraction_rollout_exclusions,alias:exrx" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	DocumentID          pulid.ID        `json:"documentId"          bun:"document_id,type:VARCHAR(100),notnull"`
	ExtractedAt         int64           `json:"extractedAt"         bun:"extracted_at,type:BIGINT,notnull"`
	CandidateProviderID pulid.ID        `json:"candidateProviderId" bun:"candidate_provider_id,type:VARCHAR(100),notnull"`
	Reason              ExclusionReason `json:"reason"              bun:"reason,type:VARCHAR(30),notnull"`

	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
}

func (e *RolloutExclusion) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		e,
		validation.Field(&e.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&e.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&e.DocumentID, validation.Required.Error("Document is required")),
		validation.Field(&e.ExtractedAt, validation.Required.Error("Extraction time is required")),
		validation.Field(
			&e.CandidateProviderID,
			validation.Required.Error("Candidate is required"),
		),
		validation.Field(&e.Reason, validation.By(func(any) error {
			if e.Reason.IsValid() {
				return nil
			}
			return validation.NewError("validation_invalid_reason", "Reason is invalid")
		})),
	))
}

func (e *RolloutExclusion) GetID() pulid.ID { return e.ID }

func (e *RolloutExclusion) GetTableName() string { return "extraction_rollout_exclusions" }

func (e *RolloutExclusion) GetOrganizationID() pulid.ID { return e.OrganizationID }

func (e *RolloutExclusion) GetBusinessUnitID() pulid.ID { return e.BusinessUnitID }

func (e *RolloutExclusion) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("exrx_")
		}
		if e.CreatedAt == 0 {
			e.CreatedAt = timeutils.NowUnix()
		}
	}

	return nil
}
