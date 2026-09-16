package lanequery_test

import (
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/lanequery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func newTestDB(t *testing.T) *bun.DB {
	t.Helper()

	sqlDB, _, err := sqlmock.New()
	require.NoError(t, err)
	db := bun.NewDB(sqlDB, pgdialect.New())
	t.Cleanup(func() { _ = db.Close() })

	return db
}

func TestNew_TenantScopedLanes(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	orgLiteral := "'" + tenant.OrgID.String() + "'"
	buLiteral := "'" + tenant.BuID.String() + "'"
	cols := buncolgen.ShipmentColumns

	sql := lanequery.New(db, lanequery.Options{
		Tenant: &tenant,
		ShipmentFilter: func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.
				Where(cols.CreatedAt.Gte(), 100).
				Where(cols.CreatedAt.Lte(), 200).
				Where(cols.Status.Ne(), shipment.StatusCanceled)
		},
	}).String()
	t.Log(sql)

	assert.Contains(
		t,
		sql,
		`WITH "lane_endpoints" AS (SELECT array_agg(stp.location_id ORDER BY sm.sequence, stp.sequence) FILTER (WHERE stp.type IN ('Pickup', 'SplitPickup')) AS pickup_location_ids, `+
			`array_agg(stp.location_id ORDER BY sm.sequence, stp.sequence) FILTER (WHERE stp.type IN ('Delivery', 'SplitDelivery')) AS delivery_location_ids FROM "shipments" AS "sp"`,
	)
	assert.Contains(
		t,
		sql,
		"JOIN shipment_moves AS sm ON sm.shipment_id = sp.id AND sm.organization_id = sp.organization_id AND sm.business_unit_id = sp.business_unit_id",
	)
	assert.Contains(
		t,
		sql,
		"JOIN stops AS stp ON stp.shipment_move_id = sm.id AND stp.organization_id = sm.organization_id AND stp.business_unit_id = sm.business_unit_id",
	)
	assert.Contains(
		t,
		sql,
		"WHERE (sp.organization_id = "+orgLiteral+") AND (sp.business_unit_id = "+buLiteral+
			") AND (sp.created_at >= 100) AND (sp.created_at <= 200) AND (sp.status != 'Canceled') GROUP BY sp.id)",
	)
	assert.Contains(
		t,
		sql,
		`"lane_pairs" AS (SELECT pickup_location_ids[1] AS origin_location_id, delivery_location_ids[cardinality(delivery_location_ids)] AS destination_location_id, COUNT(*) AS shipment_count FROM lane_endpoints `+
			"WHERE (pickup_location_ids IS NOT NULL) AND (delivery_location_ids IS NOT NULL) GROUP BY origin_location_id, destination_location_id)",
	)
	assert.Contains(
		t,
		sql,
		"JOIN locations AS loc_orig ON loc_orig.id = lp.origin_location_id AND loc_orig.organization_id = "+
			orgLiteral+" AND loc_orig.business_unit_id = "+buLiteral,
	)
	assert.Contains(
		t,
		sql,
		"JOIN locations AS loc_dest ON loc_dest.id = lp.destination_location_id AND loc_dest.organization_id = "+
			orgLiteral+" AND loc_dest.business_unit_id = "+buLiteral,
	)
	assert.Contains(t, sql, "JOIN us_states AS ust_orig ON ust_orig.id = loc_orig.state_id")
	assert.Contains(t, sql, "JOIN us_states AS ust_dest ON ust_dest.id = loc_dest.state_id")
	assert.Contains(
		t,
		sql,
		"SELECT ust_orig.abbreviation AS origin_state, ust_dest.abbreviation AS destination_state, SUM(lp.shipment_count)::int AS count FROM lane_pairs AS lp",
	)
	assert.True(t, strings.HasSuffix(sql, "GROUP BY ust_orig.abbreviation, ust_dest.abbreviation"))
	assert.NotContains(t, sql, "lp.organization_id")
	assert.NotContains(t, sql, "status IS")
}

func TestNew_InstanceWideLanesByStatus(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	cols := buncolgen.ShipmentColumns
	statuses := []shipment.Status{shipment.StatusNew, shipment.StatusInTransit}

	sql := lanequery.New(db, lanequery.Options{
		ShipmentFilter: func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where(cols.Status.In(), bun.List(statuses))
		},
		GroupByStatus: true,
	}).String()
	t.Log(sql)

	assert.Contains(
		t,
		sql,
		"AS delivery_location_ids, sp.organization_id, sp.business_unit_id, sp.status FROM",
	)
	assert.Contains(
		t,
		sql,
		"WHERE (sp.status IN ('New', 'InTransit')) GROUP BY sp.id, sp.organization_id, sp.business_unit_id, sp.status)",
	)
	assert.NotContains(t, sql, "sp.organization_id = ")
	assert.Contains(
		t,
		sql,
		"GROUP BY origin_location_id, destination_location_id, organization_id, business_unit_id, status)",
	)
	assert.Contains(
		t,
		sql,
		"JOIN locations AS loc_orig ON loc_orig.id = lp.origin_location_id AND loc_orig.organization_id = lp.organization_id AND loc_orig.business_unit_id = lp.business_unit_id",
	)
	assert.Contains(
		t,
		sql,
		"JOIN locations AS loc_dest ON loc_dest.id = lp.destination_location_id AND loc_dest.organization_id = lp.organization_id AND loc_dest.business_unit_id = lp.business_unit_id",
	)
	assert.Contains(t, sql, "SUM(lp.shipment_count)::int AS count, lp.status FROM lane_pairs AS lp")
	assert.Contains(t, sql, "GROUP BY ust_orig.abbreviation, ust_dest.abbreviation, lp.status")
}
