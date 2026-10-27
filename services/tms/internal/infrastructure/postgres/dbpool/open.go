package dbpool

import (
	"context"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/keel"
	"github.com/emoss08/keel/numeric"
	"github.com/emoss08/keel/postgis"
	"github.com/emoss08/keel/schema"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	OrganizationColumn = "organization_id"
	BusinessUnitColumn = "business_unit_id"
)

type Options struct {
	DSN               string
	MaxConns          int
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	RuntimeParams     map[string]string
	Guard             Guard
	Observers         []keel.Observer
	ObserveArgs       bool
	Tables            []*schema.Table
	VersionErrors     func(error)
	TenantConcurrency int
	Cache             *keel.CacheConfig
	SlowWaits         keel.SlowWaitConfig
	MaxWait           time.Duration
}

type Guard interface {
	OnBegin(ctx context.Context, tx keel.TxInfo) ([]keel.Statement, error)
}

func Open(ctx context.Context, opts *Options) (*keel.DB, error) {
	pc, err := pgxpool.ParseConfig(opts.DSN)
	if err != nil {
		return nil, fmt.Errorf("parse database dsn: %w", err)
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
	cfg.ObserveArgs = opts.ObserveArgs
	cfg.Tables = opts.Tables
	cfg.DataVersions = keel.DataVersionsConfig{
		Enabled:  true,
		Interval: dataVersionInterval,
		OnError:  opts.VersionErrors,
	}
	cfg.TenantConcurrency = opts.TenantConcurrency
	cfg.Cache = opts.Cache
	cfg.SlowWaits = opts.SlowWaits
	cfg.MaxWait = opts.MaxWait

	db, err := keel.Open(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open database pool: %w", err)
	}

	return db, nil
}

const (
	maxConns            = 1 << 15
	dataVersionInterval = time.Second
)

func KeelConfig(guard Guard, observers []keel.Observer) keel.Config {
	cfg := keel.Config{
		TenantColumns: []string{OrganizationColumn, BusinessUnitColumn},
		Tenant:        tenantOf,
		System:        systemOf,
		NewID:         newID,
		Now:           timeutils.Now,
		JSON: &keel.JSONCodec{
			Marshal:         sonic.Marshal,
			Unmarshal:       sonic.Unmarshal,
			UnmarshalNumber: sonic.Config{UseNumber: true}.Froze().Unmarshal,
		},
		Observers:        observers,
		AfterConnect:     registerTypes,
		ReadIsolation:    keel.RepeatableRead,
		DeadlineTimeouts: true,
	}
	if guard != nil {
		cfg.OnBegin = guard.OnBegin
	}

	return cfg
}

func registerTypes(ctx context.Context, conn *pgx.Conn) error {
	if err := postgis.Register(ctx, conn); err != nil {
		return err
	}

	return numeric.Register(ctx, conn)
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
