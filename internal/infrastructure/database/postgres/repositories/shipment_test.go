package repositories_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/services/calculator"
	"github.com/stretchr/testify/require"

	repoports "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/test/testutils"
)

func TestShipmentRepository(t *testing.T) {
	org := ts.Fixture.MustRow("Organization.trenova").(*organization.Organization)
	bu := ts.Fixture.MustRow("BusinessUnit.trenova").(*businessunit.BusinessUnit)
	smt := ts.Fixture.MustRow("Shipment.test_shipment").(*shipment.Shipment)
	serviceType := ts.Fixture.MustRow("ServiceType.std_service_type").(*servicetype.ServiceType)
	shipmentType := ts.Fixture.MustRow("ShipmentType.ftl_shipment_type").(*shipmenttype.ShipmentType)
	cus := ts.Fixture.MustRow("Customer.honeywell_customer").(*customer.Customer)
	trEquipType := ts.Fixture.MustRow("EquipmentType.tractor_equip_type").(*equipmenttype.EquipmentType)
	trlEquipType := ts.Fixture.MustRow("EquipmentType.trailer_equip_type").(*equipmenttype.EquipmentType)
	usr := ts.Fixture.MustRow("User.test_user").(*user.User)

	proNumberRepo := repositories.NewProNumberRepository(repositories.ProNumberRepositoryParams{
		Logger: logger.NewLogger(testutils.NewTestConfig()),
		DB:     ts.DB,
	})

	repo := repositories.NewShipmentRepository(repositories.ShipmentRepositoryParams{
		Logger:        logger.NewLogger(testutils.NewTestConfig()),
		DB:            ts.DB,
		ProNumberRepo: proNumberRepo,
		Calculator:    calculator.NewShipmentCalculator(calculator.ShipmentCalculatorParams{Logger: logger.NewLogger(testutils.NewTestConfig())}),
	})

	t.Run("list shipments", func(t *testing.T) {
		opts := &repoports.ListShipmentOptions{
			Filter: &ports.LimitOffsetQueryOptions{
				Limit:  10,
				Offset: 0,
				TenantOpts: &ports.TenantOptions{
					OrgID: org.ID,
					BuID:  bu.ID,
				},
			},
		}

		testutils.TestRepoList(ctx, t, repo, opts)
	})

	t.Run("list shipments with query", func(t *testing.T) {
		opts := &repoports.ListShipmentOptions{
			Filter: &ports.LimitOffsetQueryOptions{
				Limit:  10,
				Offset: 0,
				TenantOpts: &ports.TenantOptions{
					OrgID: org.ID,
					BuID:  bu.ID,
				},
				Query: "12",
			},
		}

		testutils.TestRepoList(ctx, t, repo, opts)
	})

	t.Run("list shipments with details", func(t *testing.T) {
		opts := &repoports.ListShipmentOptions{
			ShipmentOptions: repoports.ShipmentOptions{
				ExpandShipmentDetails: true,
			},
			Filter: &ports.LimitOffsetQueryOptions{
				Limit:  10,
				Offset: 0,
				TenantOpts: &ports.TenantOptions{
					OrgID: org.ID,
					BuID:  bu.ID,
				},
			},
		}

		result, err := repo.List(ctx, opts)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotEmpty(t, result.Items)
		require.NotEmpty(t, result.Items[0].Customer)
		require.NotEmpty(t, result.Items[0].Moves)
		require.NotEmpty(t, result.Items[0].Moves[0].Stops) // Include the movement stops
		require.NotEmpty(t, result.Items[0].TractorType)
		require.NotEmpty(t, result.Items[0].TrailerType)
		require.NotEmpty(t, result.Items[0].Commodities)
	})

	t.Run("get shipment by id", func(t *testing.T) {
		testutils.TestRepoGetByID(ctx, t, repo, repoports.GetShipmentByIDOptions{
			ID:    smt.ID,
			OrgID: org.ID,
			BuID:  bu.ID,
		})
	})

	t.Run("get tractor with invalid id", func(t *testing.T) {
		entity, err := repo.GetByID(ctx, repoports.GetShipmentByIDOptions{
			ID:    "invalid-id",
			OrgID: org.ID,
			BuID:  bu.ID,
		})

		require.Error(t, err)
		require.Nil(t, entity)
	})

	t.Run("get shipment by id with details", func(t *testing.T) {
		entity, err := repo.GetByID(ctx, repoports.GetShipmentByIDOptions{
			ShipmentOptions: repoports.ShipmentOptions{
				ExpandShipmentDetails: true,
			},
			ID:    smt.ID,
			OrgID: org.ID,
			BuID:  bu.ID,
		})

		require.NoError(t, err)
		require.NotNil(t, entity)
		require.NotEmpty(t, entity.Customer)
		require.NotEmpty(t, entity.Moves)
		require.NotEmpty(t, entity.Moves[0].Stops)
		require.NotEmpty(t, entity.TractorType)
		require.NotEmpty(t, entity.TrailerType)
		require.NotEmpty(t, entity.Commodities)
	})

	t.Run("get shipment by id failure", func(t *testing.T) {
		result, err := repo.GetByID(ctx, repoports.GetShipmentByIDOptions{
			ID:    "invalid-id",
			OrgID: org.ID,
			BuID:  bu.ID,
		})

		require.Error(t, err)
		require.Nil(t, result)
	})

	t.Run("create shipment", func(t *testing.T) {
		// Test Data
		newEntity := &shipment.Shipment{
			ServiceTypeID:  serviceType.ID,
			ShipmentTypeID: shipmentType.ID,
			TrailerTypeID:  &trlEquipType.ID,
			TractorTypeID:  &trEquipType.ID,
			CustomerID:     cus.ID,
			BusinessUnitID: bu.ID,
			OrganizationID: org.ID,
		}

		testutils.TestRepoCreate(ctx, t, repo, newEntity)
	})

	t.Run("create shipment with pro number", func(t *testing.T) {
		// Test Data
		newEntity := &shipment.Shipment{
			ServiceTypeID:  serviceType.ID,
			ShipmentTypeID: shipmentType.ID,
			TrailerTypeID:  &trlEquipType.ID,
			TractorTypeID:  &trEquipType.ID,
			CustomerID:     cus.ID,
			BusinessUnitID: bu.ID,
			OrganizationID: org.ID,
		}

		result, err := repo.Create(ctx, newEntity)
		require.NoError(t, err)
		require.NotNil(t, result)

		t.Logf("Pro Number: %s", result.ProNumber)
		require.NotEmpty(t, result.ProNumber)
	})

	t.Run("create shipment failure", func(t *testing.T) {
		// Test Data
		newEntity := &shipment.Shipment{
			ProNumber:      "TEST",
			ServiceTypeID:  serviceType.ID,
			ShipmentTypeID: "invalid-id",
			TrailerTypeID:  &trlEquipType.ID,
			TractorTypeID:  &trEquipType.ID,
			CustomerID:     cus.ID,
			BusinessUnitID: bu.ID,
			OrganizationID: org.ID,
		}

		results, err := repo.Create(ctx, newEntity)

		require.Error(t, err)
		require.Nil(t, results)
	})

	t.Run("update shipment", func(t *testing.T) {
		smt.ProNumber = "S12354123"

		result, err := repo.Update(ctx, smt)
		require.NoError(t, err)
		require.Equal(t, "S12354123", smt.ProNumber)
		require.NotNil(t, result)
	})

	t.Run("update shipment version lock failure", func(t *testing.T) {
		smt.ProNumber = "S12354123"
		smt.Version = 0

		results, err := repo.Update(ctx, smt)

		require.Error(t, err)
		require.Nil(t, results)
	})

	t.Run("update shipment with invalid information", func(t *testing.T) {
		smt.ProNumber = "S12354123"
		smt.ShipmentTypeID = "invalid-id"

		results, err := repo.Update(ctx, smt)

		require.Error(t, err)
		require.Nil(t, results)
	})

	t.Run("cancel shipment", func(t *testing.T) {
		now := timeutils.NowUnix()

		newEntity, err := repo.Cancel(ctx, &repoports.CancelShipmentRequest{
			ShipmentID:   smt.ID,
			OrgID:        org.ID,
			BuID:         bu.ID,
			CanceledByID: usr.ID,
			CanceledAt:   now,
			CancelReason: "Test",
		})

		require.NoError(t, err)
		require.Equal(t, shipment.StatusCanceled, newEntity.Status)
		require.Equal(t, "Test", newEntity.CancelReason)
		require.Equal(t, &now, newEntity.CanceledAt)
		require.Equal(t, &usr.ID, newEntity.CanceledByID)
	})

	t.Run("cancel shipment with invalid shipment id", func(t *testing.T) {
		now := timeutils.NowUnix()

		newEntity, err := repo.Cancel(ctx, &repoports.CancelShipmentRequest{
			ShipmentID:   "invalid-id",
			OrgID:        org.ID,
			BuID:         bu.ID,
			CanceledByID: usr.ID,
			CanceledAt:   now,
			CancelReason: "Test",
		})

		require.Error(t, err, "Shipment not found")
		require.Nil(t, newEntity)
	})
}
