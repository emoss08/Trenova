package shipmentboardrepository

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testScope() *shipmentports.ShipmentBoardScope {
	return &shipmentports.ShipmentBoardScope{
		Filter: &pagination.QueryOptions{
			FieldFilters: []domaintypes.FieldFilter{
				{Field: "status", Operator: dbtype.OpEqual, Value: "New"},
			},
		},
		Options: shipmentports.ShipmentOptions{
			QuickFilterBasis: &shipmentports.ShipmentQuickFilterBasis{
				Now:       time.Unix(1_791_318_600, 0).UTC(),
				Location:  time.UTC,
				Detention: &shipmentports.ShipmentDetentionBasis{ThresholdMinutes: 30},
				Margin: &shipmentports.ShipmentMarginBasis{
					CostPerMile:         decimal.NewFromInt(2),
					TargetMarginPercent: decimal.NewFromInt(10),
				},
			},
		},
	}
}

func renderGroupSummary(t *testing.T, groupBy shipment.BoardGrouping) (string, []any) {
	t.Helper()

	q, err := groupSummaryQuery(shipmentdb.Shipments.Query(), groupBy, "America/Chicago")
	require.NoError(t, err)

	return keeltest.Render(t, q)
}

func TestGroupSummaryBucketsShipDatesByLocalDayOfTheFirstPickup(t *testing.T) {
	t.Parallel()

	sql, args := renderGroupSummary(t, shipment.BoardGroupingShipDate)

	assert.Contains(t, sql, `LEFT JOIN "stops" grp_stop ON`)
	assert.Contains(t, sql, `grp_stop."id" = (SELECT end_stp."id"`)
	assert.Contains(
		t,
		sql,
		`ORDER BY end_sm."sequence" ASC, end_stp."sequence" ASC, (end_stp."id" COLLATE "C") ASC LIMIT 1`,
	)
	key := `coalesce(to_char(timezone($3::text, to_timestamp(grp_stop."scheduled_window_start")), $4), $5)`
	assert.Contains(
		t,
		sql,
		`SELECT `+key+`, $6::text, count(*), coalesce(sum(s."total_charge_amount"), $7) FROM`,
	)
	assert.Contains(t, sql, ` GROUP BY 1 ORDER BY`)
	assert.Contains(t, sql, `min(grp_stop."scheduled_window_start") ASC NULLS LAST`)
	assert.Equal(t, "America/Chicago", args[2])
	assert.Equal(t, []any{"", "", 0}, args[4:7])
	assert.Contains(t, sql, `end_stp."type" IN ('Pickup', 'SplitPickup')`)
}

func TestGroupSummaryBucketsDeliveryDatesByTheLastDelivery(t *testing.T) {
	t.Parallel()

	sql, _ := renderGroupSummary(t, shipment.BoardGroupingDeliveryDate)

	assert.Contains(
		t,
		sql,
		`ORDER BY end_sm."sequence" DESC, end_stp."sequence" DESC, (end_stp."id" COLLATE "C") DESC LIMIT 1`,
	)
	assert.Contains(t, sql, `end_stp."type" IN ('Delivery', 'SplitDelivery')`)
}

func TestGroupSummaryOrdersCustomersTheWayTheBoardSortsThem(t *testing.T) {
	t.Parallel()

	sql, _ := renderGroupSummary(t, shipment.BoardGroupingCustomer)

	assert.Contains(t, sql, `LEFT JOIN "customers"`)
	assert.Contains(t, sql, `GROUP BY s."customer_id", `)
	assert.Contains(t, sql, `."name" ASC NULLS LAST, s."customer_id" ASC`)
}

func TestGroupSummaryPutsShipmentsWithoutAnOwnerLast(t *testing.T) {
	t.Parallel()

	sql, _ := renderGroupSummary(t, shipment.BoardGroupingOwner)

	assert.Contains(t, sql, `LEFT JOIN "users"`)
	assert.Contains(t, sql, `SELECT coalesce(s."owner_id", $3), coalesce(o."name", $4), count(*)`)
	assert.Contains(t, sql, `s."owner_id" ASC NULLS LAST`)
}

func TestGroupSummaryRefusesAnUnknownGrouping(t *testing.T) {
	t.Parallel()

	_, err := groupSummaryQuery(
		shipmentdb.Shipments.Query(),
		shipment.BoardGrouping("Stage"),
		"UTC",
	)
	require.ErrorIs(t, err, ErrUnsupportedGrouping)
}

func TestStageSummaryReadsTheScopedShipments(t *testing.T) {
	t.Parallel()

	q, err := scoped(testScope())
	require.NoError(t, err)

	sp := shipmentdb.Shipments
	sql, args := keeltest.Render(t, q.Group(sp.StageRank).Pick(sp.StageRank, revenueSum()))
	assert.Contains(t, sql, `GROUP BY s."stage_rank"`)
	assert.Contains(t, sql, `coalesce(sum(s."total_charge_amount"), $4)`)
	assert.Contains(t, args, "New")

	_, err = scoped(nil)
	require.ErrorIs(t, err, ErrScopeMissing)
}

func TestQuickFilterTotalsIsOneQuery(t *testing.T) {
	t.Parallel()

	q, err := scoped(testScope())
	require.NoError(t, err)

	totals, err := quickFilterTotalsQuery(
		q,
		testScope().Options.QuickFilterBasis,
		[]shipment.QuickFilterSpec{
			shipment.Quick(shipment.QuickFilterLate),
			shipment.Quick(shipment.QuickFilterDetention),
		},
	)
	require.NoError(t, err)

	sql, _ := keeltest.Render(t, totals)
	assert.Contains(t, sql, `ARRAY[count(*) FILTER (WHERE s."stage_rank" = `)
	assert.Contains(t, sql, `ARRAY[CAST(coalesce(sum(s."total_charge_amount") FILTER (WHERE `)
	assert.Contains(t, sql, ` AS text)]`)
}
