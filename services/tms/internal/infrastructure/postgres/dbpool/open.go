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
	}
	if guard != nil {
		cfg.OnBegin = guard.OnBegin
	}

	return cfg
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
