//go:build integration

package ratezonerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/ratezone"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratezoneports"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest/ratingdb"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGetByIDsReturnsOnlyTheTenantsZones(t *testing.T) {
	rdb := ratingdb.NewDatabase(t)
	ctx, db := rdb.A.Ctx, rdb.Admin

	first := insertZone(t, ctx, db, rdb.A.Info, "ATL", domaintypes.StatusActive)
	second := insertZone(t, ctx, db, rdb.A.Info, "SE", domaintypes.StatusInactive)
	foreign := insertZone(t, rdb.B.Ctx, db, rdb.B.Info, "MW", domaintypes.StatusActive)

	repo := New(Params{DB: rdb.Conn, Logger: zap.NewNop()})

	zones, err := repo.GetByIDs(ctx, &ratezoneports.GetRateZonesByIDsRequest{
		TenantInfo:  rdb.A.Info,
		RateZoneIDs: []pulid.ID{second.ID, foreign.ID, first.ID, pulid.MustNew("rzn_")},
	})
	require.NoError(t, err)

	codes := make(map[pulid.ID]string, len(zones))
	for _, zone := range zones {
		codes[zone.ID] = zone.Code
	}
	assert.Equal(t, map[pulid.ID]string{first.ID: "ATL", second.ID: "SE"}, codes)

	none, err := repo.GetByIDs(ctx, &ratezoneports.GetRateZonesByIDsRequest{TenantInfo: rdb.A.Info})
	require.NoError(t, err)
	assert.Equal(t, []*ratezone.RateZone{}, none)
}
