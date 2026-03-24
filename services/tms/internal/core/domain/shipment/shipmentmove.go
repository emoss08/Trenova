package shipment

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

type ShipmentMove struct {
	bun.BaseModel `json:"-" bun:"table:shipment_moves,alias:sm"`

	ID             pulid.ID    `json:"id"                   bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID    `json:"businessUnitId"       bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID    `json:"organizationId"       bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	ShipmentID     pulid.ID    `json:"shipmentId"           bun:"shipment_id,type:VARCHAR(100),notnull"`
	Status         MoveStatus  `json:"status"               bun:"status,type:move_status_enum,notnull,default:'New'"`
	Loaded         bool        `json:"loaded"               bun:"loaded,type:BOOLEAN,notnull,default:true"`
	Sequence       int64       `json:"sequence"             bun:"sequence,type:INTEGER,notnull,default:0"`
	Distance       *float64    `json:"distance"             bun:"distance,type:FLOAT,nullzero"`
	Version        int64       `json:"version"              bun:"version,type:BIGINT"`
	CreatedAt      int64       `json:"createdAt"            bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64       `json:"updatedAt"            bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	Stops          []*Stop     `json:"stops,omitempty"      bun:"rel:has-many,join:id=shipment_move_id"`
	Assignment     *Assignment `json:"assignment,omitempty" bun:"rel:has-one,join:id=shipment_move_id"`
	Shipment       *Shipment   `json:"shipment,omitempty"   bun:"rel:belongs-to,join:shipment_id=id"`
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
