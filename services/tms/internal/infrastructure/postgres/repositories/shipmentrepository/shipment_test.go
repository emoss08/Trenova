package shipmentrepository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/zap"
)

func newShipmentListTestRepository(t *testing.T) (*repository, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	mock.MatchExpectationsInOrder(false)

	bunDB := bun.NewDB(db, pgdialect.New())
	t.Cleanup(func() {
		require.NoError(t, mock.ExpectationsWereMet())
	})

	return &repository{
		db: postgres.NewTestConnection(bunDB),
		l:  zap.NewNop(),
	}, mock
}

func TestGetUnassigned_ExcludesShipmentsWithActiveAssignments(t *testing.T) {
	t.Parallel()

	repo, mock := newShipmentListTestRepository(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	shipmentID := pulid.MustNew("shp_")

	unassignedPredicate := `(?s)NOT EXISTS .*FROM "shipment_moves" AS "sm".*JOIN "assignments" AS "a".*a\.shipment_move_id = sm\.id.*a\.organization_id = sm\.organization_id.*a\.business_unit_id = sm\.business_unit_id.*a\.archived_at IS NULL.*sm\.shipment_id = sp\.id.*sm\.organization_id = sp\.organization_id.*sm\.business_unit_id = sp\.business_unit_id`

	mock.ExpectQuery(`SELECT count\(\*\) FROM "shipments" AS "sp".*sp\.organization_id = .*sp\.business_unit_id = .*` + unassignedPredicate).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT .*FROM "shipments" AS "sp".*sp\.organization_id = .*sp\.business_unit_id = .*` + unassignedPredicate + `.*ORDER BY "sp"\."created_at" DESC LIMIT 10`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"business_unit_id",
			"organization_id",
			"service_type_id",
			"customer_id",
			"formula_template_id",
			"status",
			"pro_number",
			"rating_unit",
		}).AddRow(
			shipmentID,
			buID,
			orgID,
			pulid.MustNew("svc_"),
			pulid.MustNew("cus_"),
			pulid.MustNew("fmt_"),
			shipment.StatusNew,
			"PRO-1",
			1,
		))

	result, err := repo.GetUnassigned(t.Context(), &repositories.GetUnassignedShipmentsRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: pagination.TenantInfo{
				OrgID: orgID,
				BuID:  buID,
			},
			Pagination: pagination.Info{
				Limit:  10,
				Offset: 0,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 1, result.Total)
	require.Len(t, result.Items, 1)
	assert.Equal(t, shipmentID, result.Items[0].ID)
}
