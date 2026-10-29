package agentmemoryservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentmemorysubjects"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeMemoryRepo struct {
	agentmemoryports.AgentMemoryRepository
	created []*agentmemory.Memory
	active  []*agentmemory.Memory
	found   *agentmemory.Memory
	used    []pulid.ID
	listed  *agentmemoryports.ListActiveAgentMemoriesRequest
	sought  *agentmemoryports.FindActiveAgentMemoryRequest

	activeCount int
}

func (f *fakeMemoryRepo) CountActive(
	context.Context,
	agentmemoryports.CountActiveAgentMemoriesRequest,
) (int, error) {
	return f.activeCount, nil
}

func (f *fakeMemoryRepo) Create(
	_ context.Context,
	entity *agentmemory.Memory,
) (*agentmemory.Memory, error) {
	entity.ID = pulid.MustNew("amem_")
	f.created = append(f.created, entity)

	return entity, nil
}

func (f *fakeMemoryRepo) FindActive(
	_ context.Context,
	req agentmemoryports.FindActiveAgentMemoryRequest,
) (*agentmemory.Memory, error) {
	f.sought = &req

	return f.found, nil
}

func (f *fakeMemoryRepo) ListActive(
	_ context.Context,
	req agentmemoryports.ListActiveAgentMemoriesRequest,
) ([]*agentmemory.Memory, error) {
	f.listed = &req

	return f.active, nil
}

func (f *fakeMemoryRepo) MarkUsed(
	_ context.Context,
	req agentmemoryports.MarkAgentMemoriesUsedRequest,
) error {
	f.used = append(f.used, req.IDs...)

	return nil
}

type fakeRuns struct {
	run *agent.AgentRun
}

func (f *fakeRuns) GetByID(
	context.Context,
	agentruntimeports.GetAgentRunByIDRequest,
) (*agent.AgentRun, error) {
	return f.run, nil
}

type fakeLabeler struct {
	label string
	err   error
	asked *agentmemoryports.MemorySubjectRef
}

func (f *fakeLabeler) Label(
	_ context.Context,
	_ pagination.TenantInfo,
	ref agentmemoryports.MemorySubjectRef,
) (string, error) {
	f.asked = &ref

	return f.label, f.err
}

func newService(repo *fakeMemoryRepo, runs *fakeRuns, labeler *fakeLabeler) *Service {
	return &Service{l: zap.NewNop(), repo: repo, runs: runs, labeler: labeler}
}

// fakePermissions answers every check the same way: a manager of shared
// memory, or not.
type fakePermissions struct {
	permissionports.PermissionEngine
	manages bool
	checks  []*permissionports.PermissionCheckRequest
}

func (f *fakePermissions) Check(
	_ context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	f.checks = append(f.checks, req)

	return &permissionports.PermissionCheckResult{Allowed: f.manages}, nil
}

func agentActor() *permissionports.RequestActor {
	return &permissionports.RequestActor{
		PrincipalType: permissionports.PrincipalTypeAgent,
		PrincipalID:   pulid.MustNew("agdef_"),
	}
}

// A memory for the organization reaches everyone's conversations, so a person
// who does not manage shared memory offers it and an administrator keeps it;
// one who does keeps it at once. A run with nobody in it always offers.
func TestRemember_SharedScopeIsOfferedUnlessTheActorManagesSharedMemory(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		actor   *permissionports.RequestActor
		manages bool
		scope   agentmemory.MemoryScope
		want    agentmemory.MemoryStatus
	}{
		{
			"a person without manage offers to the organization",
			userActor(),
			false,
			agentmemory.MemoryScopeOrganization,
			agentmemory.MemoryStatusSuggested,
		},
		{
			"a person with manage keeps for the organization",
			userActor(),
			true,
			agentmemory.MemoryScopeOrganization,
			agentmemory.MemoryStatusActive,
		},
		{
			"a person without manage offers to a team",
			userActor(),
			false,
			agentmemory.MemoryScopeRole,
			agentmemory.MemoryStatusSuggested,
		},
		{
			"a person keeps their own without manage",
			userActor(),
			false,
			agentmemory.MemoryScopeUser,
			agentmemory.MemoryStatusActive,
		},
		{
			"an unattended run offers to the organization",
			agentActor(),
			true,
			agentmemory.MemoryScopeOrganization,
			agentmemory.MemoryStatusSuggested,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := &fakeMemoryRepo{}
			svc := newService(repo, &fakeRuns{run: &agent.AgentRun{}}, &fakeLabeler{})
			permissions := &fakePermissions{manages: tc.manages}
			svc.permissions = permissions

			req := &agentmemoryports.RememberRequest{
				TenantInfo: tenant(),
				Kind:       agentmemory.MemoryKindInstruction,
				Content:    "Quote in dollars.",
				Scope:      tc.scope,
			}
			switch tc.scope {
			case agentmemory.MemoryScopeUser:
				req.OwnerUserID = tc.actor.UserID
			case agentmemory.MemoryScopeRole:
				req.RoleID = pulid.MustNew("rol_")
			default:
			}
			got, err := svc.Remember(t.Context(), req, tc.actor)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got.Status)
			assert.Equal(t, tc.scope, got.Scope, "offering never narrows who the memory is for")
			if tc.actor.IsUser() && tc.scope != agentmemory.MemoryScopeUser {
				require.NotEmpty(t, permissions.checks)
				assert.Equal(
					t,
					permission.ResourceAgentMemory.String(),
					permissions.checks[0].Resource,
				)
				assert.Equal(t, permission.OpManage, permissions.checks[0].Operation)
			}
		})
	}
}

// Nobody holds the permission that cannot be checked: a memory for the
// organization is offered rather than kept.
func TestRemember_WithoutAPermissionEngineSharedMemoryIsOffered(t *testing.T) {
	t.Parallel()

	svc := newService(&fakeMemoryRepo{}, &fakeRuns{run: &agent.AgentRun{}}, &fakeLabeler{})

	got, err := svc.Remember(t.Context(), &agentmemoryports.RememberRequest{
		TenantInfo: tenant(),
		Content:    "Quote in dollars.",
		Scope:      agentmemory.MemoryScopeOrganization,
	}, userActor())
	require.NoError(t, err)
	assert.Equal(t, agentmemory.MemoryStatusSuggested, got.Status)
}

func tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

type fakeLinks struct {
	links map[agentmemory.MemoryRecordKind][]agentmemoryports.MemoryRecordLink
}

func (f *fakeLinks) ListRecordLinks(
	_ context.Context,
	req *agentmemoryports.ListMemoryRecordLinksRequest,
) ([]agentmemoryports.MemoryRecordLink, error) {
	wanted := make(map[pulid.ID]struct{}, len(req.IDs))
	for _, id := range req.IDs {
		wanted[id] = struct{}{}
	}
	out := make([]agentmemoryports.MemoryRecordLink, 0)
	for _, link := range f.links[req.Kind] {
		if _, ok := wanted[link.From]; ok {
			out = append(out, link)
		}
	}

	return out, nil
}

func userActor() *permissionports.RequestActor {
	return &permissionports.RequestActor{
		PrincipalType: permissionports.PrincipalTypeUser,
		UserID:        pulid.MustNew("usr_"),
	}
}

// A memory recorded from inside a run belongs to the agent that ran, whoever
// approved the tool call, and names the record it is about by name rather
// than id.
func TestRemember_RecordsForTheAgentAndLabelsTheSubject(t *testing.T) {
	t.Parallel()

	repo := &fakeMemoryRepo{}
	definitionID := pulid.MustNew("agdef_")
	runs := &fakeRuns{run: &agent.AgentRun{AgentDefinitionID: definitionID}}
	labeler := &fakeLabeler{label: "Acme Freight"}
	svc := newService(repo, runs, labeler)

	customerID := pulid.MustNew("cus_")
	created, err := svc.Remember(t.Context(), &agentmemoryports.RememberRequest{
		TenantInfo:  tenant(),
		Kind:        agentmemory.MemoryKindInstruction,
		Content:     "  Needs the POD within one day of delivery.  ",
		SubjectType: agentmemory.MemorySubjectCustomer,
		SubjectID:   customerID,
		RunID:       pulid.MustNew("arun_"),
	}, userActor())
	require.NoError(t, err)

	assert.Equal(t, agentmemory.MemorySourceAgent, created.Source)
	assert.Equal(t, definitionID, *created.AgentDefinitionID)
	assert.Equal(t, "Acme Freight", created.SubjectLabel)
	assert.Equal(t, "Needs the POD within one day of delivery.", created.Content)
	assert.Equal(t, customerID, labeler.asked.ID)
	assert.NotNil(t, created.CreatedByUserID, "the approver is kept alongside the agent")
}

func TestRemember_RefusesASubjectThatDoesNotExist(t *testing.T) {
	t.Parallel()

	svc := newService(&fakeMemoryRepo{}, &fakeRuns{}, &fakeLabeler{err: assert.AnError})

	_, err := svc.Remember(t.Context(), &agentmemoryports.RememberRequest{
		TenantInfo:  tenant(),
		Content:     "x",
		SubjectType: agentmemory.MemorySubjectWorker,
		SubjectID:   pulid.MustNew("wrk_"),
	}, userActor())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "No worker with that id")
}

// An agent told "Quote in dollars" on Monday and again on Friday kept two
// identical memories, both carried by every prompt. Saying it again now
// refreshes the one already kept: counted as used, so it does not go stale,
// and marked so the agent can tell the person nothing new was saved.
func TestRemember_RefreshesTheExistingRowForTheSameSentence(t *testing.T) {
	t.Parallel()

	existing := &agentmemory.Memory{ID: pulid.MustNew("amem_"), Content: "Same", UseCount: 3}
	repo := &fakeMemoryRepo{found: existing}
	svc := newService(repo, &fakeRuns{}, &fakeLabeler{})

	got, err := svc.Remember(t.Context(), &agentmemoryports.RememberRequest{
		TenantInfo: tenant(),
		Content:    "Same",
	}, userActor())
	require.NoError(t, err)

	assert.Same(t, existing, got)
	assert.Empty(t, repo.created)
	assert.True(t, got.Refreshed)
	assert.Equal(t, []pulid.ID{existing.ID}, repo.used)
	assert.Equal(t, 4, got.UseCount)
	require.NotNil(t, got.LastUsedAt)
	assert.Positive(t, *got.LastUsedAt)
}

func TestRemember_ANewMemoryIsNotMarkedRefreshed(t *testing.T) {
	t.Parallel()

	repo := &fakeMemoryRepo{}
	created, err := newService(repo, &fakeRuns{}, &fakeLabeler{}).Remember(t.Context(),
		&agentmemoryports.RememberRequest{TenantInfo: tenant(), Content: "Quote in dollars."},
		userActor(),
	)
	require.NoError(t, err)

	assert.False(t, created.Refreshed)
	assert.Empty(t, repo.used, "saving a memory is not using it")
}

func TestRemember_LooksForTheSameMemoryOnlyWhereItWouldBeRead(t *testing.T) {
	t.Parallel()

	definitionID := pulid.MustNew("agdef_")
	runs := &fakeRuns{run: &agent.AgentRun{AgentDefinitionID: definitionID}}

	clean := &fakeMemoryRepo{}
	_, err := newService(clean, runs, &fakeLabeler{}).Remember(t.Context(),
		&agentmemoryports.RememberRequest{TenantInfo: tenant(), Content: "Quote in dollars."},
		userActor(),
	)
	require.NoError(t, err)
	require.NotNil(t, clean.sought)
	assert.False(t, clean.sought.Tainted, "a person's own write never matches a tainted row")
	assert.Equal(t, agentmemory.MemoryScopeOrganization, clean.sought.Scope)
	assert.True(t, clean.sought.AgentDefinitionID.IsNil())
	assert.Positive(t, clean.sought.Now, "an expired row is not the same memory")

	tainted := &fakeMemoryRepo{}
	taint := &agent.RunTaint{}
	taint.Add(agent.TaintMark{Source: agent.TaintSourceInboundMessage})
	_, err = newService(tainted, runs, &fakeLabeler{}).Remember(t.Context(),
		&agentmemoryports.RememberRequest{
			TenantInfo: tenant(),
			Content:    "Ship to dock 9.",
			RunID:      pulid.MustNew("arun_"),
			Taint:      taint,
		},
		userActor(),
	)
	require.NoError(t, err)
	require.NotNil(t, tainted.sought)
	assert.True(t, tainted.sought.Tainted)
	assert.Equal(t, definitionID, tainted.sought.AgentDefinitionID)
	assert.Equal(t, agentmemory.MemoryScopeOrganization, tainted.sought.Scope,
		"an agent's remember is read by every agent, so it matches only organization rows")
}

func TestForContext_ReadsCandidatesWithoutCountingThemUsed(t *testing.T) {
	t.Parallel()

	one := &agentmemory.Memory{ID: pulid.MustNew("amem_")}
	repo := &fakeMemoryRepo{active: []*agentmemory.Memory{one}}
	svc := newService(repo, &fakeRuns{}, &fakeLabeler{})

	got, err := svc.ForContext(t.Context(), agentmemoryports.MemoryContextRequest{
		TenantInfo: tenant(),
		ToolNames:  []string{"dispatch_assign_move"},
	})
	require.NoError(t, err)

	assert.Equal(t, []*agentmemory.Memory{one}, got.Memories)
	assert.True(t, repo.listed.OrganizationWide)
	assert.Equal(t, []string{"dispatch_assign_move"}, repo.listed.ToolNames)
	assert.Equal(t, agentmemory.MaxMemoryCandidates, repo.listed.Limit)
	assert.Empty(t, repo.used, "a candidate is counted only once a prompt carries it")
}

func TestForContext_ReadsTheMemoriesOfTheRecordsTheTurnIsAbout(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	shipmentID := pulid.MustNew("shp_")
	locationID := pulid.MustNew("loc_")
	repo := &fakeMemoryRepo{}
	svc := newService(repo, &fakeRuns{}, &fakeLabeler{})
	links := map[agentmemory.MemoryRecordKind][]agentmemoryports.MemoryRecordLink{
		agentmemory.MemoryRecordShipment: {
			{From: shipmentID, Kind: agentmemory.MemoryRecordLocation, ID: locationID},
		},
	}
	svc.subjects = agentmemorysubjects.NewSubjectResolver(&fakeLinks{links: links})

	got, err := svc.ForContext(t.Context(), agentmemoryports.MemoryContextRequest{
		TenantInfo: tenant(),
		Records: []agentpagecontext.EntityRef{
			{Type: "customer", ID: customerID.String()},
			{Type: string(agent.SubjectShipment), ID: shipmentID.String()},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, []agentmemory.MemorySubject{
		{
			Type:     agentmemory.MemorySubjectCustomer,
			ID:       customerID,
			Relation: agentmemory.MemoryRelationDirect,
		},
		{
			Type:     agentmemory.MemorySubjectLocation,
			ID:       locationID,
			Relation: agentmemory.MemoryRelationRelated,
		},
	}, got.Subjects)
	assert.Equal(t, []agentmemoryports.MemorySubjectRef{
		{Type: agentmemory.MemorySubjectCustomer, ID: customerID},
		{Type: agentmemory.MemorySubjectLocation, ID: locationID},
	}, repo.listed.Subjects)
}

func TestRecordUse_CountsWhatAPromptCarried(t *testing.T) {
	t.Parallel()

	repo := &fakeMemoryRepo{}
	svc := newService(repo, &fakeRuns{}, &fakeLabeler{})
	ids := []pulid.ID{pulid.MustNew("amem_"), pulid.MustNew("amem_")}

	require.NoError(t, svc.RecordUse(t.Context(), agentmemoryports.RecordMemoryUseRequest{
		TenantInfo: tenant(),
		IDs:        ids,
	}))
	assert.Equal(t, ids, repo.used)

	require.NoError(t, svc.RecordUse(t.Context(), agentmemoryports.RecordMemoryUseRequest{
		TenantInfo: tenant(),
	}))
	assert.Len(t, repo.used, 2, "nothing carried is nothing to count")
}

func TestUsage_ReportsTheCountAgainstTheSoftCap(t *testing.T) {
	t.Parallel()

	repo := &fakeMemoryRepo{activeCount: 4200}
	usage, err := newService(repo, &fakeRuns{}, &fakeLabeler{}).Usage(t.Context(), tenant())
	require.NoError(t, err)

	assert.Equal(t, &agentmemoryports.AgentMemoryUsage{
		ActiveCount:   4200,
		ActiveSoftCap: agentmemory.MemoryActiveSoftCap,
		WarnAt:        agentmemory.MemoryActiveWarnAt,
	}, usage)
}

func proposal(tool string, params map[string]any) *agent.AgentProposal {
	return &agent.AgentProposal{
		ID:             pulid.MustNew("aprop_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		RunID:          pulid.MustNew("arun_"),
		ToolName:       tool,
		ToolParams:     params,
	}
}

// A person changing what the agent proposed is the correction; the reason,
// when they typed one, is kept with it. A default code the client filled in
// is not a reason and is left out.
func TestRecordCorrection_KeepsWhatAPersonChanged(t *testing.T) {
	t.Parallel()

	repo := &fakeMemoryRepo{}
	svc := newService(repo, &fakeRuns{}, &fakeLabeler{})
	p := proposal("dispatch_assign_move", map[string]any{"workerId": "wrk_a", "trailerId": "trl_1"})

	memory, err := svc.RecordCorrection(t.Context(), p, &agentdecision.AgentDecision{
		Decision: agentdecision.DecisionModified,
		Modifications: map[string]any{
			"workerId":  "wrk_b",
			"trailerId": "trl_1",
			"note":      "call first",
		},
		ReasonCode:      "approved_from_activity",
		DecidedByUserID: pulid.MustNew("usr_"),
	})
	require.NoError(t, err)
	require.NotNil(t, memory)

	assert.Equal(t, agentmemory.MemoryKindCorrection, memory.Kind)
	assert.Equal(t, agentmemory.MemorySourceDecision, memory.Source)
	assert.Equal(t, "dispatch_assign_move", memory.ToolName)
	assert.Equal(
		t,
		"When dispatch_assign_move was proposed, a person changed note to call first and workerId from wrk_a to wrk_b.",
		memory.Content,
	)
	assert.Equal(t, p.ID, *memory.SourceProposalID)
}

func TestRecordCorrection_KeepsARejectionOnlyWithAReason(t *testing.T) {
	t.Parallel()

	repo := &fakeMemoryRepo{}
	svc := newService(repo, &fakeRuns{}, &fakeLabeler{})
	p := proposal("shipment_cancel", map[string]any{"shipmentId": "shp_1"})

	none, err := svc.RecordCorrection(t.Context(), p, &agentdecision.AgentDecision{
		Decision:   agentdecision.DecisionRejected,
		ReasonCode: "rejected_from_activity",
	})
	require.NoError(t, err)
	assert.Nil(t, none, "a code the client filled in teaches nothing")

	memory, err := svc.RecordCorrection(t.Context(), p, &agentdecision.AgentDecision{
		Decision:   agentdecision.DecisionRejected,
		ReasonCode: "Never cancel a load that has already picked up (plan apl_01J)",
	})
	require.NoError(t, err)
	require.NotNil(t, memory)
	assert.Equal(t,
		"A person rejected shipment_cancel. Reason: Never cancel a load that has already picked up",
		memory.Content)
}

func TestRecordCorrection_SkipsACleanApprovalAndAChangeThatChangedNothing(t *testing.T) {
	t.Parallel()

	svc := newService(&fakeMemoryRepo{}, &fakeRuns{}, &fakeLabeler{})
	p := proposal("dispatch_assign_move", map[string]any{"workerId": "wrk_a"})

	approved, err := svc.RecordCorrection(
		t.Context(),
		p,
		&agentdecision.AgentDecision{Decision: agentdecision.DecisionAccepted},
	)
	require.NoError(t, err)
	assert.Nil(t, approved)

	same, err := svc.RecordCorrection(t.Context(), p, &agentdecision.AgentDecision{
		Decision:      agentdecision.DecisionModified,
		Modifications: map[string]any{"workerId": "wrk_a"},
	})
	require.NoError(t, err)
	assert.Nil(t, same)
}

func TestHumanReason(t *testing.T) {
	t.Parallel()

	assert.Empty(t, HumanReason("approved_from_activity"))
	assert.Empty(t, HumanReason("decided_as_plan (plan apl_1)"))
	assert.Empty(t, HumanReason("   "))
	assert.Equal(
		t,
		"Driver asked for Fridays off",
		HumanReason("Driver asked for Fridays off (plan apl_1)"),
	)
	assert.Equal(t, "Wrong trailer", HumanReason("Wrong trailer"))
}
