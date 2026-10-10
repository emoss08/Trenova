package dbtx

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type controlRow struct{ name string }

type fakeControlTable struct {
	row       *controlRow
	inserts   int
	insertErr error
}

func (f *fakeControlTable) spec() GetOrCreateSpec[*controlRow] {
	return GetOrCreateSpec[*controlRow]{
		Find: func(context.Context) (*controlRow, error) {
			if f.row == nil {
				return nil, sql.ErrNoRows
			}

			return f.row, nil
		},
		Create: func(context.Context) error {
			f.inserts++
			if f.insertErr != nil {
				return f.insertErr
			}
			f.row = &controlRow{name: "inserted"}

			return nil
		},
		Default: func() *controlRow { return &controlRow{name: "default"} },
	}
}

func TestGetOrCreate_AReadOnlyCallerGetsTheDefaultWithoutAWrite(t *testing.T) {
	t.Parallel()

	table := &fakeControlTable{}
	got, err := GetOrCreate(ports.WithReadOnly(t.Context()), outsideTx{}, table.spec())

	require.NoError(t, err)
	assert.Equal(t, "default", got.name)
	assert.Zero(t, table.inserts)
}

func TestGetOrCreate_AnExistingRowIsReadNotInserted(t *testing.T) {
	t.Parallel()

	table := &fakeControlTable{row: &controlRow{name: "kept"}}
	got, err := GetOrCreate(t.Context(), outsideTx{}, table.spec())

	require.NoError(t, err)
	assert.Equal(t, "kept", got.name)
	assert.Zero(t, table.inserts)
}

func TestGetOrCreate_AMissingRowIsInsertedThenRead(t *testing.T) {
	t.Parallel()

	table := &fakeControlTable{}
	got, err := GetOrCreate(t.Context(), outsideTx{}, table.spec())

	require.NoError(t, err)
	assert.Equal(t, "inserted", got.name)
	assert.Equal(t, 1, table.inserts)
}

func TestGetOrCreate_AReadOnlyRefusalOfTheInsertReturnsTheDefault(t *testing.T) {
	t.Parallel()

	refused := fmt.Errorf("create default: %w", &pgconn.PgError{Code: pgerrcode.ReadOnlySQLTransaction})
	table := &fakeControlTable{insertErr: refused}
	got, err := GetOrCreate(t.Context(), outsideTx{}, table.spec())

	require.NoError(t, err)
	assert.Equal(t, "default", got.name)
}

func TestGetOrCreate_OtherFailuresAreReturned(t *testing.T) {
	t.Parallel()

	broken := &fakeControlTable{insertErr: assert.AnError}
	_, err := GetOrCreate(t.Context(), outsideTx{}, broken.spec())
	require.ErrorIs(t, err, assert.AnError)

	unreadable := GetOrCreateSpec[*controlRow]{
		Find:    func(context.Context) (*controlRow, error) { return nil, assert.AnError },
		Create:  func(context.Context) error { t.Fatal("no insert after a failed read"); return nil },
		Default: func() *controlRow { return nil },
	}
	_, err = GetOrCreate(t.Context(), outsideTx{}, unreadable)
	require.ErrorIs(t, err, assert.AnError)
}

func TestGetOrCreate_TheRepositoryWordsAFailure(t *testing.T) {
	t.Parallel()

	spec := (&fakeControlTable{insertErr: assert.AnError}).spec()
	spec.MapError = func(err error) error { return fmt.Errorf("control busy: %w", err) }

	_, err := GetOrCreate(t.Context(), outsideTx{}, spec)
	require.ErrorIs(t, err, assert.AnError)
	assert.Contains(t, err.Error(), "control busy")

	table := &fakeControlTable{row: &controlRow{name: "kept"}}
	found := table.spec()
	found.MapError = func(error) error { t.Fatal("no failure to word"); return nil }
	got, err := GetOrCreate(t.Context(), outsideTx{}, found)
	require.NoError(t, err)
	assert.Equal(t, "kept", got.name)
}
