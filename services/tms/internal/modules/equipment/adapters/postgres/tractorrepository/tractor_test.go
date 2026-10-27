package tractorrepository

import (
	"slices"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/bootstrap/crossref"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor/tractordb"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/tractorports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func renderTractors(t *testing.T, includes tractorports.TractorRelationIncludes) (string, []any) {
	t.Helper()

	q, err := testRepository().tractorQuery(&includes)
	require.NoError(t, err)

	return keeltest.Render(t, q.Limit(10))
}

func TestTractorQuery_ReadsTheSelectedColumns(t *testing.T) {
	t.Parallel()

	sql, _ := renderTractors(t, tractorports.TractorRelationIncludes{
		TractorColumns: []string{"id", "code", "created_at"},
	})

	assert.Equal(
		t,
		`SELECT t."id", t."code", t."created_at", t."business_unit_id", t."organization_id" `+
			`FROM "tractors" t WHERE t."business_unit_id" = $1 AND t."organization_id" = $2 LIMIT 10`,
		sql,
	)
}

func TestTractorQuery_RefusesAnUnknownColumn(t *testing.T) {
	t.Parallel()

	_, err := testRepository().tractorQuery(
		&tractorports.TractorRelationIncludes{TractorColumns: []string{"nope"}},
	)
	require.Error(t, err)
}

func TestTractorQuery_ReadsTheLastKnownLocationThroughOneLateralJoin(t *testing.T) {
	t.Parallel()

	sql, args := renderTractors(t, tractorports.TractorRelationIncludes{
		TractorColumns:           []string{"id", "created_at"},
		IncludeLastKnownLocation: true,
	})

	assert.Contains(t, sql, `(lkc."current_location_id"), `+
		`((SELECT l."name" FROM "locations" l WHERE l."business_unit_id" = $1 `+
		`AND l."organization_id" = $2 AND l."id" = lkc."current_location_id"))`)
	assert.Contains(
		t,
		sql,
		`LEFT JOIN LATERAL (SELECT lkc2."current_location_id" FROM "equipment_continuity" lkc2 `+
			`WHERE lkc2."business_unit_id" = $1 AND lkc2."organization_id" = $2 AND lkc2."equipment_id" = t."id" `+
			`AND lkc2."equipment_type" = 'Tractor' AND lkc2."is_current" IS TRUE LIMIT 1) lkc ON TRUE`,
	)
	assert.Contains(t, sql, `SELECT l2."latitude" FROM "locations" l2`)
	assert.Contains(t, sql, `SELECT l3."longitude" FROM "locations" l3`)
	assert.Equal(t, []any{"bu_1", "org_1"}, args)
}

func TestTractorQuery_ReadsTheMoveInProgressInTheSameStatement(t *testing.T) {
	t.Parallel()

	sql, args := renderTractors(t, tractorports.TractorRelationIncludes{
		TractorColumns:        []string{"id", "created_at"},
		IncludeInProgressMove: true,
	})
	without, _ := renderTractors(t, tractorports.TractorRelationIncludes{})

	assert.Contains(t, sql, `FROM "assignments"`)
	assert.Contains(t, sql, `."tractor_id" = t."id"`)
	assert.Contains(t, sql, `."archived_at" IS NULL`)
	assert.Contains(t, sql, `."pro_number"`)
	assert.Contains(t, sql, `."sequence" DESC`)
	assert.True(t, strings.Contains(sql, `'InTransit'`) || slices.Contains(args, any("InTransit")),
		"only a move in transit counts")
	assert.NotContains(t, sql, `"trailer_id"`)
	assert.NotContains(t, without, `"assignments"`)
}

func TestTractorQuery_LeavesTheLocationOutWhenUnselected(t *testing.T) {
	t.Parallel()

	sql, _ := renderTractors(t, tractorports.TractorRelationIncludes{})

	assert.NotContains(t, sql, "equipment_continuity")
	assert.NotContains(t, sql, "locations")
}

func TestTractorQuery_JoinsTheRequestedRelationsWithTheirColumns(t *testing.T) {
	t.Parallel()

	sql, _ := renderTractors(t, tractorports.TractorRelationIncludes{
		IncludeEquipmentType: true,
		EquipmentTypeColumns: []string{"code"},
		IncludePrimaryWorker: true,
		IncludeFleetCode:     true,
	})

	assert.Contains(t, sql, `et."code", et."id"`)
	assert.Contains(t, sql, `LEFT JOIN "equipment_types" et ON et."id" = t."equipment_type_id" `+
		`AND et."business_unit_id" = $1 AND et."organization_id" = $2`)
	assert.Contains(t, sql, `LEFT JOIN "workers" pw ON pw."id" = t."primary_worker_id"`)
	assert.Contains(t, sql, `LEFT JOIN "fleet_codes" fc ON fc."id" = t."fleet_code_id"`)
	assert.NotContains(t, sql, `"organizations"`)
}

func TestTractorQuery_TenantDetailsLoadTheTenantParents(t *testing.T) {
	t.Parallel()

	sql, _ := renderTractors(t, tractorports.TractorRelationIncludes{IncludeTenantDetails: true})

	assert.Contains(t, sql, `LEFT JOIN "business_units" bu ON bu."id" = t."business_unit_id"`)
	assert.Contains(
		t,
		sql,
		`LEFT JOIN "organizations" o ON o."id" = t."organization_id" AND o."business_unit_id" = $1`,
	)
	assert.Contains(t, sql, `LEFT JOIN "us_states" s ON s."id" = o."state_id"`)
}

func TestTractorQuery_RelationIncludesAreIndependent(t *testing.T) {
	t.Parallel()

	sql, _ := renderTractors(t, tractorports.TractorRelationIncludes{IncludeOwnerWorker: true})

	assert.Contains(t, sql, `LEFT JOIN "workers" ow ON ow."id" = t."owner_worker_id"`)
	assert.NotContains(t, sql, `"equipment_types"`)
}

func TestStatusFilter(t *testing.T) {
	t.Parallel()

	repo := &repository{l: zap.NewNop()}

	assert.Nil(t, repo.statusFilter(""))
	assert.Nil(t, repo.statusFilter("Sideways"))

	sql, args := keeltest.Render(t, tractordb.Tractors.
		Where(repo.statusFilter(string(domaintypes.EquipmentStatusAvailable))).Limit(1))
	assert.Contains(t, sql, `AND t."status" = 'Available' LIMIT 1`)
	assert.Len(t, args, 2)
}

func TestNormalizeLookupKeys_LowersTrimsAndDedupes(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		[]string{"trk-1", "trk-2"},
		normalizeLookupKeys([]string{" TRK-1 ", "trk-1", "", "Trk-2"}),
	)
}

func testRepository() *repository {
	return &repository{locations: crossref.LocationColumns(), moves: crossref.MoveColumns()}
}
