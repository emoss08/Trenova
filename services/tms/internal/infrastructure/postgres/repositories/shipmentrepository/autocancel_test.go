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

func TestGetAutoCancelableShipments_UsesTenantAndThreshold(t *testing.T) {
	t.Parallel()

	repo, mock := newCancelTestRepository(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	shipmentID := pulid.MustNew("shp_")

	mock.ExpectQuery(`SELECT .*FROM "shipments" AS "sp".*sp\.organization_id = .*sp\.business_unit_id = .*sp\.status = .*sp\.created_at <= .*`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "business_unit_id", "status"}).
			AddRow(shipmentID, orgID, buID, shipment.StatusNew))

	entities, err := repo.GetAutoCancelableShipments(t.Context(), &repositories.GetAutoCancelableShipmentsRequest{
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
	}, 30)

	require.NoError(t, err)
	require.Len(t, entities, 1)
	assert.Equal(t, shipmentID, entities[0].ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAutoCancelShipments_UpdatesEligibleShipmentsAndComponents(t *testing.T) {
	t.Parallel()

	repo, mock := newCancelTestRepository(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	shipmentID := pulid.MustNew("shp_")
	moveID := pulid.MustNew("sm_")

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT .*FROM "shipments" AS "sp".*sp\.organization_id = .*sp\.business_unit_id = .*sp\.status = .*sp\.created_at <= .*`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "business_unit_id", "status"}).
			AddRow(shipmentID, orgID, buID, shipment.StatusNew))
	mock.ExpectExec(`UPDATE .*shipments.*status = .*canceled_at = .*canceled_by_id = NULL.*cancel_reason = .*updated_at = .*sp\.id = .*`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT "sm"\."id" FROM "shipment_moves" AS "sm" WHERE \(sm\.shipment_id = .*\)`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(moveID))
	mock.ExpectExec(`UPDATE .*shipment_moves.*`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE .*assignments.*`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE .*stops.*`).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	entities, err := repo.AutoCancelShipments(t.Context(), &repositories.AutoCancelShipmentsRequest{
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
	}, 30)

	require.NoError(t, err)
	require.Len(t, entities, 1)
	assert.Equal(t, shipment.StatusCanceled, entities[0].Status)
	assert.Equal(t, autoCancelReason, entities[0].CancelReason)
	require.NoError(t, mock.ExpectationsWereMet())
}
