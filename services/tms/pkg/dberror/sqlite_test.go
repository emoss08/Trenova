package dberror_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

func TestSQLiteUniqueViolationIsClassified(t *testing.T) {
	db := newDB(t)

	_, err := db.Exec(`CREATE TABLE widgets (id INTEGER PRIMARY KEY, code TEXT UNIQUE)`)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO widgets (id, code) VALUES (1, 'abc')`)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO widgets (id, code) VALUES (2, 'abc')`)
	require.Error(t, err)

	assert.True(t, dberror.IsUniqueConstraintViolation(err))
	assert.True(t, dberror.IsConstraintViolation(err))
	assert.Equal(t, "widgets.code", dberror.ExtractConstraintName(err))
}

func TestSQLiteNotNullViolationIsClassified(t *testing.T) {
	db := newDB(t)

	_, err := db.Exec(`CREATE TABLE widgets (id INTEGER PRIMARY KEY, code TEXT NOT NULL)`)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO widgets (id, code) VALUES (1, NULL)`)
	require.Error(t, err)

	assert.True(t, dberror.IsNotNullConstraintViolation(err))
}

func TestSQLiteForeignKeyViolationIsClassified(t *testing.T) {
	db := newDB(t)

	_, err := db.Exec(`PRAGMA foreign_keys = ON`)
	require.NoError(t, err)

	_, err = db.Exec(`CREATE TABLE parents (id INTEGER PRIMARY KEY)`)
	require.NoError(t, err)

	_, err = db.Exec(
		`CREATE TABLE children (id INTEGER PRIMARY KEY, parent_id INTEGER REFERENCES parents(id))`,
	)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO children (id, parent_id) VALUES (1, 999)`)
	require.Error(t, err)

	assert.True(t, dberror.IsForeignKeyConstraintViolation(err))
}

func TestSQLiteCheckViolationIsClassified(t *testing.T) {
	db := newDB(t)

	_, err := db.Exec(`CREATE TABLE widgets (id INTEGER PRIMARY KEY, qty INTEGER CHECK (qty > 0))`)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO widgets (id, qty) VALUES (1, 0)`)
	require.Error(t, err)

	assert.True(t, dberror.IsCheckConstraintViolation(err))
}

func TestNonDriverErrorsAreNotMisclassified(t *testing.T) {
	assert.False(t, dberror.IsUniqueConstraintViolation(sql.ErrNoRows))
	assert.False(t, dberror.IsConstraintViolation(sql.ErrNoRows))
	assert.Empty(t, dberror.ExtractCode(sql.ErrNoRows))
	assert.True(t, dberror.IsNotFoundError(sql.ErrNoRows))
}

func newDB(t *testing.T) *sql.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "dberror-test.db")

	db, err := sql.Open("sqlite", "file:"+path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	return db
}
