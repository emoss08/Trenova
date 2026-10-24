package documentshipmentdraft

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type Status string

const (
	StatusUnavailable Status = "Unavailable"
	StatusPending     Status = "Pending"
	StatusReady       Status = "Ready"
	StatusFailed      Status = "Failed"
)

type DocumentShipmentDraft struct {
	bun.BaseModel `bun:"table:document_shipment_drafts,alias:dsd" json:"-"`
	keel.Table    `keel:"document_shipment_drafts"`

	ID                 pulid.ID       `json:"id"                 bun:"id,type:VARCHAR(100),pk,notnull" keel:"id,pk,autoid:dsd_"`
	DocumentID         pulid.ID       `json:"documentId"         bun:"document_id,type:VARCHAR(100),notnull" keel:"document_id"`
	OrganizationID     pulid.ID       `json:"organizationId"     bun:"organization_id,type:VARCHAR(100),notnull,pk" keel:"organization_id,pk,tenant"`
	BusinessUnitID     pulid.ID       `json:"businessUnitId"     bun:"business_unit_id,type:VARCHAR(100),notnull,pk" keel:"business_unit_id,pk,tenant"`
	Status             Status         `json:"status"             bun:"status,type:document_shipment_draft_status_enum,notnull,default:'Unavailable'" keel:"status"`
	DocumentKind       string         `json:"documentKind"       bun:"document_kind,type:VARCHAR(100),nullzero" keel:"document_kind,nullzero"`
	Confidence         float64        `json:"confidence"         bun:"confidence,type:DOUBLE PRECISION,notnull" keel:"confidence"`
	DraftData          map[string]any `json:"draftData"          bun:"draft_data,type:JSONB,notnull,default:'{}'" keel:"draft_data,jsonb"`
	FailureCode        string         `json:"failureCode"        bun:"failure_code,type:VARCHAR(100),nullzero" keel:"failure_code,nullzero"`
	FailureMessage     string         `json:"failureMessage"     bun:"failure_message,type:TEXT,nullzero" keel:"failure_message,nullzero"`
	AttachedShipmentID *pulid.ID      `json:"attachedShipmentId" bun:"attached_shipment_id,type:VARCHAR(100),nullzero" keel:"attached_shipment_id"`
	AttachedAt         *int64         `json:"attachedAt"         bun:"attached_at,type:BIGINT,nullzero" keel:"attached_at"`
	AttachedByID       *pulid.ID      `json:"attachedById"       bun:"attached_by_id,type:VARCHAR(100),nullzero" keel:"attached_by_id"`
	Version            int64          `json:"version"            bun:"version,type:BIGINT,notnull" keel:"version,version"`
	CreatedAt          int64          `json:"createdAt"          bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt          int64          `json:"updatedAt"          bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
}

func (d *DocumentShipmentDraft) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if d.ID.IsNil() {
			d.ID = pulid.MustNew("dsd_")
		}
		if d.Status == "" {
			d.Status = StatusUnavailable
		}
		d.CreatedAt = now
		d.UpdatedAt = now
	case *bun.UpdateQuery:
		d.UpdatedAt = now
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
