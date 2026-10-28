package editenderports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListEDITransfersRequest struct {
	Filter        *pagination.QueryOptions `json:"filter"`
	Cursor        pagination.CursorInfo    `json:"-"`
	InboundFileID pulid.ID                 `json:"inboundFileId"`
}

type GetEDITransferByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Direction  string                `json:"direction"`
}

type GetEDITransferForUpdateRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Direction  string                `json:"direction"`
}

type SetEDITransferApprovalWorkflowRunIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	RunID      string                `json:"runId"`
}

type GetActionableInboundEDITransferByExternalReferenceRequest struct {
	TenantInfo        pagination.TenantInfo `json:"tenantInfo"`
	PartnerID         pulid.ID              `json:"partnerId"`
	ExternalReference string                `json:"externalReference"`
}

type GetEDITransferStatusCountsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Since      int64                 `json:"since"`
}

type EDITransferSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest `json:"-"`
}

type GetEDITransfersByIDsRequest struct {
	TenantInfo  pagination.TenantInfo `json:"-"`
	TransferIDs []pulid.ID            `json:"transferIds"`
}

type EDILoadTenderTransferRepository interface {
	GetInboundStatusCounts(
		ctx context.Context,
		req GetEDITransferStatusCountsRequest,
	) (map[editransfer.TransferStatus]int, error)
	ListInbound(
		ctx context.Context,
		req *ListEDITransfersRequest,
	) (*pagination.ListResult[*editransfer.EDITransfer], error)
	ListInboundCursor(
		ctx context.Context,
		req *ListEDITransfersRequest,
	) (*pagination.CursorListResult[*editransfer.EDITransfer], error)
	ListOutboundCursor(
		ctx context.Context,
		req *ListEDITransfersRequest,
	) (*pagination.CursorListResult[*editransfer.EDITransfer], error)
	ListOutbound(
		ctx context.Context,
		req *ListEDITransfersRequest,
	) (*pagination.ListResult[*editransfer.EDITransfer], error)
	GetTransferByID(
		ctx context.Context,
		req GetEDITransferByIDRequest,
	) (*editransfer.EDITransfer, error)
	GetTransfersByIDs(
		ctx context.Context,
		req GetEDITransfersByIDsRequest,
	) ([]*editransfer.EDITransfer, error)
	SelectOptions(
		ctx context.Context,
		req *EDITransferSelectOptionsRequest,
	) (*pagination.ListResult[*editransfer.EDITransfer], error)
	GetTransferForUpdate(
		ctx context.Context,
		req GetEDITransferForUpdateRequest,
	) (*editransfer.EDITransfer, error)
	CreateTransfer(
		ctx context.Context,
		entity *editransfer.EDITransfer,
	) (*editransfer.EDITransfer, error)
	UpdateTransfer(
		ctx context.Context,
		entity *editransfer.EDITransfer,
	) (*editransfer.EDITransfer, error)
	SetApprovalWorkflowRunID(
		ctx context.Context,
		req SetEDITransferApprovalWorkflowRunIDRequest,
	) (*editransfer.EDITransfer, error)
	GetActionableInboundTransferByExternalReference(
		ctx context.Context,
		req GetActionableInboundEDITransferByExternalReferenceRequest,
	) (*editransfer.EDITransfer, error)
}
