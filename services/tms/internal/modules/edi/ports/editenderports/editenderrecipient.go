package editenderports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/domain/editenderchange"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetEDITenderRecipientByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ListEDITenderRecipientsForSourceShipmentRequest struct {
	TenantInfo       pagination.TenantInfo `json:"tenantInfo"`
	SourceShipmentID pulid.ID              `json:"sourceShipmentId"`
}

type UpsertEDITenderRecipientRequest struct {
	Recipient *editenderchange.TenderRecipient `json:"recipient"`
}

type GetActiveExternalEDITenderRecipientByReferenceRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	PartnerID  pulid.ID              `json:"partnerId"`
	Reference  string                `json:"reference"`
}

type EDITenderRecipientRepository interface {
	GetTenderRecipientByID(
		ctx context.Context,
		req GetEDITenderRecipientByIDRequest,
	) (*editenderchange.TenderRecipient, error)
	ListActiveTenderRecipientsForSourceShipment(
		ctx context.Context,
		req ListEDITenderRecipientsForSourceShipmentRequest,
	) ([]*editenderchange.TenderRecipient, error)
	UpsertTenderRecipient(
		ctx context.Context,
		req UpsertEDITenderRecipientRequest,
	) (*editenderchange.TenderRecipient, error)
	UpdateTenderRecipient(
		ctx context.Context,
		entity *editenderchange.TenderRecipient,
	) (*editenderchange.TenderRecipient, error)
	GetActiveExternalRecipientByShipmentReference(
		ctx context.Context,
		req GetActiveExternalEDITenderRecipientByReferenceRequest,
	) (*editenderchange.TenderRecipient, error)
}
