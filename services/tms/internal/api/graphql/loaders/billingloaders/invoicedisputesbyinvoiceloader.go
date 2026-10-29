package billingloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
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
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *InvoiceDisputesByInvoiceIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[[]*invoicedispute.InvoiceDispute] {
	return loadercore.BatchGroupFunc(
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

var invoiceDisputesByInvoiceIDKey = loadercore.NewKey(newInvoiceDisputesByInvoiceIDLoader)

func InvoiceDisputesByInvoiceID(
	ctx context.Context,
	key string,
) ([]*invoicedispute.InvoiceDispute, error) {
	return loadercore.Load(ctx, invoiceDisputesByInvoiceIDKey, key)
}

func (f *InvoiceDisputesByInvoiceIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
