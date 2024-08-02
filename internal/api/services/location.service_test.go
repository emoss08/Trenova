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

func TestNewLocationService(t *testing.T) {
	ctx := context.Background()
	s, cleanup := testutils.SetupTestServer(t)
	defer cleanup()

	service := services.NewLocationService(s)
	org, err := factory.NewOrganizationFactory(s.DB).MustCreateOrganization(ctx)
	require.NoError(t, err)
	user, err := factory.NewUserFactory(s.DB).CreateOrGetUser(ctx)
	require.NoError(t, err)
	state, err := factory.NewStateFactory(s.DB).CreateUSState(ctx)
	require.NoError(t, err)

	locCategory := &models.LocationCategory{
		BusinessUnitID: org.BusinessUnitID,
		OrganizationID: org.ID,
		Name:           "Category",
		Description:    "Category Description",
		Color:          "#000000",
	}

	_, err = s.DB.NewInsert().Model(locCategory).Exec(ctx)
	require.NoError(t, err)

	createLocation := func(name string) *models.Location {
		return &models.Location{
			OrganizationID:     org.ID,
			BusinessUnitID:     org.BusinessUnitID,
			Status:             property.StatusActive,
			Name:               name,
			StateID:            state.ID,
			LocationCategoryID: locCategory.ID,
		}
	}

	t.Run("CreateAndGet", func(t *testing.T) {
		created, err := service.Create(ctx, createLocation("TEST"), user.ID)
		require.NoError(t, err)
		assert.NotNil(t, created)
		assert.NotEqual(t, uuid.Nil, created.ID)

		// Get the created Location
		fetched, err := service.Get(ctx, created.ID, created.OrganizationID, created.BusinessUnitID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, fetched.ID)
		assert.Equal(t, created.Name, fetched.Name)
	})

	t.Run("GetAll", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			_, err = service.Create(ctx, createLocation(fmt.Sprintf("name-%d", i)), user.ID)
			require.NoError(t, err)
		}

		filter := &services.LocationQueryFilter{
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
		// Create a new Location
		newLocation := createLocation("TEST1")
		created, err := service.Create(ctx, newLocation, user.ID)
		require.NoError(t, err)

		// Update the Location
		created.Description = "Test Description"
		updated, err := service.UpdateOne(ctx, created, user.ID)
		require.NoError(t, err)
		assert.Equal(t, "Test Description", updated.Description)

		// Fetch the updated Location
		fetched, err := service.Get(ctx, updated.ID, updated.OrganizationID, updated.BusinessUnitID)
		require.NoError(t, err)
		assert.Equal(t, "Test Description", fetched.Description)
	})

	t.Run("QueryFiltering", func(t *testing.T) {
		// Create Location with different codes
		names := []string{"TEST2", "TEST3", "TEST4"}
		for _, name := range names {
			entity := createLocation(name)
			entity.Name = name
			_, err = service.Create(ctx, entity, user.ID)
			require.NoError(t, err)
		}

		// Query with a specific code
		filter := &services.LocationQueryFilter{
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
