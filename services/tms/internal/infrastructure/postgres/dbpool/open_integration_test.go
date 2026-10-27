//go:build integration

package dbpool_test

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbpool"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeelConfigReadsDecimalsInBinary(t *testing.T) {
	ctx, db, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	conn, err := pgx.Connect(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close(ctx) })
	require.NoError(t, dbpool.KeelConfig(nil, nil).AfterConnect(ctx, conn))
	numericType, ok := conn.TypeMap().TypeForOID(pgtype.NumericOID)
	require.True(t, ok)
	assert.NotEqual(t, pgtype.NumericCodec{}, numericType.Codec)

	total, err := keel.QueryValue[decimal.Decimal](ctx, db, "SELECT 1250.5::numeric(19,4)")
	require.NoError(t, err)
	assert.Equal(t, "1250.5", total.String())
	assert.Equal(t, int32(-4), total.Exponent())

	discount, err := keel.QueryValue[decimal.NullDecimal](ctx, db, "SELECT NULL::numeric(19,4)")
	require.NoError(t, err)
	assert.False(t, discount.Valid)

	rates, err := keel.QueryValue[[]decimal.Decimal](ctx, db, "SELECT $1::numeric[]",
		[]decimal.Decimal{decimal.RequireFromString("2.15"), decimal.NewFromInt(100)})
	require.NoError(t, err)
	require.Len(t, rates, 2)
	assert.True(t, rates[0].Equal(decimal.RequireFromString("2.15")))
	assert.Equal(t, "100", rates[1].String())
}
