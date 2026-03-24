package dothazmatreference

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*DotHazmatReference)(nil)

type DotHazmatReference struct {
	bun.BaseModel `json:"-" bun:"table:dot_hazmat_references,alias:dhr"`

	ID                  pulid.ID `json:"id"                  bun:"id,pk,type:VARCHAR(100)"`
	UnNumber            string   `json:"unNumber"            bun:"un_number,notnull"`
	ProperShippingName  string   `json:"properShippingName"  bun:"proper_shipping_name,notnull"`
	HazardClass         string   `json:"hazardClass"         bun:"hazard_class,notnull"`
	SubsidiaryHazard    string   `json:"subsidiaryHazard"    bun:"subsidiary_hazard,default:''"`
	PackingGroup        string   `json:"packingGroup"        bun:"packing_group,default:''"`
	SpecialProvisions   string   `json:"specialProvisions"   bun:"special_provisions,default:''"`
	PackagingExceptions string   `json:"packagingExceptions" bun:"packaging_exceptions,default:''"`
	PackagingNonBulk    string   `json:"packagingNonBulk"    bun:"packaging_non_bulk,default:''"`
	PackagingBulk       string   `json:"packagingBulk"       bun:"packaging_bulk,default:''"`
	QuantityPassenger   string   `json:"quantityPassenger"   bun:"quantity_passenger,default:''"`
	QuantityCargo       string   `json:"quantityCargo"       bun:"quantity_cargo,default:''"`
	VesselStowage       string   `json:"vesselStowage"       bun:"vessel_stowage,default:''"`
	ErgGuide            string   `json:"ergGuide"            bun:"erg_guide,default:''"`
	Symbols             string   `json:"symbols"             bun:"symbols,default:''"`
	CreatedAt           int64    `json:"createdAt"           bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt           int64    `json:"updatedAt"           bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (d *DotHazmatReference) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := timeutils.NowUnix()

	switch q.(type) {
	case *bun.InsertQuery:
		if d.ID.IsNil() {
			d.ID = pulid.MustNew("dhr_")
		}

		d.CreatedAt = now
	case *bun.UpdateQuery:
		d.UpdatedAt = now
	}

	return nil
}
