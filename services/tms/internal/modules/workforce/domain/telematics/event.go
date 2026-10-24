package telematics

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type TelematicsEvent struct {
	bun.BaseModel `bun:"table:telematics_events,alias:tlev" json:"-"`
	keel.Table    `                                                  keel:"telematics_events"`

	ID             pulid.ID       `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"                     keel:"id,pk"`
	OrganizationID pulid.ID       `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"        keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID       `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"       keel:"business_unit_id,pk,tenant"`
	Provider       string         `json:"provider"       bun:"provider,type:VARCHAR(32),notnull,default:'Samsara'" keel:"provider"`
	EventID        string         `json:"eventId"        bun:"event_id,type:TEXT,notnull"                          keel:"event_id"`
	EventType      EventType      `json:"eventType"      bun:"event_type,type:VARCHAR(64),notnull"                 keel:"event_type"`
	OccurredAt     int64          `json:"occurredAt"     bun:"occurred_at,type:BIGINT,notnull"                     keel:"occurred_at"`
	TractorID      pulid.ID       `json:"tractorId"      bun:"tractor_id,type:VARCHAR(100),nullzero"               keel:"tractor_id,nullzero"`
	WorkerID       pulid.ID       `json:"workerId"       bun:"worker_id,type:VARCHAR(100),nullzero"                keel:"worker_id,nullzero"`
	LocationID     pulid.ID       `json:"locationId"     bun:"location_id,type:VARCHAR(100),nullzero"              keel:"location_id,nullzero"`
	AddressName    string         `json:"addressName"    bun:"address_name,type:TEXT,nullzero"                     keel:"address_name,nullzero"`
	Payload        map[string]any `json:"payload"        bun:"payload,type:JSONB,nullzero"                         keel:"payload,jsonb,nullzero"`
	CreatedAt      int64          `json:"createdAt"      bun:"created_at,type:BIGINT,notnull"                      keel:"created_at"`

	StopOutcome       StopOutcome        `json:"stopOutcome"       bun:"stop_outcome,type:VARCHAR(20),nullzero"      keel:"stop_outcome,nullzero"`
	StopVisit         shipment.VisitKind `json:"stopVisit"         bun:"stop_visit,type:VARCHAR(20),nullzero"        keel:"stop_visit,nullzero"`
	ShipmentMoveID    pulid.ID           `json:"shipmentMoveId"    bun:"shipment_move_id,type:VARCHAR(100),nullzero" keel:"shipment_move_id,nullzero"`
	StopID            pulid.ID           `json:"stopId"            bun:"stop_id,type:VARCHAR(100),nullzero"          keel:"stop_id,nullzero"`
	StopOutcomeReason string             `json:"stopOutcomeReason" bun:"stop_outcome_reason,type:TEXT,nullzero"      keel:"stop_outcome_reason,nullzero"`
	MoveShipmentID    pulid.ID           `json:"-"                 bun:"-"                                           keel:"move_shipment_id,computed,nullzero"`
}

func NewEventID() pulid.ID {
	return pulid.MustNew("tlev_")
}

func (e *TelematicsEvent) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(e,
		validation.Field(&e.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&e.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&e.Provider,
			validation.Required.Error("Provider is required"),
			validation.Length(1, maxProviderLength).
				Error("Provider cannot be longer than 32 characters"),
		),
		// The provider's own id is what makes a redelivered event recognizable
		// as one already recorded rather than a second occurrence.
		validation.Field(&e.EventID, validation.Required.Error("Event id is required")),
		validation.Field(&e.EventType,
			validation.Required.Error("Event type is required"),
			domainvalidation.ValidEnum[EventType]("Event type is invalid"),
		),
		validation.Field(&e.OccurredAt,
			validation.Required.Error("Occurred at is required"),
			validation.Min(int64(1)).Error("Occurred at must be a valid timestamp"),
		),
		validation.Field(&e.StopOutcome,
			validation.When(e.StopOutcome != "",
				domainvalidation.ValidEnum[StopOutcome]("Stop outcome is invalid"),
			),
		),
		validation.Field(&e.StopVisit,
			validation.When(e.StopOutcome != "",
				validation.Required.Error("Stop visit is required with a stop outcome"),
			),
		),
		validation.Field(&e.ShipmentMoveID,
			validation.When(e.StopOutcome.NeedsReview(),
				validation.Required.Error("A stop outcome for review must name its move"),
			),
		),
		validation.Field(&e.StopOutcomeReason,
			validation.When(e.StopOutcome.NeedsReview(),
				validation.Required.Error("A stop outcome for review must say why"),
			),
			validation.RuneLength(0, MaxStopOutcomeReasonLength).
				Error("Stop outcome reason cannot be longer than 500 characters"),
		),
	))
}

const maxProviderLength = 32
