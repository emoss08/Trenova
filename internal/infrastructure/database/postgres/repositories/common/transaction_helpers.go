package common

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

// RunInTransaction executes a function within a database transaction
func RunInTransaction(
	ctx context.Context,
	conn db.Connection,
	fn func(context.Context, bun.Tx) error,
) error {
	dba, err := conn.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}

	return dba.RunInTx(ctx, nil, fn)
}

// RunInTransactionWithResult executes a function within a transaction and returns a result
func RunInTransactionWithResult[T any](
	ctx context.Context,
	conn db.Connection,
	fn func(context.Context, bun.Tx) (T, error),
) (T, error) {
	var result T

	err := RunInTransaction(ctx, conn, func(c context.Context, tx bun.Tx) error {
		var txErr error
		result, txErr = fn(c, tx)
		return txErr
	})

	return result, err
}

// VersionedEntity interface for entities with optimistic locking
type VersionedEntity interface {
	GetID() string
	GetVersion() int64
	IncrementVersion()
}

// OptimisticUpdate performs an update with version checking
func OptimisticUpdate[T VersionedEntity](
	ctx context.Context,
	tx bun.Tx,
	entity T,
	entityName string,
) error {
	oldVersion := entity.GetVersion()
	entity.IncrementVersion()

	result, err := tx.NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", oldVersion).
		OmitZero().
		Returning("*").
		Exec(ctx)

	if err != nil {
		return err
	}

	return CheckRowsAffected(result, entityName, entity.GetID())
}

// OptimisticUpdateWithAlias performs an update with version checking using a table alias
func OptimisticUpdateWithAlias[T VersionedEntity](
	ctx context.Context,
	tx bun.Tx,
	entity T,
	entityName string,
	tableAlias string,
) error {
	oldVersion := entity.GetVersion()
	entity.IncrementVersion()

	result, err := tx.NewUpdate().
		Model(entity).
		WherePK().
		Where(tableAlias+".version = ?", oldVersion).
		OmitZero().
		Returning("*").
		Exec(ctx)

	if err != nil {
		return err
	}

	return CheckRowsAffected(result, entityName, entity.GetID())
}
