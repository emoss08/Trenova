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

func newInvoicesByShipmentIDLoader(f *InvoicesByShipmentIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*invoice.Invoice] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *InvoicesByShipmentIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[[]*invoice.Invoice] {
	return loadercore.BatchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*invoice.Invoice, error) {
			return f.invoices.ListByShipmentIDs(ctx, invoiceports.ListInvoicesByShipmentIDsRequest{
				TenantInfo:  tenantInfo,
				ShipmentIDs: ids,
			})
		},
	)
}

var invoicesByShipmentIDKey = loadercore.NewKey(newInvoicesByShipmentIDLoader)

func InvoicesByShipmentID(ctx context.Context, key string) ([]*invoice.Invoice, error) {
	return loadercore.Load(ctx, invoicesByShipmentIDKey, key)
}

func InvoicesByShipmentIDAll(ctx context.Context, keys []string) ([][]*invoice.Invoice, error) {
	return loadercore.LoadAll(ctx, invoicesByShipmentIDKey, keys)
}

func (f *InvoicesByShipmentIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
