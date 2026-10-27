package deskbench

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
)

const (
	DataFixture = "fixture seed"
	DataVolume  = "fixture and volume seeds"
)

var ErrNoFixtureData = errors.New(
	"the database holds no fixture-seed shipment (SEED-…) in the bench user's organization, " +
		"and the scenarios name those records",
)

type DataProfile struct {
	Shape string `json:"shape"`
	Drift string `json:"drift,omitempty"`
}

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
		args: []any{benchTitlePrefix + "%"},
	}
)

func dataQueries() []dataQuery {
	return []dataQuery{fixtureShipments, volumeDrivers, benchWrites}
}

func (b *Bench) ProfileData(ctx context.Context, session *Session) (DataProfile, error) {
	counts, err := dbtx.Read(
		session.Context(ctx),
		b.DB,
		func(ctx context.Context) ([]int64, error) {
			db := b.DB.Keel(ctx)
			queries := dataQueries()
			counts := make([]int64, 0, len(queries))
			for idx := range queries {
				count, err := keel.QueryValue[int64](ctx, db, queries[idx].sql, queries[idx].args...)
				if err != nil {
					return nil, fmt.Errorf("profile the bench's data: %w", err)
				}
				counts = append(counts, count)
			}

			return counts, nil
		},
	)
	if err != nil {
		return DataProfile{}, err
	}

	return dataProfileOf(counts[0], counts[1], counts[2])
}

func dataProfileOf(fixture, volume, writes int64) (DataProfile, error) {
	if fixture == 0 {
		return DataProfile{}, ErrNoFixtureData
	}
	profile := DataProfile{Shape: DataFixture}
	if volume > 0 {
		profile.Shape = DataVolume
	}
	if writes > 0 {
		profile.Drift = strconv.FormatInt(writes, 10) + " changes earlier bench runs approved " +
			"are still in the data"
	}

	return profile, nil
}

func (p DataProfile) String() string {
	if p.Drift == "" {
		return p.Shape
	}

	return p.Shape + "; " + p.Drift
}

func ComparedDataNote(before, after DataProfile) string {
	if before.Shape == "" || before.Shape == after.Shape {
		return ""
	}

	return "the earlier run ran on the " + before.Shape + " and this one on the " +
		after.Shape + ", so a moved pass rate may be the data rather than the change"
}
