package repositories_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	repoports "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/test/testutils"
	"github.com/stretchr/testify/require"
)

func TestAssignmentRepository(t *testing.T) {
	// Fixtures
	org := ts.Fixture.MustRow("Organization.trenova").(*organization.Organization)
	bu := ts.Fixture.MustRow("BusinessUnit.trenova").(*businessunit.BusinessUnit)
	worker1 := ts.Fixture.MustRow("Worker.worker_1").(*worker.Worker)
	worker2 := ts.Fixture.MustRow("Worker.worker_2").(*worker.Worker)
	trt := ts.Fixture.MustRow("Tractor.tractor_1").(*tractor.Tractor)
	trl := ts.Fixture.MustRow("Trailer.test_trailer").(*trailer.Trailer)

	// Shipment 1
	smt := ts.Fixture.MustRow("Shipment.test_shipment").(*shipment.Shipment)
	move := ts.Fixture.MustRow("ShipmentMove.test_shipment_move").(*shipment.ShipmentMove)

	// Shipment 2
	smt2 := ts.Fixture.MustRow("Shipment.test_shipment").(*shipment.Shipment)
	move2 := ts.Fixture.MustRow("ShipmentMove.test_shipment_move_8").(*shipment.ShipmentMove)

	// Logger
	log := logger.NewLogger(testutils.NewTestConfig())

	// Repositories
	moveRepo := repositories.NewShipmentMoveRepository(repositories.ShipmentMoveRepositoryParams{
		DB:     ts.DB,
		Logger: log,
	})
	proNumberRepo := repositories.NewProNumberRepository(repositories.ProNumberRepositoryParams{
		DB:     ts.DB,
		Logger: log,
	})

	shipmentCommodityRepo := repositories.NewShipmentCommodityRepository(repositories.ShipmentCommodityRepositoryParams{
		DB:     ts.DB,
		Logger: log,
	})

	shipmentRepo := repositories.NewShipmentRepository(repositories.ShipmentRepositoryParams{
		DB:                          ts.DB,
		ProNumberRepo:               proNumberRepo,
		ShipmentCommodityRepository: shipmentCommodityRepo,
		Logger:                      log,
	})
	repo := repositories.NewAssignmentRepository(repositories.AssignmentRepositoryParams{
		DB:           ts.DB,
		ShipmentRepo: shipmentRepo,
		MoveRepo:     moveRepo,
		Logger:       log,
	})

	t.Run("get assignment by id", func(t *testing.T) {
		assign := &shipment.Assignment{
			ShipmentMoveID:    move.ID,
			BusinessUnitID:    bu.ID,
			OrganizationID:    org.ID,
			PrimaryWorkerID:   worker1.ID,
			SecondaryWorkerID: &worker2.ID,
			TractorID:         trt.ID,
			TrailerID:         trl.ID,
		}

		created, err := repo.SingleAssign(ctx, assign)
		require.NoError(t, err)
		require.NotNil(t, created)

		result, err := repo.GetByID(ctx, repoports.GetAssignmentByIDOptions{
			ID:    created.ID,
			OrgID: org.ID,
			BuID:  bu.ID,
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, created.ID, result.ID)
		require.Equal(t, created.ShipmentMoveID, result.ShipmentMoveID)
	})

	t.Run("get assignment by invalid id", func(t *testing.T) {
		result, err := repo.GetByID(ctx, repoports.GetAssignmentByIDOptions{
			ID:    "invalid-id",
			OrgID: org.ID,
			BuID:  bu.ID,
		})

		require.Error(t, err)
		require.Nil(t, result)
	})

	t.Run("single assign creates partial assignment with multiple moves", func(t *testing.T) {
		assign := &shipment.Assignment{
			ShipmentMoveID:    move.ID,
			OrganizationID:    org.ID,
			BusinessUnitID:    bu.ID,
			PrimaryWorkerID:   worker1.ID,
			SecondaryWorkerID: &worker2.ID,
			TractorID:         trt.ID,
			TrailerID:         trl.ID,
		}

		result, err := repo.SingleAssign(ctx, assign)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotEmpty(t, result.ID)
		require.Equal(t, move.ID, result.ShipmentMoveID)

		// Verify move status was updated
		updatedMove, err := moveRepo.GetByID(ctx, repoports.GetMoveByIDOptions{
			MoveID: move.ID,
			OrgID:  org.ID,
			BuID:   bu.ID,
		})
		require.NoError(t, err)
		require.Equal(t, shipment.MoveStatusAssigned, updatedMove.Status)

		// Verify shipment status was updated
		updatedShipment, err := shipmentRepo.GetByID(ctx, &repoports.GetShipmentByIDOptions{
			ID:    smt.ID,
			OrgID: org.ID,
			BuID:  bu.ID,
			ShipmentOptions: repoports.ShipmentOptions{
				ExpandShipmentDetails: true,
			},
		})
		require.NoError(t, err)
		require.Equal(t, shipment.StatusPartiallyAssigned, updatedShipment.Status)
	})

	t.Run("single assign creates assignment with single move assignment", func(t *testing.T) {
		assign := &shipment.Assignment{
			ShipmentMoveID:    move2.ID,
			OrganizationID:    org.ID,
			BusinessUnitID:    bu.ID,
			PrimaryWorkerID:   worker1.ID,
			SecondaryWorkerID: &worker2.ID,
			TractorID:         trt.ID,
			TrailerID:         trl.ID,
		}

		result, err := repo.SingleAssign(ctx, assign)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotEmpty(t, result.ID)
		require.Equal(t, move2.ID, result.ShipmentMoveID)

		// Verify move status was updated
		updatedMove, err := moveRepo.GetByID(ctx, repoports.GetMoveByIDOptions{
			MoveID: move2.ID,
			OrgID:  org.ID,
			BuID:   bu.ID,
		})
		require.NoError(t, err)
		require.Equal(t, shipment.MoveStatusAssigned, updatedMove.Status)

		// Verify shipment status was updated
		updatedShipment, err := shipmentRepo.GetByID(ctx, &repoports.GetShipmentByIDOptions{
			ID:    smt2.ID,
			OrgID: org.ID,
			BuID:  bu.ID,
			ShipmentOptions: repoports.ShipmentOptions{
				ExpandShipmentDetails: true,
			},
		})
		require.NoError(t, err)
		require.Equal(t, shipment.StatusPartiallyAssigned, updatedShipment.Status)
	})

	t.Run("bulk assign", func(t *testing.T) {
		req := &repoports.AssignmentRequest{
			ShipmentID:        smt.ID,
			OrgID:             org.ID,
			BuID:              bu.ID,
			PrimaryWorkerID:   worker1.ID,
			SecondaryWorkerID: &worker2.ID,
			TractorID:         trt.ID,
			TrailerID:         trl.ID,
		}

		results, err := repo.BulkAssign(ctx, req)
		require.NoError(t, err)
		require.NotEmpty(t, results)

		// Verify all moves are assigned
		for _, result := range results {
			require.NotEmpty(t, result.ID)
			require.Equal(t, worker1.ID, result.PrimaryWorkerID)
		}

		// Verify shipment status was updated
		updatedShipment, err := shipmentRepo.GetByID(ctx, &repoports.GetShipmentByIDOptions{
			ID:    smt.ID,
			OrgID: org.ID,
			BuID:  bu.ID,
		})
		require.NoError(t, err)
		require.Equal(t, shipment.StatusAssigned, updatedShipment.Status)
	})

	t.Run("reassign", func(t *testing.T) {
		// First create an assignment
		assign := &shipment.Assignment{
			ShipmentMoveID:    move.ID,
			OrganizationID:    org.ID,
			BusinessUnitID:    bu.ID,
			PrimaryWorkerID:   worker1.ID,
			SecondaryWorkerID: &worker2.ID,
			TractorID:         trt.ID,
			TrailerID:         trl.ID,
		}

		created, err := repo.SingleAssign(ctx, assign)
		require.NoError(t, err)
		require.NotNil(t, created)

		// Now reassign with different worker
		created.PrimaryWorkerID = worker2.ID
		created.SecondaryWorkerID = &worker1.ID

		updated, err := repo.Reassign(ctx, created)
		require.NoError(t, err)
		require.NotNil(t, updated)
		require.Equal(t, worker2.ID, updated.PrimaryWorkerID)
		require.Equal(t, worker1.ID, *updated.SecondaryWorkerID)
	})
}
