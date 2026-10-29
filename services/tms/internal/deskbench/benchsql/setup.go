package benchsql

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/deskbench"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/jackc/pgx/v5"
)

// Leftovers cancels the shipments earlier bench conversations created and
// left New, before a run starts. A load a scenario booked and never ran made
// "this one" and "the acme load" ambiguous for every run after it. Only a load a
// bench thread's create or duplicate made is touched: an edit's result names
// the seed load it changed, and reading every executed proposal canceled
// SEED-SHP-001 after a re-rate. Loads people entered are never touched.
//
// It also retires what earlier bench conversations taught the agents, and
// dismisses what they suggested. A rejected hold taught "this person does not
// want a hold for missing paperwork", and the case that asks for that hold
// failed on every run after. Memories from people's own conversations are
// never touched.
func Leftovers() []string {
	return []string{
		`CREATE TEMP TABLE bench_leftovers ON COMMIT DROP AS
		SELECT s.id FROM shipments s
		WHERE s.status = 'New' AND s.pro_number NOT LIKE 'SEED-%' AND s.id IN (
			SELECT p.execution_result->'record'->>'id'
			FROM agent_proposals p
			JOIN agent_runs r ON r.id = p.run_id
			JOIN assistant_threads t ON t.id = r.subject_id
			WHERE t.title LIKE '[bench]%'
				AND p.status = 'Executed'
				AND p.tool_name IN ('shipment_create', 'shipment_duplicate')
				AND p.execution_result->'record'->>'entityType' = 'shipment')`,
		`UPDATE assignments a SET status = 'Canceled' FROM shipment_moves sm
		WHERE a.shipment_move_id = sm.id AND sm.shipment_id IN (SELECT id FROM bench_leftovers)`,
		`UPDATE shipment_moves SET status = 'Canceled'
		WHERE shipment_id IN (SELECT id FROM bench_leftovers)`,
		`UPDATE shipments SET status = 'Canceled' WHERE id IN (SELECT id FROM bench_leftovers)`,
		`CREATE TEMP TABLE bench_lessons ON COMMIT DROP AS
		SELECT m.id FROM agent_memories m
		WHERE m.status IN ('Active', 'Paused', 'Suggested') AND (
			m.reflection_id IN (
				SELECT rf.id FROM agent_reflections rf
				LEFT JOIN agent_runs r ON r.id = rf.run_id
				JOIN assistant_threads t ON t.id = COALESCE(rf.thread_id, r.subject_id)
				WHERE t.title LIKE '[bench]%')
			OR m.source_run_id IN (
				SELECT r.id FROM agent_runs r
				JOIN assistant_threads t ON t.id = r.subject_id
				WHERE t.title LIKE '[bench]%'))`,
		`UPDATE agent_memories SET
			status = CASE WHEN status = 'Suggested' THEN 'Dismissed' ELSE 'Retired' END,
			retired_at = CASE WHEN status = 'Suggested' THEN retired_at
				ELSE ` + nowEpoch + ` END,
			version = version + 1,
			updated_at = ` + nowEpoch + `
		WHERE id IN (SELECT id FROM bench_lessons)`,
	}
}

const nowEpoch = "extract(epoch from current_timestamp)::bigint"

func (d *Database) RunSetup(
	ctx context.Context,
	session *deskbench.Session,
	statements []string,
) error {
	if len(statements) == 0 {
		return nil
	}

	return d.RunStatements(ctx, session, statements)
}

func (d *Database) RunStatements(
	ctx context.Context,
	session *deskbench.Session,
	statements []string,
) error {
	ctx = session.Context(ctx)

	return dbtx.WriteErr(ctx, d.db, func(ctx context.Context) error {
		db := d.db.Keel(ctx)
		if _, err := keel.Exec(ctx, db, boundStatementTimeoutSQL); err != nil {
			return fmt.Errorf("bound the setup: %w", err)
		}
		confined, err := readConfinement(ctx, db)
		if err != nil {
			return err
		}
		if err = confined.err(); err != nil {
			return err
		}
		for idx, statement := range statements {
			if _, err = keel.Exec(ctx, db, statement); err != nil {
				return fmt.Errorf("setup %d: %w", idx+1, err)
			}
		}

		return nil
	})
}

var ErrSetupUnconfined = errors.New("setup refused: the bench's database role is not confined to " +
	"the bench user's organization")

type confinement struct {
	Bypass   bool   `db:"bypass"`
	Security string `db:"security"`
}

const confinementSQL = `SELECT r.rolsuper OR r.rolbypassrls AS bypass,
	current_setting('row_security') AS security
	FROM pg_roles r WHERE r.rolname = current_user`

func readConfinement(ctx context.Context, db *keel.DB) (confinement, error) {
	rows, err := keel.QueryAll(ctx, db, pgx.RowToStructByName[confinement], confinementSQL)
	if err != nil {
		return confinement{}, fmt.Errorf("read the role's row-level security: %w", err)
	}
	if len(rows) == 0 {
		return confinement{}, errors.New(
			"read the role's row-level security: the role has no pg_roles row",
		)
	}

	return rows[0], nil
}

func (c confinement) err() error {
	switch {
	case c.Bypass:
		return fmt.Errorf("%w: it is a superuser or bypasses row-level security",
			ErrSetupUnconfined)
	case c.Security != "on":
		return fmt.Errorf("%w: row_security is %q", ErrSetupUnconfined, c.Security)
	default:
		return nil
	}
}
