package podchaseports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// Facts is what asking for one shipment's proof of delivery needs: the
// shipment as the request names it, and who can be asked. Driver is the
// worker who drove the delivering move; Carrier the carrier that hauled it.
type Facts struct {
	ShipmentID   pulid.ID
	ProNumber    string
	BOL          string
	CustomerName string
	DeliveredAt  int64
	Driver       *Driver
	Carrier      *Carrier
}

type Driver struct {
	WorkerID pulid.ID
	Name     string
	// OnDash says the driver has a portal login, so a Dash message reaches
	// them.
	OnDash bool
}

type Carrier struct {
	CarrierID pulid.ID
	Name      string
	Email     string
}

type ListFactsRequest struct {
	TenantInfo  pagination.TenantInfo
	ShipmentIDs []pulid.ID
	// DeliveredBefore leaves out a shipment delivered since: it is not yet
	// missing anything.
	DeliveredBefore int64
}

// FactsReader reads, for the shipments still delivered without a proof of
// delivery on file, who can be asked for one, and the tenant's billing
// sender for an email request.
type FactsReader interface {
	ListPODChaseFacts(ctx context.Context, req *ListFactsRequest) ([]*Facts, error)
	BillingSenderProfile(ctx context.Context, tenantInfo pagination.TenantInfo) (pulid.ID, error)
}
