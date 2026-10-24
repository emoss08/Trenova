package dbpool

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/uptrace/bun"
)

var ErrUnsupportedDriverConn = errors.New(
	"postgres driver connection does not support scoped access",
)

type Guard interface {
	BeginCommand(ctx context.Context, command string) (string, error)
	GuardStatement(ctx context.Context) error
	OnBegin(ctx context.Context, tx keel.TxInfo) ([]keel.Statement, error)
}

type connector struct {
	base   driver.Connector
	driver poolDriver
	guard  Guard
}

type poolDriver struct {
	driver.Driver
	db *keel.DB
}

var (
	_ driver.Connector = (*connector)(nil)
	_ driver.Driver    = poolDriver{}
)

func newConnector(db *keel.DB, guard Guard) *connector {
	base := stdlib.GetPoolConnector(db.Pool())

	return &connector{
		base:   base,
		driver: poolDriver{Driver: base.Driver(), db: db},
		guard:  guard,
	}
}

func openSQL(db *keel.DB, guard Guard) *sql.DB {
	sqldb := sql.OpenDB(newConnector(db, guard))
	sqldb.SetMaxIdleConns(0)

	return sqldb
}

func KeelOf(sqldb *sql.DB) *keel.DB {
	if sqldb == nil {
		return nil
	}

	if d, ok := sqldb.Driver().(poolDriver); ok {
		return d.db
	}

	return nil
}

func PgxConn(conn bun.Conn) (*pgx.Conn, error) {
	var out *pgx.Conn
	err := conn.Raw(func(driverConn any) error {
		scoped, ok := driverConn.(*scopedConn)
		if !ok {
			return fmt.Errorf("%w: %T", ErrUnsupportedDriverConn, driverConn)
		}
		out = scoped.base.Conn()
		return nil
	})

	return out, err
}

func (c *connector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.base.Connect(ctx)
	if err != nil {
		return nil, err
	}

	scoped, err := wrapScopedConn(c.guard, conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	return scoped, nil
}

func (c *connector) Driver() driver.Driver {
	return c.driver
}

func (c *connector) Close() error {
	c.driver.db.Close()
	return nil
}

type baseConn interface {
	driver.Conn
	driver.ConnBeginTx
	driver.ConnPrepareContext
	driver.ExecerContext
	driver.QueryerContext
	driver.Pinger
	driver.SessionResetter
	driver.NamedValueChecker
	Conn() *pgx.Conn
}

type scopedConn struct {
	base  baseConn
	guard Guard
	inTx  bool
}

var (
	_ driver.Conn               = (*scopedConn)(nil)
	_ driver.ConnBeginTx        = (*scopedConn)(nil)
	_ driver.ConnPrepareContext = (*scopedConn)(nil)
	_ driver.ExecerContext      = (*scopedConn)(nil)
	_ driver.QueryerContext     = (*scopedConn)(nil)
	_ driver.Pinger             = (*scopedConn)(nil)
	_ driver.SessionResetter    = (*scopedConn)(nil)
	_ driver.Validator          = (*scopedConn)(nil)
	_ driver.NamedValueChecker  = (*scopedConn)(nil)
)

func wrapScopedConn(guard Guard, conn driver.Conn) (*scopedConn, error) {
	base, ok := conn.(baseConn)
	if !ok {
		return nil, fmt.Errorf("%w: %T", ErrUnsupportedDriverConn, conn)
	}

	return &scopedConn{base: base, guard: guard}, nil
}

func (cn *scopedConn) Prepare(query string) (driver.Stmt, error) {
	return cn.PrepareContext(context.Background(), query)
}

func (cn *scopedConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if err := cn.guardStatement(ctx); err != nil {
		return nil, err
	}

	return cn.base.PrepareContext(ctx, query)
}

func (cn *scopedConn) Close() error {
	return cn.base.Close()
}

func (cn *scopedConn) Begin() (driver.Tx, error) {
	return cn.BeginTx(context.Background(), driver.TxOptions{})
}

func (cn *scopedConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	command, err := beginCommand(opts)
	if err != nil {
		return nil, err
	}

	if cn.guard != nil {
		if command, err = cn.guard.BeginCommand(ctx, command); err != nil {
			return nil, err
		}
	}

	if _, err = cn.base.ExecContext(ctx, command, nil); err != nil {
		return nil, err
	}

	cn.inTx = true

	return &scopedTx{conn: cn}, nil
}

func (cn *scopedConn) ExecContext(
	ctx context.Context,
	query string,
	args []driver.NamedValue,
) (driver.Result, error) {
	if err := cn.guardStatement(ctx); err != nil {
		return nil, err
	}

	return cn.base.ExecContext(ctx, query, args)
}

func (cn *scopedConn) QueryContext(
	ctx context.Context,
	query string,
	args []driver.NamedValue,
) (driver.Rows, error) {
	if err := cn.guardStatement(ctx); err != nil {
		return nil, err
	}

	if len(args) == 0 {
		args = simpleProtocolArgs
	}

	return cn.base.QueryContext(ctx, query, args)
}

var simpleProtocolArgs = []driver.NamedValue{{
	Ordinal: 1,
	Value:   pgx.QueryExecModeSimpleProtocol,
}}

func (cn *scopedConn) CheckNamedValue(value *driver.NamedValue) error {
	return cn.base.CheckNamedValue(value)
}

func (cn *scopedConn) guardStatement(ctx context.Context) error {
	if cn.inTx || cn.guard == nil {
		return nil
	}

	return cn.guard.GuardStatement(ctx)
}

func (cn *scopedConn) Ping(ctx context.Context) error {
	return cn.base.Ping(ctx)
}

func (cn *scopedConn) ResetSession(ctx context.Context) error {
	if cn.inTx {
		return driver.ErrBadConn
	}

	return cn.base.ResetSession(ctx)
}

func (cn *scopedConn) IsValid() bool {
	return !cn.inTx && !cn.base.Conn().IsClosed()
}

func (cn *scopedConn) finish(command string) error {
	_, err := cn.base.ExecContext(context.Background(), command, nil)
	cn.inTx = false

	return err
}

type scopedTx struct {
	conn *scopedConn
}

func (tx *scopedTx) Commit() error {
	return tx.conn.finish("COMMIT")
}

func (tx *scopedTx) Rollback() error {
	return tx.conn.finish("ROLLBACK")
}

func beginCommand(opts driver.TxOptions) (string, error) {
	command := "BEGIN"

	switch sql.IsolationLevel(opts.Isolation) {
	case sql.LevelDefault:
	case sql.LevelReadUncommitted:
		command += " ISOLATION LEVEL READ UNCOMMITTED"
	case sql.LevelReadCommitted:
		command += " ISOLATION LEVEL READ COMMITTED"
	case sql.LevelRepeatableRead:
		command += " ISOLATION LEVEL REPEATABLE READ"
	case sql.LevelSerializable:
		command += " ISOLATION LEVEL SERIALIZABLE"
	case sql.LevelWriteCommitted, sql.LevelSnapshot, sql.LevelLinearizable:
		return "", unsupportedIsolation(opts)
	default:
		return "", unsupportedIsolation(opts)
	}

	if opts.ReadOnly {
		command += " READ ONLY"
	}

	return command, nil
}

func unsupportedIsolation(opts driver.TxOptions) error {
	return fmt.Errorf(
		"postgres: unsupported transaction isolation: %s",
		sql.IsolationLevel(opts.Isolation).String(),
	)
}
