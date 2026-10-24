package tractorrepository

import (
	"testing"

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

	q, err := tractorQuery(&includes)
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

	_, err := tractorQuery(&tractorports.TractorRelationIncludes{TractorColumns: []string{"nope"}})
	require.Error(t, err)
}

func TestTractorQuery_ReadsTheLastKnownLocationThroughOneLateralJoin(t *testing.T) {
	t.Parallel()

	sql, args := renderTractors(t, tractorports.TractorRelationIncludes{
		TractorColumns:           []string{"id", "created_at"},
		IncludeLastKnownLocation: true,
	})

	assert.Contains(t, sql, `(lkc."current_location_id"), `+
		`((SELECT l.name FROM locations l WHERE l.id = lkc."current_location_id" `+
		`AND l.organization_id = t."organization_id" AND l.business_unit_id = t."business_unit_id"))`)
	assert.Contains(
		t,
		sql,
		`LEFT JOIN LATERAL (SELECT lkc2."current_location_id" FROM "equipment_continuity" lkc2 `+
			`WHERE lkc2."business_unit_id" = $1 AND lkc2."organization_id" = $2 AND lkc2."equipment_id" = t."id" `+
			`AND lkc2."equipment_type" = $3 AND lkc2."is_current" IS TRUE LIMIT 1) lkc ON TRUE`,
	)
	assert.Contains(t, sql, "l.latitude")
	assert.Contains(t, sql, "l.longitude")
	assert.Equal(t, []any{"bu_1", "org_1", "Tractor"}, args)
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
	assert.Contains(t, sql, `AND t."status" = $3 LIMIT 1`)
	assert.Equal(t, "Available", args[2])
}

func TestNormalizeLookupKeys_LowersTrimsAndDedupes(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		[]string{"trk-1", "trk-2"},
		normalizeLookupKeys([]string{" TRK-1 ", "trk-1", "", "Trk-2"}),
	)
}
