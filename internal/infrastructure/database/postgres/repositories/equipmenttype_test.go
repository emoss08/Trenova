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
	org := ts.Fixture.MustRow("Organization.trenova").(*organization.Organization)
	bu := ts.Fixture.MustRow("BusinessUnit.trenova").(*businessunit.BusinessUnit)
	tractorType := ts.Fixture.MustRow("EquipmentType.tractor_equip_type").(*equipmenttype.EquipmentType)

	repo := repositories.NewEquipmentTypeRepository(repositories.EquipmentTypeRespositoryParams{
		Logger: logger.NewLogger(testutils.NewTestConfig()),
		DB:     ts.DB,
	})

	t.Run("list equipment types", func(t *testing.T) {
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

	t.Run("get equipment type by id", func(t *testing.T) {
		testutils.TestRepoGetByID(ctx, t, repo, repoports.GetEquipmentTypeByIDOptions{
			ID:    tractorType.ID,
			OrgID: org.ID,
			BuID:  bu.ID,
		})
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

		testutils.TestRepoCreate(ctx, t, repo, et)
	})

	t.Run("update equipment type", func(t *testing.T) {
		tractorType.Description = "Test Equipment Type 2"
		testutils.TestRepoUpdate(ctx, t, repo, tractorType)
	})
}
