package shipmentprovider

import (
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type ShipmentCountCard struct {
	Count           int `json:"count"`
	TrendPercentage int `json:"trendPercentage"` // * The percentage change in the number of shipments from the previous month
}

type CountByShipmentStatus struct {
	Status shipment.Status `json:"status"`
	Count  int             `json:"count"`
}

// ShipmentSummary contains only the essential shipment information for analytics
type ShipmentSummary struct {
	ID                 pulid.ID        `json:"id"`
	ProNumber          string          `json:"proNumber"`
	BOL                string          `json:"bol"`
	Status             shipment.Status `json:"status"`
	CustomerID         pulid.ID        `json:"customerId"`
	CustomerName       string          `json:"customerName"`
	ExpectedDelivery   int64           `json:"expectedDelivery"`
	DeliveryLocation   string          `json:"deliveryLocation"`
	DeliveryLocationID pulid.ID        `json:"deliveryLocationId"`
	CreatedAt          int64           `json:"createdAt"`
}

type ShipmentsByExpectedDeliverDateCard struct {
	Count     int                `json:"count"`
	Date      int64              `json:"date"`
	Shipments []*ShipmentSummary `json:"shipments"`
}
