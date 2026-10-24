package dbcli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/emoss08/keel/keelcli"
	"github.com/emoss08/trenova/pkg/pgvector"
	"github.com/emoss08/trenova/schema/migrations"
	"github.com/jackc/pgx/v5"
)

const VectorMigration = "20261231006110_ai_retrieval_vector"

var (
	ErrVectorExtensionTooOld = errors.New("the installed pgvector is older than 0.8")
	ErrVectorSchemaNotReady  = errors.New("the vector schema is still missing after installing it")
)

type VectorResult struct {
	Before        pgvector.VectorSupport
	After         pgvector.VectorSupport
	PoliciesAdded int
}

var EnableVectorCommand = keelcli.Command{
	Name:  "enable-vector",
	Usage: "[-dry-run]",
	Summary: "install pgvector 0.8+ and the semantic retrieval tables a migration skipped " +
		"without it; safe to run again",
	Run: runEnableVector,
}

func runEnableVector(ctx context.Context, env *keelcli.Env, args []string) error {
	fs := env.FlagSet("enable-vector")
	dryRun := fs.Bool("dry-run", false, "report what is installed without changing anything")
	if err := env.Parse(fs, args); err != nil {
		return err
	}

	db, err := env.DB(ctx)
	if err != nil {
		return err
	}
	conn, err := db.Pool().Acquire(ctx)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer conn.Release()

	if *dryRun {
		support, probeErr := pgvector.ProbeVectorPgx(ctx, conn)
		if probeErr != nil {
			return probeErr
		}
		reportVector(env.Stdout, "Current state", support)
		return nil
	}

	result, err := EnableVector(ctx, conn.Conn())
	if result.Before.State != "" {
		reportVector(env.Stdout, "Before", result.Before)
	}
	if err != nil {
		return err
	}
	reportVector(env.Stdout, "After", result.After)
	if result.PoliciesAdded > 0 {
		_, _ = fmt.Fprintf(
			env.Stdout,
			"Applied tenant isolation policies to %d new vector table(s)\n",
			result.PoliciesAdded,
		)
	}
	_, _ = fmt.Fprintln(
		env.Stdout,
		"Semantic retrieval vector storage is ready. Running services pick it up within five minutes.",
	)

	return nil
}

func EnableVector(ctx context.Context, conn *pgx.Conn) (VectorResult, error) {
	schema, err := migrations.File(VectorMigration)
	if err != nil {
		return VectorResult{}, err
	}

	before, err := pgvector.ProbeVectorPgx(ctx, conn)
	if err != nil {
		return VectorResult{}, err
	}
	result := VectorResult{Before: before}

	err = pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
		return installVector(ctx, tx, schema)
	})
	if err != nil {
		return result, err
	}

	if err = conn.QueryRow(ctx, migrations.ReconcileRLS).Scan(&result.PoliciesAdded); err != nil {
		return result, fmt.Errorf("protect the vector tables with row-level security: %w", err)
	}

	if result.After, err = pgvector.ProbeVectorPgx(ctx, conn); err != nil {
		return result, err
	}
	if !result.After.Ready() {
		return result, fmt.Errorf(
			"%w: pgvector reports %s",
			ErrVectorSchemaNotReady,
			result.After.State,
		)
	}

	return result, nil
}

func installVector(ctx context.Context, tx pgx.Tx, schema string) error {
	if _, err := tx.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS vector"); err != nil {
		return fmt.Errorf("create extension vector: %w", err)
	}

	support, err := pgvector.ProbeVectorPgx(ctx, tx)
	if err != nil {
		return err
	}

	if support.State == pgvector.VectorExtensionTooOld {
		if _, err = tx.Exec(ctx, "ALTER EXTENSION vector UPDATE"); err != nil {
			return fmt.Errorf("update extension vector: %w", err)
		}
		if support, err = pgvector.ProbeVectorPgx(ctx, tx); err != nil {
			return err
		}
		if support.State == pgvector.VectorExtensionTooOld {
			return fmt.Errorf(
				"%w: %s is installed and no newer version is available on the server",
				ErrVectorExtensionTooOld,
				support.ExtensionVersion,
			)
		}
	}

	if _, err = tx.Exec(ctx, schema); err != nil {
		return fmt.Errorf("create vector schema: %w", err)
	}

	return nil
}

func reportVector(w io.Writer, label string, support pgvector.VectorSupport) {
	version := support.ExtensionVersion
	if version == "" {
		version = "not installed"
	}

	_, _ = fmt.Fprintf(w, "%s: %s (pgvector %s)\n", label, support.State, version)
}
