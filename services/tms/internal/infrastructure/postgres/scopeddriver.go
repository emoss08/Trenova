package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"

	"github.com/emoss08/trenova/pkg/dbscope"
)

type scopePool string

const (
	scopePoolTenant scopePool = "tenant"
	scopePoolSystem scopePool = "system"
)

const (
	scopeEventTenantTx        = "tenant_tx"
	scopeEventSystemTx        = "system_tx"
	scopeEventSystemStatement = "system_statement"
	scopeEventUnscopedTx      = "unscoped_tx"
	scopeEventUnscopedStmt    = "unscoped_statement"
	scopeEventWrongPool       = "wrong_pool"

	scopeOutcomeAllowed  = "allowed"
	scopeOutcomeObserved = "observed"
	scopeOutcomeRefused  = "refused"
)

var (
	ErrNoTenantScope = errors.New(
		"database access refused: no tenant scope is bound to this context",
	)
	ErrStatementOutsideTransaction = errors.New(
		"database access refused: tenant data must be read and written inside an explicit tenant-scoped transaction",
	)
	ErrSystemScopeRequired = errors.New(
		"database access refused: the system pool only serves contexts declared with dbscope.WithSystem",
	)
	ErrSystemScopeReason = errors.New(
		"database access refused: dbscope.WithSystem requires a reason",
	)
	errUnsupportedDriverConn = errors.New(
		"postgres driver connection does not support scoped access",
	)
)

type scopeEvent struct {
	Pool    scopePool
	Event   string
	Outcome string
	Reason  string
	Caller  string
}

type scopeReporter interface {
	Report(ctx context.Context, event *scopeEvent)
}

type scopedConnectorConfig struct {
	Base           driver.Connector
	Pool           scopePool
	Signer         *scopeSigner
	Enforce        bool
	SystemFallback bool
	Reporter       scopeReporter
}

type scopedConnector struct {
	base           driver.Connector
	pool           scopePool
	signer         *scopeSigner
	enforce        bool
	systemFallback bool
	reporter       scopeReporter
}

var _ driver.Connector = (*scopedConnector)(nil)

func newScopedConnector(cfg scopedConnectorConfig) *scopedConnector {
	return &scopedConnector{
		base:           cfg.Base,
		pool:           cfg.Pool,
		signer:         cfg.Signer,
		enforce:        cfg.Enforce,
		systemFallback: cfg.SystemFallback && !cfg.Enforce,
		reporter:       cfg.Reporter,
	}
}

func (c *scopedConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.base.Connect(ctx)
	if err != nil {
		return nil, err
	}

	scoped, err := wrapScopedConn(c, conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	return scoped, nil
}

func (c *scopedConnector) Driver() driver.Driver {
	return c.base.Driver()
}

type baseConn interface {
	driver.Conn
	driver.ConnBeginTx
	driver.ConnPrepareContext
	driver.ExecerContext
	driver.QueryerContext
	driver.Pinger
	driver.SessionResetter
	driver.Validator
}

type scopedConn struct {
	base      baseConn
	connector *scopedConnector
	inTx      bool
}

var (
	_ driver.Conn               = (*scopedConn)(nil)
	_ driver.ConnBeginTx        = (*scopedConn)(nil)
	_ driver.ConnPrepareContext = (*scopedConn)(nil)
	_ driver.ExecerContext      = (*scopedConn)(nil)
	_ driver.QueryerContext     = (*scopedConn)(nil)
	_ driver.Pinger             = (*scopedConn)(nil)
	_ driver.SessionResetter    = (*scopedConn)(nil)
	_ driver.Validator          = (*scopedConn)(nil)
)

func wrapScopedConn(connector *scopedConnector, conn driver.Conn) (*scopedConn, error) {
	base, ok := conn.(baseConn)
	if !ok {
		return nil, fmt.Errorf("%w: %T", errUnsupportedDriverConn, conn)
	}

	return &scopedConn{base: base, connector: connector}, nil
}

func (cn *scopedConn) Prepare(query string) (driver.Stmt, error) {
	return cn.PrepareContext(context.Background(), query)
}

func (cn *scopedConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if err := cn.guardStatement(ctx); err != nil {
		return nil, err
	}

	return cn.base.PrepareContext(ctx, query)
}

func (cn *scopedConn) Close() error {
	return cn.base.Close()
}

func (cn *scopedConn) Begin() (driver.Tx, error) {
	return cn.BeginTx(context.Background(), driver.TxOptions{})
}

func (cn *scopedConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	command, err := beginCommand(opts)
	if err != nil {
		return nil, err
	}

	command, err = cn.scopeBegin(ctx, command)
	if err != nil {
		return nil, err
	}

	if _, err = cn.base.ExecContext(ctx, command, nil); err != nil {
		return nil, err
	}

	cn.inTx = true

	return &scopedTx{conn: cn}, nil
}

func (cn *scopedConn) scopeBegin(ctx context.Context, command string) (string, error) {
	c := cn.connector
	scope := dbscope.From(ctx)

	if c.pool == scopePoolSystem || c.servesSystemScope(scope) {
		if err := c.checkSystemScope(ctx, scope, scopeEventSystemTx); err != nil {
			return "", err
		}
		return command, nil
	}

	if tenant, ok := scope.Tenant(); ok {
		token, err := c.signer.token(tenant)
		if err != nil {
			c.report(ctx, scopeEventUnscopedTx, scopeOutcomeRefused, "")
			return "", err
		}
		c.report(ctx, scopeEventTenantTx, scopeOutcomeAllowed, "")
		return command + "; SET LOCAL trenova.scope = '" + token + "'", nil
	}

	event := scopeEventUnscopedTx
	if scope.Kind() == dbscope.KindSystem {
		event = scopeEventWrongPool
	}

	if err := c.refuseOrObserve(ctx, event, scope.Reason(), ErrNoTenantScope); err != nil {
		return "", err
	}

	return command, nil
}

func (cn *scopedConn) ExecContext(
	ctx context.Context,
	query string,
	args []driver.NamedValue,
) (driver.Result, error) {
	if err := cn.guardStatement(ctx); err != nil {
		return nil, err
	}

	return cn.base.ExecContext(ctx, query, args)
}

func (cn *scopedConn) QueryContext(
	ctx context.Context,
	query string,
	args []driver.NamedValue,
) (driver.Rows, error) {
	if err := cn.guardStatement(ctx); err != nil {
		return nil, err
	}

	return cn.base.QueryContext(ctx, query, args)
}

func (cn *scopedConn) guardStatement(ctx context.Context) error {
	if cn.inTx {
		return nil
	}

	c := cn.connector
	scope := dbscope.From(ctx)

	if c.pool == scopePoolSystem || c.servesSystemScope(scope) {
		return c.checkSystemScope(ctx, scope, scopeEventSystemStatement)
	}

	return c.refuseOrObserve(
		ctx,
		scopeEventUnscopedStmt,
		scope.Reason(),
		ErrStatementOutsideTransaction,
	)
}

func (cn *scopedConn) Ping(ctx context.Context) error {
	return cn.base.Ping(ctx)
}

func (cn *scopedConn) ResetSession(ctx context.Context) error {
	if cn.inTx {
		return driver.ErrBadConn
	}

	return cn.base.ResetSession(ctx)
}

func (cn *scopedConn) IsValid() bool {
	return !cn.inTx && cn.base.IsValid()
}

func (cn *scopedConn) finish(command string) error {
	_, err := cn.base.ExecContext(context.Background(), command, nil)
	cn.inTx = false

	return err
}

type scopedTx struct {
	conn *scopedConn
}

func (tx *scopedTx) Commit() error {
	return tx.conn.finish("COMMIT")
}

func (tx *scopedTx) Rollback() error {
	return tx.conn.finish("ROLLBACK")
}

func (c *scopedConnector) servesSystemScope(scope dbscope.Scope) bool {
	return c.systemFallback && scope.Kind() == dbscope.KindSystem
}

func (c *scopedConnector) checkSystemScope(
	ctx context.Context,
	scope dbscope.Scope,
	event string,
) error {
	if scope.Kind() != dbscope.KindSystem {
		return c.refuseOrObserve(ctx, scopeEventWrongPool, "", ErrSystemScopeRequired)
	}

	if scope.Reason() == "" {
		return c.refuseOrObserve(ctx, event, "", ErrSystemScopeReason)
	}

	c.report(ctx, event, scopeOutcomeAllowed, scope.Reason())

	return nil
}

func (c *scopedConnector) refuseOrObserve(
	ctx context.Context,
	event, reason string,
	refusal error,
) error {
	if c.enforce {
		c.report(ctx, event, scopeOutcomeRefused, reason)
		return refusal
	}

	c.report(ctx, event, scopeOutcomeObserved, reason)

	return nil
}

func (c *scopedConnector) report(ctx context.Context, event, outcome, reason string) {
	if c.reporter == nil {
		return
	}

	caller := ""
	if outcome != scopeOutcomeAllowed || event != scopeEventTenantTx {
		caller = callerOutsideDatabaseLayer()
	}

	c.reporter.Report(ctx, &scopeEvent{
		Pool:    c.pool,
		Event:   event,
		Outcome: outcome,
		Reason:  reason,
		Caller:  caller,
	})
}

func beginCommand(opts driver.TxOptions) (string, error) {
	command := "BEGIN"

	switch sql.IsolationLevel(opts.Isolation) {
	case sql.LevelDefault:
	case sql.LevelReadUncommitted:
		command += " ISOLATION LEVEL READ UNCOMMITTED"
	case sql.LevelReadCommitted:
		command += " ISOLATION LEVEL READ COMMITTED"
	case sql.LevelRepeatableRead:
		command += " ISOLATION LEVEL REPEATABLE READ"
	case sql.LevelSerializable:
		command += " ISOLATION LEVEL SERIALIZABLE"
	case sql.LevelWriteCommitted, sql.LevelSnapshot, sql.LevelLinearizable:
		return "", unsupportedIsolation(opts)
	default:
		return "", unsupportedIsolation(opts)
	}

	if opts.ReadOnly {
		command += " READ ONLY"
	}

	return command, nil
}

func unsupportedIsolation(opts driver.TxOptions) error {
	return fmt.Errorf(
		"postgres: unsupported transaction isolation: %s",
		sql.IsolationLevel(opts.Isolation).String(),
	)
}

const callerSearchDepth = 32

var databaseLayerPrefixes = [...]string{
	"runtime.",
	"database/sql.",
	"github.com/uptrace/bun",
	"github.com/emoss08/trenova/internal/infrastructure/postgres.",
	"github.com/emoss08/trenova/pkg/dbscope.",
}

var callerPCs = sync.Pool{
	New: func() any {
		pcs := make([]uintptr, callerSearchDepth)
		return &pcs
	},
}

func callerOutsideDatabaseLayer() string {
	pcsPtr, _ := callerPCs.Get().(*[]uintptr)
	defer callerPCs.Put(pcsPtr)

	pcs := *pcsPtr
	n := runtime.Callers(3, pcs)
	frames := runtime.CallersFrames(pcs[:n])

	for {
		frame, more := frames.Next()
		if !inDatabaseLayer(frame.Function) {
			return fmt.Sprintf("%s:%d", frame.Function, frame.Line)
		}
		if !more {
			return "unknown"
		}
	}
}

func inDatabaseLayer(function string) bool {
	for _, prefix := range databaseLayerPrefixes {
		if strings.HasPrefix(function, prefix) {
			return true
		}
	}

	return false
}
