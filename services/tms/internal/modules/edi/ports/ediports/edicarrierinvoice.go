package ediports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edicarrierinvoice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListEDICarrierInvoicesRequest struct {
	Filter               *pagination.QueryOptions                             `json:"filter"`
	PartnerID            pulid.ID                                             `json:"partnerId"`
	ShipmentID           pulid.ID                                             `json:"shipmentId"`
	ReconciliationStatus edicarrierinvoice.CarrierInvoiceReconciliationStatus `json:"reconciliationStatus"`
}

type GetEDICarrierInvoiceByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetEDICarrierInvoiceByNumberRequest struct {
	TenantInfo    pagination.TenantInfo
	PartnerID     pulid.ID
	InvoiceNumber string
}

type EDICarrierInvoiceRepository interface {
	ListCarrierInvoices(
		ctx context.Context,
		req *ListEDICarrierInvoicesRequest,
	) (*pagination.ListResult[*edicarrierinvoice.CarrierInvoice], error)
	GetCarrierInvoiceByID(
		ctx context.Context,
		req GetEDICarrierInvoiceByIDRequest,
	) (*edicarrierinvoice.CarrierInvoice, error)
	GetCarrierInvoiceByNumber(
		ctx context.Context,
		req *GetEDICarrierInvoiceByNumberRequest,
	) (*edicarrierinvoice.CarrierInvoice, error)
	CreateCarrierInvoice(
		ctx context.Context,
		entity *edicarrierinvoice.CarrierInvoice,
	) (*edicarrierinvoice.CarrierInvoice, error)
	UpdateCarrierInvoice(
		ctx context.Context,
		entity *edicarrierinvoice.CarrierInvoice,
	) (*edicarrierinvoice.CarrierInvoice, error)
}
