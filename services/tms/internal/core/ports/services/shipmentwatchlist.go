package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type DeliveryHourBucket struct {
	Hour      int `json:"hour"`
	Delivered int `json:"delivered"`
	Scheduled int `json:"scheduled"`
	Late      int `json:"late"`
}

type LateDelivery struct {
	ShipmentID   pulid.ID `json:"shipmentId"`
	ProNumber    string   `json:"proNumber"`
	DeltaMinutes int      `json:"deltaMinutes"`
	City         string   `json:"city"`
	CustomerName string   `json:"customerName"`
}

type ShipmentDeliveryWatch struct {
	OnTime    int                  `json:"onTime"`
	Total     int                  `json:"total"`
	LateCount int                  `json:"lateCount"`
	Buckets   []DeliveryHourBucket `json:"buckets"`
	WorstLate []LateDelivery       `json:"worstLate"`
}

type UncoveredWindowSummary struct {
	Window       shipment.PickupWindow `json:"window"`
	StartMinutes int                   `json:"startMinutes"`
	EndMinutes   *int                  `json:"endMinutes"`
	Count        int                   `json:"count"`
	Revenue      decimal.Decimal       `json:"revenue"`
}

type NextUncoveredPickup struct {
	ShipmentID      pulid.ID `json:"shipmentId"`
	PickupAt        int64    `json:"pickupAt"`
	OriginCity      string   `json:"originCity"`
	DestinationCity string   `json:"destinationCity"`
}

type ShipmentUncoveredWatch struct {
	Count   int                      `json:"count"`
	Revenue decimal.Decimal          `json:"revenue"`
	Windows []UncoveredWindowSummary `json:"windows"`
	Next    *NextUncoveredPickup     `json:"next"`
}

type DetentionAccrual struct {
	ShipmentID    pulid.ID        `json:"shipmentId"`
	StopID        pulid.ID        `json:"stopId"`
	OccurrenceID  *pulid.ID       `json:"occurrenceId"`
	FacilityName  string          `json:"facilityName"`
	CoverageName  string          `json:"coverageName"`
	BillableSince int64           `json:"billableSince"`
	RatePerHour   decimal.Decimal `json:"ratePerHour"`
	Amount        decimal.Decimal `json:"amount"`
}

type ShipmentDetentionWatch struct {
	StopCount   int                `json:"stopCount"`
	Amount      decimal.Decimal    `json:"amount"`
	RatePerHour decimal.Decimal    `json:"ratePerHour"`
	SnapshotAt  int64              `json:"snapshotAt"`
	Top         []DetentionAccrual `json:"top"`
}

type ReadyToBillCustomer struct {
	CustomerID pulid.ID        `json:"customerId"`
	Name       string          `json:"name"`
	Count      int             `json:"count"`
	Total      decimal.Decimal `json:"total"`
}

type ShipmentBillingWatch struct {
	Count         int                   `json:"count"`
	Total         decimal.Decimal       `json:"total"`
	Customers     []ReadyToBillCustomer `json:"customers"`
	MoreCustomers int                   `json:"moreCustomers"`
}

type ShipmentWatchlist struct {
	Deliveries ShipmentDeliveryWatch  `json:"deliveries"`
	Uncovered  ShipmentUncoveredWatch `json:"uncovered"`
	Detention  ShipmentDetentionWatch `json:"detention"`
	Billing    ShipmentBillingWatch   `json:"billing"`
}

type ShipmentWatchlistReader interface {
	Watchlist(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		timezone string,
	) (*ShipmentWatchlist, error)
}
