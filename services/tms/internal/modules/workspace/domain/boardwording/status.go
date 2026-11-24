package boardwording

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/i18n"
)

func ActiveStatus(ctx context.Context, status string) (string, boardcatalog.Tone) {
	switch domaintypes.Status(status) {
	case domaintypes.StatusActive:
		return i18n.T(ctx, "Active"), boardcatalog.ToneInfo
	case domaintypes.StatusInactive:
		return i18n.T(ctx, "Inactive"), boardcatalog.ToneBad
	}

	return status, ""
}
