package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/customerservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type customersByIDsGetter interface {
	GetByIDs(
		ctx context.Context,
		req repositories.GetCustomersByIDsRequest,
	) ([]*customer.Customer, error)
}

type CustomerByIDLoaderFactoryParams struct {
	fx.In

	CustomerService *customerservice.Service
}

type CustomerByIDLoaderFactory struct {
	customers customersByIDsGetter
}

func NewCustomerByIDLoaderFactory(
	p CustomerByIDLoaderFactoryParams,
) *CustomerByIDLoaderFactory {
	return &CustomerByIDLoaderFactory{
		customers: p.CustomerService,
	}
}

func (f *CustomerByIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *customer.Customer] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *CustomerByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[*customer.Customer] {
	return batchByIDFunc(func(ctx context.Context, ids []pulid.ID) ([]*customer.Customer, error) {
		return f.customers.GetByIDs(ctx, repositories.GetCustomersByIDsRequest{
			TenantInfo:  tenantInfo,
			CustomerIDs: ids,
			CustomerFilterOptions: repositories.CustomerFilterOptions{
				IncludeState: true,
			},
		})
	}, "Customer not found within your organization")
}
