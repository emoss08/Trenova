package billingloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
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

func newCustomerPaymentApplicationsByInvoiceIDLoader(
	f *CustomerPaymentApplicationsByInvoiceIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*customerpayment.Application] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *CustomerPaymentApplicationsByInvoiceIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[[]*customerpayment.Application] {
	return loadercore.BatchGroupFunc(
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

func newCreditMemoApplicationsByInvoiceIDLoader(f *CreditMemoApplicationsByInvoiceIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*customerpayment.CreditMemoApplication] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *CreditMemoApplicationsByInvoiceIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[[]*customerpayment.CreditMemoApplication] {
	return loadercore.BatchGroupFunc(
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

var (
	customerPaymentApplicationsByInvoiceIDKey = loadercore.NewKey(
		newCustomerPaymentApplicationsByInvoiceIDLoader,
	)
	creditMemoApplicationsByInvoiceIDKey = loadercore.NewKey(
		newCreditMemoApplicationsByInvoiceIDLoader,
	)
)

func CustomerPaymentApplicationsByInvoiceID(
	ctx context.Context,
	key string,
) ([]*customerpayment.Application, error) {
	return loadercore.Load(ctx, customerPaymentApplicationsByInvoiceIDKey, key)
}

func (f *CustomerPaymentApplicationsByInvoiceIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}

func CreditMemoApplicationsByInvoiceID(
	ctx context.Context,
	key string,
) ([]*customerpayment.CreditMemoApplication, error) {
	return loadercore.Load(ctx, creditMemoApplicationsByInvoiceIDKey, key)
}

func (f *CreditMemoApplicationsByInvoiceIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
