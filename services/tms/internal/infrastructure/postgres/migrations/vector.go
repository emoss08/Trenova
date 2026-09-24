package migrations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/pkg/dbdialect"
	"github.com/uptrace/bun"
)

const vectorSchemaMigration = "20261231006110_ai_retrieval_vector.tx.up.sql"

var (
	ErrVectorNotPostgres     = errors.New("pgvector needs a PostgreSQL database")
	ErrVectorExtensionTooOld = errors.New("the installed pgvector is older than 0.8")
	ErrVectorSchemaNotReady  = errors.New("the vector schema is still missing after installing it")
)

func VectorSchemaSQL() (string, error) {
	contents, err := sqlMigrations.ReadFile(vectorSchemaMigration)
	if err != nil {
		return "", fmt.Errorf("read vector schema: %w", err)
	}

	return string(contents), nil
}

type EnableVectorResult struct {
	Before dbdialect.VectorSupport
	After  dbdialect.VectorSupport
}

func EnableVector(ctx context.Context, db *bun.DB) (EnableVectorResult, error) {
	if dbdialect.FromBun(db).IsSQLite() {
		return EnableVectorResult{}, ErrVectorNotPostgres
	}

	schema, err := VectorSchemaSQL()
	if err != nil {
		return EnableVectorResult{}, err
	}

	before, err := dbdialect.ProbeVector(ctx, db)
	if err != nil {
		return EnableVectorResult{}, err
	}

	err = db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		return installVector(ctx, tx, schema)
	})
	if err != nil {
		return EnableVectorResult{Before: before}, err
	}

	after, err := dbdialect.ProbeVector(ctx, db)
	if err != nil {
		return EnableVectorResult{Before: before}, err
	}

	result := EnableVectorResult{Before: before, After: after}
	if !after.Ready() {
		return result, fmt.Errorf("%w: pgvector reports %s", ErrVectorSchemaNotReady, after.State)
	}

	return result, nil
}

func installVector(ctx context.Context, tx bun.Tx, schema string) error {
	if _, err := tx.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS vector"); err != nil {
		return fmt.Errorf("create extension vector: %w", err)
	}

	support, err := dbdialect.ProbeVector(ctx, tx)
	if err != nil {
		return err
	}

	if support.State == dbdialect.VectorExtensionTooOld {
		if _, err = tx.ExecContext(ctx, "ALTER EXTENSION vector UPDATE"); err != nil {
			return fmt.Errorf("update extension vector: %w", err)
		}

		if support, err = dbdialect.ProbeVector(ctx, tx); err != nil {
			return err
		}

		if support.State == dbdialect.VectorExtensionTooOld {
			return fmt.Errorf(
				"%w: %s is installed and no newer version is available on the server",
				ErrVectorExtensionTooOld,
				support.ExtensionVersion,
			)
		}
	}

	if _, err = tx.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("create vector schema: %w", err)
	}

	return nil
}
