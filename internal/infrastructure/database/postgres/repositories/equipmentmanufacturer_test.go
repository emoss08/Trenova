package repositories_test

import (
	"context"
	"os"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/stretchr/testify/require"

	repoports "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/test/testutils"
)

var (
	ts  *testutils.TestSetup
	ctx = context.Background()
)

func TestMain(m *testing.M) {
	setup, err := testutils.NewTestSetup(ctx)
	if err != nil {
		panic(err)
	}

	ts = setup

	os.Exit(m.Run())
}

func TestEquipmentManufacturerRepository(t *testing.T) {
	// Fixtures
	org := ts.Fixture.MustRow("Organization.trenova").(*organization.Organization)
	bu := ts.Fixture.MustRow("BusinessUnit.trenova").(*businessunit.BusinessUnit)
	emf1 := ts.Fixture.MustRow("EquipmentManufacturer.kenworth_manufacturer").(*equipmentmanufacturer.EquipmentManufacturer)
	// Test Data
	em := &equipmentmanufacturer.EquipmentManufacturer{
		Name:           "Test Equipment Manufacturer 2",
		Description:    "Test Equipment Manufacturer Description",
		Status:         domain.StatusActive,
		BusinessUnitID: bu.ID,
		OrganizationID: org.ID,
	}

	repo := repositories.NewEquipmentManufacturerRepository(repositories.EquipManuRepositoryParams{
		Logger: logger.NewLogger(testutils.NewTestConfig()),
		DB:     ts.DB,
	})

	t.Run("list equipment manufacturers", func(t *testing.T) {
		equipmentManufacturers, err := repo.List(ctx, &ports.LimitOffsetQueryOptions{
			Limit:  10,
			Offset: 0,
			TenantOpts: &ports.TenantOptions{
				OrgID: org.ID,
				BuID:  bu.ID,
			},
		})

		require.NoError(t, err)
		require.NotNil(t, equipmentManufacturers)
		require.Len(t, equipmentManufacturers.Items, 3)
	})

	t.Run("get equipment manufacturer by id", func(t *testing.T) {
		equipManu, err := repo.GetByID(ctx, repoports.GetEquipManufacturerByIDOptions{
			ID:    emf1.ID,
			OrgID: org.ID,
			BuID:  bu.ID,
		})

		require.NoError(t, err)
		require.NotNil(t, equipManu)
		require.Equal(t, emf1.ID, equipManu.ID)
	})

	t.Run("create equipment manufacturer", func(t *testing.T) {
		created, err := repo.Create(ctx, em)
		require.NoError(t, err)
		require.NotNil(t, created)
		require.Equal(t, em.ID, created.ID)
	})

	t.Run("update equipment manufacturer", func(t *testing.T) {
		em.Name = "Test Equipment Manufacturer 3"
		updated, err := repo.Update(ctx, em)
		require.NoError(t, err)
		require.NotNil(t, updated)
		require.Equal(t, em.ID, updated.ID)
	})
}
