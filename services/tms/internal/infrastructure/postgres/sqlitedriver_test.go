package postgres

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	t.Cleanup(func() { _ = db.Close() })

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
	t.Cleanup(func() { _ = db.Close() })

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
