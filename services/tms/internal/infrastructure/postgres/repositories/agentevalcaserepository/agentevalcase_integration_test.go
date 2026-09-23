//go:build integration

package agentevalcaserepository

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type definitionRow struct {
	ID             pulid.ID `bun:"id"`
	OrganizationID pulid.ID `bun:"organization_id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
}

type fixture struct {
	ctx        context.Context
	db         *bun.DB
	repo       repositories.AgentEvalCaseRepository
	tenant     pagination.TenantInfo
	definition pulid.ID
}

func setup(t *testing.T) *fixture {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	registry := seeder.NewRegistry()
	seeds.Register(registry)
	engine := seeder.NewEngine(
		db,
		registry,
		&config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}},
	)
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	cols := buncolgen.DefinitionColumns
	var row definitionRow
	require.NoError(t, db.NewSelect().
		TableExpr(buncolgen.DefinitionTable.As(buncolgen.DefinitionTable.Alias)).
		Column(cols.ID.Bare(), cols.OrganizationID.Bare(), cols.BusinessUnitID.Bare()).
		Limit(1).
		Scan(ctx, &row))

	return &fixture{
		ctx:        ctx,
		db:         db,
		repo:       New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()}),
		tenant:     pagination.TenantInfo{OrgID: row.OrganizationID, BuID: row.BusinessUnitID},
		definition: row.ID,
	}
}

func (f *fixture) newCase(t *testing.T, input string) *agentquality.EvalCase {
	t.Helper()

	evalCase := &agentquality.EvalCase{
		OrganizationID:    f.tenant.OrgID,
		BusinessUnitID:    f.tenant.BuID,
		AgentDefinitionID: f.definition,
		Title:             "Hold an unpaid shipment",
		Source:            agentquality.CaseSourceCurated,
		Status:            agentquality.CaseStatusActive,
		Trigger:           agent.RunTriggerChat,
		Input:             input,
		HeldTools:         []string{"get_shipment", "place_shipment_hold"},
		ToolFixtures: []agentquality.ToolFixture{{
			Tool:   "get_shipment",
			Args:   map[string]any{"id": "shp_1"},
			Result: map[string]any{"proNumber": "S-100", "creditLimit": "[restricted:creditLimit]"},
		}},
		Expected: agentquality.Expected{
			ToolMode: agentquality.ToolMatchAnyOrder,
			Tools: []agentquality.ExpectedTool{{
				Name: "place_shipment_hold",
				Args: map[string]any{"shipmentId": "shp_1"},
			}},
			MustMention: []string{"hold"},
		},
	}
	hash, err := evalCase.ComputeContentHash()
	require.NoError(t, err)
	evalCase.ContentHash = hash

	return evalCase
}

func otherTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func TestCreateAndRead_RoundTripsEveryFrozenField(t *testing.T) {
	f := setup(t)

	created, err := f.repo.Create(f.ctx, f.newCase(t, "Put S-100 on hold"))
	require.NoError(t, err)
	require.True(t, created.ID.IsNotNil())

	read, err := f.repo.GetByID(f.ctx, repositories.GetAgentEvalCaseByIDRequest{
		ID:         created.ID,
		TenantInfo: f.tenant,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"get_shipment", "place_shipment_hold"}, read.HeldTools)
	require.Len(t, read.ToolFixtures, 1)
	assert.Equal(t, "get_shipment", read.ToolFixtures[0].Tool)
	assert.Equal(t, agentquality.ToolMatchAnyOrder, read.Expected.ToolMode)
	require.Len(t, read.Expected.Tools, 1)
	assert.Equal(t, "place_shipment_hold", read.Expected.Tools[0].Name)
	assert.Equal(t, created.ContentHash, read.ContentHash)
}

func TestTenantIsolation_AnotherTenantSeesNothing(t *testing.T) {
	f := setup(t)

	created, err := f.repo.Create(f.ctx, f.newCase(t, "Put S-101 on hold"))
	require.NoError(t, err)
	other := otherTenant()

	_, err = f.repo.GetByID(f.ctx, repositories.GetAgentEvalCaseByIDRequest{
		ID:         created.ID,
		TenantInfo: other,
	})
	assert.True(t, errortypes.IsNotFoundError(err))

	_, err = f.repo.GetByContent(f.ctx, repositories.GetAgentEvalCaseByContentRequest{
		AgentDefinitionID: f.definition,
		ContentHash:       created.ContentHash,
		TenantInfo:        other,
	})
	assert.True(t, errortypes.IsNotFoundError(err))

	listed, err := f.repo.ListByAgent(f.ctx, repositories.ListAgentEvalCasesRequest{
		AgentDefinitionID: f.definition,
		TenantInfo:        other,
	})
	require.NoError(t, err)
	assert.Empty(t, listed)

	stolen := *created
	stolen.OrganizationID = other.OrgID
	stolen.BusinessUnitID = other.BuID
	stolen.Title = "Rewritten from elsewhere"
	_, err = f.repo.Update(f.ctx, &stolen)
	require.Error(t, err, "an update scoped to another tenant touches nothing")

	purged, err := f.repo.PurgeExpired(f.ctx, repositories.PurgeExpiredEvalCasesRequest{
		TenantInfo:    other,
		CreatedBefore: timeutils.NowUnix() + 3600,
		Now:           timeutils.NowUnix(),
	})
	require.NoError(t, err)
	assert.Zero(t, purged)

	still, err := f.repo.GetByID(f.ctx, repositories.GetAgentEvalCaseByIDRequest{
		ID:         created.ID,
		TenantInfo: f.tenant,
	})
	require.NoError(t, err)
	assert.Equal(t, "Hold an unpaid shipment", still.Title)
}

func TestCreate_RefusesTheSameFrozenInputTwice(t *testing.T) {
	f := setup(t)

	_, err := f.repo.Create(f.ctx, f.newCase(t, "Put S-102 on hold"))
	require.NoError(t, err)

	_, err = f.repo.Create(f.ctx, f.newCase(t, "Put S-102 on hold"))
	require.Error(t, err)
	assert.True(t, dberror.IsUniqueConstraintViolation(err))

	found, err := f.repo.GetByContent(f.ctx, repositories.GetAgentEvalCaseByContentRequest{
		AgentDefinitionID: f.definition,
		ContentHash:       f.newCase(t, "Put S-102 on hold").ContentHash,
		TenantInfo:        f.tenant,
	})
	require.NoError(t, err)
	assert.Equal(t, "Put S-102 on hold", found.Input)
}

func TestUpdate_ChecksTheVersion(t *testing.T) {
	f := setup(t)

	created, err := f.repo.Create(f.ctx, f.newCase(t, "Put S-103 on hold"))
	require.NoError(t, err)

	created.Status = agentquality.CaseStatusQuarantined
	updated, err := f.repo.Update(f.ctx, created)
	require.NoError(t, err)
	assert.Equal(t, int64(1), updated.Version)

	stale := *updated
	stale.Version = 0
	_, err = f.repo.Update(f.ctx, &stale)
	require.Error(t, err)
}

func TestPurge_ExpiredAndOrphanedCasesGo(t *testing.T) {
	f := setup(t)

	past := timeutils.NowUnix() - 60
	expired := f.newCase(t, "Put S-104 on hold")
	expired.ExpiresAt = &past
	_, err := f.repo.Create(f.ctx, expired)
	require.NoError(t, err)

	missingThread := pulid.MustNew("athr_")
	orphan := f.newCase(t, "Put S-105 on hold")
	orphan.SourceThreadID = &missingThread
	_, err = f.repo.Create(f.ctx, orphan)
	require.NoError(t, err)

	kept, err := f.repo.Create(f.ctx, f.newCase(t, "Put S-106 on hold"))
	require.NoError(t, err)

	purged, err := f.repo.PurgeExpired(f.ctx, repositories.PurgeExpiredEvalCasesRequest{
		TenantInfo: f.tenant,
		Now:        timeutils.NowUnix(),
	})
	require.NoError(t, err)
	assert.Equal(t, 1, purged)

	orphans, err := f.repo.PurgeOrphaned(f.ctx, repositories.PurgeOrphanedEvalCasesRequest{})
	require.NoError(t, err)
	assert.Equal(t, 1, orphans)

	remaining, err := f.repo.ListByAgent(f.ctx, repositories.ListAgentEvalCasesRequest{
		AgentDefinitionID: f.definition,
		TenantInfo:        f.tenant,
	})
	require.NoError(t, err)
	require.Len(t, remaining, 1)
	assert.Equal(t, kept.ID, remaining[0].ID)
}
