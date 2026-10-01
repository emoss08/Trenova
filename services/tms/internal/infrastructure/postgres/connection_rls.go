package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"go.uber.org/zap"
)

const (
	tenantRoleName       = "trenova_tenant"
	bypassRoleName       = "trenova_rls_bypass"
	maxDistinctScopeLogs = 10_000
	verifyRolesReason    = "verify row-level security roles at startup"
)

var (
	ErrRLSRoleCannotBypass = errors.New(
		"row-level security is enforced but the application database role can bypass it",
	)
	ErrRLSRoleNotTenant = errors.New(
		"row-level security is enforced but the application database role is not a member of " + tenantRoleName,
	)
	ErrRLSSystemRoleCannotBypass = errors.New(
		"the system database role is not a member of " + bypassRoleName,
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

func newRLSRuntime(
	cfg *config.Config,
	logger *observability.ContextLogger,
	registry *metrics.Registry,
) (*rlsRuntime, error) {
	if cfg == nil || !cfg.Database.GetDialect().IsPostgres() || !cfg.Database.RLS.Enabled() {
		return nil, nil
	}

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

func (r *rlsRuntime) connector(
	dsn string,
	params map[string]any,
	pool scopePool,
	systemFallback bool,
) *scopedConnector {
	return newScopedConnector(scopedConnectorConfig{
		Base: pgdriver.NewConnector(
			pgdriver.WithDSN(dsn),
			pgdriver.WithConnParams(params),
		),
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
	seen     sync.Map
	distinct atomic.Int64
}

func newScopeLogReporter(
	logger *observability.ContextLogger,
	registry *metrics.Registry,
) *scopeLogReporter {
	return &scopeLogReporter{logger: logger, metrics: registry}
}

func (r *scopeLogReporter) Report(ctx context.Context, event scopeEvent) {
	if r.metrics != nil && r.metrics.Database != nil {
		r.metrics.Database.RecordRLSScopeEvent(string(event.Pool), event.Event, event.Outcome)
	}

	if r.logger == nil || (event.Event == scopeEventTenantTx && event.Outcome == scopeOutcomeAllowed) {
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
		r.logger.Error(ctx, "Refused database access outside its row-level security scope", fields...)
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

func (r *scopeLogReporter) firstSighting(event scopeEvent) bool {
	if r.distinct.Load() >= maxDistinctScopeLogs {
		return false
	}

	key := string(event.Pool) + "|" + event.Event + "|" + event.Outcome + "|" + event.Reason + "|" + event.Caller
	if _, loaded := r.seen.LoadOrStore(key, struct{}{}); loaded {
		return false
	}

	r.distinct.Add(1)

	return true
}

func (c *Connection) openSystemDB() (*sql.DB, error) {
	if c.rls == nil || !c.cfg.Database.System.Configured() {
		return nil, nil
	}

	role := c.cfg.Database.System
	sqldb := sql.OpenDB(c.rls.connector(
		c.cfg.GetDSNForRole(role),
		c.settings.connParams,
		scopePoolSystem,
		false,
	))
	sqldb.SetMaxOpenConns(role.GetMaxOpenConns())
	sqldb.SetMaxIdleConns(role.GetMaxIdleConns())
	if c.settings.connMaxLifetime > 0 {
		sqldb.SetConnMaxLifetime(c.settings.connMaxLifetime)
	}
	if c.settings.connMaxIdleTime > 0 {
		sqldb.SetConnMaxIdleTime(c.settings.connMaxIdleTime)
	}

	return sqldb, nil
}

func (c *Connection) connectSystem(ctx context.Context) error {
	sqldb, err := c.openSystemDB()
	if err != nil || sqldb == nil {
		return err
	}

	c.system = c.withHooks(bun.NewDB(sqldb, pgdialect.New()))
	registerModels(c.system)
	if err = c.system.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping system database pool: %w", err)
	}

	return nil
}

type roleAttributes struct {
	Name         string `bun:"name"`
	Superuser    bool   `bun:"superuser"`
	BypassRLS    bool   `bun:"bypass_rls"`
	BypassMember bool   `bun:"bypass_member"`
	TenantMember bool   `bun:"tenant_member"`
	OwnsTables   bool   `bun:"owns_tables"`
}

const roleAttributesQuery = `
SELECT
	r.rolname AS name,
	r.rolsuper AS superuser,
	r.rolbypassrls AS bypass_rls,
	EXISTS (SELECT 1 FROM pg_roles b WHERE b.rolname = '` + bypassRoleName + `' AND pg_has_role(r.oid, b.oid, 'MEMBER')) AS bypass_member,
	EXISTS (SELECT 1 FROM pg_roles t WHERE t.rolname = '` + tenantRoleName + `' AND pg_has_role(r.oid, t.oid, 'MEMBER')) AS tenant_member,
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

func (c *Connection) verifyRLS(ctx context.Context) error {
	if c.rls == nil {
		return nil
	}

	probe := dbscope.WithTenant(ctx, dbscope.Tenant{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})

	var attrs roleAttributes
	err := c.db.RunInTx(probe, &sql.TxOptions{ReadOnly: true}, func(ctx context.Context, tx bun.Tx) error {
		if err := tx.NewRaw(roleAttributesQuery).Scan(ctx, &attrs); err != nil {
			return fmt.Errorf("read application role attributes: %w", err)
		}

		var orgID string
		if err := tx.NewRaw("SELECT trenova_rls.org_id()").Scan(ctx, &orgID); err != nil {
			return fmt.Errorf("%w: %w", ErrRLSScopeKeyNotInstalled, err)
		}

		return nil
	})
	if err != nil {
		return c.rlsProblem(ctx, err)
	}

	if attrs.Superuser || attrs.BypassRLS || attrs.BypassMember || attrs.OwnsTables {
		if err = c.rlsProblem(ctx, fmt.Errorf("%w (role %q)", ErrRLSRoleCannotBypass, attrs.Name)); err != nil {
			return err
		}
	}

	if !attrs.TenantMember {
		if err = c.rlsProblem(ctx, fmt.Errorf("%w (role %q)", ErrRLSRoleNotTenant, attrs.Name)); err != nil {
			return err
		}
	}

	return c.verifySystemRole(ctx)
}

func (c *Connection) verifySystemRole(ctx context.Context) error {
	if c.system == nil {
		return nil
	}

	var attrs roleAttributes
	if err := c.system.NewRaw(roleAttributesQuery).Scan(dbscope.WithSystem(ctx, verifyRolesReason), &attrs); err != nil {
		return c.rlsProblem(ctx, fmt.Errorf("read system role attributes: %w", err))
	}

	if !attrs.Superuser && !attrs.BypassMember {
		return c.rlsProblem(ctx, fmt.Errorf("%w (role %q)", ErrRLSSystemRoleCannotBypass, attrs.Name))
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
