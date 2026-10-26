//go:build integration

package dbcli

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbpool"
	"github.com/emoss08/trenova/internal/keeltables"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/stretchr/testify/require"
)

func TestMarkMigratedTablesBumpsEveryTableForNoTenant(t *testing.T) {
	ctx, db, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	tables := keeltables.All()[:3]
	config := func() (keel.Config, error) {
		kc := dbpool.KeelConfig(nil, nil)
		kc.DSN = dsn
		kc.Tables = tables
		kc.DataVersions = keel.DataVersionsConfig{Enabled: true}
		return kc, nil
	}

	require.NoError(t, markMigratedTables(config)(ctx, nil))

	for _, table := range tables {
		version, err := keel.QueryValue[int64](ctx, db,
			"SELECT version FROM keel_data_versions WHERE tenant_key = '' AND table_name = $1",
			table.Name)
		require.NoError(t, err, table.Name)
		require.Positive(t, version, table.Name)
	}

	_, err := keel.QueryValue[int64](ctx, db,
		"SELECT version FROM keel_data_versions WHERE tenant_key = '' AND table_name = $1",
		keeltables.All()[3].Name)
	require.ErrorIs(t, err, keel.ErrNotFound)
}
