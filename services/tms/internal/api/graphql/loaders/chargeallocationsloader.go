package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type chargeAllocationsLister interface {
	ListByShipmentIDs(
		ctx context.Context,
		req *shipmentports.ListChargeAllocationsByShipmentIDsRequest,
	) (map[pulid.ID][]*shipment.ChargeAllocation, error)
	ListByOrderChargeIDs(
		ctx context.Context,
		req *shipmentports.ListChargeAllocationsByOrderChargeIDsRequest,
	) (map[pulid.ID][]*shipment.ChargeAllocation, error)
}

type ChargeAllocationsByShipmentIDLoaderFactoryParams struct {
	fx.In

	ChargeAllocationRepo shipmentports.ChargeAllocationRepository
}

type ChargeAllocationsByShipmentIDLoaderFactory struct {
	allocations chargeAllocationsLister
}

func NewChargeAllocationsByShipmentIDLoaderFactory(
	p ChargeAllocationsByShipmentIDLoaderFactoryParams,
) *ChargeAllocationsByShipmentIDLoaderFactory {
	return &ChargeAllocationsByShipmentIDLoaderFactory{
		allocations: p.ChargeAllocationRepo,
	}
}

func newChargeAllocationsByShipmentIDLoader(f *ChargeAllocationsByShipmentIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*shipment.ChargeAllocation] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *ChargeAllocationsByShipmentIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[[]*shipment.ChargeAllocation] {
	return batchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*shipment.ChargeAllocation, error) {
			return f.allocations.ListByShipmentIDs(
				ctx,
				&shipmentports.ListChargeAllocationsByShipmentIDsRequest{
					TenantInfo:  tenantInfo,
					ShipmentIDs: ids,
				},
			)
		},
	)
}

type ChargeAllocationsByOrderChargeIDLoaderFactoryParams struct {
	fx.In

	ChargeAllocationRepo shipmentports.ChargeAllocationRepository
}

type ChargeAllocationsByOrderChargeIDLoaderFactory struct {
	allocations chargeAllocationsLister
}

func NewChargeAllocationsByOrderChargeIDLoaderFactory(
	p ChargeAllocationsByOrderChargeIDLoaderFactoryParams,
) *ChargeAllocationsByOrderChargeIDLoaderFactory {
	return &ChargeAllocationsByOrderChargeIDLoaderFactory{
		allocations: p.ChargeAllocationRepo,
	}
}

func newChargeAllocationsByOrderChargeIDLoader(f *ChargeAllocationsByOrderChargeIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*shipment.ChargeAllocation] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *ChargeAllocationsByOrderChargeIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[[]*shipment.ChargeAllocation] {
	return batchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*shipment.ChargeAllocation, error) {
			return f.allocations.ListByOrderChargeIDs(
				ctx,
				&shipmentports.ListChargeAllocationsByOrderChargeIDsRequest{
					TenantInfo:     tenantInfo,
					OrderChargeIDs: ids,
				},
			)
		},
	)
}
