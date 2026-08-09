package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetCarrierInvoiceMatchByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ListCarrierInvoiceMatchesRequest struct {
	Filter    *pagination.QueryOptions             `json:"filter"`
	CarrierID pulid.ID                             `json:"carrierId"`
	Status    carriersettlement.InvoiceMatchStatus `json:"status"`
}

type CarrierInvoiceMatchRepository interface {
	List(
		ctx context.Context,
		req *ListCarrierInvoiceMatchesRequest,
	) (*pagination.ListResult[*carriersettlement.InvoiceMatch], error)
	GetByID(
		ctx context.Context,
		req GetCarrierInvoiceMatchByIDRequest,
	) (*carriersettlement.InvoiceMatch, error)
	GetOpenByEDIInvoiceID(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		invoiceID pulid.ID,
	) (*carriersettlement.InvoiceMatch, error)
	Create(
		ctx context.Context,
		entity *carriersettlement.InvoiceMatch,
	) (*carriersettlement.InvoiceMatch, error)
	Update(
		ctx context.Context,
		entity *carriersettlement.InvoiceMatch,
	) (*carriersettlement.InvoiceMatch, error)
}
