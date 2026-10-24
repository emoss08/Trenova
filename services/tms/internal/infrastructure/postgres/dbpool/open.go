package dbpool

import (
	"context"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

const (
	OrganizationColumn = "organization_id"
	BusinessUnitColumn = "business_unit_id"
)

type Options struct {
	DSN             string
	MaxConns        int
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	RuntimeParams   map[string]string
	Guard           Guard
	Observers       []keel.Observer
}

type Pool struct {
	Bun  *bun.DB
	Keel *keel.DB
}

func Open(ctx context.Context, opts *Options) (Pool, error) {
	pc, err := pgxpool.ParseConfig(opts.DSN)
	if err != nil {
		return Pool{}, fmt.Errorf("parse database dsn: %w", err)
	}

	if opts.MaxConns > 0 {
		pc.MaxConns = int32(min(opts.MaxConns, maxConns)) //nolint:gosec // bounded by maxConns
	}
	if opts.MaxConnLifetime > 0 {
		pc.MaxConnLifetime = opts.MaxConnLifetime
	}
	if opts.MaxConnIdleTime > 0 {
		pc.MaxConnIdleTime = opts.MaxConnIdleTime
	}
	for name, value := range opts.RuntimeParams {
		pc.ConnConfig.RuntimeParams[name] = value
	}

	cfg := KeelConfig(opts.Guard, opts.Observers)
	cfg.Pool = pc

	db, err := keel.Open(ctx, cfg)
	if err != nil {
		return Pool{}, fmt.Errorf("open database pool: %w", err)
	}

	return Pool{
		Bun:  bun.NewDB(openSQL(db, opts.Guard), pgdialect.New()),
		Keel: db,
	}, nil
}

const maxConns = 1 << 15

func KeelConfig(guard Guard, observers []keel.Observer) keel.Config {
	cfg := keel.Config{
		TenantColumns: []string{OrganizationColumn, BusinessUnitColumn},
		Tenant:        tenantOf,
		System:        systemOf,
		NewID:         newID,
		Now:           timeutils.Now,
		JSON:          &keel.JSONCodec{Marshal: sonic.Marshal, Unmarshal: sonic.Unmarshal},
		Observers:     observers,
		SkipEnums:     true,
		AfterConnect:  registerEnums,
	}
	if guard != nil {
		cfg.OnBegin = guard.OnBegin
	}

	return cfg
}

const enumTypesSQL = `SELECT t.oid, t.typname, a.oid, a.typname
	FROM pg_type t
	JOIN pg_namespace n ON n.oid = t.typnamespace
	JOIN pg_type a ON a.oid = t.typarray
	WHERE t.typtype = 'e'
		AND n.nspname NOT IN ('pg_catalog', 'information_schema')`

func registerEnums(ctx context.Context, conn *pgx.Conn) error {
	rows, err := conn.Query(ctx, enumTypesSQL, pgx.QueryExecModeSimpleProtocol)
	if err != nil {
		return fmt.Errorf("list enum types: %w", err)
	}
	defer rows.Close()

	types := conn.TypeMap()
	for rows.Next() {
		var oid, arrayOID uint32
		var name, arrayName string
		if err = rows.Scan(&oid, &name, &arrayOID, &arrayName); err != nil {
			return fmt.Errorf("read enum type: %w", err)
		}

		enum := &pgtype.Type{Name: name, OID: oid, Codec: &pgtype.EnumCodec{}}
		types.RegisterType(enum)
		types.RegisterType(&pgtype.Type{
			Name:  arrayName,
			OID:   arrayOID,
			Codec: &pgtype.ArrayCodec{ElementType: enum},
		})
	}

	return rows.Err()
}

func tenantOf(ctx context.Context) ([]any, bool) {
	tenant, ok := dbscope.TenantFrom(ctx)
	if !ok || !tenant.Valid() {
		return nil, false
	}

	return []any{tenant.OrganizationID.String(), tenant.BusinessUnitID.String()}, true
}

func systemOf(ctx context.Context) (string, bool) {
	scope := dbscope.From(ctx)
	if scope.Kind() != dbscope.KindSystem {
		return "", false
	}

	return scope.Reason(), true
}

func newID(prefix string) string {
	return pulid.MustNew(prefix).String()
}
