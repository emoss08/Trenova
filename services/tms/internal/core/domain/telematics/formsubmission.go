package telematics

import (
	"github.com/emoss08/trenova/shared/pulid"
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
	Applied              bool             `json:"applied"              bun:"applied,type:BOOLEAN,notnull,default:false"`
	AppliedFields        int              `json:"appliedFields"        bun:"applied_fields,type:INT,notnull,default:0"`
	AppliedAt            *int64           `json:"appliedAt"            bun:"applied_at,type:BIGINT,nullzero"`
	CreatedAt            int64            `json:"createdAt"            bun:"created_at,type:BIGINT,notnull"`
}

func NewFormSubmissionID() pulid.ID {
	return pulid.MustNew("tfsub_")
}
