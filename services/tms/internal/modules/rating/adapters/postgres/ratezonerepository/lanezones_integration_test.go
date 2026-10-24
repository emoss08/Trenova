//go:build integration

package ratezonerepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/places/domain/rategeo"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratezone"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratezone/ratezonedb"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratezoneports"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest/ratingdb"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func insertZone(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	tenantInfo pagination.TenantInfo,
	code string,
	status domaintypes.Status,
	members ...*ratezone.RateZoneMember,
) *ratezone.RateZone {
	t.Helper()

	zone := &ratezone.RateZone{
		ID:             pulid.MustNew("rzn_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Code:           code,
		Name:           code,
		Kind:           ratezone.ZoneKindRegional,
		Status:         status,
		Members:        members,
	}
	for _, member := range members {
		member.ID = pulid.MustNew("rzm_")
	}
	require.NoError(t, zone.StampMembers(t.Context()))
	require.NoError(t, ratezonedb.RateZones.Create(ctx, db, zone))
	require.NoError(t, ratezonedb.RateZoneMembers.CreateMany(ctx, db, zone.Members))
	return zone
}

func TestResolveLaneZonesReturnsEachEndsZonesMostSpecificFirst(t *testing.T) {
	rdb := ratingdb.NewDatabase(t)
	ctx, db, tenantInfo := rdb.A.Ctx, rdb.Admin, rdb.A.Info

	state := pulid.MustNew("ust_")
	metro := insertZone(t, ctx, db, tenantInfo, "ATL", domaintypes.StatusActive,
		&ratezone.RateZoneMember{ScopeType: rategeo.ScopeTypeZip3, ScopeValue: "30301"})
	region := insertZone(t, ctx, db, tenantInfo, "SE", domaintypes.StatusActive,
		&ratezone.RateZoneMember{ScopeType: rategeo.ScopeTypeState, ScopeValue: state.String()})
	insertZone(t, ctx, db, tenantInfo, "OLD", domaintypes.StatusInactive,
		&ratezone.RateZoneMember{ScopeType: rategeo.ScopeTypeZip3, ScopeValue: "30301"})
	midwest := insertZone(t, ctx, db, tenantInfo, "MW", domaintypes.StatusActive,
		&ratezone.RateZoneMember{ScopeType: rategeo.ScopeTypeZip3, ScopeValue: "60601"})

	repo := New(Params{DB: rdb.Conn, Logger: zap.NewNop()})

	lane, err := repo.ResolveLaneZones(ctx, &ratezoneports.ResolveLaneZonesRequest{
		TenantInfo:      tenantInfo,
		OriginKeys:      []string{metro.Members[0].MatchKey, region.Members[0].MatchKey},
		DestinationKeys: []string{midwest.Members[0].MatchKey},
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"ATL", "SE"}, ratezoneports.ZoneCodes(lane.Origin),
		"the zip3 zone outranks the state zone, and an inactive zone is left out")
	assert.Equal(t, []string{"MW"}, ratezoneports.ZoneCodes(lane.Destination))

	codes, err := repo.GetCodesByIDs(ctx, tenantInfo, []pulid.ID{metro.ID, midwest.ID})
	require.NoError(t, err)
	assert.Equal(t, map[pulid.ID]string{metro.ID: "ATL", midwest.ID: "MW"}, codes)

	empty, err := repo.ResolveLaneZones(ctx, &ratezoneports.ResolveLaneZonesRequest{
		TenantInfo: tenantInfo,
	})
	require.NoError(t, err)
	assert.Empty(t, empty.Origin)
	assert.Empty(t, empty.Destination)
}

func TestResolveLaneZonesNeverReadsAnotherTenantsZones(t *testing.T) {
	rdb := ratingdb.NewDatabase(t)
	ctx, db, tenantInfo := rdb.A.Ctx, rdb.Admin, rdb.A.Info
	zone := insertZone(t, ctx, db, tenantInfo, "ATL", domaintypes.StatusActive,
		&ratezone.RateZoneMember{ScopeType: rategeo.ScopeTypeZip3, ScopeValue: "30301"})

	repo := New(Params{DB: rdb.Conn, Logger: zap.NewNop()})
	lane, err := repo.ResolveLaneZones(rdb.B.Ctx, &ratezoneports.ResolveLaneZonesRequest{
		TenantInfo: rdb.B.Info,
		OriginKeys: []string{zone.Members[0].MatchKey},
	})
	require.NoError(t, err)
	assert.Empty(t, lane.Origin)
}
