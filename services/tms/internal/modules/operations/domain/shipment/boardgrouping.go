package shipment

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
