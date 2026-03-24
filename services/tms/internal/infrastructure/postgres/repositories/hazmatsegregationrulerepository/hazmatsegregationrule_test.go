package hazmatsegregationrulerepository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/zap"
)

func TestListActiveByTenant_ReturnsOnlyTenantScopedActiveRules(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)

	bunDB := bun.NewDB(db, pgdialect.New())
	t.Cleanup(func() {
		mock.ExpectClose()
		require.NoError(t, bunDB.Close())
	})

	repo := &repository{
		db: postgres.NewTestConnection(bunDB),
		l:  zap.NewNop(),
	}

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	mock.ExpectQuery(`SELECT .* FROM "hazmat_segregation_rules" AS "hsr" .*organization_id.*business_unit_id.*status.*ORDER BY`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "organization_id", "business_unit_id", "status", "name", "class_a", "class_b", "segregation_type",
		}).AddRow(
			pulid.MustNew("hsr_"), orgID, buID, domaintypes.StatusActive, "Rule A", "Class1", "Class3", "Prohibited",
		))

	entities, err := repo.ListActiveByTenant(t.Context(), pagination.TenantInfo{
		OrgID: orgID,
		BuID:  buID,
	})

	require.NoError(t, err)
	require.Len(t, entities, 1)
	assert.Equal(t, "Rule A", entities[0].Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}
