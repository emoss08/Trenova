package telematics

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const defaultProvider = "Samsara"

type TelematicsEvent struct {
	keel.Table `keel:"telematics_events"`

	ID             pulid.ID       `json:"id"             keel:"id,pk"`
	OrganizationID pulid.ID       `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID       `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	Provider       string         `json:"provider"       keel:"provider"`
	EventID        string         `json:"eventId"        keel:"event_id"`
	EventType      EventType      `json:"eventType"      keel:"event_type"`
	OccurredAt     int64          `json:"occurredAt"     keel:"occurred_at"`
	TractorID      pulid.ID       `json:"tractorId"      keel:"tractor_id,nullzero"`
	WorkerID       pulid.ID       `json:"workerId"       keel:"worker_id,nullzero"`
	LocationID     pulid.ID       `json:"locationId"     keel:"location_id,nullzero"`
	AddressName    string         `json:"addressName"    keel:"address_name,nullzero"`
	Payload        map[string]any `json:"payload"        keel:"payload,jsonb,nullzero"`
	CreatedAt      int64          `json:"createdAt"      keel:"created_at"`

	StopOutcome       StopOutcome             `json:"stopOutcome"       keel:"stop_outcome,nullzero"`
	StopVisit         shipmentvocab.VisitKind `json:"stopVisit"         keel:"stop_visit,nullzero"`
	ShipmentMoveID    pulid.ID                `json:"shipmentMoveId"    keel:"shipment_move_id,nullzero"`
	StopID            pulid.ID                `json:"stopId"            keel:"stop_id,nullzero"`
	StopOutcomeReason string                  `json:"stopOutcomeReason" keel:"stop_outcome_reason,nullzero"`
	MoveShipmentID    pulid.ID                `json:"-"                 keel:"move_shipment_id,computed,nullzero"`
}

func (e *TelematicsEvent) BeforeCreate(_ context.Context) error {
	if e.Provider == "" {
		e.Provider = defaultProvider
	}

	return nil
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
