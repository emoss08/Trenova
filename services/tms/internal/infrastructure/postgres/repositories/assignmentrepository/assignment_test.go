package assignmentrepository

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

func newTestRepository(t *testing.T) (*repository, sqlmock.Sqlmock) {
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

func TestGetByMoveID_IgnoresArchivedAssignments(t *testing.T) {
	t.Parallel()

	repo, mock := newTestRepository(t)
	moveID := pulid.MustNew("sm_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	mock.ExpectQuery(`SELECT .*FROM "assignments" AS "a".*a\.shipment_move_id = .*a\.organization_id = .*a\.business_unit_id = .*a\.archived_at IS NULL`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"organization_id",
			"business_unit_id",
			"shipment_move_id",
			"primary_worker_id",
			"tractor_id",
			"status",
			"version",
			"created_at",
			"updated_at",
		}).AddRow(
			pulid.MustNew("asn_"),
			tenantInfo.OrgID,
			tenantInfo.BuID,
			moveID,
			pulid.MustNew("wrk_"),
			pulid.MustNew("trc_"),
			shipment.AssignmentStatusNew,
			1,
			1,
			1,
		))

	entity, err := repo.GetByMoveID(t.Context(), tenantInfo, moveID)

	require.NoError(t, err)
	require.NotNil(t, entity)
	assert.Equal(t, moveID, entity.ShipmentMoveID)
}

func TestList_ExcludesArchivedAssignments(t *testing.T) {
	t.Parallel()

	repo, mock := newTestRepository(t)
	req := &repositories.ListAssignmentsRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: pagination.TenantInfo{
				OrgID: pulid.MustNew("org_"),
				BuID:  pulid.MustNew("bu_"),
			},
			Pagination: pagination.Info{
				Limit:  10,
				Offset: 0,
			},
		},
	}

	mock.ExpectQuery(`SELECT count\(\*\) FROM "assignments" AS "a".*a\.organization_id = .*a\.business_unit_id = .*a\.archived_at IS NULL`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT .*FROM "assignments" AS "a".*a\.organization_id = .*a\.business_unit_id = .*a\.archived_at IS NULL.*ORDER BY "a"\."created_at" DESC LIMIT 10`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"organization_id",
			"business_unit_id",
			"shipment_move_id",
			"primary_worker_id",
			"tractor_id",
			"status",
			"version",
			"created_at",
			"updated_at",
		}).AddRow(
			pulid.MustNew("asn_"),
			req.Filter.TenantInfo.OrgID,
			req.Filter.TenantInfo.BuID,
			pulid.MustNew("sm_"),
			pulid.MustNew("wrk_"),
			pulid.MustNew("trc_"),
			shipment.AssignmentStatusNew,
			1,
			1,
			1,
		))

	result, err := repo.List(t.Context(), req)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Items, 1)
}

func TestUnassign_ArchivesAssignment(t *testing.T) {
	t.Parallel()

	repo, mock := newTestRepository(t)
	entity := &shipment.Assignment{
		ID:              pulid.MustNew("asn_"),
		OrganizationID:  pulid.MustNew("org_"),
		BusinessUnitID:  pulid.MustNew("bu_"),
		ShipmentMoveID:  pulid.MustNew("sm_"),
		PrimaryWorkerID: pulid.Must("wrk_"),
		TractorID:       pulid.Must("trc_"),
		Status:          shipment.AssignmentStatusNew,
		Version:         3,
	}

	mock.ExpectExec(`UPDATE .*assignments.*SET .*status.*archived_at.*version.*updated_at.*WHERE .*archived_at IS NULL.*`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT .*FROM "assignments" AS "a".*a\.id = .*a\.organization_id = .*a\.business_unit_id = .*`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"organization_id",
			"business_unit_id",
			"shipment_move_id",
			"primary_worker_id",
			"tractor_id",
			"status",
			"archived_at",
			"version",
			"created_at",
			"updated_at",
		}).AddRow(
			entity.ID,
			entity.OrganizationID,
			entity.BusinessUnitID,
			entity.ShipmentMoveID,
			entity.PrimaryWorkerID,
			entity.TractorID,
			shipment.AssignmentStatusCanceled,
			1710000000,
			4,
			1,
			2,
		))

	updated, err := repo.Unassign(t.Context(), entity)

	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, shipment.AssignmentStatusCanceled, updated.Status)
	require.NotNil(t, updated.ArchivedAt)
}
