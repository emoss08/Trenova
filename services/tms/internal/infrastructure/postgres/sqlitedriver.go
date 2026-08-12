package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strings"
	"sync"
)

const sqliteRewriteDriverName = "sqlite-trenova"

var registerSQLiteRewriteDriver sync.Once

func rewriteILike(query string) string {
	if !containsFold(query, "ilike") {
		return query
	}

	var (
		out    strings.Builder
		closer byte
		i      int
	)

	out.Grow(len(query))

	for i < len(query) {
		ch := query[i]

		switch {
		case closer != 0:
			if ch == closer {
				closer = 0
			}
		case ch == '\'' || ch == '"' || ch == '`':
			closer = ch
		case ch == '[':
			closer = ']'
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
	return b == '_' || b == '$' ||
		(b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9')
}

func containsFold(haystack, needle string) bool {
	if needle == "" {
		return true
	}

	first := needle[0] | 0x20

	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i]|0x20 != first {
			continue
		}

		if strings.EqualFold(haystack[i:i+len(needle)], needle) {
			return true
		}
	}

	return false
}

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
