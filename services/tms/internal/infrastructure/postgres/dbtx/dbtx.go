package dbtx

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/uptrace/bun"
)

type scopedTransactor interface {
	ScopedTransactions() bool
}

func Read[T any](
	ctx context.Context,
	conn ports.DBConnection,
	fn func(context.Context) (T, error),
) (T, error) {
	return run(ctx, conn, true, fn)
}

func Write[T any](
	ctx context.Context,
	conn ports.DBConnection,
	fn func(context.Context) (T, error),
) (T, error) {
	return run(ctx, conn, false, fn)
}

func ReadErr(ctx context.Context, conn ports.DBConnection, fn func(context.Context) error) error {
	return runErr(ctx, conn, true, fn)
}

func WriteErr(ctx context.Context, conn ports.DBConnection, fn func(context.Context) error) error {
	return runErr(ctx, conn, false, fn)
}

func Read2[A, B any](
	ctx context.Context,
	conn ports.DBConnection,
	fn func(context.Context) (A, B, error),
) (first A, second B, err error) {
	return run2(ctx, conn, true, fn)
}

func Write2[A, B any](
	ctx context.Context,
	conn ports.DBConnection,
	fn func(context.Context) (A, B, error),
) (first A, second B, err error) {
	return run2(ctx, conn, false, fn)
}

func Required(conn ports.DBConnection) bool {
	scoped, ok := conn.(scopedTransactor)
	return ok && scoped.ScopedTransactions()
}

func run[T any](
	ctx context.Context,
	conn ports.DBConnection,
	readOnly bool,
	fn func(context.Context) (T, error),
) (T, error) {
	if !Required(conn) {
		return fn(ctx)
	}

	var out T
	err := conn.WithTx(
		ctx,
		ports.TxOptions{ReadOnly: readOnly},
		func(ctx context.Context, _ bun.Tx) error {
			var err error
			out, err = fn(ctx)
			return err
		},
	)

	return out, err
}

func run2[A, B any](
	ctx context.Context,
	conn ports.DBConnection,
	readOnly bool,
	fn func(context.Context) (A, B, error),
) (first A, second B, err error) {
	if !Required(conn) {
		return fn(ctx)
	}

	err = conn.WithTx(
		ctx,
		ports.TxOptions{ReadOnly: readOnly},
		func(ctx context.Context, _ bun.Tx) error {
			var fnErr error
			first, second, fnErr = fn(ctx)
			return fnErr
		},
	)

	return first, second, err
}

func runErr(
	ctx context.Context,
	conn ports.DBConnection,
	readOnly bool,
	fn func(context.Context) error,
) error {
	if !Required(conn) {
		return fn(ctx)
	}

	return conn.WithTx(
		ctx,
		ports.TxOptions{ReadOnly: readOnly},
		func(ctx context.Context, _ bun.Tx) error {
			return fn(ctx)
		},
	)
}
