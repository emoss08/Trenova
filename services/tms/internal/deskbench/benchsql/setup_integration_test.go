//go:build integration

package benchsql

import (
	"testing"

	"github.com/emoss08/trenova/internal/deskbench"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/require"
)

func TestRunStatements_RunsOnlyOnAConnectionRowLevelSecurityConfines(t *testing.T) {
	ctx, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)
	data := seedtest.SeedFullTestData(t, ctx, admin)

	superuser, err := readConfinement(ctx, admin)
	require.NoError(t, err)
	require.ErrorIs(t, superuser.err(), ErrSetupUnconfined, "the admin role sees every org")

	database := &Database{db: dbtest.NewRLSConnection(t, admin, dsn)}
	session := &deskbench.Session{Tenant: pagination.TenantInfo{
		OrgID:  data.Organization.ID,
		BuID:   data.BusinessUnit.ID,
		UserID: data.User.ID,
	}}
	require.NoError(t, database.RunStatements(t.Context(), session, []string{
		"UPDATE shipments SET bol = bol WHERE pro_number = 'NONE'",
	}), "the application role is confined, so its setup runs")
}
