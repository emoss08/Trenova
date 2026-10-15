package shipmentresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
)

func (r *QueryResolver) shipmentsInvoicesBill(
	ctx context.Context,
	rawInvoiceIDs []string,
) ([]pulid.ID, error) {
	if rawInvoiceIDs == nil {
		return nil, nil
	}

	authCtx, err := r.RequirePermission(ctx, permission.ResourceInvoice, permission.OpRead)
	if err != nil {
		return nil, err
	}
	invoiceIDs, err := base.ParseIDs(rawInvoiceIDs)
	if err != nil {
		return nil, err
	}
	if len(invoiceIDs) == 0 {
		return []pulid.ID{}, nil
	}

	invoices, err := r.InvoiceService.GetByIDs(ctx, invoiceports.GetInvoicesByIDsRequest{
		TenantInfo: base.TenantInfo(authCtx),
		InvoiceIDs: invoiceIDs,
	})
	if err != nil {
		return nil, err
	}

	return invoice.BilledShipmentIDs(invoices...), nil
}

func emptyShipmentConnection() *gqlmodel.ShipmentConnection {
	total := 0

	return &gqlmodel.ShipmentConnection{
		Edges:      []*gqlmodel.ShipmentEdge{},
		TotalCount: &total,
		PageInfo:   base.PageInfo(false, nil),
	}
}
