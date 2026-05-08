package shipmentevent

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*Event)(nil)

type Event struct {
	bun.BaseModel `bun:"table:shipment_events,alias:se"`

	ID             pulid.ID `json:"id"                     bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId"         bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId"         bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	ShipmentID     pulid.ID `json:"shipmentId"             bun:"shipment_id,type:VARCHAR(100),notnull"`
	MoveID         pulid.ID `json:"moveId,omitempty"       bun:"move_id,type:VARCHAR(100),nullzero"`
	StopID         pulid.ID `json:"stopId,omitempty"       bun:"stop_id,type:VARCHAR(100),nullzero"`
	AssignmentID   pulid.ID `json:"assignmentId,omitempty" bun:"assignment_id,type:VARCHAR(100),nullzero"`
	CommentID      pulid.ID `json:"commentId,omitempty"    bun:"comment_id,type:VARCHAR(100),nullzero"`
	HoldID         pulid.ID `json:"holdId,omitempty"       bun:"hold_id,type:VARCHAR(100),nullzero"`

	Type       Type      `json:"type"              bun:"type,type:VARCHAR(50),notnull"`
	Severity   Severity  `json:"severity"          bun:"severity,type:VARCHAR(20),notnull,default:'muted'"`
	ActorType  ActorType `json:"actorType"         bun:"actor_type,type:VARCHAR(20),notnull"`
	ActorID    pulid.ID  `json:"actorId,omitempty" bun:"actor_id,type:VARCHAR(100),nullzero"`
	ActorLabel string    `json:"actorLabel"        bun:"actor_label,type:VARCHAR(100)"`
	Summary    string    `json:"summary"           bun:"summary,type:TEXT,notnull"`

	Metadata      map[string]any `json:"metadata,omitempty"      bun:"metadata,type:JSONB,default:'{}'::jsonb"`
	OccurredAt    int64          `json:"occurredAt"              bun:"occurred_at,type:BIGINT,notnull"`
	CorrelationID string         `json:"correlationId,omitempty" bun:"correlation_id,type:VARCHAR(100)"`

	Actor    *tenant.User       `json:"actor,omitempty"    bun:"rel:belongs-to,join:actor_id=id"`
	Shipment *shipment.Shipment `json:"shipment,omitempty" bun:"rel:belongs-to,join:shipment_id=id"`
}

func (e *Event) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("se_")
		}
		if e.OccurredAt == 0 {
			e.OccurredAt = timeutils.NowUnix()
		}
		if e.Severity == "" {
			e.Severity = SeverityMuted
		}
	}
	return nil
}

func (e *Event) GetID() pulid.ID {
	return e.ID
}

func (e *Event) GetOrganizationID() pulid.ID {
	return e.OrganizationID
}

func (e *Event) GetBusinessUnitID() pulid.ID {
	return e.BusinessUnitID
}
