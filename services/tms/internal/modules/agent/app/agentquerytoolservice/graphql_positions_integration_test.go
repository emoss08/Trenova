//go:build integration

package agentquerytoolservice_test

import (
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/agentread/agentreadtest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/telematicsrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/app/telematicsservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// The fleet map's positions are read under row-level security: a tenant's
// recent readings and none of another's, narrowed to the tractors named, each
// with how many minutes old it is, and a reading past the window left out
// until the call widens it.
func TestConvertedLists_PositionsReadTheTenantsRecentReadings(t *testing.T) {
	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	conn := dbtest.NewRLSConnection(t, admin, dsn)
	cases, services := convertedListCases(conn)
	repo := telematicsrepository.New(telematicsrepository.Params{DB: conn, Logger: zap.NewNop()})
	services.TelematicsService = telematicsservice.New(telematicsservice.Params{
		Repo: repo, Logger: zap.NewNop(),
	})
	idx := slices.IndexFunc(cases, func(c referenceCase) bool {
		return c.tool == "equipment_list_tractors"
	})
	require.GreaterOrEqual(t, idx, 0)

	mine := agentreadtest.SeedTenant(t, admin, "A")
	theirs := agentreadtest.SeedTenant(t, admin, "B")
	ours := writeReferenceRows(t, &cases[idx], mine, "A")
	other := writeReferenceRows(t, &cases[idx], theirs, "B")

	now := timeutils.NowUnix()
	report := func(tenant *referenceTenant, ids []string, stale string) {
		positions := make([]*telematics.VehiclePosition, 0, len(ids))
		for _, id := range ids {
			recorded := now - 60
			if id == stale {
				recorded = now - 6*3600
			}
			positions = append(positions, &telematics.VehiclePosition{
				OrganizationID:    tenant.tenant.Actor.OrganizationID,
				BusinessUnitID:    tenant.tenant.Actor.BusinessUnitID,
				TractorID:         pulid.ID(id),
				Provider:          "Samsara",
				ProviderVehicleID: id,
				Latitude:          32.7767,
				Longitude:         -96.797,
				SpeedMph:          54,
				RecordedAt:        recorded,
				ReceivedAt:        recorded,
			})
		}
		require.NoError(t, repo.UpsertVehiclePositions(tenant.ctx, positions))
	}
	stale := ours.rows[0]
	report(ours, ours.rows, stale)
	report(other, other.rows, "")

	engine := &mocks.DenyResourcesPermissionEngine{}
	reader := agentreadtest.NewReader(t, agentreadtest.Options{Engine: engine, Resolvers: services})
	tool, err := agentquerytoolservice.ConvertedListTool("equipment_list_positions", reader, engine)
	require.NoError(t, err)

	read := func(args map[string]any) (ids []string, rows []map[string]any) {
		page := callTool(t, ours.ctx, tool, mine.Actor, args)
		items, _ := page["items"].([]any)
		for _, item := range items {
			row, _ := item.(map[string]any)
			ids = append(ids, row["tractorId"].(string))
			rows = append(rows, row)
		}
		return ids, rows
	}

	t.Run("the tenant's recent readings and none of another's", func(t *testing.T) {
		ids, rows := read(map[string]any{})
		assert.ElementsMatch(t, ours.rows[1:], ids)
		for _, row := range rows {
			assert.EqualValues(t, 1, row["ageMinutes"])
			assert.NotEmpty(t, row["tractorCode"])
		}
	})

	t.Run("a wider window reaches the older reading", func(t *testing.T) {
		ids, rows := read(map[string]any{"maxAgeSeconds": 8 * 3600})
		assert.ElementsMatch(t, ours.rows, ids)
		for _, row := range rows {
			if row["tractorId"] == stale {
				assert.EqualValues(t, 360, row["ageMinutes"])
			}
		}
	})

	t.Run("only the tractors named", func(t *testing.T) {
		ids, _ := read(map[string]any{"tractorIds": []any{ours.rows[1], other.rows[1]}})
		assert.Equal(t, []string{ours.rows[1]}, ids)
	})
}
