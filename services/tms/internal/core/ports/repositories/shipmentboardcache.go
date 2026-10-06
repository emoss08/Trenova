package repositories

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ShipmentBoardSection string

const (
	ShipmentBoardSectionBriefing    = ShipmentBoardSection("briefing")
	ShipmentBoardSectionWatchlist   = ShipmentBoardSection("watchlist")
	ShipmentBoardSectionSuggestions = ShipmentBoardSection("suggestions")
)

type ShipmentBoardCacheKey struct {
	Section    ShipmentBoardSection
	TenantInfo pagination.TenantInfo
	Timezone   string
	Epoch      int64
	Variant    string
}

type ShipmentBoardEpochBumper interface {
	BumpEpoch(ctx context.Context, organizationID, businessUnitID pulid.ID) error
}

type ShipmentBoardCache interface {
	Epoch(ctx context.Context, tenantInfo pagination.TenantInfo) (int64, error)
	Get(ctx context.Context, key *ShipmentBoardCacheKey, dest any) (bool, error)
	Set(ctx context.Context, key *ShipmentBoardCacheKey, value any) error
}
