//go:build integration

package ratezonerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/places/domain/rategeo"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratezone"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratezoneports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func insertZone(
	t *testing.T,
	db *bun.DB,
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
	_, err := db.NewInsert().Model(zone).Exec(t.Context())
	require.NoError(t, err)
	_, err = db.NewInsert().Model(&zone.Members).Exec(t.Context())
	require.NoError(t, err)
	return zone
}

func TestResolveLaneZonesReturnsEachEndsZonesMostSpecificFirst(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenantInfo := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}

	state := pulid.MustNew("ust_")
	metro := insertZone(t, db, tenantInfo, "ATL", domaintypes.StatusActive,
		&ratezone.RateZoneMember{ScopeType: rategeo.ScopeTypeZip3, ScopeValue: "30301"})
	region := insertZone(t, db, tenantInfo, "SE", domaintypes.StatusActive,
		&ratezone.RateZoneMember{ScopeType: rategeo.ScopeTypeState, ScopeValue: state.String()})
	insertZone(t, db, tenantInfo, "OLD", domaintypes.StatusInactive,
		&ratezone.RateZoneMember{ScopeType: rategeo.ScopeTypeZip3, ScopeValue: "30301"})
	midwest := insertZone(t, db, tenantInfo, "MW", domaintypes.StatusActive,
		&ratezone.RateZoneMember{ScopeType: rategeo.ScopeTypeZip3, ScopeValue: "60601"})

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

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
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenantInfo := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	zone := insertZone(t, db, tenantInfo, "ATL", domaintypes.StatusActive,
		&ratezone.RateZoneMember{ScopeType: rategeo.ScopeTypeZip3, ScopeValue: "30301"})

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	lane, err := repo.ResolveLaneZones(ctx, &ratezoneports.ResolveLaneZonesRequest{
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		OriginKeys: []string{zone.Members[0].MatchKey},
	})
	require.NoError(t, err)
	assert.Empty(t, lane.Origin)
}
