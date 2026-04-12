package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type AssignBankReceiptWorkItemRequest struct {
	WorkItemID       pulid.ID              `json:"workItemId"`
	AssignedToUserID pulid.ID              `json:"assignedToUserId"`
	TenantInfo       pagination.TenantInfo `json:"tenantInfo"`
}

type ResolveBankReceiptWorkItemRequest struct {
	WorkItemID     pulid.ID                           `json:"workItemId"`
	ResolutionType bankreceiptworkitem.ResolutionType `json:"resolutionType"`
	ResolutionNote string                             `json:"resolutionNote"`
	TenantInfo     pagination.TenantInfo              `json:"tenantInfo"`
}

type DismissBankReceiptWorkItemRequest struct {
	WorkItemID     pulid.ID              `json:"workItemId"`
	ResolutionNote string                `json:"resolutionNote"`
	TenantInfo     pagination.TenantInfo `json:"tenantInfo"`
}

type GetBankReceiptWorkItemRequest struct {
	WorkItemID pulid.ID              `json:"workItemId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type BankReceiptWorkItemService interface {
	ListActive(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*bankreceiptworkitem.WorkItem, error)
	Get(
		ctx context.Context,
		req *GetBankReceiptWorkItemRequest,
	) (*bankreceiptworkitem.WorkItem, error)
	Assign(
		ctx context.Context,
		req *AssignBankReceiptWorkItemRequest,
		actor *RequestActor,
	) (*bankreceiptworkitem.WorkItem, error)
	StartReview(
		ctx context.Context,
		req *GetBankReceiptWorkItemRequest,
		actor *RequestActor,
	) (*bankreceiptworkitem.WorkItem, error)
	Resolve(
		ctx context.Context,
		req *ResolveBankReceiptWorkItemRequest,
		actor *RequestActor,
	) (*bankreceiptworkitem.WorkItem, error)
	Dismiss(
		ctx context.Context,
		req *DismissBankReceiptWorkItemRequest,
		actor *RequestActor,
	) (*bankreceiptworkitem.WorkItem, error)
}
