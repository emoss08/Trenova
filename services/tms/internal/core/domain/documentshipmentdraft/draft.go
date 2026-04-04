package documentshipmentdraft

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
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

	ID                 pulid.ID       `json:"id"                 bun:"id,type:VARCHAR(100),pk,notnull"`
	DocumentID         pulid.ID       `json:"documentId"         bun:"document_id,type:VARCHAR(100),notnull"`
	OrganizationID     pulid.ID       `json:"organizationId"     bun:"organization_id,type:VARCHAR(100),notnull,pk"`
	BusinessUnitID     pulid.ID       `json:"businessUnitId"     bun:"business_unit_id,type:VARCHAR(100),notnull,pk"`
	Status             Status         `json:"status"             bun:"status,type:document_shipment_draft_status_enum,notnull,default:'Unavailable'"`
	DocumentKind       string         `json:"documentKind"       bun:"document_kind,type:VARCHAR(100),nullzero"`
	Confidence         float64        `json:"confidence"         bun:"confidence,type:DOUBLE PRECISION,notnull,default:0"`
	DraftData          map[string]any `json:"draftData"          bun:"draft_data,type:JSONB,notnull,default:'{}'::jsonb"`
	FailureCode        string         `json:"failureCode"        bun:"failure_code,type:VARCHAR(100),nullzero"`
	FailureMessage     string         `json:"failureMessage"     bun:"failure_message,type:TEXT,nullzero"`
	AttachedShipmentID *pulid.ID      `json:"attachedShipmentId" bun:"attached_shipment_id,type:VARCHAR(100),nullzero"`
	AttachedAt         *int64         `json:"attachedAt"         bun:"attached_at,type:BIGINT,nullzero"`
	AttachedByID       *pulid.ID      `json:"attachedById"       bun:"attached_by_id,type:VARCHAR(100),nullzero"`
	Version            int64          `json:"version"            bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt          int64          `json:"createdAt"          bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt          int64          `json:"updatedAt"          bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
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
