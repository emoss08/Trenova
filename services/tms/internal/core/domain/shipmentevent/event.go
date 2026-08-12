package shipmentevent

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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

	Metadata      map[string]any `json:"metadata,omitempty"      bun:"metadata,type:JSONB,default:'{}'"`
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

func (e *Event) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(e,
		validation.Field(&e.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&e.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&e.ShipmentID, validation.Required.Error("Shipment is required")),
		validation.Field(&e.Type,
			validation.Required.Error("Type is required"),
			domainvalidation.ValidEnum[Type]("Type is invalid"),
		),
		validation.Field(&e.Severity,
			validation.Required.Error("Severity is required"),
			domainvalidation.ValidEnum[Severity]("Severity is invalid"),
		),
		validation.Field(&e.ActorType,
			validation.Required.Error("Actor type is required"),
			domainvalidation.ValidEnum[ActorType]("Actor type is invalid"),
		),
		// A user-attributed entry with no user is an unattributable line on the
		// shipment timeline, which is the one thing the timeline is for.
		validation.Field(&e.ActorID,
			validation.When(
				e.ActorType == ActorUser,
				validation.Required.Error("A user event must name the user who caused it"),
			),
		),
		validation.Field(&e.ActorLabel,
			validation.Length(0, maxActorLabelLength).
				Error("Actor label cannot be longer than 100 characters"),
		),
		validation.Field(&e.Summary, validation.Required.Error("Summary is required")),
		validation.Field(&e.CorrelationID,
			validation.Length(0, maxCorrelationIDLength).
				Error("Correlation id cannot be longer than 100 characters"),
		),
		validation.Field(&e.OccurredAt,
			validation.Required.Error("Occurred at is required"),
			validation.Min(int64(1)).Error("Occurred at must be a valid timestamp"),
		),
	))
}

const (
	maxActorLabelLength    = 100
	maxCorrelationIDLength = 100
)
