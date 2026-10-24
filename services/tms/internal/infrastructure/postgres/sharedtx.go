package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbpool"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/uptrace/bun"
)

type sharedTx struct {
	bun   bun.Tx
	ctx   context.Context
	conn  *bun.Conn
	done  func(committed bool)
	ended bool
}

func (c *Connection) begin(ctx context.Context, opts kernelports.TxOptions) (*sharedTx, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	txOpts := &sql.TxOptions{Isolation: opts.Isolation, ReadOnly: opts.ReadOnly}
	binding := &txBinding{scope: dbscope.From(ctx)}
	db := c.poolForContext(ctx)

	kdb := c.Keel(ctx)
	if kdb == nil {
		txCtx := context.WithValue(ctx, txContextKey{}, binding)
		tx, err := db.BeginTx(context.WithoutCancel(txCtx), txOpts)
		if err != nil {
			return nil, err
		}
		binding.tx = tx

		return &sharedTx{bun: tx, ctx: txCtx}, nil
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}

	pgxConn, err := dbpool.PgxConn(conn)
	if err != nil {
		return nil, errors.Join(err, conn.Close())
	}

	kctx, done := keel.JoinTx(ctx, kdb, pgxConn)
	txCtx := context.WithValue(kctx, txContextKey{}, binding)

	tx, err := conn.BeginTx(context.WithoutCancel(txCtx), txOpts)
	if err != nil {
		done(false)
		return nil, errors.Join(err, conn.Close())
	}
	binding.tx = tx

	return &sharedTx{
		bun:  tx,
		ctx:  txCtx,
		conn: &conn,
		done: done,
	}, nil
}

func (t *sharedTx) commit() error {
	if err := t.bun.Commit(); err != nil {
		return err
	}

	t.finish(true)
	return nil
}

func (t *sharedTx) end(committed bool) {
	if t.ended {
		return
	}

	if !committed {
		_ = t.bun.Rollback()
	}
	t.finish(committed)
}

func (t *sharedTx) finish(committed bool) {
	t.ended = true

	if t.conn != nil {
		_ = t.conn.Close()
	}
	if t.done != nil {
		t.done(committed)
	}
}
