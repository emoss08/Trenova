package postgres

import (
	"database/sql/driver"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// database/sql only falls back to Prepare when a connection advertises neither
// QueryerContext nor ExecerContext. Implementing either would let statements
// reach SQLite without passing through rewriteILike.
func TestRewriteConnHasNoFastPath(t *testing.T) {
	var conn any = &rewriteConn{}

	_, isQueryer := conn.(driver.QueryerContext)
	assert.False(t, isQueryer, "rewriteConn must not implement QueryerContext")

	_, isExecer := conn.(driver.ExecerContext)
	assert.False(t, isExecer, "rewriteConn must not implement ExecerContext")
}

func TestRewriteILike(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  string
	}{
		{
			name:  "leaves a query without the operator alone",
			query: `SELECT * FROM t WHERE a = ?`,
			want:  `SELECT * FROM t WHERE a = ?`,
		},
		{
			name:  "rewrites the operator",
			query: `SELECT * FROM t WHERE name ILIKE ?`,
			want:  `SELECT * FROM t WHERE name LIKE ?`,
		},
		{
			name:  "rewrites regardless of case",
			query: `SELECT * FROM t WHERE name iLiKe ?`,
			want:  `SELECT * FROM t WHERE name LIKE ?`,
		},
		{
			name:  "rewrites NOT ILIKE",
			query: `SELECT * FROM t WHERE name NOT ILIKE ?`,
			want:  `SELECT * FROM t WHERE name NOT LIKE ?`,
		},
		{
			name:  "rewrites every occurrence",
			query: `SELECT * FROM t WHERE a ILIKE ? OR b ILIKE ?`,
			want:  `SELECT * FROM t WHERE a LIKE ? OR b LIKE ?`,
		},
		{
			name:  "leaves string literals untouched",
			query: `SELECT * FROM t WHERE note = ' ILIKE ' AND name ILIKE ?`,
			want:  `SELECT * FROM t WHERE note = ' ILIKE ' AND name LIKE ?`,
		},
		{
			name:  "leaves quoted identifiers untouched",
			query: `SELECT "ilike" FROM t WHERE name ILIKE ?`,
			want:  `SELECT "ilike" FROM t WHERE name LIKE ?`,
		},
		{
			name:  "does not touch a column whose name merely contains the word",
			query: `SELECT * FROM t WHERE ilike_count > ? AND unilike > ?`,
			want:  `SELECT * FROM t WHERE ilike_count > ? AND unilike > ?`,
		},
		{
			name:  "leaves backtick quoted identifiers untouched",
			query: "SELECT `ilike` FROM t WHERE name ILIKE ?",
			want:  "SELECT `ilike` FROM t WHERE name LIKE ?",
		},
		{
			name:  "leaves bracket quoted identifiers untouched",
			query: `SELECT [ilike] FROM t WHERE name ILIKE ?`,
			want:  `SELECT [ilike] FROM t WHERE name LIKE ?`,
		},
		{
			name:  "treats a dollar sign as part of an identifier",
			query: `SELECT foo$ilike, $ilike FROM t WHERE name ILIKE ?`,
			want:  `SELECT foo$ilike, $ilike FROM t WHERE name LIKE ?`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, rewriteILike(tt.query))
		})
	}
}

func TestSQLiteDriverExecutesILike(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "ilike.db")

	db, err := openSQLiteDB("file:" + path)
	require.NoError(t, err)
	t.Cleanup(func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Errorf("failed to close database: %v", closeErr)
		}
	})

	_, err = db.ExecContext(ctx, `CREATE TABLE people (name TEXT)`)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `INSERT INTO people (name) VALUES ('Alice'), ('bob')`)
	require.NoError(t, err)

	var name string
	require.NoError(
		t,
		db.QueryRowContext(ctx, `SELECT name FROM people WHERE name ILIKE ?`, "alice").Scan(&name),
		"ILIKE must survive the rewrite and execute on SQLite",
	)
	assert.Equal(t, "Alice", name, "ILIKE must stay case-insensitive after the rewrite")

	require.NoError(
		t,
		db.QueryRowContext(ctx, `SELECT name FROM people WHERE name ILIKE ?`, "BOB").Scan(&name),
	)
	assert.Equal(t, "bob", name)
}

func TestSQLiteDriverPreservesLiteralContainingILike(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "literal.db")

	db, err := openSQLiteDB("file:" + path)
	require.NoError(t, err)
	t.Cleanup(func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Errorf("failed to close database: %v", closeErr)
		}
	})

	_, err = db.ExecContext(ctx, `CREATE TABLE notes (body TEXT)`)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `INSERT INTO notes (body) VALUES ('uses ILIKE here')`)
	require.NoError(t, err)

	var body string
	require.NoError(
		t,
		db.QueryRowContext(ctx, `SELECT body FROM notes WHERE body ILIKE ?`, "%ilike%").Scan(&body),
	)
	assert.Equal(
		t,
		"uses ILIKE here",
		body,
		"a stored value containing ILIKE must not be rewritten",
	)
}
