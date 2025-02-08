package repositories_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/ports"

	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/test/testutils"
)

func TestCommodityRepository(t *testing.T) {
	org := ts.Fixture.MustRow("Organization.trenova").(*organization.Organization)
	bu := ts.Fixture.MustRow("BusinessUnit.trenova").(*businessunit.BusinessUnit)
	// loc := ts.Fixture.MustRow("Customer.honeywell_customer").(*customer.Customer)
	// usState := ts.Fixture.MustRow("UsState.ca").(*usstate.UsState)

	repo := repositories.NewCommodityRepository(repositories.CommodityRepositoryParams{
		Logger: logger.NewLogger(testutils.NewTestConfig()),
		DB:     ts.DB,
	})

	t.Run("list commodities", func(t *testing.T) {
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

	// t.Run("list customers with query", func(t *testing.T) {
	// 	opts := &repoports.ListCustomerOptions{
	// 		Filter: &ports.LimitOffsetQueryOptions{
	// 			Limit:  10,
	// 			Offset: 0,
	// 			TenantOpts: &ports.TenantOptions{
	// 				OrgID: org.ID,
	// 				BuID:  bu.ID,
	// 			},
	// 			Query: "Honeywell",
	// 		},
	// 	}

	// 	testutils.TestRepoList(ctx, t, repo, opts)
	// })

	// t.Run("list customers with state", func(t *testing.T) {
	// 	opts := &repoports.ListCustomerOptions{
	// 		IncludeState: true,
	// 		Filter: &ports.LimitOffsetQueryOptions{
	// 			Limit:  10,
	// 			Offset: 0,
	// 			TenantOpts: &ports.TenantOptions{
	// 				OrgID: org.ID,
	// 				BuID:  bu.ID,
	// 			},
	// 		},
	// 	}

	// 	result, err := repo.List(ctx, opts)
	// 	require.NoError(t, err)
	// 	require.NotNil(t, result)
	// 	require.NotEmpty(t, result.Items)
	// 	require.NotEmpty(t, result.Items[0].State)
	// })

	// t.Run("get customer by id", func(t *testing.T) {
	// 	testutils.TestRepoGetByID(ctx, t, repo, repoports.GetCustomerByIDOptions{
	// 		ID:    loc.ID,
	// 		OrgID: org.ID,
	// 		BuID:  bu.ID,
	// 	})
	// })

	// t.Run("get customer with invalid id", func(t *testing.T) {
	// 	l, err := repo.GetByID(ctx, repoports.GetCustomerByIDOptions{
	// 		ID:    "invalid-id",
	// 		OrgID: org.ID,
	// 		BuID:  bu.ID,
	// 	})

	// 	require.Error(t, err, "customer not found")
	// 	require.Nil(t, l)
	// })

	// t.Run("get customer by id with state", func(t *testing.T) {
	// 	result, err := repo.GetByID(ctx, repoports.GetCustomerByIDOptions{
	// 		ID:           loc.ID,
	// 		OrgID:        org.ID,
	// 		BuID:         bu.ID,
	// 		IncludeState: true,
	// 	})

	// 	require.NoError(t, err)
	// 	require.NotNil(t, result)
	// 	require.NotEmpty(t, result.State)
	// })

	// t.Run("get customer by id failure", func(t *testing.T) {
	// 	result, err := repo.GetByID(ctx, repoports.GetCustomerByIDOptions{
	// 		ID:           "invalid-id",
	// 		OrgID:        org.ID,
	// 		BuID:         bu.ID,
	// 		IncludeState: true,
	// 	})

	// 	require.Error(t, err)
	// 	require.Nil(t, result)
	// })

	// t.Run("create customer", func(t *testing.T) {
	// 	// Test Data
	// 	l := &customer.Customer{
	// 		Name:           "Test customer 2",
	// 		AddressLine1:   "1234 Main St",
	// 		Code:           "TEST000001",
	// 		City:           "Los Angeles",
	// 		PostalCode:     "90001",
	// 		Status:         domain.StatusActive,
	// 		StateID:        usState.ID,
	// 		BusinessUnitID: bu.ID,
	// 		OrganizationID: org.ID,
	// 	}

	// 	testutils.TestRepoCreate(ctx, t, repo, l)
	// })

	// t.Run("create customer failure", func(t *testing.T) {
	// 	// Test Data
	// 	l := &customer.Customer{
	// 		Name:           "Test customer 2",
	// 		AddressLine1:   "1234 Main St",
	// 		Code:           "TEST000001",
	// 		City:           "Los Angeles",
	// 		PostalCode:     "90001",
	// 		Status:         domain.StatusActive,
	// 		StateID:        "invalid-id",
	// 		BusinessUnitID: bu.ID,
	// 		OrganizationID: org.ID,
	// 	}

	// 	results, err := repo.Create(ctx, l)

	// 	require.Error(t, err)
	// 	require.Nil(t, results)
	// })

	// t.Run("update customer", func(t *testing.T) {
	// 	loc.Name = "Test Customer 3"
	// 	testutils.TestRepoUpdate(ctx, t, repo, loc)
	// })

	// t.Run("update customer version lock failure", func(t *testing.T) {
	// 	loc.Name = "Test Customer 3"
	// 	loc.Version = 0

	// 	results, err := repo.Update(ctx, loc)

	// 	require.Error(t, err)
	// 	require.Nil(t, results)
	// })

	// t.Run("update customer with invalid information", func(t *testing.T) {
	// 	loc.Name = "Test customer 3"
	// 	loc.StateID = "invalid-id"

	// 	results, err := repo.Update(ctx, loc)

	// 	require.Error(t, err)
	// 	require.Nil(t, results)
	// })
}
