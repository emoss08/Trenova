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

func TestCommodityService(t *testing.T) {
	ctx := context.Background()
	s := testutils.SetupTestServer(t)
	service := services.NewCommodityService(s)
	org, err := factory.NewOrganizationFactory(s.DB).MustCreateOrganization(ctx)
	require.NoError(t, err)

	createTestCommodity := func(name string) *models.Commodity {
		return &models.Commodity{
			Name:                name,
			Status:              property.StatusActive,
			OrganizationID:      org.ID,
			BusinessUnitID:      org.BusinessUnitID,
			HazardousMaterialID: uuid.Nil,
		}
	}

	t.Run("CreateAndGet", func(t *testing.T) {
		created, err := service.Create(ctx, createTestCommodity("OKAYOKAY"))
		require.NoError(t, err)
		assert.NotNil(t, created)
		assert.NotEqual(t, uuid.Nil, created.ID)

		// Get the created Commodity
		fetched, err := service.Get(ctx, created.ID, created.OrganizationID, created.BusinessUnitID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, fetched.ID)
		assert.Equal(t, created.Name, fetched.Name)
		assert.Equal(t, created.Status, fetched.Status)
	})

	t.Run("GetAll", func(t *testing.T) {
		// Create multiple Commodity
		for i := 0; i < 5; i++ {
			_, err := service.Create(ctx, createTestCommodity(fmt.Sprintf("CODE%d", i)))
			require.NoError(t, err)
		}

		// Query all Commodity
		filter := &services.CommodityQueryFilter{
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
			Limit:          5,
			Offset:         0,
		}
		entities, total, err := service.GetAll(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, entities, 5)
		assert.Equal(t, 5, total)
	})

	t.Run("Update", func(t *testing.T) {
		created, err := service.Create(ctx, createTestCommodity("OKAYOKAY"))
		require.NoError(t, err)

		// Update the Commodity
		created.Name = "UPDATED"
		updated, err := service.UpdateOne(ctx, created)
		require.NoError(t, err)
		assert.Equal(t, created.ID, updated.ID)
		assert.Equal(t, "UPDATED", updated.Name)
	})

	t.Run("QueryFiltering", func(t *testing.T) {
		// Create multiple Commodity
		for i := 0; i < 5; i++ {
			_, err := service.Create(ctx, createTestCommodity(fmt.Sprintf("CODE%d", i)))
			require.NoError(t, err)
		}

		// Query all Commodity
		filter := &services.CommodityQueryFilter{
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
			Limit:          5,
			Offset:         0,
		}
		entities, total, err := service.GetAll(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, entities, 1)
		assert.Equal(t, 1, total)
	})
}
