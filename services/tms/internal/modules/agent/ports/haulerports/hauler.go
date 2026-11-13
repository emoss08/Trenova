package haulerports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/podchaseports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// Hauler is who is moving a load right now and how the load is named to
// them: the driver on the move, or the carrier that has it. Both are nil
// when nobody is assigned.
type Hauler struct {
	ShipmentID pulid.ID
	ProNumber  string
	Driver     *podchaseports.Driver
	Carrier    *podchaseports.Carrier
}

// HaulerRequest names the load by the most specific thing known about it:
// the move, a stop on the move, or only the shipment, whose move under way
// is then the one read.
type HaulerRequest struct {
	TenantInfo pagination.TenantInfo
	ShipmentID pulid.ID
	MoveID     pulid.ID
	StopID     pulid.ID
}

// Reader reads who has a load, and the move a tender offered. Hauler is nil
// when the load cannot be found; TenderMove is nil when the tender is gone.
type Reader interface {
	Hauler(ctx context.Context, req *HaulerRequest) (*Hauler, error)
	TenderMove(ctx context.Context, tenantInfo pagination.TenantInfo, tenderID pulid.ID) (pulid.ID, error)
}
