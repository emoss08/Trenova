package postgres

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbpool"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/readscope"
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

const (
	setScopeSQL          = "SELECT set_config('trenova.scope', $1, true)"
	setScopeAndReadScope = "SELECT set_config('trenova.scope', $1, true), set_config('trenova.read_scope', $2, true)"
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

type scopeGuardConfig struct {
	Pool           scopePool
	Signer         *scopeSigner
	Enforce        bool
	SystemFallback bool
	Reporter       scopeReporter
}

type scopeGuard struct {
	pool           scopePool
	signer         *scopeSigner
	enforce        bool
	systemFallback bool
	reporter       scopeReporter
}

type transactionScope struct {
	token     string
	readScope string
}

func (s transactionScope) tenant() bool {
	return s.token != ""
}

var _ dbpool.Guard = (*scopeGuard)(nil)

func newScopeGuard(cfg scopeGuardConfig) *scopeGuard {
	return &scopeGuard{
		pool:           cfg.Pool,
		signer:         cfg.Signer,
		enforce:        cfg.Enforce,
		systemFallback: cfg.SystemFallback && !cfg.Enforce,
		reporter:       cfg.Reporter,
	}
}

func (g *scopeGuard) begin(ctx context.Context) (transactionScope, error) {
	scope := dbscope.From(ctx)

	if g.pool == scopePoolSystem || g.servesSystemScope(scope) {
		return transactionScope{}, g.checkSystemScope(ctx, scope, scopeEventSystemTx)
	}

	if tenant, ok := scope.Tenant(); ok {
		token, err := g.signer.token(tenant)
		if err != nil {
			g.report(ctx, scopeEventUnscopedTx, scopeOutcomeRefused, "")
			return transactionScope{}, err
		}
		readScope, err := readscope.Encode(ctx)
		if err != nil {
			g.report(ctx, scopeEventUnscopedTx, scopeOutcomeRefused, "")
			return transactionScope{}, err
		}
		g.report(ctx, scopeEventTenantTx, scopeOutcomeAllowed, "")
		return transactionScope{token: token, readScope: readScope}, nil
	}

	event := scopeEventUnscopedTx
	if scope.Kind() == dbscope.KindSystem {
		event = scopeEventWrongPool
	}

	return transactionScope{}, g.refuseOrObserve(ctx, event, scope.Reason(), ErrNoTenantScope)
}

func (g *scopeGuard) BeginCommand(ctx context.Context, command string) (string, error) {
	scope, err := g.begin(ctx)
	if err != nil || !scope.tenant() {
		return command, err
	}

	command += "; SET LOCAL trenova.scope = '" + scope.token + "'"
	if scope.readScope != "" {
		command += "; SET LOCAL trenova.read_scope = '" + scope.readScope + "'"
	}

	return command, nil
}

func (g *scopeGuard) OnBegin(ctx context.Context, _ keel.TxInfo) ([]keel.Statement, error) {
	scope, err := g.begin(ctx)
	if err != nil || !scope.tenant() {
		return nil, err
	}

	if scope.readScope == "" {
		return []keel.Statement{{SQL: setScopeSQL, Args: []any{scope.token}}}, nil
	}

	return []keel.Statement{{
		SQL:  setScopeAndReadScope,
		Args: []any{scope.token, scope.readScope},
	}}, nil
}

func (g *scopeGuard) GuardStatement(ctx context.Context) error {
	scope := dbscope.From(ctx)

	if g.pool == scopePoolSystem || g.servesSystemScope(scope) {
		return g.checkSystemScope(ctx, scope, scopeEventSystemStatement)
	}

	return g.refuseOrObserve(
		ctx,
		scopeEventUnscopedStmt,
		scope.Reason(),
		ErrStatementOutsideTransaction,
	)
}

func (g *scopeGuard) servesSystemScope(scope dbscope.Scope) bool {
	return g.systemFallback && scope.Kind() == dbscope.KindSystem
}

func (g *scopeGuard) checkSystemScope(
	ctx context.Context,
	scope dbscope.Scope,
	event string,
) error {
	if scope.Kind() != dbscope.KindSystem {
		return g.refuseOrObserve(ctx, scopeEventWrongPool, "", ErrSystemScopeRequired)
	}

	if scope.Reason() == "" {
		return g.refuseOrObserve(ctx, event, "", ErrSystemScopeReason)
	}

	g.report(ctx, event, scopeOutcomeAllowed, scope.Reason())

	return nil
}

func (g *scopeGuard) refuseOrObserve(
	ctx context.Context,
	event, reason string,
	refusal error,
) error {
	if g.enforce {
		g.report(ctx, event, scopeOutcomeRefused, reason)
		return refusal
	}

	g.report(ctx, event, scopeOutcomeObserved, reason)

	return nil
}

func (g *scopeGuard) report(ctx context.Context, event, outcome, reason string) {
	if g.reporter == nil {
		return
	}

	caller := ""
	if outcome != scopeOutcomeAllowed || event != scopeEventTenantTx {
		caller = callerOutsideDatabaseLayer()
	}

	g.reporter.Report(ctx, &scopeEvent{
		Pool:    g.pool,
		Event:   event,
		Outcome: outcome,
		Reason:  reason,
		Caller:  caller,
	})
}

const callerSearchDepth = 32

var databaseLayerPrefixes = [...]string{
	"runtime.",
	"database/sql.",
	"github.com/uptrace/bun",
	"github.com/jackc/pgx",
	"github.com/emoss08/keel",
	"github.com/emoss08/trenova/internal/infrastructure/postgres.",
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbpool.",
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx.",
	"github.com/emoss08/trenova/pkg/dbscope.",
}

const callerFramesReported = 2

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

	reported := make([]string, 0, callerFramesReported)
	for len(reported) < callerFramesReported {
		frame, more := frames.Next()
		if !inDatabaseLayer(frame.Function) {
			reported = append(reported, fmt.Sprintf("%s:%d", frame.Function, frame.Line))
		}
		if !more {
			break
		}
	}

	if len(reported) == 0 {
		return "unknown"
	}

	return strings.Join(reported, " <- ")
}

func inDatabaseLayer(function string) bool {
	for _, prefix := range databaseLayerPrefixes {
		if strings.HasPrefix(function, prefix) {
			return true
		}
	}

	return false
}
