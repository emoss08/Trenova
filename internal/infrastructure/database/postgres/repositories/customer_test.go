// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package repositories_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/stretchr/testify/require"

	repoports "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/test/testutils"
)

func TestCustomerRepository(t *testing.T) {
	org := ts.Fixture.MustRow("Organization.trenova").(*organization.Organization)
	bu := ts.Fixture.MustRow("BusinessUnit.trenova").(*businessunit.BusinessUnit)
	cus := ts.Fixture.MustRow("Customer.honeywell_customer").(*customer.Customer)
	cusBillProfile := ts.Fixture.MustRow("BillingProfile.honeywell_billing_profile").(*customer.BillingProfile)
	usState := ts.Fixture.MustRow("UsState.ca").(*usstate.UsState)

	repo := repositories.NewCustomerRepository(repositories.CustomerRepositoryParams{
		Logger: logger.NewLogger(testutils.NewTestConfig()),
		DB:     ts.DB,
	})

	t.Run("list customers", func(t *testing.T) {
		opts := &repoports.ListCustomerOptions{
			Filter: &ports.QueryOptions{
				Limit:  10,
				Offset: 0,
				TenantOpts: ports.TenantOptions{
					OrgID: org.ID,
					BuID:  bu.ID,
				},
			},
		}

		testutils.TestRepoList(ctx, t, repo, opts)
	})

	t.Run("list customers with query", func(t *testing.T) {
		opts := &repoports.ListCustomerOptions{
			Filter: &ports.QueryOptions{
				Limit:  10,
				Offset: 0,
				TenantOpts: ports.TenantOptions{
					OrgID: org.ID,
					BuID:  bu.ID,
				},
				Query: "Honeywell",
			},
		}

		testutils.TestRepoList(ctx, t, repo, opts)
	})

	t.Run("list customers with state", func(t *testing.T) {
		opts := &repoports.ListCustomerOptions{
			IncludeState: true,
			Filter: &ports.QueryOptions{
				Limit:  10,
				Offset: 0,
				TenantOpts: ports.TenantOptions{
					OrgID: org.ID,
					BuID:  bu.ID,
				},
			},
		}

		result, err := repo.List(ctx, opts)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotEmpty(t, result.Items)
		require.NotEmpty(t, result.Items[0].State)
	})

	t.Run("get customer by id", func(t *testing.T) {
		testutils.TestRepoGetByID(ctx, t, repo, repoports.GetCustomerByIDOptions{
			ID:    cus.ID,
			OrgID: org.ID,
			BuID:  bu.ID,
		})
	})

	t.Run("get customer with invalid id", func(t *testing.T) {
		l, err := repo.GetByID(ctx, repoports.GetCustomerByIDOptions{
			ID:    "invalid-id",
			OrgID: org.ID,
			BuID:  bu.ID,
		})

		require.Error(t, err, "customer not found")
		require.Nil(t, l)
	})

	t.Run("get customer by id with state", func(t *testing.T) {
		result, err := repo.GetByID(ctx, repoports.GetCustomerByIDOptions{
			ID:           cus.ID,
			OrgID:        org.ID,
			BuID:         bu.ID,
			IncludeState: true,
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotEmpty(t, result.State)
	})

	t.Run("get customer by id failure", func(t *testing.T) {
		result, err := repo.GetByID(ctx, repoports.GetCustomerByIDOptions{
			ID:           "invalid-id",
			OrgID:        org.ID,
			BuID:         bu.ID,
			IncludeState: true,
		})

		require.Error(t, err)
		require.Nil(t, result)
	})

	t.Run("create customer", func(t *testing.T) {
		// Test Data
		c := &customer.Customer{
			Name:           "Test customer 2",
			AddressLine1:   "1234 Main St",
			Code:           "TEST000001",
			City:           "Los Angeles",
			PostalCode:     "90001",
			Status:         domain.StatusActive,
			StateID:        usState.ID,
			BusinessUnitID: bu.ID,
			OrganizationID: org.ID,
		}

		testutils.TestRepoCreate(ctx, t, repo, c)
	})

	t.Run("create customer with billing profile", func(t *testing.T) {
		// Test Data
		c := &customer.Customer{
			Name:           "Test customer 2",
			AddressLine1:   "1234 Main St",
			Code:           "TEST000002",
			City:           "Los Angeles",
			PostalCode:     "90001",
			Status:         domain.StatusActive,
			StateID:        usState.ID,
			BusinessUnitID: bu.ID,
			OrganizationID: org.ID,
			BillingProfile: &customer.BillingProfile{
				BusinessUnitID:   bu.ID,
				OrganizationID:   org.ID,
				BillingCycleType: customer.BillingCycleTypeMonthly,
			},
		}

		result, err := repo.Create(ctx, c)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.BillingProfile)
		require.Equal(t, c.BillingProfile.BillingCycleType, result.BillingProfile.BillingCycleType)
	})

	t.Run("create customer failure", func(t *testing.T) {
		// Test Data
		l := &customer.Customer{
			Name:           "Test customer 2",
			AddressLine1:   "1234 Main St",
			Code:           "TEST000001",
			City:           "Los Angeles",
			PostalCode:     "90001",
			Status:         domain.StatusActive,
			StateID:        "invalid-id",
			BusinessUnitID: bu.ID,
			OrganizationID: org.ID,
		}

		results, err := repo.Create(ctx, l)

		require.Error(t, err)
		require.Nil(t, results)
	})

	t.Run("update customer", func(t *testing.T) {
		cus.Name = "Test Customer 3"
		testutils.TestRepoUpdate(ctx, t, repo, cus)
	})

	t.Run("update customer with billing profile", func(t *testing.T) {
		cus.BillingProfile = cusBillProfile
		cusBillProfile.BillingCycleType = customer.BillingCycleTypeMonthly
		testutils.TestRepoUpdate(ctx, t, repo, cus)
	})

	t.Run("update customer version lock failure", func(t *testing.T) {
		cus.Name = "Test Customer 3"
		cus.Version = 0

		results, err := repo.Update(ctx, cus)

		require.Error(t, err)
		require.Nil(t, results)
	})

	t.Run("update customer billing profile version lock failure", func(t *testing.T) {
		cus.BillingProfile = cusBillProfile
		cusBillProfile.BillingCycleType = customer.BillingCycleTypeMonthly
		cusBillProfile.Version = 0

		results, err := repo.Update(ctx, cus)

		require.Error(t, err)
		require.Nil(t, results)
	})

	t.Run("update customer with invalid billing profile", func(t *testing.T) {
		cus.BillingProfile.BillingCycleType = "invalid-billing-cycle-type"

		results, err := repo.Update(ctx, cus)

		require.Error(t, err)
		require.Nil(t, results)
	})

	t.Run("update customer with invalid information", func(t *testing.T) {
		cus.Name = "Test customer 3"
		cus.StateID = "invalid-id"

		results, err := repo.Update(ctx, cus)

		require.Error(t, err)
		require.Nil(t, results)
	})
}
