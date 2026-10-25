package telematics

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type FormFieldValue struct {
	Label string `json:"label"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

type FormSubmission struct {
	keel.Table `keel:"telematics_form_submissions"`

	ID                   pulid.ID         `json:"id"                   keel:"id,pk"`
	OrganizationID       pulid.ID         `json:"organizationId"       keel:"organization_id,pk,tenant"`
	BusinessUnitID       pulid.ID         `json:"businessUnitId"       keel:"business_unit_id,pk,tenant"`
	Provider             string           `json:"provider"             keel:"provider"`
	ProviderSubmissionID string           `json:"providerSubmissionId" keel:"provider_submission_id"`
	TemplateID           string           `json:"templateId"           keel:"template_id"`
	TemplateName         string           `json:"templateName"         keel:"template_name,nullzero"`
	WorkerID             pulid.ID         `json:"workerId"             keel:"worker_id,nullzero"`
	ShipmentID           pulid.ID         `json:"shipmentId"           keel:"shipment_id,nullzero"`
	ShipmentMoveID       pulid.ID         `json:"shipmentMoveId"       keel:"shipment_move_id,nullzero"`
	StopID               pulid.ID         `json:"stopId"               keel:"stop_id,nullzero"`
	SubmittedAt          int64            `json:"submittedAt"          keel:"submitted_at"`
	Fields               []FormFieldValue `json:"fields"               keel:"fields,jsonb,nullzero"`
	Applied              bool             `json:"applied"              keel:"applied"`
	AppliedFields        int              `json:"appliedFields"        keel:"applied_fields"`
	AppliedAt            *int64           `json:"appliedAt"            keel:"applied_at"`
	CreatedAt            int64            `json:"createdAt"            keel:"created_at"`
}

func (s *FormSubmission) BeforeCreate(_ context.Context) error {
	if s.Provider == "" {
		s.Provider = defaultProvider
	}

	return nil
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
