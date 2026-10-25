package documentshipmentdraft

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type Status string

const (
	StatusUnavailable Status = "Unavailable"
	StatusPending     Status = "Pending"
	StatusReady       Status = "Ready"
	StatusFailed      Status = "Failed"
)

type DocumentShipmentDraft struct {
	keel.Table `keel:"document_shipment_drafts"`

	ID                 pulid.ID       `json:"id"                 keel:"id,pk,autoid:dsd_"`
	DocumentID         pulid.ID       `json:"documentId"         keel:"document_id"`
	OrganizationID     pulid.ID       `json:"organizationId"     keel:"organization_id,pk,tenant"`
	BusinessUnitID     pulid.ID       `json:"businessUnitId"     keel:"business_unit_id,pk,tenant"`
	Status             Status         `json:"status"             keel:"status"`
	DocumentKind       string         `json:"documentKind"       keel:"document_kind,nullzero"`
	Confidence         float64        `json:"confidence"         keel:"confidence"`
	DraftData          map[string]any `json:"draftData"          keel:"draft_data,jsonb"`
	FailureCode        string         `json:"failureCode"        keel:"failure_code,nullzero"`
	FailureMessage     string         `json:"failureMessage"     keel:"failure_message,nullzero"`
	AttachedShipmentID *pulid.ID      `json:"attachedShipmentId" keel:"attached_shipment_id"`
	AttachedAt         *int64         `json:"attachedAt"         keel:"attached_at"`
	AttachedByID       *pulid.ID      `json:"attachedById"       keel:"attached_by_id"`
	Version            int64          `json:"version"            keel:"version,version"`
	CreatedAt          int64          `json:"createdAt"          keel:"created_at,autocreate"`
	UpdatedAt          int64          `json:"updatedAt"          keel:"updated_at,autoupdate"`
}

func (d *DocumentShipmentDraft) BeforeCreate(ctx context.Context) error {
	if d.Status == "" {
		d.Status = StatusUnavailable
	}

	return nil
}

func (d *DocumentShipmentDraft) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(d,
		validation.Field(&d.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&d.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&d.DocumentID, validation.Required.Error("Document is required")),
		validation.Field(&d.Status,
			validation.Required.Error("Status is required"),
			validation.In(StatusUnavailable, StatusPending, StatusReady, StatusFailed).
				Error("Status is invalid"),
		),
		validation.Field(&d.DocumentKind,
			validation.Length(0, maxDraftKindLength).
				Error("Document kind cannot be longer than 100 characters"),
		),
		validation.Field(&d.Confidence,
			validation.Min(0.0).Error("Confidence must be between 0 and 1"),
			validation.Max(1.0).Error("Confidence must be between 0 and 1"),
		),
		validation.Field(&d.FailureCode,
			validation.Length(0, maxDraftFailureCodeLength).
				Error("Failure code cannot be longer than 100 characters"),
		),
		validation.Field(&d.FailureMessage,
			validation.When(d.Status == StatusFailed, validation.Required.Error(
				"A failed draft must record why it failed",
			)),
		),
		// Attaching a draft to a shipment is an authorship record: what it was
		// attached to, when, and by whom travel together or the audit trail is
		// incomplete.
		validation.Field(&d.AttachedAt,
			validation.When(d.AttachedShipmentID != nil, validation.Required.Error(
				"An attached draft must record when it was attached",
			)),
		),
		validation.Field(&d.AttachedByID,
			validation.When(d.AttachedShipmentID != nil, validation.Required.Error(
				"An attached draft must record who attached it",
			)),
		),
		validation.Field(&d.AttachedShipmentID,
			validation.When(d.AttachedAt != nil, validation.Required.Error(
				"An attached draft must name the shipment it was attached to",
			)),
		),
	))
}

const (
	maxDraftKindLength        = 100
	maxDraftFailureCodeLength = 100
)
