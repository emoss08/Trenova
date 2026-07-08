package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListEDICarrierInvoicesRequest struct {
	Filter               *pagination.QueryOptions               `json:"filter"`
	PartnerID            pulid.ID                               `json:"partnerId"`
	ShipmentID           pulid.ID                               `json:"shipmentId"`
	ReconciliationStatus edi.CarrierInvoiceReconciliationStatus `json:"reconciliationStatus"`
}

type GetEDICarrierInvoiceByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type EDICarrierInvoiceRepository interface {
	ListCarrierInvoices(
		ctx context.Context,
		req *ListEDICarrierInvoicesRequest,
	) (*pagination.ListResult[*edi.CarrierInvoice], error)
	GetCarrierInvoiceByID(
		ctx context.Context,
		req GetEDICarrierInvoiceByIDRequest,
	) (*edi.CarrierInvoice, error)
	CreateCarrierInvoice(
		ctx context.Context,
		entity *edi.CarrierInvoice,
	) (*edi.CarrierInvoice, error)
	UpdateCarrierInvoice(
		ctx context.Context,
		entity *edi.CarrierInvoice,
	) (*edi.CarrierInvoice, error)
}
