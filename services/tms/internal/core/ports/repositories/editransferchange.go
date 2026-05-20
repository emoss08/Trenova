package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
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

type EDITransferChangeRepository interface {
	ListTransferChanges(
		ctx context.Context,
		req *ListEDITransferChangesRequest,
	) (*pagination.ListResult[*edi.TransferChange], error)
	GetTransferChangeByID(
		ctx context.Context,
		req GetEDITransferChangeByIDRequest,
	) (*edi.TransferChange, error)
	CreateTransferChange(
		ctx context.Context,
		entity *edi.TransferChange,
	) (*edi.TransferChange, error)
	UpdateTransferChange(
		ctx context.Context,
		entity *edi.TransferChange,
	) (*edi.TransferChange, error)
}
