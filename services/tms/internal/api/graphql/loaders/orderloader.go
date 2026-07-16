package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/orderservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/graph-gophers/dataloader/v7"
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

func (f *OrderByIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloader.Loader[string, *order.Order] {
	return dataloader.NewBatchedLoader(f.batchFunc(tenantInfo))
}

func (f *OrderByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) dataloader.BatchFunc[string, *order.Order] {
	return batchByIDFunc(func(ctx context.Context, ids []pulid.ID) ([]*order.Order, error) {
		return f.orderService.GetByIDs(ctx, repositories.GetOrdersByIDsRequest{
			TenantInfo: tenantInfo,
			OrderIDs:   ids,
		})
	}, "Order not found")
}
