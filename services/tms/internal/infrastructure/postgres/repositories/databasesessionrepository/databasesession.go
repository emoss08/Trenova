package databasesessionrepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/system"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var _ repositories.DatabaseSessionRepository = (*repository)(nil)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.DatabaseSessionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("repository.databasesession"),
	}
}

func (r *repository) ListBlocked(ctx context.Context) ([]*system.DatabaseSessionChain, error) {
	var rows []*system.DatabaseSessionChain

	err := r.db.DBForContext(ctx).NewRaw(`
		SELECT
			blocked.pid AS blocked_pid,
			blocking.pid AS blocking_pid,
			blocked.datname AS database_name,
			COALESCE(blocked.state, '') AS blocked_state,
			COALESCE(blocking.state, '') AS blocking_state,
			COALESCE(blocked.wait_event_type, '') AS blocked_wait_event_type,
			COALESCE(blocked.wait_event, '') AS blocked_wait_event,
			COALESCE(blocked.application_name, '') AS blocked_application_name,
			COALESCE(blocking.application_name, '') AS blocking_application_name,
			COALESCE(blocked.usename, '') AS blocked_user,
			COALESCE(blocking.usename, '') AS blocking_user,
			LEFT(regexp_replace(COALESCE(blocked.query, ''), '\s+', ' ', 'g'), 240) AS blocked_query_preview,
			LEFT(regexp_replace(COALESCE(blocking.query, ''), '\s+', ' ', 'g'), 240) AS blocking_query_preview,
			COALESCE(EXTRACT(EPOCH FROM now() - blocked.xact_start)::bigint, 0) AS blocked_transaction_age_s,
			COALESCE(EXTRACT(EPOCH FROM now() - blocking.xact_start)::bigint, 0) AS blocking_transaction_age_s,
			COALESCE(EXTRACT(EPOCH FROM now() - blocked.query_start)::bigint, 0) AS blocked_query_age_s,
			COALESCE(EXTRACT(EPOCH FROM now() - blocking.query_start)::bigint, 0) AS blocking_query_age_s
		FROM pg_stat_activity blocked
		CROSS JOIN LATERAL unnest(pg_blocking_pids(blocked.pid)) AS blocker_pid
		JOIN pg_stat_activity blocking ON blocking.pid = blocker_pid
		WHERE blocked.datname = current_database()
		ORDER BY blocked.query_start NULLS LAST, blocked.pid, blocking.pid
	`).Scan(ctx, &rows)
	if err != nil {
		return nil, fmt.Errorf("list blocked database sessions: %w", err)
	}

	return rows, nil
}

func (r *repository) Terminate(
	ctx context.Context,
	pid int64,
) (*system.TerminateDatabaseSessionResult, error) {
	if pid <= 0 {
		return nil, errortypes.NewValidationError(
			"pid",
			errortypes.ErrInvalid,
			"PID must be greater than zero.",
		)
	}

	result := &system.TerminateDatabaseSessionResult{
		PID: pid,
	}

	err := r.db.WithTx(
		ctx,
		ports.TxOptions{ReadOnly: false},
		func(txCtx context.Context, tx bun.Tx) error {
			var currentPID int64
			if err := tx.NewRaw(`SELECT pg_backend_pid()`).Scan(txCtx, &currentPID); err != nil {
				return fmt.Errorf("get current backend pid: %w", err)
			}
			if currentPID == pid {
				return errortypes.NewConflictError(
					"Refusing to terminate the current database session.",
				)
			}

			var currentDatabase string
			if err := tx.NewRaw(`SELECT current_database()`).Scan(txCtx, &currentDatabase); err != nil {
				return fmt.Errorf("get current database: %w", err)
			}

			var targetDatabase string
			err := tx.NewRaw(`SELECT datname FROM pg_stat_activity WHERE pid = ?`, pid).
				Scan(txCtx, &targetDatabase)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return errortypes.NewNotFoundError("Database session was not found.")
				}
				return fmt.Errorf("load target database session: %w", err)
			}

			if !strings.EqualFold(targetDatabase, currentDatabase) {
				return errortypes.NewConflictError(
					"Database session belongs to a different database.",
				)
			}

			if err := tx.NewRaw(`SELECT pg_terminate_backend(?)`, pid).Scan(txCtx, &result.Terminated); err != nil {
				return fmt.Errorf("terminate database session: %w", err)
			}

			if result.Terminated {
				result.Message = "Database session terminated."
				return nil
			}

			result.Message = "Database session could not be terminated."
			return errortypes.NewConflictError(result.Message)
		},
	)
	if err != nil {
		return nil, err
	}

	return result, nil
}
