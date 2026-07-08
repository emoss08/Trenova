package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListEDITransfersRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Cursor pagination.CursorInfo    `json:"-"`
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

type ListActionableInboundEDITransfersByPartnerRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	PartnerID  pulid.ID              `json:"partnerId"`
	Statuses   []edi.TransferStatus  `json:"statuses"`
	ExcludeIDs []pulid.ID            `json:"excludeIds"`
}

type EDILoadTenderTransferRepository interface {
	GetInboundStatusCounts(
		ctx context.Context,
		req GetEDITransferStatusCountsRequest,
	) (map[edi.TransferStatus]int, error)
	ListInbound(
		ctx context.Context,
		req *ListEDITransfersRequest,
	) (*pagination.ListResult[*edi.EDITransfer], error)
	ListInboundCursor(
		ctx context.Context,
		req *ListEDITransfersRequest,
	) (*pagination.CursorListResult[*edi.EDITransfer], error)
	ListOutboundCursor(
		ctx context.Context,
		req *ListEDITransfersRequest,
	) (*pagination.CursorListResult[*edi.EDITransfer], error)
	ListOutbound(
		ctx context.Context,
		req *ListEDITransfersRequest,
	) (*pagination.ListResult[*edi.EDITransfer], error)
	GetTransferByID(
		ctx context.Context,
		req GetEDITransferByIDRequest,
	) (*edi.EDITransfer, error)
	GetTransfersByIDs(
		ctx context.Context,
		req GetEDITransfersByIDsRequest,
	) ([]*edi.EDITransfer, error)
	SelectOptions(
		ctx context.Context,
		req *EDITransferSelectOptionsRequest,
	) (*pagination.ListResult[*edi.EDITransfer], error)
	GetTransferForUpdate(
		ctx context.Context,
		req GetEDITransferForUpdateRequest,
	) (*edi.EDITransfer, error)
	CreateTransfer(
		ctx context.Context,
		entity *edi.EDITransfer,
	) (*edi.EDITransfer, error)
	UpdateTransfer(
		ctx context.Context,
		entity *edi.EDITransfer,
	) (*edi.EDITransfer, error)
	SetApprovalWorkflowRunID(
		ctx context.Context,
		req SetEDITransferApprovalWorkflowRunIDRequest,
	) (*edi.EDITransfer, error)
	GetActionableInboundTransferByExternalReference(
		ctx context.Context,
		req GetActionableInboundEDITransferByExternalReferenceRequest,
	) (*edi.EDITransfer, error)
	ListActionableInboundTransfersByPartner(
		ctx context.Context,
		req ListActionableInboundEDITransfersByPartnerRequest,
	) ([]*edi.EDITransfer, error)
}
