package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListEDITransfersRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
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

type EDILoadTenderTransferRepository interface {
	ListInbound(
		ctx context.Context,
		req *ListEDITransfersRequest,
	) (*pagination.ListResult[*edi.EDITransfer], error)
	ListOutbound(
		ctx context.Context,
		req *ListEDITransfersRequest,
	) (*pagination.ListResult[*edi.EDITransfer], error)
	GetTransferByID(
		ctx context.Context,
		req GetEDITransferByIDRequest,
	) (*edi.EDITransfer, error)
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
}
