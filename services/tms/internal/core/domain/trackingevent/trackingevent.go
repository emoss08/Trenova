package trackingevent

import (
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type TrackingEvent struct {
	bun.BaseModel `bun:"table:tracking_events,alias:tkev" json:"-"`

	ID             pulid.ID           `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID           `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	ShipmentID     pulid.ID           `json:"shipmentId"     bun:"shipment_id,type:VARCHAR(100),notnull"`
	ShipmentMoveID pulid.ID           `json:"shipmentMoveId" bun:"shipment_move_id,type:VARCHAR(100),notnull"`
	StopID         pulid.ID           `json:"stopId"         bun:"stop_id,type:VARCHAR(100),notnull"`
	Source         Source             `json:"source"         bun:"source,type:VARCHAR(20),notnull"`
	SourceKey      string             `json:"sourceKey"      bun:"source_key,type:VARCHAR(255),notnull"`
	Kind           shipment.VisitKind `json:"kind"           bun:"kind,type:VARCHAR(20),notnull"`
	MatchMethod    MatchMethod        `json:"matchMethod"    bun:"match_method,type:VARCHAR(20),notnull"`
	EventAt        int64              `json:"eventAt"        bun:"event_at,type:BIGINT,notnull"`
	ReceivedAt     int64              `json:"receivedAt"     bun:"received_at,type:BIGINT,notnull"`
	Latitude       *float64           `json:"latitude"       bun:"latitude,type:DOUBLE PRECISION,nullzero"`
	Longitude      *float64           `json:"longitude"      bun:"longitude,type:DOUBLE PRECISION,nullzero"`
	SourceStatus   string             `json:"sourceStatus"   bun:"source_status,type:VARCHAR(64),nullzero"`
	RawReference   string             `json:"rawReference"   bun:"raw_reference,type:VARCHAR(255),nullzero"`
	ReportedByID   pulid.ID           `json:"reportedById"   bun:"reported_by_id,type:VARCHAR(100),nullzero"`
	Outcome        Outcome            `json:"outcome"        bun:"outcome,type:VARCHAR(20),notnull"`
	OutcomeReason  string             `json:"outcomeReason"  bun:"outcome_reason,type:TEXT,nullzero"`
	CreatedAt      int64              `json:"createdAt"      bun:"created_at,type:BIGINT,notnull"`
	UpdatedAt      int64              `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull"`
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
