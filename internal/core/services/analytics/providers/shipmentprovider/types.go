package shipmentprovider

import "github.com/emoss08/trenova/internal/core/domain/shipment"

type ShipmentCountCard struct {
	Count           int `json:"count"`
	TrendPercentage int `json:"trendPercentage"` // * The percentage change in the number of shipments from the previous month
}

type CountByShipmentStatus struct {
	Status shipment.Status `json:"status"`
	Count  int             `json:"count"`
}
