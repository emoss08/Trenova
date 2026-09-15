package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type ListChargeAllocationsByShipmentIDsRequest struct {
	TenantInfo  pagination.TenantInfo `json:"-"`
	ShipmentIDs []pulid.ID            `json:"-"`
}

type ListChargeAllocationsByOrderChargeIDsRequest struct {
	TenantInfo     pagination.TenantInfo `json:"-"`
	OrderChargeIDs []pulid.ID            `json:"-"`
}

// SyncOrderChargeAllocationsRequest replaces the allocations of one order charge.
// A nil Allocations leaves the rows alone; an empty one removes them all.
type SyncOrderChargeAllocationsRequest struct {
	TenantInfo    pagination.TenantInfo        `json:"-"`
	OrderID       pulid.ID                     `json:"-"`
	OrderChargeID pulid.ID                     `json:"-"`
	Allocations   []*shipment.ChargeAllocation `json:"-"`
}

type MarkChargeAllocationsInvoicedRequest struct {
	TenantInfo    pagination.TenantInfo `json:"-"`
	AllocationIDs []pulid.ID            `json:"-"`
	InvoiceID     pulid.ID              `json:"-"`
	InvoicedAt    int64                 `json:"-"`
}

type ClearChargeAllocationsInvoiceRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	InvoiceID  pulid.ID              `json:"-"`
}

type ChargeAllocationRepository interface {
	// SyncForShipment reconciles the freight and accessorial allocations carried on
	// the shipment inside the caller's transaction. Nil leaves them untouched,
	// empty deletes them all. Rows whose charge no longer exists are dropped, rows
	// that named the shipment's previous customer as payer follow the new one, and
	// every open billing queue item for the shipment has its allocated total
	// refreshed.
	SyncForShipment(
		ctx context.Context,
		tx bun.IDB,
		entity *shipment.Shipment,
	) error
	SyncForOrderCharge(
		ctx context.Context,
		tx bun.IDB,
		req *SyncOrderChargeAllocationsRequest,
	) error
	ListByShipmentIDs(
		ctx context.Context,
		req *ListChargeAllocationsByShipmentIDsRequest,
	) (map[pulid.ID][]*shipment.ChargeAllocation, error)
	ListByOrderChargeIDs(
		ctx context.Context,
		req *ListChargeAllocationsByOrderChargeIDsRequest,
	) (map[pulid.ID][]*shipment.ChargeAllocation, error)
	MarkInvoiced(
		ctx context.Context,
		req *MarkChargeAllocationsInvoicedRequest,
	) (int64, error)
	ClearInvoice(
		ctx context.Context,
		req *ClearChargeAllocationsInvoiceRequest,
	) (int64, error)
	// LockedIDs reports which of the given allocations are carried on a posted
	// invoice and therefore cannot change.
	LockedIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		allocationIDs []pulid.ID,
	) (map[pulid.ID]struct{}, error)
	// RefreshQueueSnapshots recomputes the allocated total on every open billing
	// queue item of the shipment from its current charges and allocations.
	RefreshQueueSnapshots(
		ctx context.Context,
		tx bun.IDB,
		tenantInfo pagination.TenantInfo,
		shipmentID pulid.ID,
	) error
}
