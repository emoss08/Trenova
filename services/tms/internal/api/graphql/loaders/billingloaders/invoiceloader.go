package billingloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type invoicesByIDsGetter interface {
	GetByIDs(
		ctx context.Context,
		req invoiceports.GetInvoicesByIDsRequest,
	) ([]*invoice.Invoice, error)
}

type InvoiceByIDLoaderFactoryParams struct {
	fx.In

	InvoiceQueryService invoiceports.InvoiceQueryService
}

type InvoiceByIDLoaderFactory struct {
	invoices invoicesByIDsGetter
}

func NewInvoiceByIDLoaderFactory(
	p InvoiceByIDLoaderFactoryParams,
) *InvoiceByIDLoaderFactory {
	return &InvoiceByIDLoaderFactory{
		invoices: p.InvoiceQueryService,
	}
}

func newInvoiceByIDLoader(f *InvoiceByIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *invoice.Invoice] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *InvoiceByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[*invoice.Invoice] {
	return loadercore.BatchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*invoice.Invoice, error) {
			return f.invoices.GetByIDs(ctx, invoiceports.GetInvoicesByIDsRequest{
				TenantInfo: tenantInfo,
				InvoiceIDs: ids,
			})
		},
		"Invoice not found within your organization",
	)
}

var invoiceByIDKey = loadercore.NewKey(newInvoiceByIDLoader)

func InvoiceByID(ctx context.Context, key string) (*invoice.Invoice, error) {
	return loadercore.Load(ctx, invoiceByIDKey, key)
}

func (f *InvoiceByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
