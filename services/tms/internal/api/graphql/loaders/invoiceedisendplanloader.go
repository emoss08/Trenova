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

type invoiceEDISendPlanResolver interface {
	ResolveEDISendPlans(
		ctx context.Context,
		req *invoiceports.ResolveInvoiceEDISendPlansRequest,
	) (map[pulid.ID]*invoiceports.InvoiceEDISendPlan, error)
}

type ediPlanInvoicesGetter interface {
	GetByIDs(
		ctx context.Context,
		req invoiceports.GetInvoicesByIDsRequest,
	) ([]*invoice.Invoice, error)
}

type InvoiceEDISendPlanByInvoiceIDLoaderFactoryParams struct {
	fx.In

	InvoiceService invoiceports.InvoiceService
	InvoiceRepo    invoiceports.InvoiceRepository
}

// InvoiceEDISendPlanByInvoiceIDLoaderFactory resolves EDI send plans for a
// page of invoices at once: one customer lookup, one partner lookup, and one
// profile lookup per distinct partner.
type InvoiceEDISendPlanByInvoiceIDLoaderFactory struct {
	plans    invoiceEDISendPlanResolver
	invoices ediPlanInvoicesGetter
}

func NewInvoiceEDISendPlanByInvoiceIDLoaderFactory(
	p InvoiceEDISendPlanByInvoiceIDLoaderFactoryParams,
) *InvoiceEDISendPlanByInvoiceIDLoaderFactory {
	return &InvoiceEDISendPlanByInvoiceIDLoaderFactory{
		plans:    p.InvoiceService,
		invoices: p.InvoiceRepo,
	}
}

func newInvoiceEDISendPlanByInvoiceIDLoader(f *InvoiceEDISendPlanByInvoiceIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *invoiceports.InvoiceEDISendPlan] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *InvoiceEDISendPlanByInvoiceIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[*invoiceports.InvoiceEDISendPlan] {
	return func(ctx context.Context, keys []string) ([]*invoiceports.InvoiceEDISendPlan, []error) {
		results := make([]*invoiceports.InvoiceEDISendPlan, len(keys))
		errs := make([]error, len(keys))
		ids := make([]pulid.ID, 0, len(keys))
		for idx, key := range keys {
			id, err := pulid.MustParse(key)
			if err != nil {
				errs[idx] = err
				continue
			}
			ids = append(ids, id)
		}
		if len(ids) == 0 {
			return results, errs
		}

		invoices, err := f.invoices.GetByIDs(ctx, invoiceports.GetInvoicesByIDsRequest{
			TenantInfo: tenantInfo,
			InvoiceIDs: ids,
		})
		if err != nil {
			return failAll(results, errs, err)
		}
		plans, err := f.plans.ResolveEDISendPlans(
			ctx,
			&invoiceports.ResolveInvoiceEDISendPlansRequest{
				TenantInfo: tenantInfo,
				Invoices:   invoices,
			},
		)
		if err != nil {
			return failAll(results, errs, err)
		}
		for idx, key := range keys {
			if errs[idx] != nil {
				continue
			}
			plan := plans[pulid.ID(key)]
			if plan == nil {
				plan = &invoiceports.InvoiceEDISendPlan{
					InvoiceID: pulid.ID(key),
					Status:    invoice.EDISendStatusNotSent,
					Blockers:  []string{"Invoice not found"},
				}
			}
			results[idx] = plan
		}

		return results, errs
	}
}

func failAll(
	results []*invoiceports.InvoiceEDISendPlan,
	errs []error,
	err error,
) ([]*invoiceports.InvoiceEDISendPlan, []error) {
	for idx := range errs {
		if errs[idx] == nil {
			errs[idx] = err
		}
	}

	return results, errs
}
