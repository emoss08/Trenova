package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type chargeAllocationsLister interface {
	ListByShipmentIDs(
		ctx context.Context,
		req *repositories.ListChargeAllocationsByShipmentIDsRequest,
	) (map[pulid.ID][]*shipment.ChargeAllocation, error)
	ListByOrderChargeIDs(
		ctx context.Context,
		req *repositories.ListChargeAllocationsByOrderChargeIDsRequest,
	) (map[pulid.ID][]*shipment.ChargeAllocation, error)
}

type ChargeAllocationsByShipmentIDLoaderFactoryParams struct {
	fx.In

	ChargeAllocationRepo repositories.ChargeAllocationRepository
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

func (f *ChargeAllocationsByShipmentIDLoaderFactory) NewForTenant(
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
				&repositories.ListChargeAllocationsByShipmentIDsRequest{
					TenantInfo:  tenantInfo,
					ShipmentIDs: ids,
				},
			)
		},
	)
}

type ChargeAllocationsByOrderChargeIDLoaderFactoryParams struct {
	fx.In

	ChargeAllocationRepo repositories.ChargeAllocationRepository
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

func (f *ChargeAllocationsByOrderChargeIDLoaderFactory) NewForTenant(
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
				&repositories.ListChargeAllocationsByOrderChargeIDsRequest{
					TenantInfo:     tenantInfo,
					OrderChargeIDs: ids,
				},
			)
		},
	)
}
