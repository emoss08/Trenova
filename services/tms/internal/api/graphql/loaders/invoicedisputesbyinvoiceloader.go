package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicedispute"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type invoiceDisputesByInvoiceIDsLister interface {
	ListByInvoiceIDs(
		ctx context.Context,
		req *invoiceports.ListInvoiceDisputesByInvoiceIDsRequest,
	) (map[pulid.ID][]*invoicedispute.InvoiceDispute, error)
}

type InvoiceDisputesByInvoiceIDLoaderFactoryParams struct {
	fx.In

	InvoiceDisputeRepo invoiceports.InvoiceDisputeRepository
}

type InvoiceDisputesByInvoiceIDLoaderFactory struct {
	disputes invoiceDisputesByInvoiceIDsLister
}

func NewInvoiceDisputesByInvoiceIDLoaderFactory(
	p InvoiceDisputesByInvoiceIDLoaderFactoryParams,
) *InvoiceDisputesByInvoiceIDLoaderFactory {
	return &InvoiceDisputesByInvoiceIDLoaderFactory{disputes: p.InvoiceDisputeRepo}
}

func newInvoiceDisputesByInvoiceIDLoader(f *InvoiceDisputesByInvoiceIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*invoicedispute.InvoiceDispute] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *InvoiceDisputesByInvoiceIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[[]*invoicedispute.InvoiceDispute] {
	return batchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*invoicedispute.InvoiceDispute, error) {
			return f.disputes.ListByInvoiceIDs(
				ctx,
				&invoiceports.ListInvoiceDisputesByInvoiceIDsRequest{
					TenantInfo: tenantInfo,
					InvoiceIDs: ids,
				},
			)
		},
	)
}
