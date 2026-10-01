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

type GetLiveCarrierInvoiceMatchByNumberRequest struct {
	TenantInfo    pagination.TenantInfo
	CarrierID     pulid.ID
	InvoiceNumber string
}

type ListLiveCarrierInvoiceMatchesByAssignmentRequest struct {
	TenantInfo   pagination.TenantInfo
	AssignmentID pulid.ID
}

type ListResolvedMatchAssignmentsRequest struct {
	TenantInfo    pagination.TenantInfo
	AssignmentIDs []pulid.ID
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
	GetOpenByExtractionID(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		extractionID pulid.ID,
	) (*carriersettlement.InvoiceMatch, error)
	GetLiveByCarrierInvoiceNumber(
		ctx context.Context,
		req *GetLiveCarrierInvoiceMatchByNumberRequest,
	) (*carriersettlement.InvoiceMatch, error)
	ListLiveByAssignment(
		ctx context.Context,
		req ListLiveCarrierInvoiceMatchesByAssignmentRequest,
	) ([]*carriersettlement.InvoiceMatch, error)
	ListResolvedAssignmentIDs(
		ctx context.Context,
		req ListResolvedMatchAssignmentsRequest,
	) ([]pulid.ID, error)
	UpdateSettlementForAdjustmentEvents(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		costEventIDs []pulid.ID,
		settlementID pulid.ID,
	) error
	Create(
		ctx context.Context,
		entity *carriersettlement.InvoiceMatch,
	) (*carriersettlement.InvoiceMatch, error)
	Update(
		ctx context.Context,
		entity *carriersettlement.InvoiceMatch,
	) (*carriersettlement.InvoiceMatch, error)
}
