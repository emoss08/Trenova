package shipment

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var _ validationframework.TenantedEntity = (*ShipmentCommodity)(nil)

// Commodity line items on a shipment
type ShipmentCommodity struct {
	keel.Table `keel:"shipment_commodities,report=Operations"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:sc_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	ShipmentID     pulid.ID `json:"shipmentId"     keel:"shipment_id,pk"`
	CommodityID    pulid.ID `json:"commodityId"    keel:"commodity_id,pk"`
	Weight         int64    `json:"weight"         keel:"weight,format=weight"`
	Pieces         int64    `json:"pieces"         keel:"pieces,format=count"`
	LengthFeet     *float64 `json:"lengthFeet"     keel:"length_feet"`
	WidthFeet      *float64 `json:"widthFeet"      keel:"width_feet"`
	HeightFeet     *float64 `json:"heightFeet"     keel:"height_feet"`
	Version        int64    `json:"version"        keel:"version,version"`
	CreatedAt      int64    `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt"      keel:"updated_at,autoupdate"`

	Shipment     *Shipment            `json:"shipment,omitempty"     keel:"belongs:shipment_id"`
	Commodity    *commodity.Commodity `json:"commodity,omitempty"    keel:"belongs:commodity_id"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	Organization *tenant.Organization `json:"organization,omitempty"`
}

func (sc *ShipmentCommodity) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		sc,
		validation.Field(
			&sc.Pieces,
			validation.Required.Error("Pieces is required"),
			validation.Min(1).Error("Pieces must be at least 1"),
		),
		validation.Field(
			&sc.Weight,
			validation.Required.Error("Weight is required"),
			validation.Min(1).Error("Weight must be at least 1"),
		),
		validation.Field(
			&sc.CommodityID,
			validation.Required.Error("Commodity ID is required"),
		),
	))
}

func (sc *ShipmentCommodity) GetID() pulid.ID {
	return sc.ID
}

func (sc *ShipmentCommodity) GetTableName() string {
	return "shipment_commodities"
}

func (sc *ShipmentCommodity) GetOrganizationID() pulid.ID {
	return sc.OrganizationID
}

func (sc *ShipmentCommodity) GetBusinessUnitID() pulid.ID {
	return sc.BusinessUnitID
}

func (sc *ShipmentCommodity) MeetsLinearFeetPerUnitRequirement() bool {
	if sc.Commodity == nil || sc.Commodity.LinearFeetPerUnit == nil {
		return false
	}

	return sc.Pieces > 0 && *sc.Commodity.LinearFeetPerUnit > 0
}
