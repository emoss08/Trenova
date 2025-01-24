package repositories_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	repoports "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/test/testutils"
)

func TestWorkerRepository(t *testing.T) {
	org := ts.Fixture.MustRow("Organization.trenova").(*organization.Organization)
	bu := ts.Fixture.MustRow("BusinessUnit.trenova").(*businessunit.BusinessUnit)
	wrk := ts.Fixture.MustRow("Worker.worker_1").(*worker.Worker)
	usState := ts.Fixture.MustRow("UsState.ca").(*usstate.UsState)

	repo := repositories.NewWorkerRepository(repositories.WorkerRepositoryParams{
		Logger: logger.NewLogger(testutils.NewTestConfig()),
		DB:     ts.DB,
	})

	t.Run("list", func(t *testing.T) {
		opts := &repoports.ListWorkerOptions{
			Filter: &ports.LimitOffsetQueryOptions{
				Limit:  10,
				Offset: 0,
				TenantOpts: &ports.TenantOptions{
					OrgID: org.ID,
					BuID:  bu.ID,
				},
			},
		}

		testutils.TestRepoList(ctx, t, repo, opts)
	})

	t.Run("get by id", func(t *testing.T) {
		testutils.TestRepoGetByID(ctx, t, repo, repoports.GetWorkerByIDOptions{
			WorkerID: wrk.ID,
			OrgID:    org.ID,
			BuID:     bu.ID,
		})
	})

	t.Run("get with invalid id", func(t *testing.T) {
		entity, err := repo.GetByID(ctx, repoports.GetWorkerByIDOptions{
			WorkerID: "invalid-id",
			OrgID:    org.ID,
			BuID:     bu.ID,
		})

		require.Error(t, err, "entity not found")
		require.Nil(t, entity)
	})

	t.Run("create", func(t *testing.T) {
		newEntity := &worker.Worker{
			FirstName:      "John",
			LastName:       "Doe",
			Gender:         "Male",
			AddressLine1:   "123 Main St",
			City:           "Los Angeles",
			StateID:        usState.ID,
			PostalCode:     "90001",
			BusinessUnitID: bu.ID,
			OrganizationID: org.ID,
			Profile: &worker.WorkerProfile{
				DOB:             timeutils.NowUnix(),
				LicenseNumber:   "1234567890",
				LicenseStateID:  usState.ID,
				BusinessUnitID:  bu.ID,
				OrganizationID:  org.ID,
				HazmatExpiry:    timeutils.NowUnix(),
				LicenseExpiry:   timeutils.NowUnix(),
				HireDate:        timeutils.NowUnix(),
				MVRDueDate:      timeutils.NowUnixPointer(),
				PhysicalDueDate: timeutils.NowUnixPointer(),
			},
		}

		testutils.TestRepoCreate(ctx, t, repo, newEntity)
	})

	t.Run("update", func(t *testing.T) {
		// Fetch the worker along with the profile from the database
		updatedEntity, err := repo.GetByID(ctx, repoports.GetWorkerByIDOptions{
			IncludeProfile: true,
			WorkerID:       wrk.ID,
			OrgID:          org.ID,
			BuID:           bu.ID,
		})

		updatedEntity.FirstName = "Jane"
		updatedEntity.Profile.LicenseNumber = "0987654321"

		require.NoError(t, err)
		require.NotNil(t, wrk)
		require.NotNil(t, updatedEntity.Profile)
		assert.Equal(t, "Jane", updatedEntity.FirstName)
		assert.Equal(t, "0987654321", updatedEntity.Profile.LicenseNumber)
	})
}
