package partiesloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/parties/app/customerservice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type customersByIDsGetter interface {
	GetByIDs(
		ctx context.Context,
		req customerports.GetCustomersByIDsRequest,
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

func newCustomerByIDLoader(f *CustomerByIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *customer.Customer] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *CustomerByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[*customer.Customer] {
	return loadercore.BatchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*customer.Customer, error) {
			return f.customers.GetByIDs(ctx, customerports.GetCustomersByIDsRequest{
				TenantInfo:  tenantInfo,
				CustomerIDs: ids,
				CustomerFilterOptions: customerports.CustomerFilterOptions{
					IncludeState: true,
				},
			})
		},
		"Customer not found within your organization",
	)
}

var customerByIDKey = loadercore.NewKey(newCustomerByIDLoader)

func CustomerByID(ctx context.Context, key string) (*customer.Customer, error) {
	return loadercore.Load(ctx, customerByIDKey, key)
}

func (f *CustomerByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
