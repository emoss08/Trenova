package dothazmatreference

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type DotHazmatReference struct {
	keel.Table `keel:"dot_hazmat_references"`

	ID                  pulid.ID `json:"id" keel:"id,pk,autoid:dhr_"`
	UnNumber            string   `json:"unNumber" keel:"un_number"`
	ProperShippingName  string   `json:"properShippingName" keel:"proper_shipping_name"`
	HazardClass         string   `json:"hazardClass" keel:"hazard_class"`
	SubsidiaryHazard    string   `json:"subsidiaryHazard" keel:"subsidiary_hazard,nullzero"`
	PackingGroup        string   `json:"packingGroup" keel:"packing_group,nullzero"`
	SpecialProvisions   string   `json:"specialProvisions" keel:"special_provisions,nullzero"`
	PackagingExceptions string   `json:"packagingExceptions" keel:"packaging_exceptions,nullzero"`
	PackagingNonBulk    string   `json:"packagingNonBulk" keel:"packaging_non_bulk,nullzero"`
	PackagingBulk       string   `json:"packagingBulk" keel:"packaging_bulk,nullzero"`
	QuantityPassenger   string   `json:"quantityPassenger" keel:"quantity_passenger,nullzero"`
	QuantityCargo       string   `json:"quantityCargo" keel:"quantity_cargo,nullzero"`
	VesselStowage       string   `json:"vesselStowage" keel:"vessel_stowage,nullzero"`
	ErgGuide            string   `json:"ergGuide" keel:"erg_guide,nullzero"`
	Symbols             string   `json:"symbols" keel:"symbols,nullzero"`
	CreatedAt           int64    `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt           int64    `json:"updatedAt" keel:"updated_at,autoupdate"`
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
