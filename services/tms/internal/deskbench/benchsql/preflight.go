package benchsql

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/deskbench"
	"github.com/emoss08/trenova/internal/deskbench/benchresult"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
)

type dataQuery struct {
	sql  string
	args []any
}

var (
	fixtureShipments = dataQuery{
		sql:  "SELECT count(*) FROM shipments WHERE pro_number LIKE $1",
		args: []any{"SEED-%"},
	}
	volumeDrivers = dataQuery{
		sql:  "SELECT count(*) FROM workers WHERE email LIKE $1",
		args: []any{"%@drivers.example.com"},
	}
	benchWrites = dataQuery{
		sql: `SELECT count(*) FROM agent_proposals p
			JOIN agent_runs r ON r.id = p.run_id
			JOIN assistant_threads t ON t.id = r.subject_id
			WHERE t.title LIKE $1 AND p.status = 'Executed'`,
		args: []any{deskbench.BenchTitlePrefix + "%"},
	}
)

func dataQueries() []dataQuery {
	return []dataQuery{fixtureShipments, volumeDrivers, benchWrites}
}

func (d *Database) ProfileData(
	ctx context.Context,
	session *deskbench.Session,
) (benchresult.DataProfile, error) {
	counts, err := dbtx.Read(
		session.Context(ctx),
		d.db,
		func(ctx context.Context) ([]int64, error) {
			db := d.db.Keel(ctx)
			queries := dataQueries()
			counts := make([]int64, 0, len(queries))
			for idx := range queries {
				count, err := keel.QueryValue[int64](
					ctx,
					db,
					queries[idx].sql,
					queries[idx].args...)
				if err != nil {
					return nil, fmt.Errorf("profile the bench's data: %w", err)
				}
				counts = append(counts, count)
			}

			return counts, nil
		},
	)
	if err != nil {
		return benchresult.DataProfile{}, err
	}

	return benchresult.DataProfileOf(counts[0], counts[1], counts[2])
}
