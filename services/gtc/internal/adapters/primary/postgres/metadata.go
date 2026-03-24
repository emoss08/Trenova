package postgres

import (
	"context"
	"fmt"

	"github.com/emoss08/gtc/internal/core/domain"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type MetadataStore struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewMetadataStore(pool *pgxpool.Pool, logger *zap.Logger) *MetadataStore {
	return &MetadataStore{
		pool:   pool,
		logger: logger.Named("metadata_store"),
	}
}

func (s *MetadataStore) LoadTableMetadata(
	ctx context.Context,
	schema string,
	table string,
) (domain.TableMetadata, error) {
	const query = `
		SELECT a.attname
		FROM pg_index i
		JOIN pg_class c ON c.oid = i.indrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		JOIN unnest(i.indkey) WITH ORDINALITY AS cols(attnum, ord) ON TRUE
		JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = cols.attnum
		WHERE i.indisprimary
		  AND n.nspname = $1
		  AND c.relname = $2
		ORDER BY cols.ord
	`

	rows, err := s.pool.Query(ctx, query, schema, table)
	if err != nil {
		return domain.TableMetadata{}, fmt.Errorf("load primary keys for %s.%s: %w", schema, table, err)
	}
	defer rows.Close()

	keys := make([]string, 0, 2)
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return domain.TableMetadata{}, fmt.Errorf("scan primary key for %s.%s: %w", schema, table, err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return domain.TableMetadata{}, fmt.Errorf("iterate primary keys for %s.%s: %w", schema, table, err)
	}
	if len(keys) == 0 {
		return domain.TableMetadata{}, fmt.Errorf("table %s.%s has no primary key", schema, table)
	}

	return domain.TableMetadata{
		Schema:      schema,
		Table:       table,
		PrimaryKeys: keys,
	}, nil
}
