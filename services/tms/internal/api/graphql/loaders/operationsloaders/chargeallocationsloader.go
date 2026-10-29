package operationsloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/chargeallocationports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type chargeAllocationsLister interface {
	ListByShipmentIDs(
		ctx context.Context,
		req *chargeallocationports.ListChargeAllocationsByShipmentIDsRequest,
	) (map[pulid.ID][]*shipment.ChargeAllocation, error)
	ListByOrderChargeIDs(
		ctx context.Context,
		req *chargeallocationports.ListChargeAllocationsByOrderChargeIDsRequest,
	) (map[pulid.ID][]*shipment.ChargeAllocation, error)
}

type ChargeAllocationsByShipmentIDLoaderFactoryParams struct {
	fx.In

	ChargeAllocationRepo chargeallocationports.ChargeAllocationRepository
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
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *ChargeAllocationsByShipmentIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[[]*shipment.ChargeAllocation] {
	return loadercore.BatchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*shipment.ChargeAllocation, error) {
			return f.allocations.ListByShipmentIDs(
				ctx,
				&chargeallocationports.ListChargeAllocationsByShipmentIDsRequest{
					TenantInfo:  tenantInfo,
					ShipmentIDs: ids,
				},
			)
		},
	)
}

type ChargeAllocationsByOrderChargeIDLoaderFactoryParams struct {
	fx.In

	ChargeAllocationRepo chargeallocationports.ChargeAllocationRepository
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
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *ChargeAllocationsByOrderChargeIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[[]*shipment.ChargeAllocation] {
	return loadercore.BatchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*shipment.ChargeAllocation, error) {
			return f.allocations.ListByOrderChargeIDs(
				ctx,
				&chargeallocationports.ListChargeAllocationsByOrderChargeIDsRequest{
					TenantInfo:     tenantInfo,
					OrderChargeIDs: ids,
				},
			)
		},
	)
}

var (
	chargeAllocationsByShipmentIDKey    = loadercore.NewKey(newChargeAllocationsByShipmentIDLoader)
	chargeAllocationsByOrderChargeIDKey = loadercore.NewKey(
		newChargeAllocationsByOrderChargeIDLoader,
	)
)

func ChargeAllocationsByShipmentID(
	ctx context.Context,
	key string,
) ([]*shipment.ChargeAllocation, error) {
	return loadercore.Load(ctx, chargeAllocationsByShipmentIDKey, key)
}

func (f *ChargeAllocationsByShipmentIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}

func ChargeAllocationsByOrderChargeID(
	ctx context.Context,
	key string,
) ([]*shipment.ChargeAllocation, error) {
	return loadercore.Load(ctx, chargeAllocationsByOrderChargeIDKey, key)
}

func (f *ChargeAllocationsByOrderChargeIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
