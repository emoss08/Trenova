package editenderports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edisync"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListEDITransferChangesRequest struct {
	Filter         *pagination.QueryOptions `json:"filter"`
	ShipmentLinkID pulid.ID                 `json:"shipmentLinkId"`
}

type GetEDITransferChangeByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type CreateEDITransferChangeIdempotentResult struct {
	TransferChange *edisync.TransferChange `json:"transferChange"`
	Created        bool                    `json:"created"`
}

type EDITransferChangeRepository interface {
	ListTransferChanges(
		ctx context.Context,
		req *ListEDITransferChangesRequest,
	) (*pagination.ListResult[*edisync.TransferChange], error)
	GetTransferChangeByID(
		ctx context.Context,
		req GetEDITransferChangeByIDRequest,
	) (*edisync.TransferChange, error)
	CreateTransferChange(
		ctx context.Context,
		entity *edisync.TransferChange,
	) (*edisync.TransferChange, error)
	CreateTransferChangeIdempotent(
		ctx context.Context,
		entity *edisync.TransferChange,
	) (*CreateEDITransferChangeIdempotentResult, error)
	UpdateTransferChange(
		ctx context.Context,
		entity *edisync.TransferChange,
	) (*edisync.TransferChange, error)
}
