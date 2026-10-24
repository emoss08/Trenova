package lastknownlocation

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentcontinuity"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentcontinuity/equipmentcontinuitydb"
)

const (
	alias        = "lkc"
	nameSQL      = "(SELECT l.name FROM locations l WHERE l.id = ? AND l.organization_id = ? AND l.business_unit_id = ?)"
	latitudeSQL  = "(SELECT l.latitude FROM locations l WHERE l.id = ? AND l.organization_id = ? AND l.business_unit_id = ?)"
	longitudeSQL = "(SELECT l.longitude FROM locations l WHERE l.id = ? AND l.organization_id = ? AND l.business_unit_id = ?)"
)

type Tenant struct {
	OrganizationID keel.Expr
	BusinessUnitID keel.Expr
}

type Fields struct {
	LocationID keel.Expr
	Name       keel.Expr
	Latitude   keel.Expr
	Longitude  keel.Expr
}

func Continuity() *equipmentcontinuitydb.EquipmentContinuitiesTable {
	return equipmentcontinuitydb.EquipmentContinuities.As(alias)
}

func Current(
	ec *equipmentcontinuitydb.EquipmentContinuitiesTable,
	equipmentID keel.Expr,
	kind equipmentcontinuity.EquipmentType,
) []keel.Pred {
	return []keel.Pred{
		ec.EquipmentID.EqCol(equipmentID),
		ec.EquipmentType.Eq(kind),
		ec.IsCurrent.IsTrue(),
	}
}

func Of(ec *equipmentcontinuitydb.EquipmentContinuitiesTable, tenant Tenant) Fields {
	current := ec.CurrentLocationID
	return Fields{
		LocationID: current,
		Name:       keel.Raw(nameSQL, current, tenant.OrganizationID, tenant.BusinessUnitID),
		Latitude:   keel.Raw(latitudeSQL, current, tenant.OrganizationID, tenant.BusinessUnitID),
		Longitude:  keel.Raw(longitudeSQL, current, tenant.OrganizationID, tenant.BusinessUnitID),
	}
}
