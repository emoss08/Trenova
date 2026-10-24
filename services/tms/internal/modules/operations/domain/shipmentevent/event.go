package shipmentevent

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type Event struct {
	keel.Table `keel:"shipment_events"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:se_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	ShipmentID     pulid.ID `json:"shipmentId" keel:"shipment_id"`
	MoveID         pulid.ID `json:"moveId,omitempty" keel:"move_id,nullzero"`
	StopID         pulid.ID `json:"stopId,omitempty" keel:"stop_id,nullzero"`
	AssignmentID   pulid.ID `json:"assignmentId,omitempty" keel:"assignment_id,nullzero"`
	CommentID      pulid.ID `json:"commentId,omitempty" keel:"comment_id,nullzero"`
	HoldID         pulid.ID `json:"holdId,omitempty" keel:"hold_id,nullzero"`

	Type       Type      `json:"type" keel:"type"`
	Severity   Severity  `json:"severity" keel:"severity"`
	ActorType  ActorType `json:"actorType" keel:"actor_type"`
	ActorID    pulid.ID  `json:"actorId,omitempty" keel:"actor_id,nullzero"`
	ActorLabel string    `json:"actorLabel" keel:"actor_label,nullzero"`
	Summary    string    `json:"summary" keel:"summary"`

	Metadata      map[string]any `json:"metadata,omitempty" keel:"metadata,jsonb"`
	OccurredAt    int64          `json:"occurredAt" keel:"occurred_at"`
	CorrelationID string         `json:"correlationId,omitempty" keel:"correlation_id,nullzero"`

	Actor    *tenant.User       `json:"actor,omitempty" keel:"belongs:actor_id"`
	Shipment *shipment.Shipment `json:"shipment,omitempty" keel:"belongs:shipment_id"`
}

func (e *Event) BeforeCreate(ctx context.Context) error {
	if e.Severity == "" {
		e.Severity = SeverityMuted
	}
	if e.OccurredAt == 0 {
		e.OccurredAt = timeutils.NowUnix()
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
