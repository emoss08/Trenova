package services_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/emoss08/trenova/pkg/testutils"
	"github.com/emoss08/trenova/pkg/testutils/factory"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChargeTypeService(t *testing.T) {
	ctx := context.Background()
	s := testutils.SetupTestServer(t)
	service := services.NewChargeTypeService(s)
	org, err := factory.NewOrganizationFactory(s.DB).MustCreateOrganization(ctx)
	require.NoError(t, err)

	createTestChargeType := func(name string) *models.ChargeType {
		return &models.ChargeType{
			Status:         property.StatusActive,
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
			Name:           name,
			Description:    "Test Accessorial Charge",
		}
	}

	t.Run("CreateAndGet", func(t *testing.T) {
		created, err := service.Create(ctx, createTestChargeType("OKAYOKAY"))
		require.NoError(t, err)
		assert.NotNil(t, created)
		assert.NotEqual(t, uuid.Nil, created.ID)

		// Get the created AccessorialCharge
		fetched, err := service.Get(ctx, created.ID, created.OrganizationID, created.BusinessUnitID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, fetched.ID)
		assert.Equal(t, created.Name, fetched.Name)
		assert.Equal(t, created.Description, fetched.Description)
	})

	t.Run("GetAll", func(t *testing.T) {
		// Create multiple AccessorialCharges
		for i := 0; i < 5; i++ {
			_, err := service.Create(ctx, createTestChargeType(fmt.Sprintf("CODE%d", i)))
			require.NoError(t, err)
		}

		// Query all AccessorialCharges
		filter := &services.ChargeTypeQueryFilter{
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
			Limit:          10,
			Offset:         0,
		}
		charges, count, err := service.GetAll(ctx, filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, 5)
		assert.GreaterOrEqual(t, len(charges), 5)
	})

	t.Run("Update", func(t *testing.T) {
		// Create a new AccessorialCharge
		newCharge := createTestChargeType("WEIRD")
		created, err := service.Create(ctx, newCharge)
		require.NoError(t, err)

		// Update the AccessorialCharge
		created.Description = "Updated Description"
		updated, err := service.UpdateOne(ctx, created)
		require.NoError(t, err)
		assert.Equal(t, "Updated Description", updated.Description)

		// Fetch the updated AccessorialCharge
		fetched, err := service.Get(ctx, updated.ID, updated.OrganizationID, updated.BusinessUnitID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Description", fetched.Description)
	})

	t.Run("QueryFiltering", func(t *testing.T) {
		// Create AccessorialCharges with different codes
		codes := []string{"ABC", "DEF", "GHI"}
		for _, code := range codes {
			charge := createTestChargeType(code)
			charge.Name = code
			_, err := service.Create(ctx, charge)
			require.NoError(t, err)
		}

		// Query with a specific code
		filter := &services.ChargeTypeQueryFilter{
			Query:          "ABC",
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
			Limit:          10,
			Offset:         0,
		}
		charges, count, err := service.GetAll(ctx, filter)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
		assert.Equal(t, "ABC", charges[0].Name)
	})
}
