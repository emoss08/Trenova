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

	repo := repositories.NewEquipmentManufacturerRepository(repositories.EquipManuRepositoryParams{
		Logger: logger.NewLogger(testutils.NewTestConfig()),
		DB:     ts.DB,
	})

	t.Run("list equipment manufacturers", func(t *testing.T) {
		opts := &ports.LimitOffsetQueryOptions{
			Limit:  10,
			Offset: 0,
			TenantOpts: &ports.TenantOptions{
				OrgID: org.ID,
				BuID:  bu.ID,
			},
		}

		testutils.TestRepoList(ctx, t, repo, opts)
	})

	t.Run("get equipment manufacturer by id", func(t *testing.T) {
		testutils.TestRepoGetByID(ctx, t, repo, repoports.GetEquipManufacturerByIDOptions{
			ID:    emf1.ID,
			OrgID: org.ID,
			BuID:  bu.ID,
		})
	})

	t.Run("get equipment manufacturer with invalid id", func(t *testing.T) {
		equipManu, err := repo.GetByID(ctx, repoports.GetEquipManufacturerByIDOptions{
			ID:    "invalid-id",
			OrgID: org.ID,
			BuID:  bu.ID,
		})

		require.Error(t, err, "equipment manufacturer not found")
		require.Nil(t, equipManu)
	})

	t.Run("create equipment manufacturer", func(t *testing.T) {
		em := &equipmentmanufacturer.EquipmentManufacturer{
			Name:           "Test Equipment Manufacturer 2",
			Description:    "Test Equipment Manufacturer Description",
			Status:         domain.StatusActive,
			BusinessUnitID: bu.ID,
			OrganizationID: org.ID,
		}

		testutils.TestRepoCreate(ctx, t, repo, em)
	})

	t.Run("update equipment manufacturer", func(t *testing.T) {
		emf1.Name = "Test Equipment Manufacturer 3"
		testutils.TestRepoUpdate(ctx, t, repo, emf1)
	})
}
