//go:build integration

package userrepository

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	inttestutil "github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func TestUpdatePasswordUpdatesOnlyScopedUser(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	conn := postgres.NewTestConnection(db)
	repo := New(Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})

	data := seedtest.SeedFullTestData(t, ctx, db)

	user, err := repo.GetByID(ctx, repositories.GetUserByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  data.Organization.ID,
			BuID:   data.BusinessUnit.ID,
			UserID: data.User.ID,
		},
	})
	require.NoError(t, err)
	originalHash := user.Password

	updatedHash, err := user.GeneratePassword("new-password-123")
	require.NoError(t, err)

	err = repo.UpdatePassword(ctx, repositories.UpdateUserPasswordRequest{
		UserID:             data.User.ID,
		OrganizationID:     data.Organization.ID,
		BusinessUnitID:     data.BusinessUnit.ID,
		Password:           updatedHash,
		MustChangePassword: false,
	})
	require.NoError(t, err)

	updatedUser, err := repo.GetByID(ctx, repositories.GetUserByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  data.Organization.ID,
			BuID:   data.BusinessUnit.ID,
			UserID: data.User.ID,
		},
	})
	require.NoError(t, err)
	assert.NotEqual(t, originalHash, updatedUser.Password)
	assert.Equal(t, updatedHash, updatedUser.Password)
	assert.False(t, updatedUser.MustChangePassword)
}

func TestReplaceOrganizationMembershipsReturnsConflictWhenMembershipsLocked(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	conn := postgres.NewTestConnection(db)
	repo := New(Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})

	data := seedtest.SeedFullTestData(t, ctx, db)

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	secondOrg := seedtest.NewOrganization(data.BusinessUnit.ID, data.State.ID).
		WithName("Second Organization").
		WithScacCode("SECN").
		WithDOTNumber("7654321").
		WithBucketName("second-org-bucket").
		Build(t, ctx, tx)
	require.NoError(t, tx.Commit())

	initialMemberships := []*tenant.OrganizationMembership{
		{
			UserID:         data.User.ID,
			BusinessUnitID: data.BusinessUnit.ID,
			OrganizationID: data.Organization.ID,
			GrantedByID:    data.User.ID,
			IsDefault:      true,
		},
		{
			UserID:         data.User.ID,
			BusinessUnitID: data.BusinessUnit.ID,
			OrganizationID: secondOrg.ID,
			GrantedByID:    data.User.ID,
			IsDefault:      false,
		},
	}
	_, err = db.NewInsert().Model(&initialMemberships).Exec(ctx)
	require.NoError(t, err)

	lock := inttestutil.HoldTxLock(
		t,
		conn,
		ports.TxOptions{},
		func(lockCtx context.Context, tx bun.Tx) error {
			locked := make([]*tenant.OrganizationMembership, 0)
			return tx.NewSelect().
				Model(&locked).
				Where("uom.user_id = ?", data.User.ID).
				Where("uom.business_unit_id = ?", data.BusinessUnit.ID).
				Order("uom.organization_id ASC").
				For("UPDATE").
				Scan(lockCtx)
		},
	)
	lock.WaitLocked(t)

	memberships, err := repo.ReplaceOrganizationMemberships(
		ctx,
		repositories.ReplaceOrganizationMembershipsRequest{
			ActorID:         data.User.ID,
			UserID:          data.User.ID,
			BusinessUnitID:  data.BusinessUnit.ID,
			OrganizationIDs: []pulid.ID{secondOrg.ID},
		},
	)

	require.Nil(t, memberships)
	require.Error(t, err)
	assert.True(t, errortypes.IsConflictError(err))
	assert.Contains(t, err.Error(), "organization memberships are busy")

	lock.Release()
	lock.Wait(t)

	finalMemberships, err := repo.ListOrganizationMemberships(
		ctx,
		data.User.ID,
		data.BusinessUnit.ID,
	)
	require.NoError(t, err)
	require.Len(t, finalMemberships, 2)

	defaultCount := 0
	for _, membership := range finalMemberships {
		if membership.IsDefault {
			defaultCount++
			assert.Equal(t, data.Organization.ID, membership.OrganizationID)
		}
	}
	assert.Equal(t, 1, defaultCount)
}
