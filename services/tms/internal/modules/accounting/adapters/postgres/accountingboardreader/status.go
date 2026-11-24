package accountingboardreader

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/shared/i18n"
)

func entryStatus(ctx context.Context, status string) (string, boardcatalog.Tone) {
	switch journalentry.Status(status) {
	case journalentry.StatusDraft:
		return i18n.T(ctx, "Draft"), ""
	case journalentry.StatusPending:
		return i18n.T(ctx, "Pending"), boardcatalog.ToneWarn
	case journalentry.StatusApproved:
		return i18n.T(ctx, "Approved"), boardcatalog.ToneInfo
	case journalentry.StatusPosted:
		return i18n.T(ctx, "Posted"), boardcatalog.ToneOK
	case journalentry.StatusReversed:
		return i18n.T(ctx, "Reversed"), ""
	case journalentry.StatusRejected:
		return i18n.T(ctx, "Rejected"), boardcatalog.ToneBad
	case journalentry.StatusVoid:
		return i18n.T(ctx, "Void"), boardcatalog.ToneBad
	}

	return status, ""
}
