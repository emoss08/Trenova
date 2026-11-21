package tenantcopyrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/cloud/archiveplan"
	"github.com/jackc/pgx/v5"
)

const (
	columnsQuery = `SELECT c.relname::text AS table_name,
       c.relkind = 'p' AS partitioned,
       a.attname::text AS column_name,
       format_type(a.atttypid, a.atttypmod) AS column_type,
       NOT a.attnotnull AS nullable,
       a.attgenerated <> '' AS generated,
       a.attidentity <> '' AS identity,
       COALESCE(pg_get_serial_sequence(format('%I.%I', n.nspname, c.relname), a.attname), '') AS serial_sequence
FROM pg_catalog.pg_class c
JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
JOIN pg_catalog.pg_attribute a ON a.attrelid = c.oid
WHERE n.nspname = 'public'
  AND c.relkind IN ('r', 'p')
  AND NOT c.relispartition
  AND a.attnum > 0
  AND NOT a.attisdropped
ORDER BY c.relname, a.attnum`

	primaryKeysQuery = `SELECT c.relname::text AS table_name,
       ARRAY(
           SELECT a.attname::text
           FROM unnest(i.indkey::int2[]) WITH ORDINALITY AS k(attnum, ord)
           JOIN pg_catalog.pg_attribute a ON a.attrelid = c.oid AND a.attnum = k.attnum
           ORDER BY k.ord
       ) AS key_columns
FROM pg_catalog.pg_index i
JOIN pg_catalog.pg_class c ON c.oid = i.indrelid
JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
WHERE i.indisprimary
  AND n.nspname = 'public'
  AND NOT c.relispartition`

	foreignKeysQuery = `SELECT con.conname::text AS constraint_name,
       child.relname::text AS child_table,
       parent.relname::text AS parent_table,
       ARRAY(
           SELECT att.attname::text
           FROM unnest(con.conkey) WITH ORDINALITY AS k(attnum, ord)
           JOIN pg_catalog.pg_attribute att ON att.attrelid = con.conrelid AND att.attnum = k.attnum
           ORDER BY k.ord
       ) AS child_columns,
       ARRAY(
           SELECT att.attname::text
           FROM unnest(con.confkey) WITH ORDINALITY AS k(attnum, ord)
           JOIN pg_catalog.pg_attribute att ON att.attrelid = con.confrelid AND att.attnum = k.attnum
           ORDER BY k.ord
       ) AS parent_columns
FROM pg_catalog.pg_constraint con
JOIN pg_catalog.pg_class child ON child.oid = con.conrelid
JOIN pg_catalog.pg_class parent ON parent.oid = con.confrelid
JOIN pg_catalog.pg_namespace n ON n.oid = child.relnamespace
WHERE con.contype = 'f'
  AND n.nspname = 'public'
  AND NOT child.relispartition
ORDER BY con.conname`

	rollupsQuery = `SELECT rollup::text FROM public.keel_rollup_state ORDER BY 1`
	globalsQuery = `SELECT table_name::text FROM trenova_rls.global_tables ORDER BY 1`
)

type runner func(ctx context.Context, sql string, fn func(pgx.CollectableRow) error) error

func txRunner(tx pgx.Tx) runner {
	return func(ctx context.Context, sql string, fn func(pgx.CollectableRow) error) error {
		rows, err := tx.Query(ctx, sql)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			if err = fn(rows); err != nil {
				return err
			}
		}
		return rows.Err()
	}
}

func keelRunner(db *keel.DB) runner {
	return func(ctx context.Context, sql string, fn func(pgx.CollectableRow) error) error {
		return keel.QueryRows(ctx, db, fn, sql)
	}
}

type columnRow struct {
	Table       string `db:"table_name"`
	Partitioned bool   `db:"partitioned"`
	archiveplan.Column
}

type keyRow struct {
	Table   string   `db:"table_name"`
	Columns []string `db:"key_columns"`
}

func readCatalog(ctx context.Context, q runner) (*archiveplan.Catalog, error) {
	cat := &archiveplan.Catalog{
		Tables:  make(map[string]*archiveplan.Table),
		Globals: make(map[string]string),
	}
	columns, err := collect(ctx, q, columnsQuery, pgx.RowToStructByName[columnRow])
	if err != nil {
		return nil, fmt.Errorf("read the tables' columns: %w", err)
	}
	for _, row := range columns {
		t := cat.Tables[row.Table]
		if t == nil {
			t = &archiveplan.Table{Name: row.Table, Partitions: row.Partitioned}
			cat.Tables[row.Table] = t
		}
		t.Columns = append(t.Columns, row.Column)
	}
	keys, err := collect(ctx, q, primaryKeysQuery, pgx.RowToStructByName[keyRow])
	if err != nil {
		return nil, fmt.Errorf("read the primary keys: %w", err)
	}
	for _, row := range keys {
		if t, ok := cat.Tables[row.Table]; ok {
			t.PrimaryKey = row.Columns
		}
	}
	if cat.ForeignKeys, err = collect(
		ctx,
		q,
		foreignKeysQuery,
		pgx.RowToAddrOfStructByName[archiveplan.ForeignKey],
	); err != nil {
		return nil, fmt.Errorf("read the foreign keys: %w", err)
	}
	if cat.Rollups, err = collect(ctx, q, rollupsQuery, pgx.RowTo[string]); err != nil {
		return nil, fmt.Errorf("read the rollups: %w", err)
	}
	globals, err := collect(ctx, q, globalsQuery, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("read the global tables: %w", err)
	}
	for _, name := range globals {
		cat.Globals[name] = name
	}
	return cat, nil
}

func collect[T any](
	ctx context.Context,
	run runner,
	sql string,
	to pgx.RowToFunc[T],
) ([]T, error) {
	out := make([]T, 0)
	err := run(ctx, sql, func(row pgx.CollectableRow) error {
		value, err := to(row)
		if err != nil {
			return err
		}
		out = append(out, value)
		return nil
	})
	return out, err
}
