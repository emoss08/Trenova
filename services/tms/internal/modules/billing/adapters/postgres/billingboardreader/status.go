package billingboardreader

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicedispute"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/shared/i18n"
)

func invoiceStatus(ctx context.Context, status string) (string, boardcatalog.Tone) {
	switch invoice.Status(status) {
	case invoice.StatusDraft:
		return i18n.T(ctx, "Draft"), ""
	case invoice.StatusPosted:
		return i18n.T(ctx, "Posted"), boardcatalog.ToneOK
	case invoice.StatusVoided:
		return i18n.T(ctx, "Voided"), boardcatalog.ToneBad
	}

	return status, ""
}

func disputeStatus(ctx context.Context, status string) (string, boardcatalog.Tone) {
	switch invoicedispute.DisputeCaseStatus(status) {
	case invoicedispute.DisputeCaseStatusOpen:
		return i18n.T(ctx, "Open"), boardcatalog.ToneInfo
	case invoicedispute.DisputeCaseStatusResolved:
		return i18n.T(ctx, "Resolved"), boardcatalog.ToneOK
	case invoicedispute.DisputeCaseStatusWithdrawn:
		return i18n.T(ctx, "Withdrawn"), ""
	}

	return status, ""
}

func receiptStatus(ctx context.Context, status string) (string, boardcatalog.Tone) {
	switch bankreceipt.Status(status) {
	case bankreceipt.StatusImported:
		return i18n.T(ctx, "Imported"), ""
	case bankreceipt.StatusMatched:
		return i18n.T(ctx, "Matched"), boardcatalog.ToneOK
	case bankreceipt.StatusException:
		return i18n.T(ctx, "Exception"), boardcatalog.ToneBad
	}

	return status, ""
}
