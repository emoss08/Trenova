package services_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/testutils/factory"

	"github.com/emoss08/trenova/pkg/testutils"
)

func TestNewLocationCategoryService(t *testing.T) {
	ctx := context.Background()
	s, cleanup := testutils.SetupTestServer(t)
	defer cleanup()

	service := services.NewLocationCategoryService(s)
	org, err := factory.NewOrganizationFactory(s.DB).MustCreateOrganization(ctx)
	require.NoError(t, err)
	user, err := factory.NewUserFactory(s.DB).CreateOrGetUser(ctx)
	require.NoError(t, err)

	createLocationCategory := func(name string) *models.LocationCategory {
		return &models.LocationCategory{
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
			Name:           name,
			Description:    "Test Description",
		}
	}

	t.Run("CreateAndGet", func(t *testing.T) {
		created, err := service.Create(ctx, createLocationCategory("TEST"), user.ID)
		require.NoError(t, err)
		assert.NotNil(t, created)
		assert.NotEqual(t, uuid.Nil, created.ID)

		// Get the created LocationCategory
		fetched, err := service.Get(ctx, created.ID, created.OrganizationID, created.BusinessUnitID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, fetched.ID)
		assert.Equal(t, created.Name, fetched.Name)
	})

	t.Run("GetAll", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			_, err = service.Create(ctx, createLocationCategory(fmt.Sprintf("code%d", i)), user.ID)
			require.NoError(t, err)
		}

		filter := &services.LocationCategoryQueryFilter{
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
		// Create a new LocationCategory
		newLocationCategory := createLocationCategory("TEST1")
		created, err := service.Create(ctx, newLocationCategory, user.ID)
		require.NoError(t, err)

		// Update the LocationCategory
		created.Description = "Test Description 2"
		updated, err := service.UpdateOne(ctx, created, user.ID)
		require.NoError(t, err)
		assert.Equal(t, "Test Description 2", updated.Description)

		// Fetch the updated LocationCategory
		fetched, err := service.Get(ctx, updated.ID, updated.OrganizationID, updated.BusinessUnitID)
		require.NoError(t, err)
		assert.Equal(t, "Test Description 2", fetched.Description)
	})

	t.Run("QueryFiltering", func(t *testing.T) {
		// Create LocationCategory with different codes
		names := []string{"TEST2", "TEST3", "TEST4"}
		for _, name := range names {
			entity := createLocationCategory(name)
			entity.Name = name
			_, err = service.Create(ctx, entity, user.ID)
			require.NoError(t, err)
		}

		// Query with a specific code
		filter := &services.LocationCategoryQueryFilter{
			Query:          "TEST2",
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
			Limit:          10,
			Offset:         0,
		}

		results, count, err := service.GetAll(ctx, filter)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
		assert.Equal(t, "TEST2", results[0].Name)
	})
}
