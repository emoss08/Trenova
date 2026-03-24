package postgres

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/gtc/internal/core/domain"
	"github.com/emoss08/gtc/internal/core/ports"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type SnapshotReader struct {
	pool        *pgxpool.Pool
	checkpoints ports.CheckpointStore
	batchSize   int
	concurrency int
	logger      *zap.Logger
}

func NewSnapshotReader(
	pool *pgxpool.Pool,
	checkpoints ports.CheckpointStore,
	batchSize int,
	concurrency int,
	logger *zap.Logger,
) *SnapshotReader {
	return &SnapshotReader{
		pool:        pool,
		checkpoints: checkpoints,
		batchSize:   batchSize,
		concurrency: concurrency,
		logger:      logger.Named("snapshot_reader"),
	}
}

func (r *SnapshotReader) CurrentLSN(ctx context.Context) (string, error) {
	var lsn string
	if err := r.pool.QueryRow(ctx, "SELECT pg_current_wal_lsn()::text").Scan(&lsn); err != nil {
		return "", fmt.Errorf("query current wal lsn: %w", err)
	}
	return lsn, nil
}

func (r *SnapshotReader) Run(
	ctx context.Context,
	bindings []domain.SnapshotBinding,
	handler ports.RecordHandler,
) error {
	return r.run(ctx, bindings, handler, true)
}

func (r *SnapshotReader) Backfill(
	ctx context.Context,
	bindings []domain.SnapshotBinding,
	handler ports.RecordHandler,
) error {
	return r.run(ctx, bindings, handler, false)
}

func (r *SnapshotReader) run(
	ctx context.Context,
	bindings []domain.SnapshotBinding,
	handler ports.RecordHandler,
	persistProgress bool,
) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sem := make(chan struct{}, r.concurrency)
	errCh := make(chan error, 1)
	var wg sync.WaitGroup

	for _, binding := range bindings {
		if persistProgress {
			progress, err := r.checkpoints.LoadSnapshotProgress(ctx, binding.FullTableName())
			if err != nil {
				return err
			}
			if progress.Completed {
				continue
			}
		}

		binding := binding
		wg.Add(1)
		go func() {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()

			if err := r.snapshotTable(ctx, binding, handler, persistProgress); err != nil {
				select {
				case errCh <- err:
				default:
				}
				cancel()
			}
		}()
	}

	wg.Wait()

	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}

func (r *SnapshotReader) HealthCheck(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

func (r *SnapshotReader) snapshotTable(
	ctx context.Context,
	binding domain.SnapshotBinding,
	handler ports.RecordHandler,
	persistProgress bool,
) error {
	tableName := binding.FullTableName()
	progress := ports.SnapshotProgress{TableName: tableName}
	var err error
	if persistProgress {
		progress, err = r.checkpoints.LoadSnapshotProgress(ctx, tableName)
		if err != nil {
			return err
		}
	}

	cursor, err := domain.ParseCursor(progress.Cursor)
	if err != nil {
		return fmt.Errorf("parse snapshot cursor for %s: %w", tableName, err)
	}

	r.logger.Info("snapshotting table", zap.String("table", tableName), zap.Any("cursor", cursor.Values))

	for {
		query, args := buildSnapshotQuery(binding, cursor, r.batchSize)
		rows, err := r.pool.Query(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("query snapshot rows for %s: %w", tableName, err)
		}

		count := 0
		for rows.Next() {
			values, mapErr := scanRowMap(rows)
			if mapErr != nil {
				rows.Close()
				return fmt.Errorf("scan snapshot row for %s: %w", tableName, mapErr)
			}

			record := domain.SourceRecord{
				Operation: domain.OperationSnapshot,
				Schema:    binding.Schema,
				Table:     binding.Table,
				NewData:   values,
				Metadata: domain.RecordMetadata{
					Timestamp: time.Now().UTC(),
					Snapshot:  true,
				},
			}

			if err := handler(ctx, record); err != nil {
				rows.Close()
				return err
			}

			cursorValues, err := domain.RecordKey(values, binding.PrimaryKeys)
			if err != nil {
				rows.Close()
				return fmt.Errorf("build snapshot cursor for %s: %w", tableName, err)
			}
			cursor = domain.Cursor{Values: cursorValues}
			count++
		}

		if err := rows.Err(); err != nil {
			rows.Close()
			return fmt.Errorf("iterate snapshot rows for %s: %w", tableName, err)
		}
		rows.Close()

		cursorPayload, err := cursor.Marshal()
		if err != nil {
			return err
		}

		if persistProgress {
			if err := r.checkpoints.SaveSnapshotProgress(ctx, ports.SnapshotProgress{
				TableName: tableName,
				Cursor:    cursorPayload,
				Completed: count < r.batchSize,
			}); err != nil {
				return err
			}
		}

		if count < r.batchSize {
			r.logger.Info("snapshot complete", zap.String("table", tableName), zap.Any("cursor", cursor.Values))
			return nil
		}
	}
}

func buildSnapshotQuery(binding domain.SnapshotBinding, cursor domain.Cursor, batchSize int) (string, []any) {
	var (
		predicate string
		args      []any
	)

	if !cursor.IsZero() && len(cursor.Values) == len(binding.PrimaryKeys) {
		predicate, args = buildCursorPredicate(binding.PrimaryKeys, cursor.Values)
	}

	whereClause := ""
	if predicate != "" {
		whereClause = " WHERE " + predicate
	}

	args = append(args, batchSize)

	return fmt.Sprintf(
		`SELECT * FROM %s.%s%s ORDER BY %s LIMIT $%d`,
		quoteIdentifier(binding.Schema),
		quoteIdentifier(binding.Table),
		whereClause,
		buildOrderBy(binding.PrimaryKeys),
		len(args),
	), args
}

func buildCursorPredicate(fields []string, values []any) (string, []any) {
	parts := make([]string, 0, len(fields))
	args := make([]any, 0, len(values)*len(values))
	argIndex := 1

	for idx := range fields {
		clauses := make([]string, 0, idx+1)
		for prev := 0; prev < idx; prev++ {
			clauses = append(clauses, fmt.Sprintf("%s = $%d", quoteIdentifier(fields[prev]), argIndex))
			args = append(args, values[prev])
			argIndex++
		}

		clauses = append(clauses, fmt.Sprintf("%s > $%d", quoteIdentifier(fields[idx]), argIndex))
		args = append(args, values[idx])
		argIndex++
		parts = append(parts, "("+strings.Join(clauses, " AND ")+")")
	}

	return strings.Join(parts, " OR "), args
}

func buildOrderBy(fields []string) string {
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		parts = append(parts, quoteIdentifier(field))
	}
	return strings.Join(parts, ", ")
}

func scanRowMap(rows pgx.Rows) (map[string]any, error) {
	values, err := rows.Values()
	if err != nil {
		return nil, err
	}

	fieldDescriptions := rows.FieldDescriptions()
	record := make(map[string]any, len(fieldDescriptions))
	for idx, field := range fieldDescriptions {
		record[string(field.Name)] = normalizeValue(values[idx])
	}

	return record, nil
}

func normalizeValue(value any) any {
	switch typed := value.(type) {
	case []byte:
		return string(typed)
	case time.Time:
		return typed.UTC().Format(time.RFC3339Nano)
	default:
		return typed
	}
}
