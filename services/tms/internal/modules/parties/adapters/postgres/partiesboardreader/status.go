package partiesboardreader

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/shared/i18n"
)

func complianceStatus(ctx context.Context, status string) (string, boardcatalog.Tone) {
	switch carrier.ComplianceStatus(status) {
	case carrier.ComplianceStatusPending:
		return i18n.T(ctx, "Pending"), boardcatalog.ToneWarn
	case carrier.ComplianceStatusQualified:
		return i18n.T(ctx, "Qualified"), boardcatalog.ToneOK
	case carrier.ComplianceStatusDisqualified:
		return i18n.T(ctx, "Disqualified"), boardcatalog.ToneBad
	case carrier.ComplianceStatusExpired:
		return i18n.T(ctx, "Expired"), boardcatalog.ToneBad
	}

	return status, ""
}
