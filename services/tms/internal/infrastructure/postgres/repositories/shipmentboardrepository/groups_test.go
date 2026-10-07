package shipmentboardrepository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func renderGroupSummary(t *testing.T, groupBy shipment.BoardGrouping) string {
	t.Helper()

	db := bun.NewDB(nil, pgdialect.New())
	q, err := groupSummaryQuery(
		db.NewSelect().Model((*shipment.Shipment)(nil)),
		groupBy,
		"America/Chicago",
	)
	require.NoError(t, err)
	return q.String()
}

func TestGroupSummaryBucketsShipDatesByLocalDayOfTheFirstPickup(t *testing.T) {
	t.Parallel()

	sql := renderGroupSummary(t, shipment.BoardGroupingShipDate)

	assert.Contains(t, sql, "LEFT JOIN stops AS grp_stop ON (grp_stop.id = (SELECT end_stp.id")
	assert.Contains(t, sql, "end_stp.type IN ('Pickup', 'SplitPickup')")
	assert.Contains(t, sql, `ORDER BY end_sm.sequence ASC, end_stp.sequence ASC, end_stp.id COLLATE "C" ASC LIMIT 1`)
	assert.Contains(t, sql, `COALESCE(to_char(to_timestamp(grp_stop.scheduled_window_start) AT TIME ZONE 'America/Chicago', 'YYYY-MM-DD'), '') AS "group_key"`)
	assert.Contains(t, sql, `GROUP BY "group_key"`)
	assert.Contains(t, sql, "ORDER BY MIN(grp_stop.scheduled_window_start) ASC NULLS LAST")
}

func TestGroupSummaryBucketsDeliveryDatesByTheLastDelivery(t *testing.T) {
	t.Parallel()

	sql := renderGroupSummary(t, shipment.BoardGroupingDeliveryDate)

	assert.Contains(t, sql, "end_stp.type IN ('Delivery', 'SplitDelivery')")
	assert.Contains(t, sql, `ORDER BY end_sm.sequence DESC, end_stp.sequence DESC, end_stp.id COLLATE "C" DESC LIMIT 1`)
}

func TestGroupSummaryOrdersCustomersTheWayTheBoardSortsThem(t *testing.T) {
	t.Parallel()

	sql := renderGroupSummary(t, shipment.BoardGroupingCustomer)

	assert.Contains(t, sql, "LEFT JOIN customers AS grp_cus ON (grp_cus.id = sp.customer_id)")
	assert.Contains(t, sql, "sp.customer_id AS group_key")
	assert.Contains(t, sql, "COALESCE(grp_cus.name, '') AS group_label")
	assert.Contains(t, sql, "GROUP BY sp.customer_id, grp_cus.name")
	assert.Contains(t, sql, "ORDER BY grp_cus.name ASC NULLS LAST, sp.customer_id ASC")
}

func TestGroupSummaryPutsShipmentsWithoutAnOwnerLast(t *testing.T) {
	t.Parallel()

	sql := renderGroupSummary(t, shipment.BoardGroupingOwner)

	assert.Contains(t, sql, "LEFT JOIN users AS grp_own ON (grp_own.id = sp.owner_id)")
	assert.Contains(t, sql, "COALESCE(sp.owner_id, '') AS group_key")
	assert.Contains(t, sql, "ORDER BY grp_own.name ASC NULLS LAST, sp.owner_id ASC NULLS LAST")
}

func TestGroupSummaryRefusesAnUnknownGrouping(t *testing.T) {
	t.Parallel()

	db := bun.NewDB(nil, pgdialect.New())
	_, err := groupSummaryQuery(
		db.NewSelect().Model((*shipment.Shipment)(nil)),
		shipment.BoardGrouping("Stage"),
		"UTC",
	)

	require.ErrorIs(t, err, ErrUnsupportedGrouping)
}

func TestGroupSummaryScansTheGroupsInsideTheBoardScope(t *testing.T) {
	t.Parallel()

	repo, mock := newTestRepository(t)
	mock.ExpectQuery(`SELECT COUNT\(\*\) AS count, COALESCE\(SUM\(sp\.total_charge_amount\), 0\) AS revenue, sp\.customer_id AS group_key, .* FROM "shipments" AS "sp" LEFT JOIN customers AS grp_cus .* WHERE .*sp\.organization_id.* GROUP BY sp\.customer_id, grp_cus\.name`).
		WillReturnRows(sqlmock.NewRows([]string{"count", "revenue", "group_key", "group_label"}).
			AddRow(4, "2400.25", "cus_1", "Acme Foods").
			AddRow(1, "0", "cus_2", ""))

	rows, err := repo.GroupSummary(t.Context(), &repositories.SummarizeShipmentBoardGroupsRequest{
		Scope:    testScope(),
		GroupBy:  shipment.BoardGroupingCustomer,
		Timezone: "UTC",
	})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "cus_1", rows[0].Key)
	assert.Equal(t, "Acme Foods", rows[0].Label)
	assert.Equal(t, 4, rows[0].Count)
	assert.True(t, rows[0].Revenue.Equal(decimal.RequireFromString("2400.25")))
	assert.Empty(t, rows[1].Label)
	require.NoError(t, mock.ExpectationsWereMet())
}
