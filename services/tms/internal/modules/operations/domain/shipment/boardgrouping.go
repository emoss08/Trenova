package shipment

import "strings"

type BoardGrouping string

const (
	BoardGroupingShipDate     = BoardGrouping("ShipDate")
	BoardGroupingDeliveryDate = BoardGrouping("DeliveryDate")
	BoardGroupingCustomer     = BoardGrouping("Customer")
	BoardGroupingOwner        = BoardGrouping("Owner")
)

const (
	ShipperStopRelationship   = "shipperStop"
	ConsigneeStopRelationship = "consigneeStop"
	ShipperStopAlias          = "shipper_stop"
	ConsigneeStopAlias        = "consignee_stop"
)

func (g BoardGrouping) IsValid() bool {
	switch g {
	case BoardGroupingShipDate,
		BoardGroupingDeliveryDate,
		BoardGroupingCustomer,
		BoardGroupingOwner:
		return true
	default:
		return false
	}
}

func (g BoardGrouping) IsDate() bool {
	return g == BoardGroupingShipDate || g == BoardGroupingDeliveryDate
}

func ShipperStopIDSQL(shipmentAlias string) string {
	return endStopIDSQL(shipmentAlias, []StopType{StopTypePickup, StopTypeSplitPickup}, "ASC")
}

func ConsigneeStopIDSQL(shipmentAlias string) string {
	return endStopIDSQL(
		shipmentAlias,
		[]StopType{StopTypeDelivery, StopTypeSplitDelivery},
		"DESC",
	)
}

func endStopIDSQL(shipmentAlias string, types []StopType, direction string) string {
	var b strings.Builder
	b.WriteString("(SELECT end_stp.id FROM shipment_moves AS end_sm JOIN stops AS end_stp")
	b.WriteString(" ON end_stp.shipment_move_id = end_sm.id")
	b.WriteString(" AND end_stp.organization_id = end_sm.organization_id")
	b.WriteString(" AND end_stp.business_unit_id = end_sm.business_unit_id")
	b.WriteString(" WHERE end_sm.shipment_id = ")
	b.WriteString(shipmentAlias)
	b.WriteString(".id AND end_sm.organization_id = ")
	b.WriteString(shipmentAlias)
	b.WriteString(".organization_id AND end_sm.business_unit_id = ")
	b.WriteString(shipmentAlias)
	b.WriteString(".business_unit_id AND end_stp.type IN (")
	for i, stopType := range types {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString("'")
		b.WriteString(string(stopType))
		b.WriteString("'")
	}
	b.WriteString(") ORDER BY end_sm.sequence ")
	b.WriteString(direction)
	b.WriteString(", end_stp.sequence ")
	b.WriteString(direction)
	b.WriteString(", end_stp.id COLLATE \"C\" ")
	b.WriteString(direction)
	b.WriteString(" LIMIT 1)")
	return b.String()
}
