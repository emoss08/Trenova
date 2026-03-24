package domainregistry

import (
	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/distanceoverride"
	"github.com/emoss08/trenova/internal/core/domain/equipmentcontinuity"
	"github.com/emoss08/trenova/internal/core/domain/fleetcode"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
)

func RegisterEntities() []any {
	return []any{
		&usstate.UsState{},
		&tenant.BusinessUnit{},
		&formulatemplate.FormulaTemplate{},
		&fleetcode.FleetCode{},
		&trailer.Trailer{},
		&equipmentcontinuity.EquipmentContinuity{},
		&accessorialcharge.AccessorialCharge{},
		&servicetype.ServiceType{},
		&distanceoverride.DistanceOverride{},
		&audit.Entry{},
		&shipment.Assignment{},
		&shipment.Stop{},
		&shipment.ShipmentMove{},
		&shipment.AdditionalCharge{},
		&shipment.ShipmentCommodity{},
		&shipment.ShipmentComment{},
		&shipment.ShipmentCommentMention{},
		&shipment.ShipmentHold{},
		&shipment.Shipment{},
		&customer.CustomerBillingProfileDocumentType{},
	}
}
