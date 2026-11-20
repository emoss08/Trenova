//go:build integration

package documentmeter_test

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/documentmeter"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentstorage/documentstoragedb"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type storageRow struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	Bytes          int64
}

func TestStorageIsMeasuredForTheInstanceAndForOneOrganization(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	system := dbscope.WithSystem(ctx, "test")
	totals := documentstoragedb.Totals
	rows, err := keel.Into[storageRow](system, db,
		totals.AllTenants().Pick(totals.OrganizationID, totals.BusinessUnitID, totals.Bytes))
	require.NoError(t, err)
	var all int64
	for _, row := range rows {
		all += row.Bytes
	}

	reader := documentmeter.New(documentmeter.Params{DB: postgres.WrapDB(db)}).Reader
	levelFor := func(scope managedscope.Scope) int64 {
		t.Helper()
		level, levelErr := reader.Level(ctx, scope)
		require.NoError(t, levelErr)
		return level
	}

	assert.Equal(
		t,
		all,
		levelFor(managedscope.Instance()),
		"the instance adds up every organization",
	)
	for _, row := range rows {
		assert.Equal(t, row.Bytes, levelFor(managedscope.Organization(dbscope.Tenant{
			OrganizationID: row.OrganizationID,
			BusinessUnitID: row.BusinessUnitID,
		})), "an organization's scope reads its own total")
	}
	assert.Zero(t, levelFor(managedscope.Organization(dbscope.Tenant{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})), "and nothing of another's")
}
