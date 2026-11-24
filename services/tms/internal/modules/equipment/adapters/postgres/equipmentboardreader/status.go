package equipmentboardreader

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/i18n"
)

func equipmentStatus(ctx context.Context, status string) (string, boardcatalog.Tone) {
	switch domaintypes.EquipmentStatus(status) {
	case domaintypes.EquipmentStatusAvailable:
		return i18n.T(ctx, "Available"), boardcatalog.ToneOK
	case domaintypes.EquipmentStatusAtMaintenance:
		return i18n.T(ctx, "At Maintenance"), boardcatalog.ToneInfo
	case domaintypes.EquipmentStatusOOS:
		return i18n.T(ctx, "Out of Service"), boardcatalog.ToneBad
	case domaintypes.EquipmentStatusSold:
		return i18n.T(ctx, "Sold"), boardcatalog.ToneWarn
	}

	return status, ""
}
