package editenderports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type LockEDITenderShipmentRequest struct {
	ShipmentID pulid.ID
	TenantInfo pagination.TenantInfo
}

type SetEDIShipmentTenderStatusRequest struct {
	ShipmentID pulid.ID
	TenantInfo pagination.TenantInfo
	Status     shipmentvocab.TenderStatus
}

type EDIShipmentTenderRepository interface {
	LockShipment(ctx context.Context, req LockEDITenderShipmentRequest) (*shipment.Shipment, error)
	SetShipmentTenderStatus(ctx context.Context, req SetEDIShipmentTenderStatusRequest) error
}
