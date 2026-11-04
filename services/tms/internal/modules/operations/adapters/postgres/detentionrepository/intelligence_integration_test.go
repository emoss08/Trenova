//go:build integration

package detentionrepository_test

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/analytics/rollup"
	"github.com/emoss08/trenova/internal/bootstrap/crossref"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/detentionrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/detentionrollup"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// settledOccurrence is one settled stop as detention intelligence reads it. Times are Unix
// seconds; minutes past free time are raw dwell less free minutes.
type settledOccurrence struct {
	stopID           pulid.ID
	locationID       pulid.ID
	stopType         string
	scheduleType     string
	handling         string
	clockStartAt     int64
	arrivedAt        int64
	appointmentStart *int64
	arrivedLate      bool
	rawDwell         int
	freeMinutes      int
	status           string
	suppressed       bool
	gross            string
	billable         string
}

func insertSettled(ctx context.Context, t *testing.T, db *keel.DB, occ settledOccurrence) {
	t.Helper()

	_, err := keel.Exec(dbscope.WithSystem(ctx, "test"), db, `
		INSERT INTO detention_occurrences (
			id, business_unit_id, organization_id, shipment_id, shipment_move_id, stop_id,
			customer_id, location_id, stop_type, schedule_type, handling, clock_start_at,
			free_time_expires_at, is_open, arrived_at, appointment_start, arrived_late,
			raw_dwell_minutes, free_minutes_granted, rounded_minutes, gross_amount,
			billable_amount, status, suppressed_by_gate
		)
		SELECT $1, s.business_unit_id, s.organization_id, sm.shipment_id, s.shipment_move_id, s.id,
			sp.customer_id, $2, $3::stop_type_enum, $4::stop_schedule_type_enum,
			$5::stop_handling_enum, $6, $6 + 60, FALSE, $7, $8, $9, $10, $11,
			GREATEST($10 - $11, 0), $12::numeric, $13::numeric, $14, $15
		FROM stops AS s
		JOIN shipment_moves AS sm
			ON sm.id = s.shipment_move_id
			AND sm.organization_id = s.organization_id
			AND sm.business_unit_id = s.business_unit_id
		JOIN shipments AS sp
			ON sp.id = sm.shipment_id
			AND sp.organization_id = sm.organization_id
			AND sp.business_unit_id = sm.business_unit_id
		WHERE s.id = $16`,
		pulid.MustNew("dto_").String(),
		occ.locationID.String(),
		occ.stopType,
		occ.scheduleType,
		occ.handling,
		occ.clockStartAt,
		occ.arrivedAt,
		occ.appointmentStart,
		occ.arrivedLate,
		occ.rawDwell,
		occ.freeMinutes,
		occ.gross,
		occ.billable,
		occ.status,
		occ.suppressed,
		occ.stopID.String(),
	)
	require.NoError(t, err)
}

func freeStops(ctx context.Context, t *testing.T, db *keel.DB, count int) []seededStop {
	t.Helper()

	stops, err := keel.QueryAll(
		dbscope.WithSystem(ctx, "test"),
		db,
		pgx.RowToStructByName[seededStop],
		`
		SELECT s.id AS stop_id, s.organization_id, s.business_unit_id
		FROM stops AS s
		WHERE NOT EXISTS (
			SELECT 1 FROM detention_occurrences AS d
			WHERE d.stop_id = s.id
				AND d.organization_id = s.organization_id
				AND d.business_unit_id = s.business_unit_id
		)
		ORDER BY s.organization_id, s.business_unit_id, s.shipment_move_id, s.sequence
		LIMIT $1`,
		count,
	)
	require.NoError(t, err)
	require.Len(t, stops, count, "the development seed should hold stops with no detention yet")
	require.Equal(t, stops[0].OrganizationID, stops[count-1].OrganizationID)
	require.Equal(t, stops[0].BusinessUnitID, stops[count-1].BusinessUnitID)

	return stops
}

func ptr[T any](v T) *T { return &v }

var chicago = func() *time.Location {
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		panic(err)
	}
	return loc
}()

func chicagoDay(year int, month time.Month, day int) int64 {
	return time.Date(year, month, day, 0, 0, 0, 0, chicago).Unix()
}

// The window is the week of Monday 2030-01-07 in America/Chicago, far from any seeded occurrence,
// so the figures are only these.
var (
	monday               = chicagoDay(2030, time.January, 7)
	mondayMorningChicago = monday + 8*3600 + 30*60
	nextMonday           = chicagoDay(2030, time.January, 14)
	weekBefore           = chicagoDay(2029, time.December, 31)
	december             = chicagoDay(2029, time.December, 1)
	january              = chicagoDay(2030, time.January, 1)
)

// days is [from, end) as whole days with no whole month inside.
func days(from, end int64) detentionports.IntelligenceWindow {
	return detentionports.IntelligenceWindow{
		From:       from,
		To:         end - 1,
		MonthsFrom: end,
		MonthsEnd:  end,
	}
}

// countIn is the organization's day counted in Chicago, and the rollups refreshed from what the
// test wrote, as the worker's schedule refreshes them.
func countIn(ctx context.Context, t *testing.T, db *keel.DB, tenant pagination.TenantInfo) {
	t.Helper()
	system := dbscope.WithSystem(ctx, "test")
	_, err := keel.Exec(
		system,
		db,
		`UPDATE organizations SET timezone = 'America/Chicago' WHERE id = $1 AND business_unit_id = $2`,
		tenant.OrgID.String(),
		tenant.BuID.String(),
	)
	require.NoError(t, err)
	refresh(ctx, t, db)
}

func refresh(ctx context.Context, t *testing.T, db *keel.DB) {
	t.Helper()
	_, err := rollup.Refresh(dbscope.WithSystem(ctx, "test"), db, detentionrollup.All()...)
	require.NoError(t, err)
}

type intelligenceFixture struct {
	repo   detentionports.DetentionIntelligenceRepository
	tenant pagination.TenantInfo
	dock   pulid.ID
	other  pulid.ID
	ctx    context.Context
}

// settleTheWeek writes five settled stops in the week of 2030-01-07 and one the week before, and
// refreshes the rollups.
func settleTheWeek(t *testing.T) (*intelligenceFixture, *keel.DB) {
	t.Helper()
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	stops := freeStops(ctx, t, db, 6)
	dock := pulid.MustNew("loc_")
	other := pulid.MustNew("loc_")
	appointment := mondayMorningChicago

	// On time for its appointment and still 90 minutes past free time: the dock's.
	insertSettled(ctx, t, db, settledOccurrence{
		stopID: stops[0].StopID, locationID: dock, stopType: "Pickup", scheduleType: "Appointment",
		handling: "Live", clockStartAt: mondayMorningChicago, arrivedAt: mondayMorningChicago,
		appointmentStart: ptr(appointment), rawDwell: 210, freeMinutes: 120,
		status: "Billed", gross: "150", billable: "120",
	})
	// Arrived 90 minutes early and waited 60 past free time: an early arrival.
	insertSettled(ctx, t, db, settledOccurrence{
		stopID: stops[1].StopID, locationID: dock, stopType: "Delivery", scheduleType: "Appointment",
		handling: "Drop", clockStartAt: mondayMorningChicago, arrivedAt: appointment - 5400,
		appointmentStart: ptr(appointment), rawDwell: 180, freeMinutes: 120,
		status: "Waived", gross: "100", billable: "0",
	})
	// Late, so not billable, whatever else is true: a late arrival.
	insertSettled(ctx, t, db, settledOccurrence{
		stopID: stops[2].StopID, locationID: dock, stopType: "Pickup", scheduleType: "Open",
		handling: "Live", clockStartAt: mondayMorningChicago, arrivedAt: mondayMorningChicago,
		arrivedLate: true, rawDwell: 140, freeMinutes: 0,
		status: "NotBillable", gross: "80", billable: "0",
	})
	// An open dock, held for want of notice.
	insertSettled(ctx, t, db, settledOccurrence{
		stopID: stops[3].StopID, locationID: other, stopType: "Delivery", scheduleType: "Open",
		handling: "Live", clockStartAt: mondayMorningChicago, arrivedAt: mondayMorningChicago,
		rawDwell: 150, freeMinutes: 120, status: "Pending", suppressed: true,
		gross: "60", billable: "60",
	})
	// Inside free time: a stop, but not detained.
	insertSettled(ctx, t, db, settledOccurrence{
		stopID: stops[4].StopID, locationID: other, stopType: "Pickup", scheduleType: "Appointment",
		handling: "Live", clockStartAt: mondayMorningChicago, arrivedAt: mondayMorningChicago,
		appointmentStart: ptr(appointment), rawDwell: 60, freeMinutes: 120,
		status: "NotBillable", gross: "0", billable: "0",
	})
	// The week before, on New Year's Eve: 30 minutes past free time at the dock.
	insertSettled(ctx, t, db, settledOccurrence{
		stopID: stops[5].StopID, locationID: dock, stopType: "Pickup", scheduleType: "Appointment",
		handling: "Live", clockStartAt: weekBefore + 9*3600, arrivedAt: weekBefore + 9*3600,
		appointmentStart: ptr(weekBefore + 9*3600), rawDwell: 150, freeMinutes: 120,
		status: "Billed", gross: "50", billable: "50",
	})

	tenant := pagination.TenantInfo{OrgID: stops[0].OrganizationID, BuID: stops[0].BusinessUnitID}
	countIn(ctx, t, db, tenant)

	return &intelligenceFixture{
		repo: detentionrepository.NewIntelligenceRepository(detentionrepository.IntelligenceParams{
			DB:       postgres.WrapDB(db),
			Logger:   zap.NewNop(),
			Invoices: crossref.InvoiceSettlementColumns(),
		}),
		tenant: tenant,
		dock:   dock,
		other:  other,
		ctx:    dbscope.WithTenant(ctx, tenant.DBTenant()),
	}, db
}

func TestIntelligence_PutsEachWaitDownToOneCauseAndFollowsTheMoney(t *testing.T) {
	f, _ := settleTheWeek(t)

	rollup, err := f.repo.Intelligence(f.ctx, &detentionports.IntelligenceRequest{
		TenantInfo: f.tenant,
		Current:    days(monday, nextMonday),
		Previous:   days(weekBefore, monday),
		Timezone:   "America/Chicago",
		Limit:      100,
	})
	require.NoError(t, err)

	totals := rollup.Totals
	assert.Equal(t, 5, totals.StopCount)
	assert.Equal(t, 4, totals.DetainedCount)
	assert.Equal(t, int64(90+60+140+30), totals.PastFreeMinutes)
	assert.Equal(t, int64(90), totals.DockMinutes)
	assert.Equal(t, int64(60), totals.EarlyMinutes)
	assert.Equal(t, int64(140), totals.LateMinutes)
	assert.Equal(t, int64(30), totals.OpenMinutes)
	assert.Equal(t, int64(30), rollup.Previous.PastFreeMinutes)

	require.Len(t, rollup.Facilities, 2)
	assert.Equal(t, f.dock, rollup.Facilities[0].LocationID, "most hours past free time first")
	assert.Equal(t, int64(290), rollup.Facilities[0].PastFreeMinutes)
	assert.Equal(t, 177, rollup.Facilities[0].AvgDwellMinutes, "(210 + 180 + 140) / 3, rounded")
	assert.Equal(t, 120.0, rollup.Facilities[0].MedianFreeMinutes,
		"a late arrival forfeits its free time and does not pull the median down")
	require.Len(t, rollup.PreviousFacilities, 1)
	assert.Equal(t, f.dock, rollup.PreviousFacilities[0].LocationID)

	buckets := make(map[string]detentionports.RecoveryBucket, len(rollup.Recovery))
	for _, bucket := range rollup.Recovery {
		buckets[bucket.Bucket] = bucket
	}
	assert.Equal(t, "120", buckets[detentionports.RecoveryBilled].BillableAmount.String())
	assert.Equal(t, "0", buckets[detentionports.RecoveryBilled].CollectedAmount.String(),
		"a billed charge on no invoice is not collected")
	assert.Equal(t, "100", buckets[detentionports.RecoveryWaived].GrossAmount.String())
	assert.Equal(t, "80", buckets[detentionports.RecoveryLate].GrossAmount.String())
	assert.Equal(t, "60", buckets[detentionports.RecoveryNotice].GrossAmount.String(),
		"a charge held for notice is lost to notice whatever its status")

	assert.Equal(t, 1, rollup.Splits.Drop.StopCount)
	assert.Equal(t, 4, rollup.Splits.Live.StopCount)
	assert.Equal(t, 3, rollup.Splits.Pickup.StopCount)
	assert.Equal(t, 2, rollup.Splits.Open.StopCount)

	require.Len(t, rollup.Days, 1)
	assert.Equal(t, monday, rollup.Days[0].Day.Unix(), "the day is the organization's Monday")

	morning := 0
	for _, cell := range rollup.Arrivals {
		if cell.Weekday == 1 && cell.Hour == 8 {
			morning += cell.StopCount
		}
	}
	assert.Equal(t, 4, morning, "arrivals are counted in the organization's time zone")
}

func TestIntelligence_NarrowsEveryFigureToOneFacility(t *testing.T) {
	f, _ := settleTheWeek(t)

	scoped, err := f.repo.Intelligence(f.ctx, &detentionports.IntelligenceRequest{
		TenantInfo: f.tenant,
		Current:    days(monday, nextMonday),
		Previous:   days(weekBefore, monday),
		Timezone:   "America/Chicago",
		LocationID: f.other,
		Limit:      100,
	})
	require.NoError(t, err)

	assert.Equal(t, 2, scoped.Totals.StopCount)
	assert.Equal(t, int64(30), scoped.Totals.OpenMinutes)
	assert.Zero(t, scoped.Previous.StopCount)
	require.Len(t, scoped.Facilities, 1)
	assert.Equal(t, f.other, scoped.Facilities[0].LocationID)
	arrived := 0
	for _, cell := range scoped.Arrivals {
		arrived += cell.StopCount
	}
	assert.Equal(t, 2, arrived, "the facility's own arrivals")
	customers := 0
	for _, customer := range scoped.Customers {
		customers += customer.StopCount
	}
	assert.Equal(t, 2, customers, "the facility's own customers")
}

func TestIntelligence_TakesWholeMonthsFromTheMonthlyFiguresAndTheEdgesFromTheDays(t *testing.T) {
	f, _ := settleTheWeek(t)

	rollup, err := f.repo.Intelligence(f.ctx, &detentionports.IntelligenceRequest{
		TenantInfo: f.tenant,
		Current: detentionports.IntelligenceWindow{
			From:       december,
			To:         nextMonday - 1,
			MonthsFrom: december,
			MonthsEnd:  january,
		},
		Previous: days(december-7*86_400, december),
		Timezone: "America/Chicago",
		Limit:    100,
	})
	require.NoError(t, err)

	assert.Equal(t, 6, rollup.Totals.StopCount, "December's stop and the week of the 7th")
	require.Len(t, rollup.Facilities, 2)
	assert.Equal(t, f.dock, rollup.Facilities[0].LocationID)
	assert.Equal(t, 4, rollup.Facilities[0].StopCount,
		"New Year's Eve from December's month and three from January's days")
	assert.Equal(t, int64(320), rollup.Facilities[0].PastFreeMinutes)
}

func TestIntelligence_KeepsEachTenantsRollupsAndMarksToItself(t *testing.T) {
	f, db := settleTheWeek(t)
	system := dbscope.WithSystem(context.Background(), "test")

	_, err := keel.Exec(
		system,
		db,
		`INSERT INTO detention_daily (business_unit_id, organization_id, bucket) VALUES ($1, 'org_another', now())`,
		f.tenant.BuID.String(),
	)
	require.NoError(t, err)
	others, err := keel.QueryValue[int64](f.ctx, db,
		`SELECT count(*) FROM detention_daily WHERE organization_id = 'org_another'`)
	require.NoError(t, err)
	assert.Zero(t, others, "a tenant reads no other organization's rollup rows")

	_, err = keel.Exec(f.ctx, db,
		`UPDATE detention_daily SET row_count = row_count + 1 WHERE organization_id = $1`,
		f.tenant.OrgID.String())
	require.Error(t, err, "a tenant never writes its rollup rows; the refresh does")

	_, err = keel.Exec(f.ctx, db,
		`UPDATE detention_occurrences SET gross_amount = gross_amount WHERE location_id = $1`,
		f.dock.String())
	require.NoError(t, err)
	marked, err := keel.QueryValue[int64](
		system,
		db,
		`SELECT count(*) FROM keel_rollup_dirty WHERE rollup = 'detention_daily' AND bucket = to_timestamp($1)`,
		monday,
	)
	require.NoError(t, err)
	assert.Equal(t, int64(1), marked,
		"a tenant's write marks the organization's local day, through the marks' insert policy")
	seen, err := keel.QueryValue[int64](f.ctx, db, `SELECT count(*) FROM keel_rollup_dirty`)
	require.NoError(t, err)
	assert.Zero(t, seen, "a tenant reads no refresh marks, its own included")

	refresh(f.ctx, t, db)
	left, err := keel.QueryValue[int64](system, db, `SELECT count(*) FROM keel_rollup_dirty`)
	require.NoError(t, err)
	assert.Zero(t, left, "the refresh clears what it recomputed")
}
