package shipmentports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// ShipmentSearchCard is one shipment as a search result draws it: what a
// person recognises it by (PRO, BOL, customer), where it runs and who drives
// it. It is read in one query for every hit at once, so a result list never
// loads whole shipments to show one line apiece.
type ShipmentSearchCard struct {
	ShipmentID       pulid.ID        `bun:"shipment_id"`
	ProNumber        string          `bun:"pro_number"`
	BOL              string          `bun:"bol"`
	Status           shipment.Status `bun:"status"`
	CustomerID       pulid.ID        `bun:"customer_id"`
	CustomerName     string          `bun:"customer_name"`
	CustomerCode     string          `bun:"customer_code"`
	ServiceTypeCode  string          `bun:"service_type_code"`
	OriginCity       string          `bun:"origin_city"`
	OriginState      string          `bun:"origin_state"`
	DestinationCity  string          `bun:"destination_city"`
	DestinationState string          `bun:"destination_state"`
	DriverID         pulid.ID        `bun:"driver_id"`
	DriverName       string          `bun:"driver_name"`
}

// ShipmentPartyLink ties a shipment to the customer or the worker a search
// found it through. WorkerID is set on a worker's current load, CustomerID on
// a customer's shipments.
type ShipmentPartyLink struct {
	ShipmentID pulid.ID `bun:"shipment_id"`
	CustomerID pulid.ID `bun:"customer_id"`
	WorkerID   pulid.ID `bun:"worker_id"`
}

type ListShipmentSearchCardsRequest struct {
	TenantInfo  pagination.TenantInfo
	ShipmentIDs []pulid.ID
}

type ListShipmentPartyLinksRequest struct {
	TenantInfo    pagination.TenantInfo
	CustomerIDs   []pulid.ID
	CustomerLimit int
	WorkerIDs     []pulid.ID
}

type ShipmentSearchCardReader interface {
	ListSearchCards(
		ctx context.Context,
		req *ListShipmentSearchCardsRequest,
	) ([]*ShipmentSearchCard, error)
	ListPartyLinks(
		ctx context.Context,
		req *ListShipmentPartyLinksRequest,
	) ([]*ShipmentPartyLink, error)
}
