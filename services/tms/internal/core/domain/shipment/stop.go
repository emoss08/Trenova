package shipment

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type Stop struct {
	bun.BaseModel `json:"-" bun:"table:stops,alias:stp"`

	ID                     pulid.ID           `json:"id"                     bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID         pulid.ID           `json:"businessUnitId"         bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID         pulid.ID           `json:"organizationId"         bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	ShipmentMoveID         pulid.ID           `json:"shipmentMoveId"         bun:"shipment_move_id,type:VARCHAR(100),notnull"`
	LocationID             pulid.ID           `json:"locationId"             bun:"location_id,type:VARCHAR(100),notnull"`
	Status                 StopStatus         `json:"status"                 bun:"status,type:stop_status_enum,notnull,default:'New'"`
	Type                   StopType           `json:"type"                   bun:"type,type:stop_type_enum,notnull,default:'Pickup'"`
	ScheduleType           StopScheduleType   `json:"scheduleType"           bun:"schedule_type,type:stop_schedule_type_enum,notnull,default:'Open'"`
	Sequence               int64              `json:"sequence"               bun:"sequence,type:INTEGER,notnull"`
	Pieces                 *int64             `json:"pieces"                 bun:"pieces,type:INTEGER,nullzero"`
	Weight                 *int64             `json:"weight"                 bun:"weight,type:INTEGER,nullzero"`
	ScheduledWindowStart   int64              `json:"scheduledWindowStart"   bun:"scheduled_window_start,type:BIGINT,notnull"`
	ScheduledWindowEnd     *int64             `json:"scheduledWindowEnd"     bun:"scheduled_window_end,type:BIGINT,nullzero"`
	ActualArrival          *int64             `json:"actualArrival"          bun:"actual_arrival,type:BIGINT,nullzero"`
	ActualDeparture        *int64             `json:"actualDeparture"        bun:"actual_departure,type:BIGINT,nullzero"`
	CountLateOverride      *bool              `json:"countLateOverride"      bun:"count_late_override,type:BOOLEAN,nullzero"`
	CountDetentionOverride *bool              `json:"countDetentionOverride" bun:"count_detention_override,type:BOOLEAN,nullzero"`
	AddressLine            string             `json:"addressLine"            bun:"address_line,type:VARCHAR(200),nullzero"`
	Version                int64              `json:"version"                bun:"version,type:BIGINT"`
	CreatedAt              int64              `json:"createdAt"              bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt              int64              `json:"updatedAt"              bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	Location               *location.Location `json:"location,omitempty"     bun:"rel:belongs-to,join:location_id=id"`
}

func (s *Stop) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	if s.ScheduleType == "" {
		s.ScheduleType = StopScheduleTypeOpen
	}

	switch query.(type) {
	case *bun.InsertQuery:
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("stp_")
		}
		s.CreatedAt = now
	case *bun.UpdateQuery:
		s.UpdatedAt = now
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
