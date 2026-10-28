package editenderports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/domain/editenderchange"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListEDITenderChangesRequest struct {
	Filter           *pagination.QueryOptions           `json:"filter"`
	RecipientID      pulid.ID                           `json:"recipientId"`
	SourceShipmentID pulid.ID                           `json:"sourceShipmentId"`
	Status           editenderchange.TenderChangeStatus `json:"status"`
}

type GetEDITenderChangeByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetEDITenderChangeByOutboundMessageIDRequest struct {
	OutboundMessageID pulid.ID              `json:"outboundMessageId"`
	TenantInfo        pagination.TenantInfo `json:"tenantInfo"`
}

type CreateEDITenderChangeIdempotentResult struct {
	TenderChange *editenderchange.TenderChange `json:"tenderChange"`
	Created      bool                          `json:"created"`
}

type SupersedeActionableEDITenderChangesRequest struct {
	RecipientID     pulid.ID                             `json:"recipientId"`
	ExcludeChangeID pulid.ID                             `json:"excludeChangeId"`
	Statuses        []editenderchange.TenderChangeStatus `json:"statuses"`
}

type EDITenderChangeRepository interface {
	ListTenderChanges(
		ctx context.Context,
		req *ListEDITenderChangesRequest,
	) (*pagination.ListResult[*editenderchange.TenderChange], error)
	GetTenderChangeByID(
		ctx context.Context,
		req GetEDITenderChangeByIDRequest,
	) (*editenderchange.TenderChange, error)
	GetTenderChangeByOutboundMessageID(
		ctx context.Context,
		req GetEDITenderChangeByOutboundMessageIDRequest,
	) (*editenderchange.TenderChange, error)
	CreateTenderChangeIdempotent(
		ctx context.Context,
		entity *editenderchange.TenderChange,
	) (*CreateEDITenderChangeIdempotentResult, error)
	SupersedeActionableTenderChanges(
		ctx context.Context,
		req SupersedeActionableEDITenderChangesRequest,
	) error
	UpdateTenderChange(
		ctx context.Context,
		entity *editenderchange.TenderChange,
	) (*editenderchange.TenderChange, error)
}
