package dbtx

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

func Read[T any](
	ctx context.Context,
	conn kernelports.DBConnection,
	fn func(context.Context) (T, error),
) (T, error) {
	return run(ctx, conn, true, fn)
}

func Write[T any](
	ctx context.Context,
	conn kernelports.DBConnection,
	fn func(context.Context) (T, error),
) (T, error) {
	return run(ctx, conn, false, fn)
}

func ReadErr(
	ctx context.Context,
	conn kernelports.DBConnection,
	fn func(context.Context) error,
) error {
	return runErr(ctx, conn, true, fn)
}

func WriteErr(
	ctx context.Context,
	conn kernelports.DBConnection,
	fn func(context.Context) error,
) error {
	return runErr(ctx, conn, false, fn)
}

func Read2[A, B any](
	ctx context.Context,
	conn kernelports.DBConnection,
	fn func(context.Context) (A, B, error),
) (first A, second B, err error) {
	return run2(ctx, conn, true, fn)
}

func Write2[A, B any](
	ctx context.Context,
	conn kernelports.DBConnection,
	fn func(context.Context) (A, B, error),
) (first A, second B, err error) {
	return run2(ctx, conn, false, fn)
}

func run[T any](
	ctx context.Context,
	conn kernelports.DBConnection,
	readOnly bool,
	fn func(context.Context) (T, error),
) (T, error) {
	var out T
	err := conn.Transact(
		ctx,
		kernelports.TxOptions{ReadOnly: readOnly},
		func(ctx context.Context) error {
			var err error
			out, err = fn(ctx)
			return err
		},
	)

	return out, err
}

func run2[A, B any](
	ctx context.Context,
	conn kernelports.DBConnection,
	readOnly bool,
	fn func(context.Context) (A, B, error),
) (first A, second B, err error) {
	err = conn.Transact(
		ctx,
		kernelports.TxOptions{ReadOnly: readOnly},
		func(ctx context.Context) error {
			var fnErr error
			first, second, fnErr = fn(ctx)
			return fnErr
		},
	)

	return first, second, err
}

func runErr(
	ctx context.Context,
	conn kernelports.DBConnection,
	readOnly bool,
	fn func(context.Context) error,
) error {
	return conn.Transact(ctx, kernelports.TxOptions{ReadOnly: readOnly}, fn)
}

func Savepoint(
	ctx context.Context,
	conn kernelports.DBConnection,
	fn func(context.Context) error,
) error {
	db := conn.Keel(ctx)
	if db == nil || !keel.InTx(ctx, db) {
		return fn(ctx)
	}

	return keel.Savepoint(ctx, db, fn)
}
