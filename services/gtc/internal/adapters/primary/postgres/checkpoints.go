package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/gtc/internal/core/ports"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

const (
	bootstrapLSNKey = "bootstrap_lsn"
	walLSNKey       = "wal_lsn"
)

type CheckpointStore struct {
	pool          *pgxpool.Pool
	schema        string
	table         string
	snapshotTable string
	logger        *zap.Logger
}

func NewCheckpointStore(
	pool *pgxpool.Pool,
	schema string,
	table string,
	logger *zap.Logger,
) *CheckpointStore {
	return &CheckpointStore{
		pool:          pool,
		schema:        schema,
		table:         table,
		snapshotTable: table + "_snapshot_progress",
		logger:        logger.Named("checkpoint_store"),
	}
}

func (s *CheckpointStore) Ensure(ctx context.Context) error {
	queries := []string{
		fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", quoteIdentifier(s.schema)),
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s.%s (
				name TEXT PRIMARY KEY,
				value TEXT NOT NULL,
				updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)
		`, quoteIdentifier(s.schema), quoteIdentifier(s.table)),
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s.%s (
				table_name TEXT PRIMARY KEY,
				cursor TEXT NOT NULL DEFAULT '',
				completed BOOLEAN NOT NULL DEFAULT FALSE,
				updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)
		`, quoteIdentifier(s.schema), quoteIdentifier(s.snapshotTable)),
		fmt.Sprintf(
			"ALTER TABLE %s.%s ADD COLUMN IF NOT EXISTS cursor TEXT NOT NULL DEFAULT ''",
			quoteIdentifier(s.schema),
			quoteIdentifier(s.snapshotTable),
		),
		fmt.Sprintf(`
			DO $$
			BEGIN
				IF EXISTS (
					SELECT 1
					FROM information_schema.columns
					WHERE table_schema = %s
					  AND table_name = %s
					  AND column_name = 'last_pk'
				) THEN
					EXECUTE 'UPDATE %s.%s SET cursor = last_pk WHERE cursor = ''''';
				END IF;
			END
			$$
		`, pgLiteral(s.schema), pgLiteral(s.snapshotTable), quoteIdentifier(s.schema), quoteIdentifier(s.snapshotTable)),
	}

	for _, query := range queries {
		if _, err := s.pool.Exec(ctx, query); err != nil {
			return fmt.Errorf("ensure checkpoint tables: %w", err)
		}
	}

	return nil
}

func (s *CheckpointStore) HealthCheck(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func (s *CheckpointStore) LoadBootstrapLSN(ctx context.Context) (string, error) {
	return s.loadKey(ctx, bootstrapLSNKey)
}

func (s *CheckpointStore) SaveBootstrapLSN(ctx context.Context, lsn string) error {
	return s.saveKey(ctx, bootstrapLSNKey, lsn)
}

func (s *CheckpointStore) LoadWALLSN(ctx context.Context) (string, error) {
	return s.loadKey(ctx, walLSNKey)
}

func (s *CheckpointStore) SaveWALLSN(ctx context.Context, lsn string) error {
	return s.saveKey(ctx, walLSNKey, lsn)
}

func (s *CheckpointStore) LoadSnapshotProgress(
	ctx context.Context,
	tableName string,
) (ports.SnapshotProgress, error) {
	query := fmt.Sprintf(
		"SELECT table_name, cursor, completed FROM %s.%s WHERE table_name = $1",
		quoteIdentifier(s.schema),
		quoteIdentifier(s.snapshotTable),
	)

	var progress ports.SnapshotProgress
	err := s.pool.QueryRow(ctx, query, tableName).Scan(&progress.TableName, &progress.Cursor, &progress.Completed)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ports.SnapshotProgress{TableName: tableName}, nil
		}
		return ports.SnapshotProgress{}, fmt.Errorf("load snapshot progress: %w", err)
	}

	return progress, nil
}

func (s *CheckpointStore) SaveSnapshotProgress(
	ctx context.Context,
	progress ports.SnapshotProgress,
) error {
	query := fmt.Sprintf(`
		INSERT INTO %s.%s (table_name, cursor, completed, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (table_name)
		DO UPDATE SET cursor = EXCLUDED.cursor, completed = EXCLUDED.completed, updated_at = NOW()
	`, quoteIdentifier(s.schema), quoteIdentifier(s.snapshotTable))

	_, err := s.pool.Exec(ctx, query, progress.TableName, progress.Cursor, progress.Completed)
	if err != nil {
		return fmt.Errorf("save snapshot progress: %w", err)
	}

	return nil
}

func (s *CheckpointStore) loadKey(ctx context.Context, name string) (string, error) {
	query := fmt.Sprintf(
		"SELECT value FROM %s.%s WHERE name = $1",
		quoteIdentifier(s.schema),
		quoteIdentifier(s.table),
	)

	var value string
	err := s.pool.QueryRow(ctx, query, name).Scan(&value)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("load checkpoint key %s: %w", name, err)
	}

	return value, nil
}

func (s *CheckpointStore) saveKey(ctx context.Context, name string, value string) error {
	query := fmt.Sprintf(`
		INSERT INTO %s.%s (name, value, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (name)
		DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()
	`, quoteIdentifier(s.schema), quoteIdentifier(s.table))

	if _, err := s.pool.Exec(ctx, query, name, value); err != nil {
		return fmt.Errorf("save checkpoint key %s: %w", name, err)
	}

	return nil
}

func quoteIdentifier(value string) string {
	return pgx.Identifier{value}.Sanitize()
}

func pgLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
