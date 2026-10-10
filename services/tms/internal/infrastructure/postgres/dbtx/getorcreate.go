package dbtx

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/dberror"
)

// GetOrCreateSpec is how to read a row that may not exist yet, how to insert
// its default, the default itself, and how the repository words a failure.
type GetOrCreateSpec[T any] struct {
	Find     func(context.Context) (T, error)
	Create   func(context.Context) error
	Default  func() T
	MapError func(error) error
}

// GetOrCreate reads a row, creating its default when there is none. A preview
// reads inside a read-only snapshot, where an insert fails and aborts the whole
// transaction, so every query after it fails ("current transaction is
// aborted"). The read comes first, the insert runs in a savepoint, and a
// read-only caller gets the default without anything being written. Find must
// report a missing row as sql.ErrNoRows.
func GetOrCreate[T any](
	ctx context.Context,
	conn ports.DBConnection,
	spec GetOrCreateSpec[T],
) (T, error) {
	found, err := getOrCreate(ctx, conn, spec)
	if err != nil && spec.MapError != nil {
		var zero T
		return zero, spec.MapError(err)
	}

	return found, err
}

func getOrCreate[T any](
	ctx context.Context,
	conn ports.DBConnection,
	spec GetOrCreateSpec[T],
) (T, error) {
	if ports.IsReadOnly(ctx) {
		return Read(ctx, conn, func(ctx context.Context) (T, error) {
			found, err := spec.Find(ctx)
			if dberror.IsNotFoundError(err) {
				return spec.Default(), nil
			}

			return found, err
		})
	}

	return Write(ctx, conn, func(ctx context.Context) (T, error) {
		found, err := spec.Find(ctx)
		if !dberror.IsNotFoundError(err) {
			return found, err
		}

		if err = Savepoint(ctx, conn, spec.Create); err != nil {
			if dberror.IsReadOnlyTransaction(err) {
				return spec.Default(), nil
			}

			var zero T
			return zero, err
		}

		return spec.Find(ctx)
	})
}
