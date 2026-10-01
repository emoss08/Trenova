package dbtx

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

type fakeConn struct {
	ports.DBConnection
	scoped bool
	opened []ports.TxOptions
}

func (f *fakeConn) ScopedTransactions() bool { return f.scoped }

func (f *fakeConn) WithTx(
	ctx context.Context,
	opts ports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) error {
	f.opened = append(f.opened, opts)
	return fn(ctx, bun.Tx{})
}

func TestHelpersRunDirectlyWhenScopedTransactionsAreOff(t *testing.T) {
	t.Parallel()

	conn := &fakeConn{}
	got, err := Read(t.Context(), conn, func(context.Context) (int, error) { return 7, nil })
	require.NoError(t, err)
	assert.Equal(t, 7, got)
	require.NoError(t, WriteErr(t.Context(), conn, func(context.Context) error { return nil }))
	assert.Empty(t, conn.opened)
	assert.False(t, Required(conn))
}

func TestHelpersOpenATransactionPerCallWhenScoped(t *testing.T) {
	t.Parallel()

	conn := &fakeConn{scoped: true}
	failure := errors.New("boom")

	got, err := Read(t.Context(), conn, func(context.Context) (string, error) { return "row", nil })
	require.NoError(t, err)
	assert.Equal(t, "row", got)

	_, err = Write(t.Context(), conn, func(context.Context) (*int, error) { return nil, failure })
	require.ErrorIs(t, err, failure)

	first, second, err := Read2(t.Context(), conn, func(context.Context) ([]int, int, error) {
		return []int{1}, 1, nil
	})
	require.NoError(t, err)
	assert.Equal(t, []int{1}, first)
	assert.Equal(t, 1, second)

	require.NoError(t, ReadErr(t.Context(), conn, func(context.Context) error { return nil }))
	_, _, err = Write2(t.Context(), conn, func(context.Context) (int, int, error) { return 0, 0, failure })
	require.ErrorIs(t, err, failure)

	assert.Equal(t, []ports.TxOptions{
		{ReadOnly: true},
		{ReadOnly: false},
		{ReadOnly: true},
		{ReadOnly: true},
		{ReadOnly: false},
	}, conn.opened)
	assert.True(t, Required(conn))
}
