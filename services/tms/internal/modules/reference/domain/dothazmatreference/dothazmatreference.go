package dothazmatreference

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*DotHazmatReference)(nil)

type DotHazmatReference struct {
	bun.BaseModel `json:"-" bun:"table:dot_hazmat_references,alias:dhr"`
	keel.Table    `keel:"dot_hazmat_references"`

	ID                  pulid.ID `json:"id"                  bun:"id,pk,type:VARCHAR(100)" keel:"id,pk,autoid:dhr_"`
	UnNumber            string   `json:"unNumber"            bun:"un_number,notnull" keel:"un_number"`
	ProperShippingName  string   `json:"properShippingName"  bun:"proper_shipping_name,notnull" keel:"proper_shipping_name"`
	HazardClass         string   `json:"hazardClass"         bun:"hazard_class,notnull" keel:"hazard_class"`
	SubsidiaryHazard    string   `json:"subsidiaryHazard"    bun:"subsidiary_hazard,default:''" keel:"subsidiary_hazard,nullzero"`
	PackingGroup        string   `json:"packingGroup"        bun:"packing_group,default:''" keel:"packing_group,nullzero"`
	SpecialProvisions   string   `json:"specialProvisions"   bun:"special_provisions,default:''" keel:"special_provisions,nullzero"`
	PackagingExceptions string   `json:"packagingExceptions" bun:"packaging_exceptions,default:''" keel:"packaging_exceptions,nullzero"`
	PackagingNonBulk    string   `json:"packagingNonBulk"    bun:"packaging_non_bulk,default:''" keel:"packaging_non_bulk,nullzero"`
	PackagingBulk       string   `json:"packagingBulk"       bun:"packaging_bulk,default:''" keel:"packaging_bulk,nullzero"`
	QuantityPassenger   string   `json:"quantityPassenger"   bun:"quantity_passenger,default:''" keel:"quantity_passenger,nullzero"`
	QuantityCargo       string   `json:"quantityCargo"       bun:"quantity_cargo,default:''" keel:"quantity_cargo,nullzero"`
	VesselStowage       string   `json:"vesselStowage"       bun:"vessel_stowage,default:''" keel:"vessel_stowage,nullzero"`
	ErgGuide            string   `json:"ergGuide"            bun:"erg_guide,default:''" keel:"erg_guide,nullzero"`
	Symbols             string   `json:"symbols"             bun:"symbols,default:''" keel:"symbols,nullzero"`
	CreatedAt           int64    `json:"createdAt"           bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt           int64    `json:"updatedAt"           bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
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

func (r *DotHazmatReference) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		// This table is the regulatory reference the hazmat surfaces read from,
		// so a row missing its identity would answer a lookup with nothing.
		validation.Field(&r.UnNumber, validation.Required.Error("UN number is required")),
		validation.Field(&r.ProperShippingName,
			validation.Required.Error("Proper shipping name is required"),
		),
		validation.Field(&r.HazardClass, validation.Required.Error("Hazard class is required")),
	))
}
