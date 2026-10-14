package factoringservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type watchedInvoices struct {
	invoiceports.InvoiceRepository
	requester factoringports.EvaluationRequester
}

func WatchInvoices(
	repo invoiceports.InvoiceRepository,
	requester factoringports.EvaluationRequester,
) invoiceports.InvoiceRepository {
	if requester == nil {
		return repo
	}
	return &watchedInvoices{InvoiceRepository: repo, requester: requester}
}

func (w *watchedInvoices) Create(
	ctx context.Context,
	entity *invoice.Invoice,
) (*invoice.Invoice, error) {
	created, err := w.InvoiceRepository.Create(ctx, entity)
	if err == nil {
		w.request(ctx, created)
	}
	return created, err
}

func (w *watchedInvoices) Update(
	ctx context.Context,
	entity *invoice.Invoice,
) (*invoice.Invoice, error) {
	updated, err := w.InvoiceRepository.Update(ctx, entity)
	if err == nil {
		w.request(ctx, updated)
	}
	return updated, err
}

func (w *watchedInvoices) request(ctx context.Context, inv *invoice.Invoice) {
	if inv == nil || inv.FactoringCompanyID == nil || inv.FactoringCompanyID.IsNil() {
		return
	}
	w.requester.RequestEvaluation(
		ctx,
		pagination.TenantInfo{OrgID: inv.OrganizationID, BuID: inv.BusinessUnitID},
		&factoringports.EvaluationTargets{InvoiceIDs: []pulid.ID{inv.ID}},
	)
}
