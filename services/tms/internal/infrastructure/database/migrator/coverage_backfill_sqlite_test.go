package migrator_test

import (
	"context"
	"testing"

	sqlitemigrations "github.com/emoss08/trenova/internal/infrastructure/sqlite/migrations"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"

	_ "modernc.org/sqlite"
)

const coverageBackfillVersion = "20260916000000"

func TestSQLiteCoverageTypeBackfillSplitsOnLiveAssignment(t *testing.T) {
	ctx := t.Context()
	db := newSQLiteDB(t)

	migrator := migrateBefore(ctx, t, db, coverageBackfillVersion)

	const (
		orgID = "org_backfill"
		buID  = "bu_backfill"
	)

	var err error

	moves := []struct {
		id           string
		coverageType string
		want         string
	}{
		{id: "sm_live_driver", coverageType: "driver", want: "driver"},
		{id: "sm_archived_driver", coverageType: "driver", want: "unassigned"},
		{id: "sm_no_assignment", coverageType: "driver", want: "unassigned"},
		{id: "sm_carrier", coverageType: "carrier", want: "carrier"},
		{id: "sm_carrier_with_driver", coverageType: "carrier", want: "carrier"},
		{id: "sm_driver_label_live_carrier", coverageType: "driver", want: "carrier"},
		{id: "sm_driver_label_canceled_carrier", coverageType: "driver", want: "unassigned"},
		{id: "sm_live_driver_and_carrier", coverageType: "driver", want: "driver"},
	}

	for _, move := range moves {
		_, err = db.NewRaw(
			`INSERT INTO shipment_moves
                (id, organization_id, business_unit_id, shipment_id, status, loaded, sequence, coverage_type, version, created_at, updated_at)
             VALUES (?, ?, ?, ?, 'New', 1, 0, ?, 0, 0, 0)`,
			move.id, orgID, buID, "shp_backfill", move.coverageType,
		).Exec(ctx)
		require.NoError(t, err)
	}

	insertAssignment := func(id, moveID string, archivedAt *int64) {
		t.Helper()

		_, execErr := db.NewRaw(
			`INSERT INTO assignments
                (id, organization_id, business_unit_id, shipment_move_id, primary_worker_id, tractor_id, status, archived_at, version, created_at, updated_at)
             VALUES (?, ?, ?, ?, 'wrk_1', 'trc_1', ?, ?, 0, 0, 0)`,
			id, orgID, buID, moveID,
			assignmentStatusFor(archivedAt), archivedAt,
		).Exec(ctx)
		require.NoError(t, execErr)
	}

	insertCarrierAssignment := func(id, moveID, status string) {
		t.Helper()

		_, execErr := db.NewRaw(
			`INSERT INTO carrier_assignments
                (id, organization_id, business_unit_id, shipment_move_id, carrier_id, status, version, created_at, updated_at)
             VALUES (?, ?, ?, ?, 'car_1', ?, 0, 0, 0)`,
			id, orgID, buID, moveID, status,
		).Exec(ctx)
		require.NoError(t, execErr)
	}

	archivedAt := int64(1700000000)
	insertAssignment("asn_live", "sm_live_driver", nil)
	insertAssignment("asn_archived", "sm_archived_driver", &archivedAt)
	insertAssignment("asn_live_carrier", "sm_carrier_with_driver", nil)
	insertAssignment("asn_live_both", "sm_live_driver_and_carrier", nil)

	insertCarrierAssignment("ca_live", "sm_driver_label_live_carrier", "Confirmed")
	insertCarrierAssignment("ca_canceled", "sm_driver_label_canceled_carrier", "Canceled")
	insertCarrierAssignment("ca_live_with_driver", "sm_live_driver_and_carrier", "Pending")

	runMigrationUp(ctx, t, migrator, coverageBackfillVersion)

	for _, move := range moves {
		var got string
		require.NoError(t, db.NewRaw(
			`SELECT coverage_type FROM shipment_moves WHERE id = ?`, move.id,
		).Scan(ctx, &got))

		require.Equalf(t, move.want, got, "coverage type for move %q", move.id)
	}
}

func TestSQLiteCoverageTypeIndexTargetsCarrierCoverage(t *testing.T) {
	ctx := t.Context()
	db := newSQLiteDB(t)

	migrator := migrate.NewMigrator(db, sqlitemigrations.Migrations)
	require.NoError(t, migrator.Init(ctx))
	_, err := migrator.Migrate(ctx)
	require.NoError(t, err)

	var definition string
	require.NoError(t, db.NewRaw(
		`SELECT sql FROM sqlite_master WHERE type = 'index' AND name = 'idx_shipment_moves_coverage_type'`,
	).Scan(ctx, &definition))

	require.Contains(t, definition, "'carrier'")
	require.NotContains(t, definition, "<>")
}

func assignmentStatusFor(archivedAt *int64) string {
	if archivedAt == nil {
		return "New"
	}

	return "Canceled"
}

func migrateBefore(
	ctx context.Context,
	t *testing.T,
	db *bun.DB,
	version string,
) *migrate.Migrator {
	t.Helper()

	priors := migrate.NewMigrations()
	for _, candidate := range sqlitemigrations.Migrations.Sorted() {
		if candidate.Name >= version {
			break
		}

		priors.Add(candidate)
	}

	migrator := migrate.NewMigrator(db, priors)
	require.NoError(t, migrator.Init(ctx))

	_, err := migrator.Migrate(ctx)
	require.NoError(t, err)

	return migrator
}

func runMigrationUp(
	ctx context.Context,
	t *testing.T,
	migrator *migrate.Migrator,
	version string,
) {
	t.Helper()

	for _, candidate := range sqlitemigrations.Migrations.Sorted() {
		if candidate.Name != version {
			continue
		}

		require.NoError(t, candidate.Up(ctx, migrator, &candidate))
		return
	}

	t.Fatalf("migration %s not found in the SQLite migration set", version)
}
