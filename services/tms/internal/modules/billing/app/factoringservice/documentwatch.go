package factoringservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type DocumentObserverParams struct {
	fx.In

	Invoices  invoiceports.InvoiceRepository
	Requester factoringports.EvaluationRequester `optional:"true"`
	Logger    *zap.Logger
}

type DocumentObserver struct {
	invoices  invoiceports.InvoiceRepository
	requester factoringports.EvaluationRequester
	l         *zap.Logger
}

var _ documentports.DocumentChangeObserver = (*DocumentObserver)(nil)

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func NewDocumentObserver(p DocumentObserverParams) documentports.DocumentChangeObserver {
	return &DocumentObserver{
		invoices:  p.Invoices,
		requester: p.Requester,
		l:         p.Logger.Named("factoring.document-observer"),
	}
}

func (o *DocumentObserver) DocumentsChanged(
	ctx context.Context,
	change *documentports.DocumentChange,
) {
	if o.requester == nil || change == nil {
		return
	}
	ids := parseIDs(change.ResourceIDs)
	if len(ids) == 0 {
		return
	}

	switch change.ResourceType {
	case invoiceResourceType:
		o.requester.RequestEvaluation(ctx, change.TenantInfo, &factoringports.EvaluationTargets{
			InvoiceIDs: ids,
		})
	case shipmentResourceType:
		byShipment, err := o.invoices.ListByShipmentIDs(ctx, invoiceports.ListInvoicesByShipmentIDsRequest{
			TenantInfo:  change.TenantInfo,
			ShipmentIDs: ids,
		})
		if err != nil {
			o.l.Warn("could not find the invoices behind changed shipment documents", zap.Error(err))
			return
		}
		invoiceIDs := make([]pulid.ID, 0, len(byShipment))
		for _, invoices := range byShipment {
			for _, inv := range invoices {
				if inv != nil && inv.FactoringCompanyID != nil && !inv.FactoringCompanyID.IsNil() {
					invoiceIDs = append(invoiceIDs, inv.ID)
				}
			}
		}
		if len(invoiceIDs) == 0 {
			return
		}
		o.requester.RequestEvaluation(ctx, change.TenantInfo, &factoringports.EvaluationTargets{
			InvoiceIDs: invoiceIDs,
		})
	}
}

func parseIDs(raw []string) []pulid.ID {
	ids := make([]pulid.ID, 0, len(raw))
	for _, value := range raw {
		id, err := pulid.Parse(value)
		if err != nil || id.IsNil() {
			continue
		}
		ids = append(ids, id)
	}
	return ids
}
