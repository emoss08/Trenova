package repositories_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/stretchr/testify/require"

	repoports "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/test/testutils"
)

func TestEquipmentTypeRepository(t *testing.T) {
	t.Parallel()
	org := ts.Fixture.MustRow("Organization.trenova").(*organization.Organization)
	bu := ts.Fixture.MustRow("BusinessUnit.trenova").(*businessunit.BusinessUnit)
	tractorType := ts.Fixture.MustRow("EquipmentType.tractor_equip_type").(*equipmenttype.EquipmentType)

	// Test Data
	et := &equipmenttype.EquipmentType{
		Code:           "TEST000001",
		Description:    "Test Equipment Type",
		Class:          equipmenttype.ClassTractor,
		Color:          "#000000",
		Status:         domain.StatusActive,
		BusinessUnitID: bu.ID,
		OrganizationID: org.ID,
	}

	repo := repositories.NewEquipmentTypeRepository(repositories.EquipmentTypeRespositoryParams{
		Logger: logger.NewLogger(testutils.NewTestConfig()),
		DB:     ts.DB,
	})

	t.Run("list equipment types", func(t *testing.T) {
		equipmentTypes, err := repo.List(ctx, &ports.LimitOffsetQueryOptions{
			Limit:  10,
			Offset: 0,
			TenantOpts: &ports.TenantOptions{
				OrgID: org.ID,
				BuID:  bu.ID,
			},
		})

		require.NoError(t, err)
		require.NotNil(t, equipmentTypes)
		require.NotEmpty(t, equipmentTypes.Items)
	})

	t.Run("get equipment type by id", func(t *testing.T) {
		equipmentType, err := repo.GetByID(ctx, repoports.GetEquipmentTypeByIDOptions{
			ID:    tractorType.ID,
			OrgID: org.ID,
			BuID:  bu.ID,
		})

		require.NoError(t, err)
		require.NotNil(t, equipmentType)
		require.Equal(t, tractorType.ID, equipmentType.ID)
	})

	t.Run("get equipment type with invalid id", func(t *testing.T) {
		equipmentType, err := repo.GetByID(ctx, repoports.GetEquipmentTypeByIDOptions{
			ID:    "invalid-id",
			OrgID: org.ID,
			BuID:  bu.ID,
		})

		require.Error(t, err, "equipment type not found")
		require.Nil(t, equipmentType)
	})

	t.Run("create equipment type", func(t *testing.T) {
		created, err := repo.Create(ctx, et)
		require.NoError(t, err)
		require.NotNil(t, created)
		require.Equal(t, et.ID, created.ID)
	})

	t.Run("update equipment type", func(t *testing.T) {
		et.Description = "Test Equipment Type 2"
		updated, err := repo.Update(ctx, et)
		require.NoError(t, err)
		require.NotNil(t, updated)
		require.Equal(t, et.ID, updated.ID)
	})
}
