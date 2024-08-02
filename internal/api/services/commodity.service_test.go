package services_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/models/property"

	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/pkg/testutils/factory"

	"github.com/emoss08/trenova/pkg/testutils"
)

func TestNewCommodityService(t *testing.T) {
	ctx := context.Background()
	s, cleanup := testutils.SetupTestServer(t)
	defer cleanup()

	service := services.NewCommodityService(s)
	org, err := factory.NewOrganizationFactory(s.DB).MustCreateOrganization(ctx)
	require.NoError(t, err)
	user, err := factory.NewUserFactory(s.DB).CreateOrGetUser(ctx)
	require.NoError(t, err)

	createTestCommodity := func(name string) *models.Commodity {
		return &models.Commodity{
			Status:         property.StatusActive,
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
			Name:           name,
			UnitOfMeasure:  property.UnitOfMeasureBottle,
		}
	}

	t.Run("CreateAndGet", func(t *testing.T) {
		created, err := service.Create(ctx, createTestCommodity("OKAYOKAY"), user.ID)
		require.NoError(t, err)
		assert.NotNil(t, created)
		assert.NotEqual(t, uuid.Nil, created.ID)

		// Get the created Commodity
		fetched, err := service.Get(ctx, created.ID, created.OrganizationID, created.BusinessUnitID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, fetched.ID)
		assert.Equal(t, created.Name, fetched.Name)
	})

	t.Run("GetAll", func(t *testing.T) {
		// Create multiple Commodities
		for i := 0; i < 5; i++ {
			_, err = service.Create(ctx, createTestCommodity(fmt.Sprintf("CODE%d", i)), user.ID)
			require.NoError(t, err)
		}

		// Query all Commodities
		filter := &services.CommodityQueryFilter{
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
		// Create a new Commodity
		newCommodity := createTestCommodity("WEIRD")
		created, err := service.Create(ctx, newCommodity, user.ID)
		require.NoError(t, err)

		// Update the Commodity
		created.UnitOfMeasure = property.UnitOfMeasureCase
		updated, err := service.UpdateOne(ctx, created, user.ID)
		require.NoError(t, err)
		assert.Equal(t, property.UnitOfMeasureCase, updated.UnitOfMeasure)

		// Fetch the updated Commodity
		fetched, err := service.Get(ctx, updated.ID, updated.OrganizationID, updated.BusinessUnitID)
		require.NoError(t, err)
		assert.Equal(t, "Case", fetched.UnitOfMeasure.String())
	})

	t.Run("QueryFiltering", func(t *testing.T) {
		// Create Commodity with different names
		names := []string{"ABC", "DEF", "GHI"}
		for _, name := range names {
			commodity := createTestCommodity(name)
			commodity.Name = name
			_, err = service.Create(ctx, commodity, user.ID)
			require.NoError(t, err)
		}

		// Query with a specific code
		filter := &services.CommodityQueryFilter{
			Query:          "ABC",
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
			Limit:          10,
			Offset:         0,
		}

		results, count, err := service.GetAll(ctx, filter)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
		assert.Equal(t, "ABC", results[0].Name)
	})
}
