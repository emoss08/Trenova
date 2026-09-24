package agentmemoryservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeMemoryRepo struct {
	repositories.AgentMemoryRepository
	created []*agent.Memory
	active  []*agent.Memory
	found   *agent.Memory
	used    []pulid.ID
	listed  *repositories.ListActiveAgentMemoriesRequest
	sought  *repositories.FindActiveAgentMemoryRequest

	activeCount int
}

func (f *fakeMemoryRepo) CountActive(
	context.Context,
	repositories.CountActiveAgentMemoriesRequest,
) (int, error) {
	return f.activeCount, nil
}

func (f *fakeMemoryRepo) Create(_ context.Context, entity *agent.Memory) (*agent.Memory, error) {
	entity.ID = pulid.MustNew("amem_")
	f.created = append(f.created, entity)

	return entity, nil
}

func (f *fakeMemoryRepo) FindActive(
	_ context.Context,
	req repositories.FindActiveAgentMemoryRequest,
) (*agent.Memory, error) {
	f.sought = &req

	return f.found, nil
}

func (f *fakeMemoryRepo) ListActive(
	_ context.Context,
	req repositories.ListActiveAgentMemoriesRequest,
) ([]*agent.Memory, error) {
	f.listed = &req

	return f.active, nil
}

func (f *fakeMemoryRepo) MarkUsed(
	_ context.Context,
	req repositories.MarkAgentMemoriesUsedRequest,
) error {
	f.used = append(f.used, req.IDs...)

	return nil
}

type fakeRuns struct {
	run *agent.AgentRun
}

func (f *fakeRuns) GetByID(
	context.Context,
	repositories.GetAgentRunByIDRequest,
) (*agent.AgentRun, error) {
	return f.run, nil
}

type fakeLabeler struct {
	label string
	err   error
	asked *repositories.MemorySubjectRef
}

func (f *fakeLabeler) Label(
	_ context.Context,
	_ pagination.TenantInfo,
	ref repositories.MemorySubjectRef,
) (string, error) {
	f.asked = &ref

	return f.label, f.err
}

func newService(repo *fakeMemoryRepo, runs *fakeRuns, labeler *fakeLabeler) *Service {
	return &Service{l: zap.NewNop(), repo: repo, runs: runs, labeler: labeler}
}

func tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func userActor() *services.RequestActor {
	return &services.RequestActor{
		PrincipalType: services.PrincipalTypeUser,
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
	created, err := svc.Remember(t.Context(), &services.RememberRequest{
		TenantInfo:  tenant(),
		Kind:        agent.MemoryKindInstruction,
		Content:     "  Needs the POD within one day of delivery.  ",
		SubjectType: agent.MemorySubjectCustomer,
		SubjectID:   customerID,
		RunID:       pulid.MustNew("arun_"),
	}, userActor())
	require.NoError(t, err)

	assert.Equal(t, agent.MemorySourceAgent, created.Source)
	assert.Equal(t, definitionID, *created.AgentDefinitionID)
	assert.Equal(t, "Acme Freight", created.SubjectLabel)
	assert.Equal(t, "Needs the POD within one day of delivery.", created.Content)
	assert.Equal(t, customerID, labeler.asked.ID)
	assert.NotNil(t, created.CreatedByUserID, "the approver is kept alongside the agent")
}

func TestRemember_RefusesASubjectThatDoesNotExist(t *testing.T) {
	t.Parallel()

	svc := newService(&fakeMemoryRepo{}, &fakeRuns{}, &fakeLabeler{err: assert.AnError})

	_, err := svc.Remember(t.Context(), &services.RememberRequest{
		TenantInfo:  tenant(),
		Content:     "x",
		SubjectType: agent.MemorySubjectWorker,
		SubjectID:   pulid.MustNew("wrk_"),
	}, userActor())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "No worker with that id")
}

func TestRemember_ReturnsTheExistingRowForTheSameSentence(t *testing.T) {
	t.Parallel()

	existing := &agent.Memory{ID: pulid.MustNew("amem_"), Content: "Same"}
	repo := &fakeMemoryRepo{found: existing}
	svc := newService(repo, &fakeRuns{}, &fakeLabeler{})

	got, err := svc.Remember(t.Context(), &services.RememberRequest{
		TenantInfo: tenant(),
		Content:    "Same",
	}, userActor())
	require.NoError(t, err)

	assert.Same(t, existing, got)
	assert.Empty(t, repo.created)
}

func TestRemember_LooksForTheSameMemoryOnlyWhereItWouldBeRead(t *testing.T) {
	t.Parallel()

	definitionID := pulid.MustNew("agdef_")
	runs := &fakeRuns{run: &agent.AgentRun{AgentDefinitionID: definitionID}}

	clean := &fakeMemoryRepo{}
	_, err := newService(clean, runs, &fakeLabeler{}).Remember(t.Context(),
		&services.RememberRequest{TenantInfo: tenant(), Content: "Quote in dollars."},
		userActor(),
	)
	require.NoError(t, err)
	require.NotNil(t, clean.sought)
	assert.False(t, clean.sought.Tainted, "a person's own write never matches a tainted row")
	assert.Equal(t, agent.MemoryScopeOrganization, clean.sought.Scope)
	assert.True(t, clean.sought.AgentDefinitionID.IsNil())
	assert.Positive(t, clean.sought.Now, "an expired row is not the same memory")

	tainted := &fakeMemoryRepo{}
	taint := &agent.RunTaint{}
	taint.Add(agent.TaintMark{Source: agent.TaintSourceInboundMessage})
	_, err = newService(tainted, runs, &fakeLabeler{}).Remember(t.Context(),
		&services.RememberRequest{
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
	assert.Equal(t, agent.MemoryScopeOrganization, tainted.sought.Scope,
		"an agent's remember is read by every agent, so it matches only organization rows")
}

func TestForContext_ReadsCandidatesWithoutCountingThemUsed(t *testing.T) {
	t.Parallel()

	one := &agent.Memory{ID: pulid.MustNew("amem_")}
	repo := &fakeMemoryRepo{active: []*agent.Memory{one}}
	svc := newService(repo, &fakeRuns{}, &fakeLabeler{})

	got, err := svc.ForContext(t.Context(), services.MemoryContextRequest{
		TenantInfo: tenant(),
		ToolNames:  []string{"assign_move"},
	})
	require.NoError(t, err)

	assert.Equal(t, []*agent.Memory{one}, got.Memories)
	assert.True(t, repo.listed.OrganizationWide)
	assert.Equal(t, []string{"assign_move"}, repo.listed.ToolNames)
	assert.Equal(t, agent.MaxMemoryCandidates, repo.listed.Limit)
	assert.Empty(t, repo.used, "a candidate is counted only once a prompt carries it")
}

func TestForContext_ReadsTheMemoriesOfTheRecordsTheTurnIsAbout(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	shipmentID := pulid.MustNew("shp_")
	locationID := pulid.MustNew("loc_")
	repo := &fakeMemoryRepo{}
	svc := newService(repo, &fakeRuns{}, &fakeLabeler{})
	svc.subjects = NewSubjectResolver(&fakeLinks{links: map[agent.MemoryRecordKind][]repositories.MemoryRecordLink{
		agent.MemoryRecordShipment: {
			{From: shipmentID, Kind: agent.MemoryRecordLocation, ID: locationID},
		},
	}})

	got, err := svc.ForContext(t.Context(), services.MemoryContextRequest{
		TenantInfo: tenant(),
		Records: []agent.EntityRef{
			{Type: "customer", ID: customerID.String()},
			{Type: string(agent.SubjectShipment), ID: shipmentID.String()},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, []agent.MemorySubject{
		{Type: agent.MemorySubjectCustomer, ID: customerID, Relation: agent.MemoryRelationDirect},
		{Type: agent.MemorySubjectLocation, ID: locationID, Relation: agent.MemoryRelationRelated},
	}, got.Subjects)
	assert.Equal(t, []repositories.MemorySubjectRef{
		{Type: agent.MemorySubjectCustomer, ID: customerID},
		{Type: agent.MemorySubjectLocation, ID: locationID},
	}, repo.listed.Subjects)
}

func TestRecordUse_CountsWhatAPromptCarried(t *testing.T) {
	t.Parallel()

	repo := &fakeMemoryRepo{}
	svc := newService(repo, &fakeRuns{}, &fakeLabeler{})
	ids := []pulid.ID{pulid.MustNew("amem_"), pulid.MustNew("amem_")}

	require.NoError(t, svc.RecordUse(t.Context(), services.RecordMemoryUseRequest{
		TenantInfo: tenant(),
		IDs:        ids,
	}))
	assert.Equal(t, ids, repo.used)

	require.NoError(t, svc.RecordUse(t.Context(), services.RecordMemoryUseRequest{
		TenantInfo: tenant(),
	}))
	assert.Len(t, repo.used, 2, "nothing carried is nothing to count")
}

func TestUsage_ReportsTheCountAgainstTheSoftCap(t *testing.T) {
	t.Parallel()

	repo := &fakeMemoryRepo{activeCount: 4200}
	usage, err := newService(repo, &fakeRuns{}, &fakeLabeler{}).Usage(t.Context(), tenant())
	require.NoError(t, err)

	assert.Equal(t, &services.AgentMemoryUsage{
		ActiveCount:     4200,
		ActiveSoftCap:   agent.MemoryActiveSoftCap,
		WarnAt:          agent.MemoryActiveWarnAt,
		ContentMaxChars: agent.MaxMemoryContentChars,
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
	p := proposal("assign_move", map[string]any{"workerId": "wrk_a", "trailerId": "trl_1"})

	memory, err := svc.RecordCorrection(t.Context(), p, &agent.AgentDecision{
		Decision: agent.DecisionModified,
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

	assert.Equal(t, agent.MemoryKindCorrection, memory.Kind)
	assert.Equal(t, agent.MemorySourceDecision, memory.Source)
	assert.Equal(t, "assign_move", memory.ToolName)
	assert.Equal(
		t,
		"When assign_move was proposed, a person changed note to call first and workerId from wrk_a to wrk_b.",
		memory.Content,
	)
	assert.Equal(t, p.ID, *memory.SourceProposalID)
}

func TestRecordCorrection_KeepsARejectionOnlyWithAReason(t *testing.T) {
	t.Parallel()

	repo := &fakeMemoryRepo{}
	svc := newService(repo, &fakeRuns{}, &fakeLabeler{})
	p := proposal("cancel_shipment", map[string]any{"shipmentId": "shp_1"})

	none, err := svc.RecordCorrection(t.Context(), p, &agent.AgentDecision{
		Decision:   agent.DecisionRejected,
		ReasonCode: "rejected_from_activity",
	})
	require.NoError(t, err)
	assert.Nil(t, none, "a code the client filled in teaches nothing")

	memory, err := svc.RecordCorrection(t.Context(), p, &agent.AgentDecision{
		Decision:   agent.DecisionRejected,
		ReasonCode: "Never cancel a load that has already picked up (plan apl_01J)",
	})
	require.NoError(t, err)
	require.NotNil(t, memory)
	assert.Equal(t,
		"A person rejected cancel_shipment. Reason: Never cancel a load that has already picked up",
		memory.Content)
}

func TestRecordCorrection_SkipsACleanApprovalAndAChangeThatChangedNothing(t *testing.T) {
	t.Parallel()

	svc := newService(&fakeMemoryRepo{}, &fakeRuns{}, &fakeLabeler{})
	p := proposal("assign_move", map[string]any{"workerId": "wrk_a"})

	approved, err := svc.RecordCorrection(
		t.Context(),
		p,
		&agent.AgentDecision{Decision: agent.DecisionAccepted},
	)
	require.NoError(t, err)
	assert.Nil(t, approved)

	same, err := svc.RecordCorrection(t.Context(), p, &agent.AgentDecision{
		Decision:      agent.DecisionModified,
		Modifications: map[string]any{"workerId": "wrk_a"},
	})
	require.NoError(t, err)
	assert.Nil(t, same)
}

func TestHumanReason(t *testing.T) {
	t.Parallel()

	assert.Empty(t, humanReason("approved_from_activity"))
	assert.Empty(t, humanReason("decided_as_plan (plan apl_1)"))
	assert.Empty(t, humanReason("   "))
	assert.Equal(
		t,
		"Driver asked for Fridays off",
		humanReason("Driver asked for Fridays off (plan apl_1)"),
	)
	assert.Equal(t, "Wrong trailer", humanReason("Wrong trailer"))
}
