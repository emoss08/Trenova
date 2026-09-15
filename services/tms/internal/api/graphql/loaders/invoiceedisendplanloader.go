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

type invoiceEDISendPlanResolver interface {
	ResolveEDISendPlans(
		ctx context.Context,
		req *services.ResolveInvoiceEDISendPlansRequest,
	) (map[pulid.ID]*services.InvoiceEDISendPlan, error)
}

type ediPlanInvoicesGetter interface {
	GetByIDs(ctx context.Context, req repositories.GetInvoicesByIDsRequest) ([]*invoice.Invoice, error)
}

type InvoiceEDISendPlanByInvoiceIDLoaderFactoryParams struct {
	fx.In

	InvoiceService services.InvoiceService
	InvoiceRepo    repositories.InvoiceRepository
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
	return &InvoiceEDISendPlanByInvoiceIDLoaderFactory{plans: p.InvoiceService, invoices: p.InvoiceRepo}
}

func (f *InvoiceEDISendPlanByInvoiceIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *services.InvoiceEDISendPlan] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *InvoiceEDISendPlanByInvoiceIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[*services.InvoiceEDISendPlan] {
	return func(ctx context.Context, keys []string) ([]*services.InvoiceEDISendPlan, []error) {
		results := make([]*services.InvoiceEDISendPlan, len(keys))
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

		invoices, err := f.invoices.GetByIDs(ctx, repositories.GetInvoicesByIDsRequest{
			TenantInfo: tenantInfo,
			InvoiceIDs: ids,
		})
		if err != nil {
			return failAll(results, errs, err)
		}
		plans, err := f.plans.ResolveEDISendPlans(ctx, &services.ResolveInvoiceEDISendPlansRequest{
			TenantInfo: tenantInfo,
			Invoices:   invoices,
		})
		if err != nil {
			return failAll(results, errs, err)
		}
		for idx, key := range keys {
			if errs[idx] != nil {
				continue
			}
			plan := plans[pulid.ID(key)]
			if plan == nil {
				plan = &services.InvoiceEDISendPlan{
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
	results []*services.InvoiceEDISendPlan,
	errs []error,
	err error,
) ([]*services.InvoiceEDISendPlan, []error) {
	for idx := range errs {
		if errs[idx] == nil {
			errs[idx] = err
		}
	}

	return results, errs
}
