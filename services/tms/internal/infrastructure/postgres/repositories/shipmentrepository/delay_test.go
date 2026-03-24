package shipmentrepository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDelayedShipments_UsesTenantAndThreshold(t *testing.T) {
	t.Parallel()

	repo, mock := newCancelTestRepository(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	shipmentID := pulid.MustNew("shp_")

	mock.ExpectQuery(`WITH "stop_cte" AS .*COALESCE\(stp\.scheduled_window_end, stp\.scheduled_window_start\) \+ .* < .*"move_cte" AS .*sp\.organization_id = .*sp\.business_unit_id = .*sp\.status NOT IN .*`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "business_unit_id", "status"}).
			AddRow(shipmentID, orgID, buID, shipment.StatusAssigned))

	entities, err := repo.GetDelayedShipments(t.Context(), &repositories.GetDelayedShipmentsRequest{
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
	}, 15)

	require.NoError(t, err)
	require.Len(t, entities, 1)
	assert.Equal(t, shipmentID, entities[0].ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDelayShipments_UpdatesEligibleShipments(t *testing.T) {
	t.Parallel()

	repo, mock := newCancelTestRepository(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	shipmentID := pulid.MustNew("shp_")

	mock.ExpectBegin()
	mock.ExpectQuery(`WITH "stop_cte" AS .*"move_cte" AS .*sp\.organization_id = .*sp\.business_unit_id = .*`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "business_unit_id", "status"}).
			AddRow(shipmentID, orgID, buID, shipment.StatusAssigned))
	mock.ExpectExec(`UPDATE .*shipments.*status = .*updated_at = .*sp\.id IN .*`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	entities, err := repo.DelayShipments(t.Context(), &repositories.DelayShipmentsRequest{
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
	}, 30)

	require.NoError(t, err)
	require.Len(t, entities, 1)
	assert.Equal(t, shipment.StatusDelayed, entities[0].Status)
	require.NoError(t, mock.ExpectationsWereMet())
}
