package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type invoiceDisputesByInvoiceIDsLister interface {
	ListByInvoiceIDs(
		ctx context.Context,
		req *repositories.ListInvoiceDisputesByInvoiceIDsRequest,
	) (map[pulid.ID][]*invoice.InvoiceDispute, error)
}

type InvoiceDisputesByInvoiceIDLoaderFactoryParams struct {
	fx.In

	InvoiceDisputeRepo repositories.InvoiceDisputeRepository
}

type InvoiceDisputesByInvoiceIDLoaderFactory struct {
	disputes invoiceDisputesByInvoiceIDsLister
}

func NewInvoiceDisputesByInvoiceIDLoaderFactory(
	p InvoiceDisputesByInvoiceIDLoaderFactoryParams,
) *InvoiceDisputesByInvoiceIDLoaderFactory {
	return &InvoiceDisputesByInvoiceIDLoaderFactory{disputes: p.InvoiceDisputeRepo}
}

func (f *InvoiceDisputesByInvoiceIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*invoice.InvoiceDispute] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *InvoiceDisputesByInvoiceIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[[]*invoice.InvoiceDispute] {
	return batchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*invoice.InvoiceDispute, error) {
			return f.disputes.ListByInvoiceIDs(
				ctx,
				&repositories.ListInvoiceDisputesByInvoiceIDsRequest{
					TenantInfo: tenantInfo,
					InvoiceIDs: ids,
				},
			)
		},
	)
}
