package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type invoicesByShipmentIDsLister interface {
	ListByShipmentIDs(
		ctx context.Context,
		req invoiceports.ListInvoicesByShipmentIDsRequest,
	) (map[pulid.ID][]*invoice.Invoice, error)
}

type InvoicesByShipmentIDLoaderFactoryParams struct {
	fx.In

	InvoiceRepo invoiceports.InvoiceRepository
}

type InvoicesByShipmentIDLoaderFactory struct {
	invoices invoicesByShipmentIDsLister
}

func NewInvoicesByShipmentIDLoaderFactory(
	p InvoicesByShipmentIDLoaderFactoryParams,
) *InvoicesByShipmentIDLoaderFactory {
	return &InvoicesByShipmentIDLoaderFactory{
		invoices: p.InvoiceRepo,
	}
}

func (f *InvoicesByShipmentIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*invoice.Invoice] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *InvoicesByShipmentIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[[]*invoice.Invoice] {
	return batchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*invoice.Invoice, error) {
			return f.invoices.ListByShipmentIDs(ctx, invoiceports.ListInvoicesByShipmentIDsRequest{
				TenantInfo:  tenantInfo,
				ShipmentIDs: ids,
			})
		},
	)
}
