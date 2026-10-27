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
	ShipmentID       pulid.ID        `db:"shipment_id"`
	ProNumber        string          `db:"pro_number"`
	BOL              string          `db:"bol"`
	Status           shipment.Status `db:"status"`
	CustomerID       pulid.ID        `db:"customer_id"`
	CustomerName     string          `db:"customer_name"`
	CustomerCode     string          `db:"customer_code"`
	ServiceTypeCode  string          `db:"service_type_code"`
	OriginCity       string          `db:"origin_city"`
	OriginState      string          `db:"origin_state"`
	DestinationCity  string          `db:"destination_city"`
	DestinationState string          `db:"destination_state"`
	DriverID         pulid.ID        `db:"driver_id"`
	DriverName       string          `db:"driver_name"`
}

// ShipmentPartyLink ties a shipment to the customer or the worker a search
// found it through. WorkerID is set on a worker's current load, CustomerID on
// a customer's shipments.
type ShipmentPartyLink struct {
	ShipmentID pulid.ID `db:"shipment_id"`
	CustomerID pulid.ID `db:"customer_id"`
	WorkerID   pulid.ID `db:"worker_id"`
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
