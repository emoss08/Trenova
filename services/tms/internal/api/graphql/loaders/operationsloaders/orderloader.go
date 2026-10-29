package operationsloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
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
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *OrderByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[*order.Order] {
	return loadercore.BatchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*order.Order, error) {
			return f.orderService.GetByIDs(ctx, orderports.GetOrdersByIDsRequest{
				TenantInfo: tenantInfo,
				OrderIDs:   ids,
			})
		},
		"Order not found",
	)
}

var orderByIDKey = loadercore.NewKey(newOrderByIDLoader)

func OrderByID(ctx context.Context, key string) (*order.Order, error) {
	return loadercore.Load(ctx, orderByIDKey, key)
}

func (f *OrderByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
