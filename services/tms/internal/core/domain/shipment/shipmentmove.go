package shipment

import (
	"context"

	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type ShipmentMove struct {
	bun.BaseModel `json:"-" bun:"table:shipment_moves,alias:sm"`

	ID                     pulid.ID       `json:"id"                   bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID         pulid.ID       `json:"businessUnitId"       bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID         pulid.ID       `json:"organizationId"       bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	ShipmentID             pulid.ID       `json:"shipmentId"           bun:"shipment_id,type:VARCHAR(100),notnull"`
	Status                 MoveStatus     `json:"status"               bun:"status,type:move_status_enum,notnull,default:'New'"`
	Loaded                 bool           `json:"loaded"               bun:"loaded,type:BOOLEAN,notnull,default:true"`
	Sequence               int64          `json:"sequence"             bun:"sequence,type:INTEGER,notnull,default:0"`
	Distance               *float64       `json:"distance"             bun:"distance,type:FLOAT,nullzero"`
	DistanceSource         string         `json:"distanceSource"       bun:"distance_source,type:VARCHAR(50),nullzero"`
	DistanceProvider       string         `json:"distanceProvider"     bun:"distance_provider,type:VARCHAR(50),nullzero"`
	DistanceCalculatedAt   *int64         `json:"distanceCalculatedAt" bun:"distance_calculated_at,type:BIGINT,nullzero"`
	DistanceRouteSignature string         `json:"distanceRouteSignature" bun:"distance_route_signature,type:TEXT,nullzero"`
	DistanceDataVersion    string         `json:"distanceDataVersion"  bun:"distance_data_version,type:VARCHAR(50),nullzero"`
	DistanceRoutingType    string         `json:"distanceRoutingType"  bun:"distance_routing_type,type:VARCHAR(50),nullzero"`
	DistanceUnits          string         `json:"distanceUnits"        bun:"distance_units,type:VARCHAR(50),nullzero"`
	DistanceMetadata       map[string]any `json:"distanceMetadata,omitempty" bun:"distance_metadata,type:JSONB,nullzero"`
	Version                int64          `json:"version"              bun:"version,type:BIGINT"`
	CreatedAt              int64          `json:"createdAt"            bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt              int64          `json:"updatedAt"            bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	Stops                  []*Stop        `json:"stops,omitempty"      bun:"rel:has-many,join:id=shipment_move_id"`
	Assignment             *Assignment    `json:"assignment,omitempty" bun:"rel:has-one,join:id=shipment_move_id"`
	Shipment               *Shipment      `json:"shipment,omitempty"   bun:"rel:belongs-to,join:shipment_id=id"`
}

func (m *ShipmentMove) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if m.ID.IsNil() {
			m.ID = pulid.MustNew("sm_")
		}
		m.CreatedAt = now
	case *bun.UpdateQuery:
		m.UpdatedAt = now
	}

	return nil
}

func (m *ShipmentMove) IsCompleted() bool {
	return m.Status == MoveStatusCompleted
}

func (m *ShipmentMove) IsInTransit() bool {
	return m.Status == MoveStatusInTransit
}

func (m *ShipmentMove) IsAssigned() bool {
	return m.Status == MoveStatusAssigned
}

func (m *ShipmentMove) HasAssignment() bool {
	return m != nil && m.Assignment != nil
}

func (m *ShipmentMove) IsNew() bool {
	return m.Status == MoveStatusNew
}

func (m *ShipmentMove) IsCanceled() bool {
	return m.Status == MoveStatusCanceled
}

func (sm *ShipmentMove) Validate(multiErr *errortypes.MultiError) {
	// Tenancy and the owning shipment are stamped by the repository when the
	// move is written, so they are not the caller's to supply and are not
	// checked here — a create would otherwise fail on fields it never sends.
	multiErr.AddOzzoError(validation.ValidateStruct(sm,
		validation.Field(&sm.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[MoveStatus]("Status is invalid"),
		),
		validation.Field(&sm.Sequence,
			validation.Min(int64(0)).Error("Sequence cannot be negative"),
		),
		// Distance is what the move is paid and billed on, so a negative one
		// would credit mileage rather than charge it.
		validation.Field(&sm.Distance,
			validation.Min(0.0).Error("Distance cannot be negative"),
		),
		validation.Field(&sm.DistanceSource,
			validation.Length(0, maxDistanceFieldLength).
				Error("Distance source cannot be longer than 50 characters"),
		),
		validation.Field(&sm.DistanceProvider,
			validation.Length(0, maxDistanceFieldLength).
				Error("Distance provider cannot be longer than 50 characters"),
		),
		validation.Field(&sm.DistanceDataVersion,
			validation.Length(0, maxDistanceFieldLength).
				Error("Distance data version cannot be longer than 50 characters"),
		),
		validation.Field(&sm.DistanceRoutingType,
			validation.Length(0, maxDistanceFieldLength).
				Error("Distance routing type cannot be longer than 50 characters"),
		),
		validation.Field(&sm.DistanceUnits,
			validation.Length(0, maxDistanceFieldLength).
				Error("Distance units cannot be longer than 50 characters"),
		),
	))

	for i, stop := range sm.Stops {
		if stop == nil {
			continue
		}
		stop.Validate(multiErr.WithIndex("stops", i))
	}
}

const maxDistanceFieldLength = 50
