package volumeseed

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

const backdateChunk = 5_000

type stamp struct {
	id      pulid.ID
	created int64
	updated int64
}

func idAt(prefix string, unix int64) pulid.ID {
	return pulid.MustNewAt(prefix, time.Unix(unix, 0))
}

func insertBatches[T any](ctx context.Context, idb bun.IDB, rows []T, size int) error {
	for begin := 0; begin < len(rows); begin += size {
		chunk := rows[begin:min(begin+size, len(rows))]
		if _, err := idb.NewInsert().Model(&chunk).Exec(ctx); err != nil {
			return err
		}
	}

	return nil
}

// backdate restores creation and update times after insert, because every
// model's BeforeAppendModel hook stamps the current time on insert.
func backdate(
	ctx context.Context,
	idb bun.IDB,
	table string,
	orgID pulid.ID,
	stamps []stamp,
) error {
	if len(stamps) == 0 {
		return nil
	}
	if err := analyze(ctx, idb, table); err != nil {
		return err
	}
	for begin := 0; begin < len(stamps); begin += backdateChunk {
		chunk := stamps[begin:min(begin+backdateChunk, len(stamps))]
		ids := make([]string, len(chunk))
		created := make([]int64, len(chunk))
		updated := make([]int64, len(chunk))
		for i, s := range chunk {
			ids[i] = s.id.String()
			created[i] = s.created
			updated[i] = max(s.updated, s.created)
		}
		if _, err := idb.NewRaw(`
			UPDATE ? AS t
			SET created_at = d.created_at, updated_at = d.updated_at
			FROM unnest(?::varchar[], ?::bigint[], ?::bigint[]) AS d(id, created_at, updated_at)
			WHERE t.id = d.id AND t.organization_id = ?`,
			bun.Ident(table),
			pgdialect.Array(ids),
			pgdialect.Array(created),
			pgdialect.Array(updated),
			orgID,
		).Exec(ctx); err != nil {
			return fmt.Errorf("backdate %s: %w", table, err)
		}
	}

	return nil
}

// analyze refreshes a table's planner statistics before an update keyed by id.
// Rows written earlier in the same transaction are otherwise invisible to the
// planner, which then rescans the whole table once per id.
func analyze(ctx context.Context, idb bun.IDB, table string) error {
	if _, err := idb.NewRaw("ANALYZE ?", bun.Ident(table)).Exec(ctx); err != nil {
		return fmt.Errorf("analyze %s: %w", table, err)
	}
	return nil
}

func (g *generator) inTx(ctx context.Context, fn func(context.Context, bun.Tx) error) error {
	return g.db.RunInTx(ctx, nil, fn)
}

func (g *generator) progress(entity string, done, total int) {
	g.opts.Logf("  • %s: %d/%d", entity, done, total)
}

func unix(t time.Time) int64 {
	return t.Unix()
}

func ptr[T any](v T) *T {
	return &v
}
