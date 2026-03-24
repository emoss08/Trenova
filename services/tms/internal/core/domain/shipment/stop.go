package shipment

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

type Stop struct {
	bun.BaseModel `json:"-" bun:"table:stops,alias:stp"`

	ID                     pulid.ID           `json:"id"                 bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID         pulid.ID           `json:"businessUnitId"     bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID         pulid.ID           `json:"organizationId"     bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	ShipmentMoveID         pulid.ID           `json:"shipmentMoveId"     bun:"shipment_move_id,type:VARCHAR(100),notnull"`
	LocationID             pulid.ID           `json:"locationId"         bun:"location_id,type:VARCHAR(100),notnull"`
	Status                 StopStatus         `json:"status"             bun:"status,type:stop_status_enum,notnull,default:'New'"`
	Type                   StopType           `json:"type"               bun:"type,type:stop_type_enum,notnull,default:'Pickup'"`
	ScheduleType           StopScheduleType   `json:"scheduleType"       bun:"schedule_type,type:stop_schedule_type_enum,notnull,default:'Open'"`
	Sequence               int64              `json:"sequence"           bun:"sequence,type:INTEGER,notnull,default:0"`
	Pieces                 *int64             `json:"pieces"             bun:"pieces,type:INTEGER,nullzero"`
	Weight                 *int64             `json:"weight"             bun:"weight,type:INTEGER,nullzero"`
	ScheduledWindowStart   int64              `json:"scheduledWindowStart" bun:"scheduled_window_start,type:BIGINT,notnull"`
	ScheduledWindowEnd     *int64             `json:"scheduledWindowEnd"   bun:"scheduled_window_end,type:BIGINT,nullzero"`
	ActualArrival          *int64             `json:"actualArrival"        bun:"actual_arrival,type:BIGINT,nullzero"`
	ActualDeparture        *int64             `json:"actualDeparture"      bun:"actual_departure,type:BIGINT,nullzero"`
	CountLateOverride      *bool              `json:"countLateOverride"    bun:"count_late_override,type:BOOLEAN,nullzero"`
	CountDetentionOverride *bool              `json:"countDetentionOverride" bun:"count_detention_override,type:BOOLEAN,nullzero"`
	AddressLine            string             `json:"addressLine"          bun:"address_line,type:VARCHAR(200),nullzero"`
	Version                int64              `json:"version"              bun:"version,type:BIGINT"`
	CreatedAt              int64              `json:"createdAt"            bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt              int64              `json:"updatedAt"            bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	Location               *location.Location `json:"location,omitempty"   bun:"rel:belongs-to,join:location_id=id"`
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

func (s *Stop) EffectiveScheduledWindowEnd() int64 {
	if s == nil {
		return 0
	}

	if s.ScheduledWindowEnd != nil {
		return *s.ScheduledWindowEnd
	}

	return s.ScheduledWindowStart
}

func (s *Stop) HasScheduledWindow() bool {
	return s != nil && s.ScheduledWindowEnd != nil
}
