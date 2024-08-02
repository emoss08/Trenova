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

func TestNewCustomerService(t *testing.T) {
	ctx := context.Background()
	s, cleanup := testutils.SetupTestServer(t)
	defer cleanup()

	service := services.NewCustomerService(s)
	org, err := factory.NewOrganizationFactory(s.DB).MustCreateOrganization(ctx)
	require.NoError(t, err)
	user, err := factory.NewUserFactory(s.DB).CreateOrGetUser(ctx)
	require.NoError(t, err)
	state, err := factory.NewStateFactory(s.DB).CreateUSState(ctx)
	require.NoError(t, err)

	createTestCustomer := func(name string) *models.Customer {
		return &models.Customer{
			OrganizationID:      org.ID,
			BusinessUnitID:      org.BusinessUnitID,
			Name:                name,
			Status:              property.StatusActive,
			AddressLine1:        "123 Main St",
			City:                "Minneapolis",
			StateID:             state.ID,
			PostalCode:          "55401",
			AutoMarkReadyToBill: true,
		}
	}

	t.Run("CreateAndGet", func(t *testing.T) {
		created, err := service.Create(ctx, createTestCustomer("OKAYOKAY"), user.ID)
		require.NoError(t, err)
		assert.NotNil(t, created)
		assert.NotEqual(t, uuid.Nil, created.ID)

		// Get the created Customer
		fetched, err := service.Get(ctx, created.ID, created.OrganizationID, created.BusinessUnitID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, fetched.ID)
		assert.Equal(t, created.Name, fetched.Name)
	})

	t.Run("GetAll", func(t *testing.T) {
		// Create multiple customers
		for i := 0; i < 5; i++ {
			_, err = service.Create(ctx, createTestCustomer(fmt.Sprintf("CODE%d", i)), user.ID)
			require.NoError(t, err)
		}

		// Query all customers
		filter := &services.CustomerQueryFilter{
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
		// Create a new Customer
		newCustomer := createTestCustomer("WEIRD")
		created, err := service.Create(ctx, newCustomer, user.ID)
		require.NoError(t, err)

		// Update the Customer
		created.AutoMarkReadyToBill = false
		updated, err := service.UpdateOne(ctx, created, user.ID)
		require.NoError(t, err)
		assert.False(t, updated.AutoMarkReadyToBill)

		// Fetch the updated Customer
		fetched, err := service.Get(ctx, updated.ID, updated.OrganizationID, updated.BusinessUnitID)
		require.NoError(t, err)
		assert.False(t, fetched.AutoMarkReadyToBill)
	})

	t.Run("QueryFiltering", func(t *testing.T) {
		// Create Customer with different names
		names := []string{"ABC", "DEF", "GHI"}
		for _, name := range names {
			customer := createTestCustomer(name)
			customer.Name = name
			_, err = service.Create(ctx, customer, user.ID)
			require.NoError(t, err)
		}

		// Query with a specific code
		filter := &services.CustomerQueryFilter{
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
