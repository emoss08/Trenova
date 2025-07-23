package common

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
)

// BaseRepository provides common functionality for all repositories
type BaseRepository struct {
	DB         db.Connection
	Logger     *zerolog.Logger
	TableName  string
	EntityName string
}

// SetupReadWrite returns read-write DB connection and operation logger
// Use this for operations that need both read and write access (e.g., transactions with reads)
func (br *BaseRepository) SetupReadWrite(
	ctx context.Context,
	operation string,
	fields ...any,
) (*bun.DB, *zerolog.Logger, error) {
	dba, err := br.DB.DB(ctx)
	if err != nil {
		return nil, nil, err
	}

	logger := br.buildLogger(operation, fields...)
	return dba, logger, nil
}

// SetupWriteOnly returns write-only DB connection and operation logger
// Use this for INSERT, UPDATE, DELETE operations
func (br *BaseRepository) SetupWriteOnly(
	ctx context.Context,
	operation string,
	fields ...any,
) (*bun.DB, *zerolog.Logger, error) {
	dba, err := br.DB.WriteDB(ctx)
	if err != nil {
		return nil, nil, err
	}

	logger := br.buildLogger(operation, fields...)
	return dba, logger, nil
}

// SetupReadOnly returns read-only DB connection and operation logger
// Use this for SELECT operations
func (br *BaseRepository) SetupReadOnly(
	ctx context.Context,
	operation string,
	fields ...any,
) (*bun.DB, *zerolog.Logger, error) {
	dba, err := br.DB.ReadDB(ctx)
	if err != nil {
		return nil, nil, err
	}

	logger := br.buildLogger(operation, fields...)
	return dba, logger, nil
}

// buildLogger creates a logger with operation context and custom fields
func (br *BaseRepository) buildLogger(operation string, fields ...any) *zerolog.Logger {
	log := br.Logger.With().
		Str("operation", operation).
		Str("repository", br.TableName)

	// Process field pairs (key, value, key, value...)
	for i := 0; i < len(fields)-1; i += 2 {
		if key, ok := fields[i].(string); ok {
			switch v := fields[i+1].(type) {
			case string:
				log = log.Str(key, v)
			case int64:
				log = log.Int64(key, v)
			case int:
				log = log.Int(key, v)
			case bool:
				log = log.Bool(key, v)
			default:
				log = log.Interface(key, v)
			}
		}
	}

	logger := log.Logger()
	return &logger
}
