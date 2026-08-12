package dbdialect_test

import (
	"testing"

	"github.com/emoss08/trenova/pkg/dbdialect"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	tests := []struct {
		input   string
		want    dbdialect.Kind
		wantErr bool
	}{
		{input: "", want: dbdialect.Postgres},
		{input: "postgres", want: dbdialect.Postgres},
		{input: "PostgreSQL", want: dbdialect.Postgres},
		{input: "  pg  ", want: dbdialect.Postgres},
		{input: "sqlite", want: dbdialect.SQLite},
		{input: "SQLite3", want: dbdialect.SQLite},
		{input: "mysql", wantErr: true},
		{input: "oracle", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := dbdialect.Parse(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				require.ErrorIs(t, err, dbdialect.ErrUnknownKind)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPostgresSupportsEverything(t *testing.T) {
	for _, capability := range allCapabilities() {
		assert.Truef(
			t,
			dbdialect.Postgres.Supports(capability),
			"postgres must support %s", capability,
		)
	}
}

func TestSQLiteRejectsPostgresOnlyCapabilities(t *testing.T) {
	for _, capability := range allCapabilities() {
		assert.Falsef(
			t,
			dbdialect.SQLite.Supports(capability),
			"sqlite must not claim %s", capability,
		)
	}
}

func TestRequireReturnsNotImplemented(t *testing.T) {
	require.NoError(t, dbdialect.Require(dbdialect.Postgres, dbdialect.CapPostGIS))

	err := dbdialect.Require(dbdialect.SQLite, dbdialect.CapPostGIS)
	require.Error(t, err)
	require.True(t, errortypes.IsNotImplementedError(err))

	assert.Equal(t, 501, errortypes.HTTPStatus(err))
	assert.Contains(t, err.Error(), "PostGIS")
	assert.Contains(t, err.Error(), "SQLite")
}

func TestUnsupportedCarriesDriverAndCapability(t *testing.T) {
	err := dbdialect.Unsupported(dbdialect.SQLite, dbdialect.CapFullTextSearch)

	assert.Equal(t, "sqlite", err.Driver)
	assert.Equal(t, string(dbdialect.CapFullTextSearch), err.Capability)
	assert.Equal(t, errortypes.ErrNotImplemented, err.Code)
}

func allCapabilities() []dbdialect.Capability {
	return []dbdialect.Capability{
		dbdialect.CapPostGIS,
		dbdialect.CapFullTextSearch,
		dbdialect.CapChangeDataCapture,
		dbdialect.CapExactDecimal,
		dbdialect.CapMaterializedView,
		dbdialect.CapAdvisoryLock,
		dbdialect.CapRowLocking,
		dbdialect.CapExclusionConstr,
		dbdialect.CapTrigram,
		dbdialect.CapConcurrentIndex,
		dbdialect.CapPartitionedTable,
	}
}
