package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type paymentApplicationsByInvoiceIDsLister interface {
	ListApplicationsByInvoiceIDs(
		ctx context.Context,
		req *customerpaymentports.ListApplicationsByInvoiceIDsRequest,
	) (map[pulid.ID][]*customerpayment.Application, error)
}

type creditMemoApplicationsByInvoiceIDsLister interface {
	ListCreditMemoApplicationsByInvoiceIDs(
		ctx context.Context,
		req *customerpaymentports.ListApplicationsByInvoiceIDsRequest,
	) (map[pulid.ID][]*customerpayment.CreditMemoApplication, error)
}

type CustomerPaymentApplicationsByInvoiceIDLoaderFactoryParams struct {
	fx.In

	CustomerPaymentRepo customerpaymentports.CustomerPaymentRepository
}

type CustomerPaymentApplicationsByInvoiceIDLoaderFactory struct {
	applications paymentApplicationsByInvoiceIDsLister
}

func NewCustomerPaymentApplicationsByInvoiceIDLoaderFactory(
	p CustomerPaymentApplicationsByInvoiceIDLoaderFactoryParams,
) *CustomerPaymentApplicationsByInvoiceIDLoaderFactory {
	return &CustomerPaymentApplicationsByInvoiceIDLoaderFactory{
		applications: p.CustomerPaymentRepo,
	}
}

func (f *CustomerPaymentApplicationsByInvoiceIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*customerpayment.Application] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *CustomerPaymentApplicationsByInvoiceIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[[]*customerpayment.Application] {
	return batchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*customerpayment.Application, error) {
			return f.applications.ListApplicationsByInvoiceIDs(
				ctx,
				&customerpaymentports.ListApplicationsByInvoiceIDsRequest{
					TenantInfo: tenantInfo,
					InvoiceIDs: ids,
				},
			)
		},
	)
}

type CreditMemoApplicationsByInvoiceIDLoaderFactoryParams struct {
	fx.In

	CustomerPaymentRepo customerpaymentports.CustomerPaymentRepository
}

type CreditMemoApplicationsByInvoiceIDLoaderFactory struct {
	applications creditMemoApplicationsByInvoiceIDsLister
}

func NewCreditMemoApplicationsByInvoiceIDLoaderFactory(
	p CreditMemoApplicationsByInvoiceIDLoaderFactoryParams,
) *CreditMemoApplicationsByInvoiceIDLoaderFactory {
	return &CreditMemoApplicationsByInvoiceIDLoaderFactory{
		applications: p.CustomerPaymentRepo,
	}
}

func (f *CreditMemoApplicationsByInvoiceIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*customerpayment.CreditMemoApplication] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *CreditMemoApplicationsByInvoiceIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[[]*customerpayment.CreditMemoApplication] {
	return batchGroupFunc(
		func(
			ctx context.Context,
			ids []pulid.ID,
		) (map[pulid.ID][]*customerpayment.CreditMemoApplication, error) {
			return f.applications.ListCreditMemoApplicationsByInvoiceIDs(
				ctx,
				&customerpaymentports.ListApplicationsByInvoiceIDsRequest{
					TenantInfo: tenantInfo,
					InvoiceIDs: ids,
				},
			)
		},
	)
}
