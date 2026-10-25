package shipment

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// Pickup and delivery stops on a move
type Stop struct {
	keel.Table `keel:"stops,report=Operations"`

	ID                     pulid.ID           `json:"id"                     keel:"id,pk,autoid:stp_"`
	BusinessUnitID         pulid.ID           `json:"businessUnitId"         keel:"business_unit_id,pk,tenant"`
	OrganizationID         pulid.ID           `json:"organizationId"         keel:"organization_id,pk,tenant"`
	ShipmentMoveID         pulid.ID           `json:"shipmentMoveId"         keel:"shipment_move_id"`
	LocationID             pulid.ID           `json:"locationId"             keel:"location_id"`
	Status                 StopStatus         `json:"status"                 keel:"status"`
	Type                   StopType           `json:"type"                   keel:"type"`
	ScheduleType           StopScheduleType   `json:"scheduleType"           keel:"schedule_type"`
	Sequence               int64              `json:"sequence"               keel:"sequence,dimension"`
	Pieces                 *int64             `json:"pieces"                 keel:"pieces,format=count"`
	Weight                 *int64             `json:"weight"                 keel:"weight,format=weight"`
	ScheduledWindowStart   int64              `json:"scheduledWindowStart"   keel:"scheduled_window_start,type=epoch"`
	ScheduledWindowEnd     *int64             `json:"scheduledWindowEnd"     keel:"scheduled_window_end,type=epoch"`
	ActualArrival          *int64             `json:"actualArrival"          keel:"actual_arrival,type=epoch"`
	ActualDeparture        *int64             `json:"actualDeparture"        keel:"actual_departure,type=epoch"`
	CountLateOverride      *bool              `json:"countLateOverride"      keel:"count_late_override"`
	CountDetentionOverride *bool              `json:"countDetentionOverride" keel:"count_detention_override"`
	AddressLine            string             `json:"addressLine"            keel:"address_line,nullzero"`
	Version                int64              `json:"version"                keel:"version,version"`
	CreatedAt              int64              `json:"createdAt"              keel:"created_at,autocreate"`
	UpdatedAt              int64              `json:"updatedAt"              keel:"updated_at,autoupdate"`
	Location               *location.Location `json:"location,omitempty"     keel:"belongs:location_id"`
}

func (s *Stop) BeforeCreate(ctx context.Context) error {
	if s.Status == "" {
		s.Status = StopStatusNew
	}
	if s.Type == "" {
		s.Type = StopTypePickup
	}
	if s.ScheduleType == "" {
		s.ScheduleType = StopScheduleTypeOpen
	}

	return nil
}

func (s *Stop) BeforeUpdate(ctx context.Context) error {
	if s.ScheduleType == "" {
		s.ScheduleType = StopScheduleTypeOpen
	}

	return nil
}

func (s *Stop) IsOriginStop() bool {
	return s.Type == StopTypePickup || s.Type == StopTypeSplitPickup
}

func (s *Stop) StatusEquals(status StopStatus) bool {
	return s.Status == status
}

func (s *Stop) IsDestinationStop() bool {
	return s.Type == StopTypeDelivery || s.Type == StopTypeSplitDelivery
}

func (s *Stop) IsCompleted() bool {
	return s.Status == StopStatusCompleted
}

func (s *Stop) IsCanceled() bool {
	return s.Status == StopStatusCanceled
}

func (s *Stop) IsInTransit() bool {
	return s.Status == StopStatusInTransit
}

func (s *Stop) IsNew() bool {
	return s.Status == StopStatusNew
}

func (s *Stop) EffectiveScheduledCutoff() int64 {
	if s == nil {
		return 0
	}

	if s.ScheduledWindowEnd != nil {
		return *s.ScheduledWindowEnd
	}

	return s.ScheduledWindowStart
}

func (s *Stop) EffectiveScheduledWindowEnd() int64 {
	return s.EffectiveScheduledCutoff()
}

func (s *Stop) HasScheduledWindow() bool {
	return s != nil && s.ScheduledWindowEnd != nil
}

func (s *Stop) Validate(multiErr *errortypes.MultiError) {
	// Read once: the rule below is built before ozzo decides whether to run it,
	// so the dereference cannot be left inside the condition.
	var arrivedAt int64
	if s.ActualArrival != nil {
		arrivedAt = *s.ActualArrival
	}

	// Tenancy and the owning move are stamped by the repository at write time,
	// so a stop arriving as part of a shipment does not carry them yet.
	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.LocationID, validation.Required.Error("Location is required")),
		validation.Field(&s.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[StopStatus]("Status is invalid"),
		),
		validation.Field(&s.Type,
			validation.Required.Error("Type is required"),
			domainvalidation.ValidEnum[StopType]("Type is invalid"),
		),
		validation.Field(&s.ScheduleType,
			validation.Required.Error("Schedule type is required"),
			domainvalidation.ValidEnum[StopScheduleType]("Schedule type is invalid"),
		),
		validation.Field(&s.Sequence,
			validation.Min(int64(0)).Error("Sequence cannot be negative"),
		),
		validation.Field(&s.Pieces,
			validation.Min(int64(0)).Error("Pieces cannot be negative"),
		),
		validation.Field(&s.Weight,
			validation.Min(int64(0)).Error("Weight cannot be negative"),
		),
		// The appointment window is what lateness and detention are measured
		// against, so an inverted one would compute both against a window that
		// closed before it opened.
		validation.Field(&s.ScheduledWindowStart,
			validation.Required.Error("Scheduled window start is required"),
			validation.Min(int64(1)).Error("Scheduled window start must be a valid timestamp"),
		),
		validation.Field(&s.ScheduledWindowEnd,
			validation.Min(s.ScheduledWindowStart).
				Error("Scheduled window end cannot precede its start"),
		),
		// A departure before the arrival would report negative dwell, which is
		// what a detention charge is calculated from.
		validation.Field(&s.ActualDeparture,
			validation.When(s.ActualArrival != nil,
				validation.Min(arrivedAt).Error("Departure cannot precede arrival"),
			),
		),
		validation.Field(&s.AddressLine,
			validation.Length(0, maxStopAddressLineLength).
				Error("Address line cannot be longer than 200 characters"),
		),
	))
}

const maxStopAddressLineLength = 200
