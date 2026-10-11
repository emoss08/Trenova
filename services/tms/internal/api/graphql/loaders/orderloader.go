package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/app/orderservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type OrderByIDLoaderFactoryParams struct {
	fx.In

	OrderService *orderservice.Service
}

type OrderByIDLoaderFactory struct {
	orderService *orderservice.Service
}

func NewOrderByIDLoaderFactory(
	p OrderByIDLoaderFactoryParams,
) *OrderByIDLoaderFactory {
	return &OrderByIDLoaderFactory{
		orderService: p.OrderService,
	}
}

func newOrderByIDLoader(f *OrderByIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *order.Order] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *OrderByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[*order.Order] {
	return batchByIDFunc(func(ctx context.Context, ids []pulid.ID) ([]*order.Order, error) {
		return f.orderService.GetByIDs(ctx, orderports.GetOrdersByIDsRequest{
			TenantInfo: tenantInfo,
			OrderIDs:   ids,
		})
	}, "Order not found")
}
