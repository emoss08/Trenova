//go:build integration

package cloud_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantkeys"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type bareTenant struct {
	org string
	bu  string
}

type plantColumn struct {
	Name    string `db:"column_name"`
	Type    string `db:"column_type"`
	Primary bool   `db:"primary_key"`
}

const plantColumnsQuery = `SELECT a.attname::text AS column_name,
       format_type(a.atttypid, a.atttypmod) AS column_type,
       EXISTS (
           SELECT 1 FROM pg_catalog.pg_index i
           WHERE i.indrelid = a.attrelid AND i.indisprimary AND a.attnum = ANY(i.indkey)
       ) AS primary_key
FROM pg_catalog.pg_attribute a
WHERE a.attrelid = to_regclass($1) AND a.attnum > 0 AND NOT a.attisdropped AND a.attgenerated = ''
ORDER BY a.attnum`

const relaxQuery = `SELECT format('ALTER TABLE %I DISABLE TRIGGER USER', c.relname)
FROM pg_catalog.pg_class c
JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public' AND c.relkind IN ('r', 'p') AND NOT c.relispartition
UNION ALL
SELECT format('ALTER TABLE %I DROP CONSTRAINT %I', c.relname, con.conname)
FROM pg_catalog.pg_constraint con
JOIN pg_catalog.pg_class c ON c.oid = con.conrelid
JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public' AND con.contype IN ('c', 'x') AND NOT c.relispartition
  AND con.conparentid = 0
UNION ALL
SELECT format('ALTER TABLE %I ALTER COLUMN %I DROP NOT NULL', c.relname, a.attname)
FROM pg_catalog.pg_class c
JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
JOIN pg_catalog.pg_attribute a ON a.attrelid = c.oid
WHERE n.nspname = 'public' AND c.relkind IN ('r', 'p') AND NOT c.relispartition
  AND a.attnum > 0 AND NOT a.attisdropped AND a.attnotnull AND a.attidentity = ''
  AND NOT EXISTS (
      SELECT 1 FROM pg_catalog.pg_index i
      WHERE i.indrelid = c.oid AND i.indisprimary AND a.attnum = ANY(i.indkey)
  )`

const otherKeysQuery = `SELECT format('ALTER TABLE %I DROP CONSTRAINT %I', c.relname, con.conname)
FROM pg_catalog.pg_constraint con
JOIN pg_catalog.pg_class c ON c.oid = con.conrelid
WHERE con.contype = 'f' AND c.relname = ANY($1::text[]) AND con.conname <> $2 AND con.conparentid = 0`

func randomID(prefix string) string {
	b := make([]byte, 10)
	_, _ = rand.Read(b)
	return prefix + hex.EncodeToString(b)
}

func uuidValue() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b)
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
}

func valueFor(columnType string) (string, error) {
	switch {
	case strings.HasPrefix(columnType, "character varying"), columnType == "text",
		strings.HasPrefix(columnType, "character("):
		return randomID("tk"), nil
	case columnType == "uuid":
		return uuidValue(), nil
	case columnType == "bigint", columnType == "integer", columnType == "smallint",
		strings.HasPrefix(columnType, "numeric"):
		return "7", nil
	case columnType == "date":
		return "2026-01-01", nil
	case strings.HasPrefix(columnType, "timestamp"):
		return "2026-01-01 00:00:00+00", nil
	case columnType == "boolean":
		return "false", nil
	default:
		return "", fmt.Errorf("no test value for a key column of type %s", columnType)
	}
}

type planter struct {
	ctx     context.Context
	conn    *pgx.Conn
	columns map[string][]plantColumn
	home    bareTenant
	other   bareTenant
}

func execAll(ctx context.Context, conn *pgx.Conn, statements []string) error {
	for _, s := range statements {
		if _, err := conn.Exec(ctx, s); err != nil {
			return fmt.Errorf("%s: %w", s, err)
		}
	}
	return nil
}

func newPlanter(t *testing.T) *planter {
	t.Helper()
	_, _, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)
	ctx := t.Context()
	conn, err := pgx.Connect(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close(context.Background()) })

	rows, err := conn.Query(ctx, relaxQuery)
	require.NoError(t, err)
	relax, err := pgx.CollectRows(rows, pgx.RowTo[string])
	require.NoError(t, err)
	require.NoError(t, execAll(ctx, conn, relax))

	p := &planter{
		ctx:     ctx,
		conn:    conn,
		columns: make(map[string][]plantColumn),
		home:    bareTenant{org: randomID("org_tk"), bu: randomID("bu_tk")},
		other:   bareTenant{org: randomID("org_tk"), bu: randomID("bu_tk")},
	}
	for _, tenant := range []bareTenant{p.home, p.other} {
		_, err = conn.Exec(ctx, `INSERT INTO business_units (id) VALUES ($1)`, tenant.bu)
		require.NoError(t, err)
		_, err = conn.Exec(ctx, `INSERT INTO organizations (id, business_unit_id) VALUES ($1, $2)`,
			tenant.org, tenant.bu)
		require.NoError(t, err)
	}
	return p
}

func (p *planter) columnsOf(table string) ([]plantColumn, error) {
	if cols, ok := p.columns[table]; ok {
		return cols, nil
	}
	rows, err := p.conn.Query(p.ctx, plantColumnsQuery, `public.`+ident(table))
	if err != nil {
		return nil, err
	}
	cols, err := pgx.CollectRows(rows, pgx.RowToStructByName[plantColumn])
	if err != nil {
		return nil, err
	}
	p.columns[table] = cols
	return cols, nil
}

type plantRow map[string]string

func (p *planter) row(table string, tenant bareTenant, fixed plantRow) (plantRow, error) {
	cols, err := p.columnsOf(table)
	if err != nil {
		return nil, err
	}
	row := plantRow{}
	for _, c := range cols {
		switch {
		case tenantKindOf(c.Name) == "unit":
			row[c.Name] = tenant.bu
		case tenantKindOf(c.Name) == "organization":
			row[c.Name] = tenant.org
		case c.Primary:
			v, verr := valueFor(c.Type)
			if verr != nil {
				return nil, fmt.Errorf("%s.%s: %w", table, c.Name, verr)
			}
			row[c.Name] = v
		}
	}
	for k, v := range fixed {
		row[k] = v
	}
	return row, nil
}

func (p *planter) insert(table string, row plantRow) error {
	cols := make([]string, 0, len(row))
	for c := range row {
		cols = append(cols, c)
	}
	slices.Sort(cols)
	types := make(map[string]string)
	all, err := p.columnsOf(table)
	if err != nil {
		return err
	}
	for _, c := range all {
		types[c.Name] = c.Type
	}
	values := make([]string, 0, len(cols))
	args := make([]any, 0, len(cols))
	for i, c := range cols {
		values = append(values, fmt.Sprintf("$%d::%s", i+1, types[c]))
		args = append(args, row[c])
	}
	_, err = p.conn.Exec(p.ctx, "INSERT INTO "+ident(table)+" ("+strings.Join(quoted(cols), ", ")+
		") VALUES ("+strings.Join(values, ", ")+")", args...)
	return err
}

func (p *planter) parentRow(k *tenantkeys.Key, tenant bareTenant) (plantRow, error) {
	if k.FK.Parent == tenantkeys.Organizations {
		return plantRow{"id": tenant.org, tenantkeys.BusinessUnitColumn: tenant.bu}, nil
	}
	cols, err := p.columnsOf(k.FK.Parent)
	if err != nil {
		return nil, err
	}
	types := make(map[string]string, len(cols))
	for _, c := range cols {
		types[c.Name] = c.Type
	}
	fixed := plantRow{}
	for _, c := range k.FK.ParentColumns {
		if tenantKindOf(c) != "" {
			continue
		}
		v, verr := valueFor(types[c])
		if verr != nil {
			return nil, fmt.Errorf("%s.%s: %w", k.FK.Parent, c, verr)
		}
		fixed[c] = v
	}
	row, err := p.row(k.FK.Parent, tenant, fixed)
	if err != nil {
		return nil, err
	}
	return row, p.insert(k.FK.Parent, row)
}

func tenantKindOf(column string) string {
	switch {
	case column == tenantkeys.BusinessUnitColumn || strings.HasSuffix(column, "_business_unit_id"):
		return "unit"
	case column == tenantkeys.OrganizationColumn || strings.HasSuffix(column, "_organization_id"):
		return "organization"
	default:
		return ""
	}
}

func referencing(k *tenantkeys.Key, parent plantRow) plantRow {
	out := plantRow{}
	for i, c := range k.FK.ChildColumns {
		p := k.FK.ParentColumns[i]
		if kind := tenantKindOf(c); kind != "" && kind == tenantKindOf(p) {
			continue
		}
		out[c] = parent[p]
	}
	return out
}

func violationOf(err error) (string, string) {
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		return pg.Code, pg.ConstraintName
	}
	return "", ""
}

type plantOutcome struct {
	refused  error
	accepted error
}

func (p *planter) plant(k *tenantkeys.Key) (out plantOutcome, err error) {
	tx, err := p.conn.Begin(p.ctx)
	if err != nil {
		return out, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	rows, err := tx.Query(p.ctx, otherKeysQuery, []string{k.FK.Child, k.FK.Parent}, k.FK.Name)
	if err != nil {
		return out, err
	}
	drops, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return out, err
	}
	if err = execAll(p.ctx, p.conn, drops); err != nil {
		return out, err
	}
	home, err := p.parentRow(k, p.home)
	if err != nil {
		return out, fmt.Errorf("make the home tenant's %s row: %w", k.FK.Parent, err)
	}
	other, err := p.parentRow(k, p.other)
	if err != nil {
		return out, fmt.Errorf("make the other tenant's %s row: %w", k.FK.Parent, err)
	}

	attempt := func(parent plantRow) error {
		if _, serr := p.conn.Exec(p.ctx, "SAVEPOINT plant"); serr != nil {
			return serr
		}
		var aerr error
		if k.FK.Child == k.FK.Parent {
			sets := make([]string, 0)
			args := make([]any, 0)
			for c, v := range referencing(k, parent) {
				args = append(args, v)
				sets = append(sets, fmt.Sprintf("%s = $%d", ident(c), len(args)))
			}
			args = append(args, home["id"])
			_, aerr = p.conn.Exec(
				p.ctx,
				"UPDATE "+ident(k.FK.Child)+" SET "+strings.Join(sets, ", ")+
					fmt.Sprintf(" WHERE id = $%d", len(args)),
				args...)
		} else {
			row, rerr := p.row(k.FK.Child, p.home, referencing(k, parent))
			if rerr != nil {
				return rerr
			}
			aerr = p.insert(k.FK.Child, row)
		}
		if _, serr := p.conn.Exec(p.ctx, "ROLLBACK TO SAVEPOINT plant"); serr != nil {
			return errors.Join(aerr, serr)
		}
		return aerr
	}
	out.refused = attempt(other)
	out.accepted = attempt(home)
	return out, nil
}

func TestEveryKeyBetweenTenantTablesRefusesAnotherTenantsRecord(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	cat, err := tenantkeys.Load(ctx, db)
	require.NoError(t, err)
	inv := tenantkeys.Classify(cat)

	held := inv.Of(tenantkeys.ClassHeld)
	require.NotEmpty(t, held)
	for _, k := range held {
		assert.True(t, k.FK.Validated, "%s is validated", k.FK.Name)
	}

	p := newPlanter(t)
	refused := 0
	for _, k := range held {
		out, perr := p.plant(k)
		if !assert.NoError(t, perr, k.FK.Name) {
			continue
		}
		code, constraint := violationOf(out.refused)
		if assert.Equal(t, pgerrcode.ForeignKeyViolation, code,
			"%s accepted a %s row of another tenant: %v", k.FK.Name, k.FK.Parent, out.refused) {
			assert.Equal(t, k.FK.Name, constraint, k.FK.Name)
			refused++
		}
		assert.NoError(
			t,
			out.accepted,
			"%s refused a %s row of the row's own tenant",
			k.FK.Name,
			k.FK.Parent,
		)
	}
	t.Logf(
		"%d of %d keys between tenant tables refused another tenant's record and accepted their own",
		refused,
		len(held),
	)
}
