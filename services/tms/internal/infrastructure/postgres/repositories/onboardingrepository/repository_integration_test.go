//go:build integration

package onboardingrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/onboarding"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestOnboardingRepository_CompletesOnce(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}

	created, err := repo.Create(ctx, onboarding.NewPending(tenant.OrgID, tenant.BuID))
	require.NoError(t, err)

	read, err := repo.Get(ctx, repositories.GetOnboardingRequest{TenantInfo: tenant})
	require.NoError(t, err)
	assert.Equal(t, created.ID, read.ID)
	assert.Equal(t, onboarding.StatusPending, read.Status)

	read.Complete(onboarding.CompleteParams{
		UserID:           data.User.ID,
		OperationType:    onboarding.OperationTypeAsset,
		SampleDataLoaded: true,
	})
	completed, err := repo.Complete(ctx, read)
	require.NoError(t, err)
	assert.Equal(t, onboarding.StatusCompleted, completed.Status)
	assert.Equal(t, onboarding.OperationTypeAsset, completed.OperationType)
	assert.True(t, completed.SampleDataLoaded)
	assert.Equal(t, read.Version+1, completed.Version)

	_, err = repo.Complete(ctx, completed)
	require.True(t, errortypes.IsVersionMismatchError(err), "a completed wizard cannot complete again")
}
