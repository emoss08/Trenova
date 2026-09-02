package services

import (
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
