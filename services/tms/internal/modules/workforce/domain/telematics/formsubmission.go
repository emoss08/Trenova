package telematics

import (
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type FormFieldValue struct {
	Label string `json:"label"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

type FormSubmission struct {
	bun.BaseModel `bun:"table:telematics_form_submissions,alias:tfsub" json:"-"`

	ID                   pulid.ID         `json:"id"                   bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID       pulid.ID         `json:"organizationId"       bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID       pulid.ID         `json:"businessUnitId"       bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	Provider             string           `json:"provider"             bun:"provider,type:VARCHAR(32),notnull,default:'Samsara'"`
	ProviderSubmissionID string           `json:"providerSubmissionId" bun:"provider_submission_id,type:TEXT,notnull"`
	TemplateID           string           `json:"templateId"           bun:"template_id,type:TEXT,notnull"`
	TemplateName         string           `json:"templateName"         bun:"template_name,type:TEXT,nullzero"`
	WorkerID             pulid.ID         `json:"workerId"             bun:"worker_id,type:VARCHAR(100),nullzero"`
	ShipmentID           pulid.ID         `json:"shipmentId"           bun:"shipment_id,type:VARCHAR(100),nullzero"`
	ShipmentMoveID       pulid.ID         `json:"shipmentMoveId"       bun:"shipment_move_id,type:VARCHAR(100),nullzero"`
	StopID               pulid.ID         `json:"stopId"               bun:"stop_id,type:VARCHAR(100),nullzero"`
	SubmittedAt          int64            `json:"submittedAt"          bun:"submitted_at,type:BIGINT,notnull"`
	Fields               []FormFieldValue `json:"fields"               bun:"fields,type:JSONB,nullzero"`
	Applied              bool             `json:"applied"              bun:"applied,type:BOOLEAN,notnull"`
	AppliedFields        int              `json:"appliedFields"        bun:"applied_fields,type:INT,notnull"`
	AppliedAt            *int64           `json:"appliedAt"            bun:"applied_at,type:BIGINT,nullzero"`
	CreatedAt            int64            `json:"createdAt"            bun:"created_at,type:BIGINT,notnull"`
}

func NewFormSubmissionID() pulid.ID {
	return pulid.MustNew("tfsub_")
}

func (s *FormSubmission) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&s.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&s.Provider,
			validation.Required.Error("Provider is required"),
			validation.Length(1, maxProviderLength).
				Error("Provider cannot be longer than 32 characters"),
		),
		validation.Field(&s.ProviderSubmissionID,
			validation.Required.Error("Provider submission id is required"),
		),
		validation.Field(&s.TemplateID, validation.Required.Error("Template is required")),
		validation.Field(&s.SubmittedAt,
			validation.Required.Error("Submitted at is required"),
			validation.Min(int64(1)).Error("Submitted at must be a valid timestamp"),
		),
		validation.Field(&s.AppliedFields,
			validation.Min(0).Error("Applied fields cannot be negative"),
		),
		// Applying a submission writes its answers onto a shipment, so the
		// record of when that happened travels with the flag that says it did.
		validation.Field(&s.AppliedAt,
			validation.When(
				s.Applied,
				validation.Required.Error("An applied submission must record when it was applied"),
			),
		),
	))
}
