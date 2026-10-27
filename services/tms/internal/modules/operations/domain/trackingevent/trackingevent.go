package trackingevent

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type TrackingEvent struct {
	keel.Table `keel:"tracking_events"`

	ID             pulid.ID           `json:"id"             keel:"id,pk"`
	OrganizationID pulid.ID           `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	ShipmentID     pulid.ID           `json:"shipmentId"     keel:"shipment_id"`
	ShipmentMoveID pulid.ID           `json:"shipmentMoveId" keel:"shipment_move_id"`
	StopID         pulid.ID           `json:"stopId"         keel:"stop_id"`
	Source         Source             `json:"source"         keel:"source"`
	SourceKey      string             `json:"sourceKey"      keel:"source_key"`
	Kind           shipment.VisitKind `json:"kind"           keel:"kind"`
	MatchMethod    MatchMethod        `json:"matchMethod"    keel:"match_method"`
	EventAt        int64              `json:"eventAt"        keel:"event_at"`
	ReceivedAt     int64              `json:"receivedAt"     keel:"received_at"`
	Latitude       *float64           `json:"latitude"       keel:"latitude"`
	Longitude      *float64           `json:"longitude"      keel:"longitude"`
	SourceStatus   string             `json:"sourceStatus"   keel:"source_status,nullzero"`
	RawReference   string             `json:"rawReference"   keel:"raw_reference,nullzero"`
	ReportedByID   pulid.ID           `json:"reportedById"   keel:"reported_by_id,nullzero"`
	Outcome        Outcome            `json:"outcome"        keel:"outcome"`
	OutcomeReason  string             `json:"outcomeReason"  keel:"outcome_reason,nullzero"`
	CreatedAt      int64              `json:"createdAt"      keel:"created_at"`
	UpdatedAt      int64              `json:"updatedAt"      keel:"updated_at"`
}

func NewID() pulid.ID {
	return pulid.MustNew("tkev_")
}

func (e *TrackingEvent) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(e,
		validation.Field(&e.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&e.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&e.ShipmentID, validation.Required.Error("Shipment is required")),
		validation.Field(&e.ShipmentMoveID, validation.Required.Error("Move is required")),
		validation.Field(&e.StopID, validation.Required.Error("Stop is required")),
		validation.Field(&e.Source,
			validation.Required.Error("Source is required"),
			domainvalidation.ValidEnum[Source]("Source is invalid"),
		),
		validation.Field(&e.SourceKey,
			validation.Required.Error("Source key is required"),
			validation.RuneLength(1, MaxSourceKeyLength).
				Error("Source key cannot be longer than 255 characters"),
		),
		validation.Field(&e.Kind,
			validation.Required.Error("Kind is required"),
			validation.In(shipment.VisitArrival, shipment.VisitDeparture).
				Error("Kind must be Arrival or Departure"),
		),
		validation.Field(&e.MatchMethod,
			validation.Required.Error("Match method is required"),
			domainvalidation.ValidEnum[MatchMethod]("Match method is invalid"),
		),
		validation.Field(&e.EventAt,
			validation.Required.Error("Event time is required"),
			validation.Min(int64(1)).Error("Event time must be a valid timestamp"),
		),
		validation.Field(&e.ReceivedAt,
			validation.Required.Error("Received time is required"),
			validation.Min(int64(1)).Error("Received time must be a valid timestamp"),
		),
		validation.Field(&e.Latitude,
			validation.When(e.Latitude != nil,
				validation.Min(-90.0).Error("Latitude must be between -90 and 90"),
				validation.Max(90.0).Error("Latitude must be between -90 and 90"),
			),
			validation.When(e.Longitude != nil && e.Latitude == nil,
				validation.Required.Error("Latitude is required with a longitude"),
			),
		),
		validation.Field(&e.Longitude,
			validation.When(e.Longitude != nil,
				validation.Min(-180.0).Error("Longitude must be between -180 and 180"),
				validation.Max(180.0).Error("Longitude must be between -180 and 180"),
			),
			validation.When(e.Latitude != nil && e.Longitude == nil,
				validation.Required.Error("Longitude is required with a latitude"),
			),
		),
		validation.Field(&e.SourceStatus,
			validation.RuneLength(0, MaxSourceStatusLength).
				Error("Source status cannot be longer than 64 characters"),
		),
		validation.Field(&e.RawReference,
			validation.RuneLength(0, MaxRawReferenceLength).
				Error("Raw reference cannot be longer than 255 characters"),
		),
		validation.Field(&e.Outcome,
			validation.Required.Error("Outcome is required"),
			domainvalidation.ValidEnum[Outcome]("Outcome is invalid"),
		),
		validation.Field(&e.OutcomeReason,
			validation.When(e.Outcome.NeedsReason(),
				validation.Required.Error("This outcome needs a reason"),
			),
			validation.RuneLength(0, MaxOutcomeReasonLength).
				Error("Outcome reason cannot be longer than 500 characters"),
		),
	))
}

func (TrackingEvent) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "fk_tracking_events_shipment",
			Message: "Shipment does not exist in your organization",
		},
	}
}
