package lastknownlocation

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentcontinuity"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentcontinuity/equipmentcontinuitydb"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmentcontinuityports"
)

const (
	alias       = "lkc"
	filterAlias = "lkf"
)

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

func Of(
	ec *equipmentcontinuitydb.EquipmentContinuitiesTable,
	locations equipmentcontinuityports.LocationColumns,
) Fields {
	current := ec.CurrentLocationID
	lookup := func(column keel.Expr) keel.Expr {
		return keel.Scalar(
			keel.Handle[*keel.Row]{Ref: locations.ID.Ref}.
				Pick(column).
				Where(locations.ID.EqCol(current)),
		)
	}
	return Fields{
		LocationID: current,
		Name:       lookup(locations.Name),
		Latitude:   lookup(locations.Latitude),
		Longitude:  lookup(locations.Longitude),
	}
}

func LocationID(equipmentID keel.Expr, kind equipmentcontinuity.EquipmentType) keel.Expr {
	ec := equipmentcontinuitydb.EquipmentContinuities.As(filterAlias)
	return keel.Scalar(
		ec.Pick(ec.CurrentLocationID).Where(Current(ec, equipmentID, kind)...).Limit(1),
	)
}
