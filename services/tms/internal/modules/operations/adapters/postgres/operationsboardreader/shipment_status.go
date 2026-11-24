package operationsboardreader

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/shared/i18n"
)

func shipmentStatus(ctx context.Context, status string) (string, boardcatalog.Tone) {
	switch shipmentvocab.Status(status) {
	case shipmentvocab.StatusNew:
		return i18n.T(ctx, "New"), ""
	case shipmentvocab.StatusPartiallyAssigned:
		return i18n.T(ctx, "Partially assigned"), boardcatalog.ToneInfo
	case shipmentvocab.StatusAssigned:
		return i18n.T(ctx, "Assigned"), boardcatalog.ToneWarn
	case shipmentvocab.StatusInTransit:
		return i18n.T(ctx, "In transit"), boardcatalog.ToneInfo
	case shipmentvocab.StatusDelayed:
		return i18n.T(ctx, "Delayed"), boardcatalog.ToneWarn
	case shipmentvocab.StatusPartiallyCompleted:
		return i18n.T(ctx, "Partially completed"), boardcatalog.ToneInfo
	case shipmentvocab.StatusReadyToInvoice:
		return i18n.T(ctx, "Ready to invoice"), boardcatalog.ToneWarn
	case shipmentvocab.StatusCompleted:
		return i18n.T(ctx, "Completed"), boardcatalog.ToneOK
	case shipmentvocab.StatusInvoiced:
		return i18n.T(ctx, "Invoiced"), boardcatalog.ToneOK
	case shipmentvocab.StatusCanceled:
		return i18n.T(ctx, "Canceled"), boardcatalog.ToneBad
	}

	return status, ""
}
