package dbtx

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
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

// GetOrCreate reads a row, creating its default when there is none. The read
// runs alone in a read-only transaction, so the result cache can serve it; only
// a missing row opens a write. A preview reads inside a read-only snapshot,
// where an insert fails and aborts the whole transaction, so every query after
// it fails ("current transaction is aborted"): the insert runs in a savepoint
// after a second read, and a read-only caller gets the default without
// anything being written. Find must report a missing row as sql.ErrNoRows.
func GetOrCreate[T any](
	ctx context.Context,
	conn kernelports.DBConnection,
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
	conn kernelports.DBConnection,
	spec GetOrCreateSpec[T],
) (T, error) {
	found, err := Read(ctx, conn, spec.Find)
	if !dberror.IsNotFoundError(err) {
		return found, err
	}

	if kernelports.IsReadOnly(ctx) {
		return spec.Default(), nil
	}

	return Write(ctx, conn, func(ctx context.Context) (T, error) {
		row, findErr := spec.Find(ctx)
		if !dberror.IsNotFoundError(findErr) {
			return row, findErr
		}

		if createErr := Savepoint(ctx, conn, spec.Create); createErr != nil {
			if dberror.IsReadOnlyTransaction(createErr) {
				return spec.Default(), nil
			}

			var zero T
			return zero, createErr
		}

		return spec.Find(ctx)
	})
}
