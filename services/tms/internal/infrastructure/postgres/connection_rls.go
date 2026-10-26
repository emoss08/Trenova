package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbpool"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

const (
	rlsTenantRole        = "trenova_tenant"
	rlsOverrideRole      = "trenova_rls_bypass"
	maxDistinctScopeLogs = 10_000
	verifyRolesReason    = "verify row-level security roles at startup"
)

var (
	ErrRLSRoleCannotBypass = errors.New(
		"row-level security is enforced but the application database role can bypass it",
	)
	ErrRLSRoleNotTenant = errors.New(
		"row-level security is enforced but the application database role is not a member of " + rlsTenantRole,
	)
	ErrRLSSystemRoleCannotBypass = errors.New(
		"the system database role is not a member of " + rlsOverrideRole,
	)
	ErrRLSScopeKeyNotInstalled = errors.New(
		"the row-level security scope key is not installed in the database; run the migrations with database.rls configured",
	)
)

type rlsRuntime struct {
	mode     string
	signer   *scopeSigner
	reporter *scopeLogReporter
}

func rlsConfigured(cfg *config.Config) bool {
	return cfg != nil && cfg.Database.RLS.Enabled()
}

func newRLSRuntime(
	cfg *config.Config,
	logger *observability.ContextLogger,
	registry *metrics.Registry,
) (*rlsRuntime, error) {
	key, err := cfg.Database.RLS.DecodeScopeKey()
	if err != nil {
		return nil, err
	}

	signer, err := newScopeSigner(
		strings.TrimSpace(cfg.Database.RLS.ScopeKeyID),
		key,
		cfg.Database.RLS.GetScopeTTL(),
	)
	if err != nil {
		return nil, err
	}

	return &rlsRuntime{
		mode:     cfg.Database.RLS.GetMode(),
		signer:   signer,
		reporter: newScopeLogReporter(logger, registry),
	}, nil
}

func (r *rlsRuntime) enforced() bool {
	return r != nil && r.mode == config.RLSModeEnforce
}

func (r *rlsRuntime) guard(pool scopePool, systemFallback bool) *scopeGuard {
	return newScopeGuard(scopeGuardConfig{
		Pool:           pool,
		Signer:         r.signer,
		Enforce:        r.enforced(),
		SystemFallback: systemFallback,
		Reporter:       r.reporter,
	})
}

type scopeLogReporter struct {
	logger   *observability.ContextLogger
	metrics  *metrics.Registry
	mu       sync.Mutex
	seen     map[string]struct{}
	distinct atomic.Int64
}

func newScopeLogReporter(
	logger *observability.ContextLogger,
	registry *metrics.Registry,
) *scopeLogReporter {
	return &scopeLogReporter{
		logger:  logger,
		metrics: registry,
		seen:    make(map[string]struct{}),
	}
}

func (r *scopeLogReporter) Report(ctx context.Context, event *scopeEvent) {
	if r.metrics != nil && r.metrics.Database != nil {
		r.metrics.Database.RecordRLSScopeEvent(string(event.Pool), event.Event, event.Outcome)
	}

	if r.logger == nil ||
		(event.Event == scopeEventTenantTx && event.Outcome == scopeOutcomeAllowed) {
		return
	}

	fields := []zap.Field{
		zap.String("pool", string(event.Pool)),
		zap.String("event", event.Event),
		zap.String("outcome", event.Outcome),
		zap.String("caller", event.Caller),
	}
	if event.Reason != "" {
		fields = append(fields, zap.String("reason", event.Reason))
	}

	if event.Outcome == scopeOutcomeRefused {
		r.logger.Error(
			ctx,
			"Refused database access outside its row-level security scope",
			fields...)
		return
	}

	first := r.firstSighting(event)

	switch {
	case event.Outcome == scopeOutcomeObserved && first:
		r.logger.Warn(ctx, "Database access outside a row-level security scope", fields...)
	case event.Outcome == scopeOutcomeObserved:
		r.logger.Debug(ctx, "Database access outside a row-level security scope", fields...)
	case first:
		r.logger.Info(ctx, "System database access bypassing tenant isolation", fields...)
	default:
		r.logger.Debug(ctx, "System database access bypassing tenant isolation", fields...)
	}
}

func (r *scopeLogReporter) firstSighting(event *scopeEvent) bool {
	if r.distinct.Load() >= maxDistinctScopeLogs {
		return false
	}

	key := string(
		event.Pool,
	) + "|" + event.Event + "|" + event.Outcome + "|" + event.Reason + "|" + event.Caller
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, loaded := r.seen[key]; loaded {
		return false
	}
	r.seen[key] = struct{}{}

	r.distinct.Add(1)

	return true
}

func (c *Connection) connectSystem(ctx context.Context) error {
	if c.rls == nil || !c.cfg.Database.System.Configured() {
		return nil
	}

	role := c.cfg.Database.System
	system, err := c.openPool(ctx, &dbpool.Options{
		DSN:             c.cfg.GetDSNForRole(role),
		MaxConns:        role.GetMaxOpenConns(),
		MaxConnLifetime: c.settings.connMaxLifetime,
		MaxConnIdleTime: c.settings.connMaxIdleTime,
		RuntimeParams:   c.settings.connParams,
		Guard:           c.rls.guard(scopePoolSystem, false),
		SlowWaits:       c.slowWaits(scopePoolSystem),
	})
	if err != nil {
		return err
	}
	c.systemKeel = system

	if err = c.systemKeel.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping system database pool: %w", err)
	}

	return nil
}

type roleAttributes struct {
	Name         string
	Superuser    bool
	BypassRLS    bool
	BypassMember bool
	TenantMember bool
	OwnsTables   bool
}

const roleAttributesQuery = `
SELECT
	r.rolname AS name,
	r.rolsuper AS superuser,
	r.rolbypassrls AS bypass_rls,
	EXISTS (SELECT 1 FROM pg_roles b WHERE b.rolname = '` + rlsOverrideRole + `' AND pg_has_role(r.oid, b.oid, 'MEMBER')) AS bypass_member,
	EXISTS (SELECT 1 FROM pg_roles t WHERE t.rolname = '` + rlsTenantRole + `' AND pg_has_role(r.oid, t.oid, 'MEMBER')) AS tenant_member,
	EXISTS (
		SELECT 1
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public'
			AND c.relkind IN ('r', 'p')
			AND pg_has_role(r.oid, c.relowner, 'MEMBER')
	) AS owns_tables
FROM pg_roles r
WHERE r.rolname = current_user`

func readRoleAttributes(ctx context.Context, db *keel.DB) (roleAttributes, error) {
	var attrs roleAttributes
	err := keel.QueryRows(ctx, db, func(row pgx.CollectableRow) error {
		return row.Scan(
			&attrs.Name,
			&attrs.Superuser,
			&attrs.BypassRLS,
			&attrs.BypassMember,
			&attrs.TenantMember,
			&attrs.OwnsTables,
		)
	}, roleAttributesQuery)

	return attrs, err
}

func (c *Connection) verifyRLS(ctx context.Context) error {
	if c.rls == nil {
		return nil
	}

	probe := dbscope.WithTenant(ctx, dbscope.Tenant{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})

	var attrs roleAttributes
	err := keel.Read(probe, c.tenantKeel, func(ctx context.Context) error {
		var err error
		if attrs, err = readRoleAttributes(ctx, c.tenantKeel); err != nil {
			return fmt.Errorf("read application role attributes: %w", err)
		}

		if _, err = keel.QueryValue[string](
			ctx,
			c.tenantKeel,
			"SELECT trenova_rls.org_id()",
		); err != nil {
			return fmt.Errorf("%w: %w", ErrRLSScopeKeyNotInstalled, err)
		}

		return nil
	})
	if err != nil {
		return c.rlsProblem(ctx, err)
	}

	if attrs.Superuser || attrs.BypassRLS || attrs.BypassMember || attrs.OwnsTables {
		if err = c.rlsProblem(
			ctx,
			fmt.Errorf("%w (role %q)", ErrRLSRoleCannotBypass, attrs.Name),
		); err != nil {
			return err
		}
	}

	if !attrs.TenantMember {
		if err = c.rlsProblem(
			ctx,
			fmt.Errorf("%w (role %q)", ErrRLSRoleNotTenant, attrs.Name),
		); err != nil {
			return err
		}
	}

	return c.verifySystemRole(ctx)
}

func (c *Connection) verifySystemRole(ctx context.Context) error {
	if c.systemKeel == nil {
		return nil
	}

	attrs, err := readRoleAttributes(dbscope.WithSystem(ctx, verifyRolesReason), c.systemKeel)
	if err != nil {
		return c.rlsProblem(ctx, fmt.Errorf("read system role attributes: %w", err))
	}

	if !attrs.Superuser && !attrs.BypassMember {
		return c.rlsProblem(
			ctx,
			fmt.Errorf("%w (role %q)", ErrRLSSystemRoleCannotBypass, attrs.Name),
		)
	}

	return nil
}

func (c *Connection) rlsProblem(ctx context.Context, err error) error {
	if c.rls.enforced() {
		return err
	}

	c.logger.Warn(ctx, "Row-level security is not ready to be enforced", zap.Error(err))

	return nil
}
