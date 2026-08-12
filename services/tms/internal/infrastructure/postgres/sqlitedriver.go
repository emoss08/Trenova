package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strings"
	"sync"
)

// SQLite has no ILIKE operator, and unlike a missing function it cannot be
// polyfilled: ILIKE is a parser keyword, so SQLite rejects the statement before
// any user-defined function could run.
//
// The operator appears in roughly fifty places, but almost none of them can be
// fixed at the call site. buncolgen and querybuilder precompute their SQL
// fragments into package-level values at init, long before configuration is
// read, and most of buncolgen is generated. Rewriting the statement as it
// reaches the driver is therefore the only single point that covers all of them.
//
// The substitution is sound because SQLite's LIKE is already case-insensitive
// for ASCII, which is what ILIKE is used for here. It is not equivalent for
// non-ASCII text, and that is one more reason SQLite is development-only.
const sqliteRewriteDriverName = "sqlite-trenova"

var registerSQLiteRewriteDriver sync.Once

// rewriteILike replaces the ILIKE operator with LIKE outside string literals and
// quoted identifiers, so a value or column name containing the word is untouched.
func rewriteILike(query string) string {
	if !containsFold(query, "ilike") {
		return query
	}

	var (
		out      strings.Builder
		inSingle bool
		inDouble bool
		i        int
	)

	out.Grow(len(query))

	for i < len(query) {
		ch := query[i]

		switch {
		case inSingle:
			if ch == '\'' {
				inSingle = false
			}
		case inDouble:
			if ch == '"' {
				inDouble = false
			}
		case ch == '\'':
			inSingle = true
		case ch == '"':
			inDouble = true
		case (ch == 'i' || ch == 'I') && isILikeAt(query, i):
			out.WriteString("LIKE")
			i += len("ILIKE")
			continue
		}

		out.WriteByte(ch)
		i++
	}

	return out.String()
}

func isILikeAt(query string, i int) bool {
	const keyword = "ilike"

	if i+len(keyword) > len(query) {
		return false
	}

	if !strings.EqualFold(query[i:i+len(keyword)], keyword) {
		return false
	}

	if i > 0 && isIdentifierByte(query[i-1]) {
		return false
	}

	end := i + len(keyword)

	return end == len(query) || !isIdentifierByte(query[end])
}

func isIdentifierByte(b byte) bool {
	return b == '_' ||
		(b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9')
}

func containsFold(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), needle)
}

// openSQLiteDB returns a database handle whose statements pass through
// rewriteILike on the way to modernc.org/sqlite.
func openSQLiteDB(dsn string) (*sql.DB, error) {
	base, err := sql.Open(sqliteDriverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	inner := base.Driver()
	if closeErr := base.Close(); closeErr != nil {
		return nil, fmt.Errorf("failed to close probe sqlite handle: %w", closeErr)
	}

	registerSQLiteRewriteDriver.Do(func() {
		sql.Register(sqliteRewriteDriverName, &rewriteDriver{inner: inner})
	})

	return sql.OpenDB(&rewriteConnector{dsn: dsn, inner: inner}), nil
}

type rewriteDriver struct {
	inner driver.Driver
}

func (d *rewriteDriver) Open(name string) (driver.Conn, error) {
	conn, err := d.inner.Open(name)
	if err != nil {
		return nil, err
	}

	return &rewriteConn{inner: conn}, nil
}

type rewriteConnector struct {
	dsn   string
	inner driver.Driver
}

func (c *rewriteConnector) Connect(_ context.Context) (driver.Conn, error) {
	conn, err := c.inner.Open(c.dsn)
	if err != nil {
		return nil, err
	}

	return &rewriteConn{inner: conn}, nil
}

func (c *rewriteConnector) Driver() driver.Driver {
	return &rewriteDriver{inner: c.inner}
}

// rewriteConn deliberately implements only the prepare path. database/sql falls
// back to Prepare when a connection does not advertise QueryerContext or
// ExecerContext, which keeps every statement going through the rewrite rather
// than slipping past it on a fast path.
type rewriteConn struct {
	inner driver.Conn
}

func (c *rewriteConn) Prepare(query string) (driver.Stmt, error) {
	return c.inner.Prepare(rewriteILike(query))
}

func (c *rewriteConn) PrepareContext(
	ctx context.Context,
	query string,
) (driver.Stmt, error) {
	if preparer, ok := c.inner.(driver.ConnPrepareContext); ok {
		return preparer.PrepareContext(ctx, rewriteILike(query))
	}

	return c.inner.Prepare(rewriteILike(query))
}

func (c *rewriteConn) Close() error { return c.inner.Close() }

func (c *rewriteConn) Begin() (driver.Tx, error) { //nolint:staticcheck // required by driver.Conn
	return c.inner.Begin() //nolint:staticcheck // delegating to the wrapped driver
}

func (c *rewriteConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if beginner, ok := c.inner.(driver.ConnBeginTx); ok {
		return beginner.BeginTx(ctx, opts)
	}

	return c.inner.Begin() //nolint:staticcheck // fallback for drivers without BeginTx
}

func (c *rewriteConn) Ping(ctx context.Context) error {
	if pinger, ok := c.inner.(driver.Pinger); ok {
		return pinger.Ping(ctx)
	}

	return nil
}

func (c *rewriteConn) ResetSession(ctx context.Context) error {
	if resetter, ok := c.inner.(driver.SessionResetter); ok {
		return resetter.ResetSession(ctx)
	}

	return nil
}

func (c *rewriteConn) IsValid() bool {
	if validator, ok := c.inner.(driver.Validator); ok {
		return validator.IsValid()
	}

	return true
}
