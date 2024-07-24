package models

import (
	"context"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"time"
)

type ShipmentControl struct {
	bun.BaseModel `bun:"table:shipment_controls,alias:sc" json:"-"`
	ID            uuid.UUID `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`

	EnforceRevCode           bool      `bun:"type:BOOLEAN,notnull,default:false" json:"enforceRevCode"`
	EnforceVoidedComm        bool      `bun:"type:BOOLEAN,notnull,default:false" json:"enforceVoidedComm"`
	AutoTotalShipment        bool      `bun:"type:BOOLEAN,notnull,default:false" json:"autoTotalShipment"`
	CompareOriginDestination bool      `bun:"type:BOOLEAN,notnull,default:false" json:"compareOriginDestination"`
	CheckForDuplicateBol     bool      `bun:"type:BOOLEAN,notnull,default:false" json:"checkForDuplicateBol"`
	CreatedAt                time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt                time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	BusinessUnitID uuid.UUID `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID `bun:"type:uuid,notnull" json:"organizationId"`

	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}

func (sc *ShipmentControl) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		sc.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		sc.UpdatedAt = time.Now()
	}

	return nil
}
