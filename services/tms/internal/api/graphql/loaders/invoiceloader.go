package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type invoicesByIDsGetter interface {
	GetByIDs(
		ctx context.Context,
		req repositories.GetInvoicesByIDsRequest,
	) ([]*invoice.Invoice, error)
}

type InvoiceByIDLoaderFactoryParams struct {
	fx.In

	InvoiceService services.InvoiceService
}

type InvoiceByIDLoaderFactory struct {
	invoices invoicesByIDsGetter
}

func NewInvoiceByIDLoaderFactory(
	p InvoiceByIDLoaderFactoryParams,
) *InvoiceByIDLoaderFactory {
	return &InvoiceByIDLoaderFactory{
		invoices: p.InvoiceService,
	}
}

func (f *InvoiceByIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *invoice.Invoice] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *InvoiceByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[*invoice.Invoice] {
	return batchByIDFunc(func(ctx context.Context, ids []pulid.ID) ([]*invoice.Invoice, error) {
		return f.invoices.GetByIDs(ctx, repositories.GetInvoicesByIDsRequest{
			TenantInfo: tenantInfo,
			InvoiceIDs: ids,
		})
	}, "Invoice not found within your organization")
}
