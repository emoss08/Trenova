package trailerrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer/trailerdb"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/trailerports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func renderTrailers(t *testing.T, includes trailerports.TrailerRelationIncludes) (string, []any) {
	t.Helper()

	q, err := trailerQuery(&includes)
	require.NoError(t, err)

	return keeltest.Render(t, q.Limit(10))
}

func TestTrailerQuery_ReadsTheSelectedColumns(t *testing.T) {
	t.Parallel()

	sql, _ := renderTrailers(t, trailerports.TrailerRelationIncludes{
		TrailerColumns: []string{"id", "code"},
	})

	assert.Contains(
		t,
		sql,
		`SELECT t."id", t."code", t."business_unit_id", t."organization_id" FROM "trailers" t`,
	)
}

func TestTrailerQuery_ReadsTheLocationsCoordinates(t *testing.T) {
	t.Parallel()

	sql, args := renderTrailers(
		t,
		trailerports.TrailerRelationIncludes{IncludeLastKnownLocation: true},
	)

	assert.Contains(t, sql, "SELECT l.latitude FROM locations l")
	assert.Contains(t, sql, "SELECT l.longitude FROM locations l")
	assert.Contains(
		t,
		sql,
		`LEFT JOIN LATERAL (SELECT lkc2."current_location_id" FROM "equipment_continuity" lkc2`,
	)
	assert.Contains(t, sql, `lkc2."equipment_type" = 'Trailer'`)
	assert.Equal(t, []any{"bu_1", "org_1"}, args)
}

func TestTrailerQuery_LeavesTheLocationOutWhenUnselected(t *testing.T) {
	t.Parallel()

	sql, _ := renderTrailers(t, trailerports.TrailerRelationIncludes{})

	assert.NotContains(t, sql, "equipment_continuity")
}

func TestTrailerQuery_LoadsTheOwnerOperatorOnlyWhenAsked(t *testing.T) {
	t.Parallel()

	with, _ := renderTrailers(t, trailerports.TrailerRelationIncludes{IncludeOwnerWorker: true})
	without, _ := renderTrailers(t, trailerports.TrailerRelationIncludes{})

	assert.Contains(t, with, `LEFT JOIN "workers" ow ON ow."id" = t."owner_worker_id"`)
	assert.NotContains(t, without, `"workers"`)
}

func TestTrailerQuery_TheFleetManagerLoadsThroughTheFleetCode(t *testing.T) {
	t.Parallel()

	sql, _ := renderTrailers(t, trailerports.TrailerRelationIncludes{IncludeFleetManager: true})

	assert.Contains(t, sql, `LEFT JOIN "fleet_codes" fc ON fc."id" = t."fleet_code_id"`)
	assert.Contains(t, sql, `LEFT JOIN "users" m ON m."id" = fc."manager_id"`)
}

func TestTrailerStatusFilter(t *testing.T) {
	t.Parallel()

	repo := &repository{l: zap.NewNop()}

	assert.Nil(t, repo.statusFilter(""))
	assert.Nil(t, repo.statusFilter("Sideways"))

	sql, _ := keeltest.Render(t, trailerdb.Trailers.
		Where(repo.statusFilter(string(domaintypes.EquipmentStatusAvailable))).Limit(1))
	assert.Contains(t, sql, `AND t."status" = 'Available' LIMIT 1`)
}
