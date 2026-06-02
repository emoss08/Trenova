package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
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
	Recipient *edi.TenderRecipient `json:"recipient"`
}

type EDITenderRecipientRepository interface {
	GetTenderRecipientByID(
		ctx context.Context,
		req GetEDITenderRecipientByIDRequest,
	) (*edi.TenderRecipient, error)
	ListActiveTenderRecipientsForSourceShipment(
		ctx context.Context,
		req ListEDITenderRecipientsForSourceShipmentRequest,
	) ([]*edi.TenderRecipient, error)
	UpsertTenderRecipient(
		ctx context.Context,
		req UpsertEDITenderRecipientRequest,
	) (*edi.TenderRecipient, error)
	UpdateTenderRecipient(
		ctx context.Context,
		entity *edi.TenderRecipient,
	) (*edi.TenderRecipient, error)
}
