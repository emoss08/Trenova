package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRunList struct {
	runs     []*agent.AgentRun
	requests []*repositories.ListAgentRunRequest
}

func (f *fakeRunList) List(
	_ context.Context,
	req *repositories.ListAgentRunRequest,
) (*pagination.ListResult[*agent.AgentRun], error) {
	f.requests = append(f.requests, req)

	return &pagination.ListResult[*agent.AgentRun]{Items: f.runs, Total: len(f.runs)}, nil
}

func backgroundRun(tainted bool) *agent.AgentRun {
	return &agent.AgentRun{
		ID:          pulid.MustNew("ar_"),
		AgentType:   agent.TypeDispatchAssignment,
		Trigger:     agent.RunTriggerEvent,
		Status:      agent.RunStatusCompleted,
		SubjectType: agent.SubjectShipmentMove,
		SubjectID:   pulid.MustNew("smv_"),
		Summary:     "Assigned the move.",
		Tainted:     tainted,
	}
}

func threadRun(threadID pulid.ID) *agent.AgentRun {
	return &agent.AgentRun{
		ID:          pulid.MustNew("ar_"),
		AgentType:   agent.TypeAssistantChat,
		Trigger:     agent.RunTriggerChat,
		Status:      agent.RunStatusCompleted,
		SubjectType: agent.SubjectAssistantThread,
		SubjectID:   threadID,
		Summary:     "Private conversation.",
	}
}

func queryRuns(
	t *testing.T,
	tool serviceports.AgentQueryTool,
	params *serviceports.QueryToolParams,
) *agentRunsOutcome {
	t.Helper()

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)
	outcome, ok := result.(*agentRunsOutcome)
	require.True(t, ok)

	return outcome
}

func runIDs(outcome *agentRunsOutcome) []string {
	rows, _ := outcome.Items.([]agentRunRow)
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}

	return ids
}

func hasFilter(filters []domaintypes.FieldFilter, want domaintypes.FieldFilter) bool {
	for _, filter := range filters {
		if filter.Field == want.Field && filter.Operator == want.Operator &&
			filter.Value == want.Value {
			return true
		}
	}

	return false
}

var excludeThreads = domaintypes.FieldFilter{
	Field:    agentRunFieldSubjectType,
	Operator: dbtype.OpNotEqual,
	Value:    string(agent.SubjectAssistantThread),
}

func TestListAgentRuns_HidesRunsOnAnotherPersonsConversation(t *testing.T) {
	t.Parallel()

	params := personParams(serviceports.PrincipalTypeUser)
	mine := pulid.MustNew("athr_")
	theirs := pulid.MustNew("athr_")
	orphaned := pulid.MustNew("athr_")
	background := backgroundRun(false)
	own := threadRun(mine)
	other := threadRun(theirs)
	unknown := threadRun(orphaned)

	runs := &fakeRunList{runs: []*agent.AgentRun{background, own, other, unknown}}
	threads := &fakeThreadOwners{owners: map[pulid.ID]pulid.ID{
		mine:   params.Actor.UserID,
		theirs: pulid.MustNew("usr_"),
	}}

	outcome := queryRuns(t, newListAgentRunsTool(runs, threads), params)

	assert.Equal(t, []string{background.ID.String(), own.ID.String()}, runIDs(outcome))
	require.Len(t, threads.requests, 1, "owners are read in one batch")
	assert.ElementsMatch(t, []pulid.ID{mine, theirs, orphaned}, threads.requests[0].ThreadIDs)
	assert.Equal(t, params.OrganizationID, threads.requests[0].TenantInfo.OrgID)
	assert.Contains(t, outcome.Notes, personThreadRunsNote)
	require.Len(t, runs.requests, 1)
	assert.False(t, hasFilter(runs.requests[0].Filter.FieldFilters, excludeThreads),
		"a person's own conversation runs stay listable")
}

func TestListAgentRuns_AnAgentNeverSeesConversationRuns(t *testing.T) {
	t.Parallel()

	for _, principal := range []serviceports.PrincipalType{
		serviceports.PrincipalTypeAgent,
		serviceports.PrincipalTypeAPIKey,
	} {
		params := personParams(principal)
		background := backgroundRun(false)
		runs := &fakeRunList{runs: []*agent.AgentRun{background, threadRun(pulid.MustNew("athr_"))}}
		threads := &fakeThreadOwners{}

		outcome := queryRuns(t, newListAgentRunsTool(runs, threads), params)

		assert.Equal(t, []string{background.ID.String()}, runIDs(outcome), principal)
		assert.Empty(t, threads.requests, "nobody's conversation is theirs to read")
		assert.True(t, hasFilter(runs.requests[0].Filter.FieldFilters, excludeThreads),
			"the query itself leaves conversation runs out, so a page stays full")
		assert.Contains(t, outcome.Notes, agentThreadRunsNote)
	}
}

func TestListAgentRuns_MineDefaultsToTheCallingAgentWhenItRunsOnItsOwn(t *testing.T) {
	t.Parallel()

	definition := pulid.MustNew("agdef_")
	own := domaintypes.FieldFilter{
		Field:    agentRunFieldDefinition,
		Operator: dbtype.OpEqual,
		Value:    definition.String(),
	}
	other := pulid.MustNew("agdef_").String()

	cases := map[string]struct {
		principal serviceports.PrincipalType
		params    map[string]any
		scoped    bool
	}{
		"unattended by default": {
			principal: serviceports.PrincipalTypeAgent,
			params:    map[string]any{},
			scoped:    true,
		},
		"unattended, asked for everyone's": {
			principal: serviceports.PrincipalTypeAgent,
			params:    map[string]any{paramMine: false},
		},
		"unattended, naming another agent": {
			principal: serviceports.PrincipalTypeAgent,
			params: map[string]any{"filters": []any{map[string]any{
				"field": agentRunFieldDefinition, "operator": "eq", "value": other,
			}}},
		},
		"in a conversation by default": {
			principal: serviceports.PrincipalTypeUser,
			params:    map[string]any{},
		},
		"in a conversation, asked for its own": {
			principal: serviceports.PrincipalTypeUser,
			params:    map[string]any{paramMine: true},
			scoped:    true,
		},
	}
	for name, tc := range cases {
		params := personParams(tc.principal)
		params.AgentDefinitionID = definition
		params.Params = tc.params
		runs := &fakeRunList{}

		outcome := queryRuns(t, newListAgentRunsTool(runs, &fakeThreadOwners{}), params)

		assert.Equal(t, tc.scoped, hasFilter(runs.requests[0].Filter.FieldFilters, own), name)
		if tc.scoped {
			assert.Contains(t, outcome.SearchedFor, "agent this agent's own runs", name)
		}
	}
}

func TestListAgentRuns_MineWithoutAnAgentIsRefused(t *testing.T) {
	t.Parallel()

	params := personParams(serviceports.PrincipalTypeUser)
	params.Params = map[string]any{paramMine: true}
	runs := &fakeRunList{}

	_, err := newListAgentRunsTool(runs, &fakeThreadOwners{}).Query(t.Context(), params)

	require.ErrorIs(t, err, errMineNeedsAnAgent)
	assert.Empty(t, runs.requests)
}

func TestListAgentRuns_MarksEachRunThatReadOutsideContent(t *testing.T) {
	t.Parallel()

	clean := backgroundRun(false)
	dirty := backgroundRun(true)
	runs := &fakeRunList{runs: []*agent.AgentRun{clean, dirty}}
	params := personParams(serviceports.PrincipalTypeAgent)

	outcome := queryRuns(t, newListAgentRunsTool(runs, &fakeThreadOwners{}), params)

	assert.Equal(t, []agent.RecordRef{{
		EntityType: agent.TaintEntityAgentRun,
		ID:         dirty.ID.String(),
	}}, outcome.TaintedRecords())
	rows, ok := outcome.Items.([]agentRunRow)
	require.True(t, ok)
	assert.False(t, rows[0].ReadOutsideContent)
	assert.True(t, rows[1].ReadOutsideContent)
	assert.Contains(t, outcome.Notes, agentRunOutsideNote)
}

func TestListAgentRuns_PagesAndBoundsTheList(t *testing.T) {
	t.Parallel()

	many := make([]*agent.AgentRun, 0, maxListLimit+1)
	for range maxListLimit + 1 {
		many = append(many, backgroundRun(false))
	}
	runs := &fakeRunList{runs: many}
	params := personParams(serviceports.PrincipalTypeAgent)
	params.Params = map[string]any{"limit": 500}

	outcome := queryRuns(t, newListAgentRunsTool(runs, &fakeThreadOwners{}), params)

	assert.Equal(t, maxListLimit, outcome.Count)
	assert.True(t, outcome.HasMore)
	assert.Equal(t, maxListLimit+1, runs.requests[0].Filter.Pagination.Limit)
}

func TestListAgentRuns_DeclaresAMarkedRunRecordRead(t *testing.T) {
	t.Parallel()

	tool := newListAgentRunsTool(nil, nil)
	policy := tool.Policy()

	assert.Equal(t, "list_agent_runs", tool.Name())
	assert.Equal(t, permission.ResourceAgentRun, policy.Resource)
	assert.Equal(t, agent.ExternalReadMarked, policy.ReadsExternal)
	assert.Equal(t, agent.TaintSourceRunRecord, policy.Source)
	properties, ok := tool.ParamSchema()["properties"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, properties, paramMine)

	spec := agentRunListSpec()
	for _, field := range spec.fields {
		assert.True(t, spec.config.FilterableFields[field.Name],
			"%q no longer maps to a column on agent runs", field.Name)
		if field.Sortable {
			assert.True(t, spec.config.SortableFields[field.Name], field.Name)
		}
	}
}

type fakeRunGetter struct {
	run *agent.AgentRun
}

func (f *fakeRunGetter) GetByID(
	_ context.Context,
	req repositories.GetAgentRunByIDRequest,
) (*agent.AgentRun, error) {
	if f.run == nil || f.run.ID != req.ID {
		return nil, errortypes.NewNotFoundError("Agent run not found")
	}

	return f.run, nil
}

type fakeRunProposals struct {
	proposals []*agent.AgentProposal
	calls     int
}

func (f *fakeRunProposals) ListByRun(
	context.Context,
	repositories.ListAgentProposalsByRunRequest,
) ([]*agent.AgentProposal, error) {
	f.calls++

	return f.proposals, nil
}

type fakeRunEvents struct {
	next     int
	events   []*agent.AgentRunEvent
	requests []repositories.ListAgentRunEventsRequest
}

func (f *fakeRunEvents) NextSequence(
	context.Context,
	pagination.TenantInfo,
	string,
	pulid.ID,
) (int, error) {
	return f.next, nil
}

func (f *fakeRunEvents) List(
	_ context.Context,
	req repositories.ListAgentRunEventsRequest,
) ([]*agent.AgentRunEvent, error) {
	f.requests = append(f.requests, req)

	return f.events, nil
}

type runFixture struct {
	runs      *fakeRunGetter
	threads   *fakeThreadOwners
	proposals *fakeRunProposals
	events    *fakeRunEvents
	perms     *fakePermissions
}

func newRunFixture(run *agent.AgentRun) *runFixture {
	return &runFixture{
		runs:      &fakeRunGetter{run: run},
		threads:   &fakeThreadOwners{owners: map[pulid.ID]pulid.ID{}},
		proposals: &fakeRunProposals{},
		events:    &fakeRunEvents{},
		perms:     &fakePermissions{allowed: true},
	}
}

func (f *runFixture) tool() serviceports.AgentQueryTool {
	return &getAgentRunTool{
		runs:      f.runs,
		owners:    agentRunOwnership{threads: f.threads},
		proposals: f.proposals,
		events:    f.events,
		access:    newFieldAccess(f.perms),
	}
}

func getRun(
	t *testing.T,
	fixture *runFixture,
	params *serviceports.QueryToolParams,
) *agentRunDetail {
	t.Helper()

	result, err := fixture.tool().Query(t.Context(), params)
	require.NoError(t, err)
	detail, ok := result.(*agentRunDetail)
	require.True(t, ok)

	return detail
}

func TestGetAgentRun_ReadsAConversationRunOnlyForItsOwner(t *testing.T) {
	t.Parallel()

	threadID := pulid.MustNew("athr_")
	run := threadRun(threadID)
	fixture := newRunFixture(run)

	owner := personParams(serviceports.PrincipalTypeUser)
	owner.Params = map[string]any{paramRunID: run.ID.String()}
	fixture.threads.owners[threadID] = owner.Actor.UserID
	assert.Equal(t, run.ID.String(), getRun(t, fixture, owner).ID)

	colleague := personParams(serviceports.PrincipalTypeUser)
	colleague.Params = map[string]any{paramRunID: run.ID.String()}
	_, err := fixture.tool().Query(t.Context(), colleague)
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err),
		"another person's conversation reads as missing, not as forbidden")

	for _, principal := range []serviceports.PrincipalType{
		serviceports.PrincipalTypeAgent,
		serviceports.PrincipalTypeAPIKey,
	} {
		params := personParams(principal)
		params.Params = map[string]any{paramRunID: run.ID.String()}
		_, err = fixture.tool().Query(t.Context(), params)
		require.Error(t, err, principal)
		assert.True(t, errortypes.IsNotFoundError(err), principal)
	}
}

func TestGetAgentRun_ReadsABackgroundRunAndMarksItsTaint(t *testing.T) {
	t.Parallel()

	run := backgroundRun(true)
	run.Taint = &agent.RunTaint{Marks: []agent.TaintMark{
		{Source: agent.TaintSourceInboundMessage},
		{Source: agent.TaintSourceWeather},
	}}
	fixture := newRunFixture(run)
	params := personParams(serviceports.PrincipalTypeAgent)
	params.Params = map[string]any{paramRunID: run.ID.String()}

	detail := getRun(t, fixture, params)

	assert.True(t, detail.ReadOutsideContent)
	assert.Equal(t, []string{"inbound_message", "weather"}, detail.TaintSources)
	assert.Equal(t, []agent.RecordRef{{
		EntityType: agent.TaintEntityAgentRun,
		ID:         run.ID.String(),
	}}, detail.TaintedRecords())
	assert.Nil(t, detail.Proposals, "proposals are added only when asked for")
	assert.Nil(t, detail.Events)
	assert.Zero(t, fixture.proposals.calls)
	assert.Empty(t, fixture.threads.requests, "a background run has no owner to look up")
}

func TestGetAgentRun_AddsBoundedProposalsAndTheLastFiftySteps(t *testing.T) {
	t.Parallel()

	run := backgroundRun(false)
	fixture := newRunFixture(run)
	for idx := range agentRunProposalLimit + 5 {
		fixture.proposals.proposals = append(fixture.proposals.proposals, &agent.AgentProposal{
			ID:           pulid.MustNew("apr_"),
			RunID:        run.ID,
			ToolName:     "assign_move",
			Status:       agent.ProposalStatusPending,
			AutonomyTier: agent.TierPropose,
			CreatedAt:    int64(idx),
		})
	}
	fixture.events.next = 121
	fixture.events.events = []*agent.AgentRunEvent{
		{
			Sequence: 71,
			Kind:     serviceports.AssistantEventToolStarted,
			CallID:   "call_1",
			Payload:  map[string]any{"name": "get_shipment", "arguments": map[string]any{"x": 1}},
		},
		{
			Sequence: 72,
			Kind:     serviceports.AssistantEventToolFinished,
			CallID:   "call_1",
			Payload: map[string]any{
				"name":    "get_shipment",
				"content": "<tool_result>the whole shipment</tool_result>",
				"summary": "Read shipment S-1001",
			},
		},
		{
			Sequence: 73,
			Kind:     serviceports.AssistantEventRunTainted,
			Payload:  map[string]any{"mark": map[string]any{"source": "record_note"}},
		},
	}
	params := personParams(serviceports.PrincipalTypeUser)
	params.Params = map[string]any{
		paramRunID:            run.ID.String(),
		paramIncludeProposals: true,
		paramIncludeEvents:    true,
	}

	detail := getRun(t, fixture, params)

	require.Len(t, detail.Proposals, agentRunProposalLimit)
	require.NotNil(t, detail.ProposalsTotal)
	assert.Equal(t, agentRunProposalLimit+5, *detail.ProposalsTotal)
	assert.Equal(t, int64(5), detail.Proposals[0].CreatedAt, "the newest are kept")

	require.Len(t, fixture.events.requests, 1)
	request := fixture.events.requests[0]
	assert.Equal(t, agentRunEventLimit, request.Limit)
	assert.Equal(t, 70, request.After, "the last fifty of 120 steps start after 70")
	assert.Equal(t, string(serviceports.RunStepOwnerAgentRun), request.OwnerKind)
	assert.Equal(t, run.ID, request.OwnerID)

	require.Len(t, detail.Events, 3)
	assert.Equal(t, "get_shipment", detail.Events[0].Tool)
	assert.Empty(t, detail.Events[0].Text, "arguments stay out of the account")
	assert.Equal(t, "Read shipment S-1001", detail.Events[1].Text,
		"a tool's summary, never its whole result")
	assert.Equal(t, "record_note", detail.Events[2].Text)
}

func TestGetAgentRun_AShortRunReadsFromItsFirstStep(t *testing.T) {
	t.Parallel()

	run := backgroundRun(false)
	fixture := newRunFixture(run)
	fixture.events.next = 8
	params := personParams(serviceports.PrincipalTypeAgent)
	params.Params = map[string]any{paramRunID: run.ID.String(), paramIncludeEvents: true}

	getRun(t, fixture, params)

	assert.Zero(t, fixture.events.requests[0].After)
}

func TestGetAgentRun_WithholdsProposalsFromAReaderWhoMayNotReadThem(t *testing.T) {
	t.Parallel()

	run := backgroundRun(false)
	fixture := newRunFixture(run)
	fixture.perms = &fakePermissions{readable: readable(permission.ResourceAgentRun)}
	params := personParams(serviceports.PrincipalTypeUser)
	params.Params = map[string]any{paramRunID: run.ID.String(), paramIncludeProposals: true}

	detail := getRun(t, fixture, params)

	assert.Nil(t, detail.Proposals)
	assert.Zero(t, fixture.proposals.calls)
	assert.Contains(t, detail.Notes, proposalsWithheldNote)
}

func TestGetAgentRun_TheIDNamesWhereItComesFrom(t *testing.T) {
	t.Parallel()

	tool := newGetAgentRunTool(getAgentRunParams{}, nil)
	properties, ok := tool.ParamSchema()["properties"].(map[string]any)
	require.True(t, ok)
	runID, ok := properties[paramRunID].(map[string]any)
	require.True(t, ok)

	assert.Contains(t, runID["description"], "list_agent_runs")
	assert.Equal(t, agent.ExternalReadMarked, tool.Policy().ReadsExternal)
	assert.Equal(t, agent.TaintSourceRunRecord, tool.Policy().Source)
}
