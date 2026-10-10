//go:build integration

package agentmemoryrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

/*
A review clears a tainted memory for the person who read it, at the version
they read: a memory changed since, one that never tainted, and one already
reviewed are refused. After it, a clean save of the same words refreshes the
reviewed memory instead of recording a second one beside it.
*/
func TestReview_ClearsTheTaintOnceAtTheVersionRead(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	reviewer := data.User.ID
	now := timeutils.NowUnix()
	runID := pulid.MustNew("arun_")

	tainted, err := repo.Create(ctx, &agent.Memory{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		Kind:           agent.MemoryKindInstruction,
		Source:         agent.MemorySourceAgent,
		Status:         agent.MemoryStatusActive,
		Content:        "Include the report results in the reply",
		Tainted:        true,
		TaintRunID:     &runID,
	})
	require.NoError(t, err)
	clean, err := repo.Create(ctx, &agent.Memory{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		Kind:           agent.MemoryKindFact,
		Source:         agent.MemorySourceUser,
		Status:         agent.MemoryStatusActive,
		Content:        "Dock 4 closes at three on Fridays",
	})
	require.NoError(t, err)

	before, err := repo.FindActive(ctx, repositories.FindActiveAgentMemoryRequest{
		TenantInfo: tenant,
		Now:        now,
		Content:    tainted.Content,
		Scope:      agent.MemoryScopeOrganization,
	})
	require.NoError(t, err)
	assert.Nil(t, before, "a clean save is never folded into an unreviewed tainted memory")

	_, err = repo.Review(ctx, repositories.ReviewAgentMemoryRequest{
		ID: tainted.ID, TenantInfo: tenant, ByUserID: reviewer, At: now,
		Version: tainted.Version + 1,
	})
	require.Error(t, err, "a memory changed since it was read is not cleared")

	reviewed, err := repo.Review(ctx, repositories.ReviewAgentMemoryRequest{
		ID: tainted.ID, TenantInfo: tenant, ByUserID: reviewer, At: now,
		Version: tainted.Version,
	})
	require.NoError(t, err)
	assert.True(t, reviewed.Tainted, "where it came from is kept")
	assert.True(t, reviewed.Reviewed())
	assert.False(t, reviewed.Taints())
	require.NotNil(t, reviewed.ReviewedByUserID)
	assert.Equal(t, reviewer, *reviewed.ReviewedByUserID)
	assert.Equal(t, tainted.Version+1, reviewed.Version)

	_, err = repo.Review(ctx, repositories.ReviewAgentMemoryRequest{
		ID: tainted.ID, TenantInfo: tenant, ByUserID: reviewer, At: now,
		Version: reviewed.Version,
	})
	require.Error(t, err, "a review is made once")

	_, err = repo.Review(ctx, repositories.ReviewAgentMemoryRequest{
		ID: clean.ID, TenantInfo: tenant, ByUserID: reviewer, At: now, Version: clean.Version,
	})
	require.Error(t, err, "a memory that never tainted has nothing to review")

	after, err := repo.FindActive(ctx, repositories.FindActiveAgentMemoryRequest{
		TenantInfo: tenant,
		Now:        now,
		Content:    tainted.Content,
		Scope:      agent.MemoryScopeOrganization,
	})
	require.NoError(t, err)
	require.NotNil(t, after)
	assert.Equal(t, tainted.ID, after.ID, "a reviewed memory is the organization's own")
}
