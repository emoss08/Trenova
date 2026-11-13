//go:build integration

package agentevalcaserepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentevaluation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentevaluation/agentevaluationdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentquality"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentqualitycontrol"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentqualitycontrol/agentqualitycontroldb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentqualitysuite"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentreplay"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentqualityports"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type definitionRow struct {
	ID             pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

type fixture struct {
	base       context.Context
	ctx        context.Context
	db         *keel.DB
	repo       agentqualityports.AgentEvalCaseRepository
	tenant     pagination.TenantInfo
	definition pulid.ID
}

func setup(t *testing.T) *fixture {
	t.Helper()

	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	rows, err := keel.QueryAll(
		ctx,
		db,
		pgx.RowToStructByPos[definitionRow],
		"SELECT id, organization_id, business_unit_id FROM agent_definitions LIMIT 1",
	)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	row := rows[0]
	tenant := pagination.TenantInfo{OrgID: row.OrganizationID, BuID: row.BusinessUnitID}

	return &fixture{
		base:       ctx,
		ctx:        dbscope.WithTenant(ctx, tenant.DBTenant()),
		db:         db,
		repo:       New(Params{DB: postgres.WrapDB(db), Logger: zap.NewNop()}),
		tenant:     tenant,
		definition: row.ID,
	}
}

func (f *fixture) scoped(tenant pagination.TenantInfo) context.Context {
	return dbscope.WithTenant(f.base, tenant.DBTenant())
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
		HeldTools:         []string{"shipment_get", "shipment_place_hold"},
		ToolFixtures: []agentquality.ToolFixture{{
			Tool:   "shipment_get",
			Args:   map[string]any{"id": "shp_1"},
			Result: map[string]any{"proNumber": "S-100", "creditLimit": "[restricted:creditLimit]"},
		}},
		Expected: agentquality.Expected{
			ToolMode: agentquality.ToolMatchAnyOrder,
			Tools: []agentquality.ExpectedTool{{
				Name: "shipment_place_hold",
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

	frozen := f.newCase(t, "Put S-100 on hold")
	asked := int64(1_772_530_200)
	frozen.AskedAt = &asked
	created, err := f.repo.Create(f.ctx, frozen)
	require.NoError(t, err)
	require.True(t, created.ID.IsNotNil())

	read, err := f.repo.GetByID(f.ctx, agentqualityports.GetAgentEvalCaseByIDRequest{
		ID:         created.ID,
		TenantInfo: f.tenant,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"shipment_get", "shipment_place_hold"}, read.HeldTools)
	require.Len(t, read.ToolFixtures, 1)
	assert.Equal(t, "shipment_get", read.ToolFixtures[0].Tool)
	assert.Equal(t, agentquality.ToolMatchAnyOrder, read.Expected.ToolMode)
	require.Len(t, read.Expected.Tools, 1)
	assert.Equal(t, "shipment_place_hold", read.Expected.Tools[0].Name)
	assert.Equal(t, created.ContentHash, read.ContentHash)
	assert.Equal(t, asked, read.Asked())

	written, err := f.repo.Create(f.ctx, f.newCase(t, "Put S-110 on hold"))
	require.NoError(t, err)
	undated, err := f.repo.GetByID(f.ctx, agentqualityports.GetAgentEvalCaseByIDRequest{
		ID:         written.ID,
		TenantInfo: f.tenant,
	})
	require.NoError(t, err)
	assert.Nil(t, undated.AskedAt, "a case written by hand was never asked at a moment")
}

func TestTenantIsolation_AnotherTenantSeesNothing(t *testing.T) {
	f := setup(t)

	created, err := f.repo.Create(f.ctx, f.newCase(t, "Put S-101 on hold"))
	require.NoError(t, err)
	other := otherTenant()
	otherCtx := f.scoped(other)

	_, err = f.repo.GetByID(otherCtx, agentqualityports.GetAgentEvalCaseByIDRequest{
		ID:         created.ID,
		TenantInfo: other,
	})
	assert.True(t, errortypes.IsNotFoundError(err))

	_, err = f.repo.GetByContent(otherCtx, agentqualityports.GetAgentEvalCaseByContentRequest{
		AgentDefinitionID: f.definition,
		ContentHash:       created.ContentHash,
		TenantInfo:        other,
	})
	assert.True(t, errortypes.IsNotFoundError(err))

	listed, err := f.repo.ListByAgent(otherCtx, agentqualityports.ListAgentEvalCasesRequest{
		AgentDefinitionID: f.definition,
		TenantInfo:        other,
	})
	require.NoError(t, err)
	assert.Empty(t, listed)

	stolen := *created
	stolen.OrganizationID = other.OrgID
	stolen.BusinessUnitID = other.BuID
	stolen.Title = "Rewritten from elsewhere"
	_, err = f.repo.Update(otherCtx, &stolen)
	require.Error(t, err, "an update scoped to another tenant touches nothing")

	purged, err := f.repo.PurgeExpired(otherCtx, agentqualityports.PurgeExpiredEvalCasesRequest{
		TenantInfo:    other,
		CreatedBefore: timeutils.NowUnix() + 3600,
		Now:           timeutils.NowUnix(),
	})
	require.NoError(t, err)
	assert.Zero(t, purged)

	still, err := f.repo.GetByID(f.ctx, agentqualityports.GetAgentEvalCaseByIDRequest{
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

	found, err := f.repo.GetByContent(f.ctx, agentqualityports.GetAgentEvalCaseByContentRequest{
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

	purged, err := f.repo.PurgeExpired(f.ctx, agentqualityports.PurgeExpiredEvalCasesRequest{
		TenantInfo: f.tenant,
		Now:        timeutils.NowUnix(),
	})
	require.NoError(t, err)
	assert.Equal(t, 1, purged)

	orphans, err := f.repo.PurgeOrphaned(f.ctx, agentqualityports.PurgeOrphanedEvalCasesRequest{})
	require.NoError(t, err)
	assert.Equal(t, 1, orphans)

	remaining, err := f.repo.ListByAgent(f.ctx, agentqualityports.ListAgentEvalCasesRequest{
		AgentDefinitionID: f.definition,
		TenantInfo:        f.tenant,
	})
	require.NoError(t, err)
	require.Len(t, remaining, 1)
	assert.Equal(t, kept.ID, remaining[0].ID)
}

func (f *fixture) automaticCase(t *testing.T, input, tool string) *agentquality.EvalCase {
	t.Helper()

	proposalID := pulid.MustNew("aprop_")
	evalCase := f.newCase(t, input)
	evalCase.Source = agentquality.CaseSourceDecidedProposal
	evalCase.SourceProposalID = &proposalID
	evalCase.Automatic = true
	evalCase.Expected = agentquality.Expected{
		ToolMode: agentquality.ToolMatchAnyOrder,
		Proposals: []agentquality.ExpectedProposal{{
			ToolName:         tool,
			Params:           map[string]any{"rate": 1350},
			SourceProposalID: proposalID.String(),
		}},
	}

	return evalCase
}

func (f *fixture) statusOf(t *testing.T, id pulid.ID) *agentquality.EvalCase {
	t.Helper()

	evalCase, err := f.repo.GetByID(f.ctx, agentqualityports.GetAgentEvalCaseByIDRequest{
		ID:         id,
		TenantInfo: f.tenant,
	})
	require.NoError(t, err)

	return evalCase
}

func TestListAutomatic_ReturnsTheSystemsActiveCasesWithTheToolTheyExpect(t *testing.T) {
	f := setup(t)

	automatic, err := f.repo.Create(f.ctx, f.automaticCase(t, "Rate S-201", "update_rate"))
	require.NoError(t, err)
	retired := f.automaticCase(t, "Rate S-202", "update_rate")
	retired.Status = agentquality.CaseStatusRetired
	_, err = f.repo.Create(f.ctx, retired)
	require.NoError(t, err)
	_, err = f.repo.Create(f.ctx, f.newCase(t, "Put S-203 on hold"))
	require.NoError(t, err)

	listed, err := f.repo.ListAutomatic(f.ctx, agentqualityports.ListAutomaticEvalCasesRequest{
		TenantInfo:        f.tenant,
		AgentDefinitionID: f.definition,
	})
	require.NoError(t, err)

	require.Len(t, listed, 1, "a retired case and a case a person wrote are not the system's to rotate")
	assert.Equal(t, automatic.ID, listed[0].ID)
	assert.Equal(t, "update_rate", listed[0].Tool)
	assert.Equal(t, automatic.CreatedAt, listed[0].CreatedAt)

	other := otherTenant()
	elsewhere, err := f.repo.ListAutomatic(
		f.scoped(other),
		agentqualityports.ListAutomaticEvalCasesRequest{
			TenantInfo:        other,
			AgentDefinitionID: f.definition,
		},
	)
	require.NoError(t, err)
	assert.Empty(t, elsewhere)
}

func TestRetireAutomatic_RetiresOnlyCasesTheSystemStillKeeps(t *testing.T) {
	f := setup(t)

	automatic, err := f.repo.Create(f.ctx, f.automaticCase(t, "Rate S-211", "update_rate"))
	require.NoError(t, err)
	person, err := f.repo.Create(f.ctx, f.newCase(t, "Put S-212 on hold"))
	require.NoError(t, err)
	ids := []pulid.ID{automatic.ID, person.ID}
	at := timeutils.NowUnix() + 5

	other := otherTenant()
	elsewhere, err := f.repo.RetireAutomatic(
		f.scoped(other),
		&agentqualityports.RetireAutomaticEvalCasesRequest{TenantInfo: other, IDs: ids, At: at},
	)
	require.NoError(t, err)
	assert.Zero(t, elsewhere, "another organization cannot retire these cases")
	assert.Equal(t, agentquality.CaseStatusActive, f.statusOf(t, automatic.ID).Status)

	retired, err := f.repo.RetireAutomatic(
		f.ctx,
		&agentqualityports.RetireAutomaticEvalCasesRequest{TenantInfo: f.tenant, IDs: ids, At: at},
	)
	require.NoError(t, err)
	assert.Equal(t, 1, retired)

	gone := f.statusOf(t, automatic.ID)
	assert.Equal(t, agentquality.CaseStatusRetired, gone.Status)
	assert.Equal(t, at, gone.UpdatedAt)
	assert.Equal(t, automatic.Version+1, gone.Version,
		"a save from a sheet opened before the rotation is refused, not written over it")
	assert.Equal(t, agentquality.CaseStatusActive, f.statusOf(t, person.ID).Status,
		"a case a person keeps stays even when its id is named")

	none, err := f.repo.RetireAutomatic(
		f.ctx,
		&agentqualityports.RetireAutomaticEvalCasesRequest{TenantInfo: f.tenant, At: at},
	)
	require.NoError(t, err)
	assert.Zero(t, none)
}

func TestRetireStaleAutomatic_RetiresAgedAutomaticCasesAndLeavesTheRest(t *testing.T) {
	f := setup(t)

	automatic, err := f.repo.Create(f.ctx, f.automaticCase(t, "Rate S-221", "update_rate"))
	require.NoError(t, err)
	person, err := f.repo.Create(f.ctx, f.newCase(t, "Put S-222 on hold"))
	require.NoError(t, err)
	now := timeutils.NowUnix()

	fresh, err := f.repo.RetireStaleAutomatic(
		f.base,
		agentqualityports.RetireStaleAutomaticEvalCasesRequest{
			CreatedBefore: automatic.CreatedAt,
			At:            now,
		},
	)
	require.NoError(t, err)
	assert.Zero(t, fresh, "a case created at the cutoff is not yet past it")

	aged, err := f.repo.RetireStaleAutomatic(
		f.base,
		agentqualityports.RetireStaleAutomaticEvalCasesRequest{
			CreatedBefore: automatic.CreatedAt + 1,
			At:            now,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, 1, aged)
	assert.Equal(t, agentquality.CaseStatusRetired, f.statusOf(t, automatic.ID).Status)
	assert.Equal(t, agentquality.CaseStatusActive, f.statusOf(t, person.ID).Status)

	again, err := f.repo.RetireStaleAutomatic(
		f.base,
		agentqualityports.RetireStaleAutomaticEvalCasesRequest{
			CreatedBefore: automatic.CreatedAt + 1,
			At:            now,
		},
	)
	require.NoError(t, err)
	assert.Zero(t, again)
}

func (f *fixture) failedReplay(t *testing.T, caseID pulid.ID, trialProvider *pulid.ID) {
	t.Helper()

	completed := timeutils.NowUnix()
	id := caseID
	err := agentevaluationdb.Evaluations.Create(f.ctx, f.db, &agentevaluation.Evaluation{
		ID:                pulid.MustNew(agentevaluation.EvaluationIDPrefix),
		OrganizationID:    f.tenant.OrgID,
		BusinessUnitID:    f.tenant.BuID,
		AgentDefinitionID: f.definition,
		EvalCaseID:        &id,
		Status:            agentevaluation.EvaluationStatusFailed,
		Trigger:           agent.RunTriggerChat,
		Actions:           []agentreplay.ReplayAction{},
		CompletedAt:       &completed,
		TrialProviderID:   trialProvider,
	})
	require.NoError(t, err)
}

func (f *fixture) sampling(t *testing.T) map[pulid.ID]agentqualitysuite.SamplingCase {
	t.Helper()

	cases, err := f.repo.ListSamplingCases(f.ctx, agentqualityports.ListEvalCaseSamplingRequest{
		TenantInfo:        f.tenant,
		AgentDefinitionID: f.definition,
	})
	require.NoError(t, err)
	byID := make(map[pulid.ID]agentqualitysuite.SamplingCase, len(cases))
	for _, evalCase := range cases {
		byID[evalCase.ID] = evalCase
	}

	return byID
}

func TestListSamplingCases_ACaseThatFailedOnlyOnATrialIsNotFailing(t *testing.T) {
	f := setup(t)

	tried, err := f.repo.Create(f.ctx, f.newCase(t, "Put S-231 on hold"))
	require.NoError(t, err)
	broken, err := f.repo.Create(f.ctx, f.newCase(t, "Put S-232 on hold"))
	require.NoError(t, err)
	providerID := pulid.MustNew("aip_")
	f.failedReplay(t, tried.ID, &providerID)
	f.failedReplay(t, broken.ID, nil)

	cases := f.sampling(t)

	require.Contains(t, cases, tried.ID)
	require.Contains(t, cases, broken.ID)
	assert.False(t, cases[tried.ID].Failing(),
		"a failure on a model the agent does not run on does not put the case first in its next run")
	assert.True(t, cases[broken.ID].Failing())
}

func (f *fixture) candidate(t *testing.T, input string) *agentquality.EvalCase {
	t.Helper()

	evalCase := f.automaticCase(t, input, "update_rate")
	evalCase.Status = agentquality.CaseStatusCandidate
	evalCase.Automatic = false
	created, err := f.repo.Create(f.ctx, evalCase)
	require.NoError(t, err)

	return created
}

func TestAdoptCandidates_ActivatesUntouchedCandidatesOfOrganizationsThatKeepTheirSet(t *testing.T) {
	f := setup(t)

	captured := f.candidate(t, "Rate S-241")
	written := f.newCase(t, "Put S-242 on hold")
	written.Status = agentquality.CaseStatusCandidate
	written, err := f.repo.Create(f.ctx, written)
	require.NoError(t, err)
	now := timeutils.NowUnix()

	none, err := f.repo.AdoptCandidates(f.base, agentqualityports.AdoptEvalCaseCandidatesRequest{
		CreatedAfter: captured.CreatedAt + 1,
		At:           now,
	})
	require.NoError(t, err)
	assert.Empty(t, none, "a candidate older than an automatic case lives is left a candidate")

	adopted, err := f.repo.AdoptCandidates(f.base, agentqualityports.AdoptEvalCaseCandidatesRequest{
		CreatedAfter: captured.CreatedAt,
		At:           now,
	})
	require.NoError(t, err)
	require.Len(t, adopted, 1)
	assert.Equal(t, captured.ID, adopted[0].ID)
	assert.Equal(t, f.tenant.OrgID, adopted[0].OrganizationID)
	assert.Equal(t, f.definition, adopted[0].AgentDefinitionID)

	active := f.statusOf(t, captured.ID)
	assert.Equal(t, agentquality.CaseStatusActive, active.Status)
	assert.True(t, active.Automatic)
	assert.Equal(t, captured.Version+1, active.Version)
	assert.Equal(t, agentquality.CaseStatusCandidate, f.statusOf(t, written.ID).Status,
		"a case a person wrote waits for a person to activate it")

	again, err := f.repo.AdoptCandidates(f.base, agentqualityports.AdoptEvalCaseCandidatesRequest{
		CreatedAfter: captured.CreatedAt,
		At:           now,
	})
	require.NoError(t, err)
	assert.Empty(t, again)
}

func TestAdoptCandidates_LeavesAnOrganizationThatTurnedTheAutomaticSetOff(t *testing.T) {
	f := setup(t)

	control := agentqualitycontrol.DefaultControl(f.tenant.OrgID, f.tenant.BuID)
	control.AutoActivate = false
	require.NoError(t, agentqualitycontroldb.Controls.Create(f.ctx, f.db, control))
	captured := f.candidate(t, "Rate S-251")

	adopted, err := f.repo.AdoptCandidates(f.base, agentqualityports.AdoptEvalCaseCandidatesRequest{
		CreatedAfter: captured.CreatedAt,
		At:           timeutils.NowUnix(),
	})
	require.NoError(t, err)

	assert.Empty(t, adopted)
	assert.Equal(t, agentquality.CaseStatusCandidate, f.statusOf(t, captured.ID).Status)
}
