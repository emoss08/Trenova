package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListEDITenderChangesRequest struct {
	Filter           *pagination.QueryOptions `json:"filter"`
	RecipientID      pulid.ID                 `json:"recipientId"`
	SourceShipmentID pulid.ID                 `json:"sourceShipmentId"`
	Status           edi.TenderChangeStatus   `json:"status"`
}

type GetEDITenderChangeByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type CreateEDITenderChangeIdempotentResult struct {
	TenderChange *edi.TenderChange `json:"tenderChange"`
	Created      bool              `json:"created"`
}

type SupersedeActionableEDITenderChangesRequest struct {
	RecipientID     pulid.ID                 `json:"recipientId"`
	ExcludeChangeID pulid.ID                 `json:"excludeChangeId"`
	Statuses        []edi.TenderChangeStatus `json:"statuses"`
}

type EDITenderChangeRepository interface {
	ListTenderChanges(
		ctx context.Context,
		req *ListEDITenderChangesRequest,
	) (*pagination.ListResult[*edi.TenderChange], error)
	GetTenderChangeByID(
		ctx context.Context,
		req GetEDITenderChangeByIDRequest,
	) (*edi.TenderChange, error)
	CreateTenderChangeIdempotent(
		ctx context.Context,
		entity *edi.TenderChange,
	) (*CreateEDITenderChangeIdempotentResult, error)
	SupersedeActionableTenderChanges(
		ctx context.Context,
		req SupersedeActionableEDITenderChangesRequest,
	) error
	UpdateTenderChange(
		ctx context.Context,
		entity *edi.TenderChange,
	) (*edi.TenderChange, error)
}
