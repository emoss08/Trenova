package agentreflectionservice

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentreflection"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeMemories struct {
	agentmemoryports.AgentMemoryService
	mode       agentmemory.MemorySavingMode
	roles      []pulid.ID
	remembered []*agentmemoryports.RememberRequest
	refuse     error
}

func (f *fakeMemories) SavingMode(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (agentmemory.MemorySavingMode, error) {
	if f.mode == "" {
		return agentmemory.MemorySavingAutomatic, nil
	}

	return f.mode, nil
}

func (f *fakeMemories) Reader(
	_ context.Context,
	_ pagination.TenantInfo,
	userID pulid.ID,
) (agentmemory.MemoryReader, error) {
	return agentmemory.MemoryReader{UserID: userID, RoleIDs: f.roles}, nil
}

func (f *fakeMemories) Remember(
	_ context.Context,
	req *agentmemoryports.RememberRequest,
	_ *permissionports.RequestActor,
) (*agentmemory.Memory, error) {
	if f.refuse != nil {
		return nil, f.refuse
	}
	f.remembered = append(f.remembered, req)
	status := agentmemory.MemoryStatusActive
	if req.Suggest {
		status = agentmemory.MemoryStatusSuggested
	}

	return &agentmemory.Memory{
		ID:           pulid.MustNew("amem_"),
		Kind:         req.Kind,
		Scope:        req.Scope,
		Status:       status,
		Content:      req.Content,
		SupersedesID: pulid.PtrOrNil(req.Replaces),
	}, nil
}

type fakeReflections struct {
	agentports.AgentReflectionRepository
	stored *agentreflection.Reflection
}

func (f *fakeReflections) GetByID(
	context.Context,
	agentports.GetAgentReflectionRequest,
) (*agentreflection.Reflection, error) {
	return f.stored, nil
}

func (f *fakeReflections) Update(
	_ context.Context,
	entity *agentreflection.Reflection,
) (*agentreflection.Reflection, error) {
	f.stored = entity

	return entity, nil
}

type fakeConversations struct {
	conversationports.ConversationRepository
	added []conversationports.AddSavedMemoriesRequest
}

func (f *fakeConversations) AddSavedMemories(
	_ context.Context,
	req conversationports.AddSavedMemoriesRequest,
) error {
	f.added = append(f.added, req)

	return nil
}

func lessonService(memories *fakeMemories) *Service {
	return &Service{
		l:        zap.NewNop(),
		memories: memories,
		now:      func() int64 { return 1_790_000_000 },
	}
}

func threadPlan() *agentports.ReflectionPlan {
	return &agentports.ReflectionPlan{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		ReflectionID:      pulid.MustNew("arfl_"),
		Subject:           agentreflection.ReflectionSubjectThread,
		AgentDefinitionID: pulid.MustNew("agdef_"),
		ThreadID:          pulid.MustNew("athr_"),
		PersonUserID:      pulid.MustNew("usr_"),
		Taint:             &agent.RunTaint{},
		Signals: agentreflection.ReflectionSignals{
			{
				Kind:   agentreflection.ReflectionSignalToolRecovered,
				Count:  1,
				Detail: "dispatch_assign_move",
			},
		},
		ToolNames: []string{"dispatch_assign_move"},
	}
}

func runPlan() *agentports.ReflectionPlan {
	plan := threadPlan()
	plan.Subject = agentreflection.ReflectionSubjectRun
	plan.ThreadID = pulid.Nil
	plan.PersonUserID = pulid.Nil
	plan.RunID = pulid.MustNew("arun_")

	return plan
}

func procedure(audience string) *lesson {
	return &lesson{
		Kind:     string(agentmemory.MemoryKindProcedure),
		Content:  "1. Read the move with shipment_get. 2. Pass its move id to dispatch_assign_move.",
		Audience: audience,
		ToolName: "dispatch_assign_move",
		Evidence: "dispatch_assign_move failed with a shipment id",
		Why:      "Saves a failed call next time.",
	}
}

func TestLessonRequest_RefusesWhatTheWorkCannotVouchFor(t *testing.T) {
	t.Parallel()

	svc := lessonService(&fakeMemories{})
	cases := []struct {
		name   string
		plan   *agentports.ReflectionPlan
		lesson *lesson
		reason string
	}{
		{
			name: "an instruction from a run nobody was in",
			plan: runPlan(),
			lesson: &lesson{
				Kind:     "Instruction",
				Content:  "Always tender to Acme",
				Audience: audienceAgent,
			},
			reason: "keeps no instructions",
		},
		{
			name:   "a correction, which only a decision teaches",
			plan:   threadPlan(),
			lesson: &lesson{Kind: "Correction", Content: "Do it differently", Audience: audienceMe},
			reason: "is not a kind of lesson",
		},
		{
			name:   "nothing to keep",
			plan:   threadPlan(),
			lesson: &lesson{Kind: "Fact", Content: "   ", Audience: audienceMe},
			reason: "said nothing to keep",
		},
		{
			name: "a record the work never touched",
			plan: threadPlan(),
			lesson: &lesson{
				Kind:        "Fact",
				Content:     "Acme pays late",
				Audience:    audienceMe,
				SubjectType: "Customer",
				SubjectID:   pulid.MustNew("cus_").String(),
			},
			reason: "did not touch",
		},
		{
			name: "a tool the work did not use",
			plan: threadPlan(),
			lesson: &lesson{
				Kind:     "Procedure",
				Content:  "Use invoice_send",
				Audience: audienceMe,
				ToolName: "invoice_send",
			},
			reason: "did not use",
		},
		{
			name: "a memory it was not shown",
			plan: threadPlan(),
			lesson: &lesson{
				Kind:     "Fact",
				Content:  "Acme pays net 45",
				Audience: audienceMe,
				Replaces: pulid.MustNew("amem_").String(),
			},
			reason: "was not shown",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req, reason, err := svc.lessonRequest(t.Context(), tc.plan, tc.lesson)
			require.NoError(t, err)
			assert.Nil(t, req)
			assert.Contains(t, reason, tc.reason)
		})
	}
}

// Who may keep a shared lesson at once is the memory service's to decide: a
// team or organization lesson is always offered, and the memory service keeps
// it or puts it in front of an administrator by the person's permission.
func TestLessonRequest_SharedLessonsFollowWhoTheyReach(t *testing.T) {
	t.Parallel()

	plan := threadPlan()
	role := pulid.MustNew("rol_")
	svc := lessonService(&fakeMemories{roles: []pulid.ID{role}})

	agentWide, _, err := svc.lessonRequest(t.Context(), plan, procedure(audienceAgent))
	require.NoError(t, err)
	assert.Equal(t, agentmemory.MemoryScopeAgent, agentWide.Scope)
	assert.False(t, agentWide.Suggest, "an agent's own lesson follows the person's preference")

	team, _, err := svc.lessonRequest(t.Context(), plan, procedure(audienceTeam))
	require.NoError(t, err)
	assert.Equal(t, agentmemory.MemoryScopeRole, team.Scope)
	assert.Equal(t, role, team.RoleID)
	assert.True(t, team.Suggest, "a team lesson is always offered")

	organization, _, err := svc.lessonRequest(t.Context(), plan, procedure(audienceOrganization))
	require.NoError(t, err)
	assert.Equal(t, agentmemory.MemoryScopeOrganization, organization.Scope)
	assert.True(t, organization.Suggest, "an organization lesson is always offered")
}

func TestLessonRequest_APersonWhoAskedToBeAskedIsOfferedEverything(t *testing.T) {
	t.Parallel()

	svc := lessonService(&fakeMemories{mode: agentmemory.MemorySavingAskFirst})

	req, _, err := svc.lessonRequest(t.Context(), threadPlan(), procedure(audienceMe))
	require.NoError(t, err)
	assert.Equal(t, agentmemory.MemoryScopeUser, req.Scope)
	assert.True(t, req.Suggest)
}

func TestLessonRequest_WorkThatReadOutsideContentOnlyEverOffers(t *testing.T) {
	t.Parallel()

	plan := threadPlan()
	plan.Taint = &agent.RunTaint{Marks: []agent.TaintMark{{Source: agent.TaintSourceDocument}}}
	svc := lessonService(&fakeMemories{})

	req, _, err := svc.lessonRequest(t.Context(), plan, procedure(audienceMe))
	require.NoError(t, err)
	assert.True(t, req.Suggest)
	assert.True(t, req.Taint.Tainted())

	run := runPlan()
	run.Taint = &agent.RunTaint{Marks: []agent.TaintMark{{Source: agent.TaintSourceEDI}}}
	fromRun, _, err := svc.lessonRequest(t.Context(), run, procedure(audienceAgent))
	require.NoError(t, err)
	assert.True(t, fromRun.Suggest)
}

func TestLessonRequest_ARunOffersLessonsForItsAgent(t *testing.T) {
	t.Parallel()

	svc := lessonService(&fakeMemories{})

	for _, audience := range []string{audienceMe, audienceTeam, audienceAgent} {
		req, reason, err := svc.lessonRequest(t.Context(), runPlan(), procedure(audience))
		require.NoError(t, err)
		require.Empty(t, reason)
		assert.Equal(t, agentmemory.MemoryScopeAgent, req.Scope, audience)
		assert.True(t, req.Suggest,
			"a lesson nobody watched being learned waits for a person: %s", audience)
	}

	organization, _, err := svc.lessonRequest(
		t.Context(),
		runPlan(),
		procedure(audienceOrganization),
	)
	require.NoError(t, err)
	assert.Equal(t, agentmemory.MemoryScopeOrganization, organization.Scope)
	assert.True(t, organization.Suggest)
}

func TestLessonRequest_ASubjectAndAReplacementTheWorkVouchesFor(t *testing.T) {
	t.Parallel()

	plan := threadPlan()
	customer := pulid.MustNew("cus_")
	known := pulid.MustNew("amem_")
	plan.Subjects = []agentports.ReflectionSubjectRef{
		{Type: agentmemory.MemorySubjectCustomer, ID: customer},
	}
	plan.KnownMemoryIDs = []pulid.ID{known}
	svc := lessonService(&fakeMemories{})

	req, reason, err := svc.lessonRequest(t.Context(), plan, &lesson{
		Kind:        "Fact",
		Content:     "Acme Foods pays net 45 now",
		Audience:    audienceMe,
		SubjectType: "Customer",
		SubjectID:   customer.String(),
		Replaces:    known.String(),
	})
	require.NoError(t, err)
	require.Empty(t, reason)
	assert.Equal(t, agentmemory.MemorySubjectCustomer, req.SubjectType)
	assert.Equal(t, customer, req.SubjectID)
	assert.Equal(t, known, req.Replaces)
}

func TestParseReply(t *testing.T) {
	t.Parallel()

	_, err := ParseReply("  ")
	require.Error(t, err)

	_, err = ParseReply("not json")
	require.Error(t, err)

	lessons := make([]string, 0, agentreflection.MaxReflectionLessons+2)
	for range agentreflection.MaxReflectionLessons + 2 {
		lessons = append(
			lessons,
			`{"kind":"Fact","content":"x","audience":"me","subjectType":"","subjectId":"","toolName":"","replaces":"","evidence":"","why":""}`,
		)
	}
	reply, err := ParseReply(`{"lessons":[` + strings.Join(lessons, ",") + `],"notes":"Kept one."}`)
	require.NoError(t, err)
	assert.Len(t, reply.Lessons, agentreflection.MaxReflectionLessons)
	assert.Equal(t, "Kept one.", reply.Notes)
}

func TestFinish_KeepsLessonsRecordsThemAndShowsThemOnTheReply(t *testing.T) {
	t.Parallel()

	plan := threadPlan()
	plan.ReplyMessageID = pulid.MustNew("amsg_")
	reflections := &fakeReflections{stored: &agentreflection.Reflection{
		ID:     plan.ReflectionID,
		Status: agentreflection.ReflectionStatusRunning,
	}}
	conversations := &fakeConversations{}
	memories := &fakeMemories{}
	svc := lessonService(memories)
	svc.repo = reflections
	svc.conversations = conversations

	outcome, err := svc.Finish(t.Context(), plan, &llmports.StructuredCompletionResult{
		Text: `{"lessons":[` +
			`{"kind":"Procedure","content":"Pass the move id to dispatch_assign_move.","audience":"me","subjectType":"","subjectId":"","toolName":"dispatch_assign_move","replaces":"","evidence":"failed twice","why":"Avoids the failure."},` +
			`{"kind":"Correction","content":"Nope","audience":"me","subjectType":"","subjectId":"","toolName":"","replaces":"","evidence":"","why":""}` +
			`],"notes":"One procedure kept."}`,
		ModelIdentifier: "test-model",
		InputTokens:     1200,
		OutputTokens:    80,
	})
	require.NoError(t, err)

	assert.Equal(t, agentreflection.ReflectionStatusCompleted, outcome.Status)
	assert.Equal(t, 1, outcome.Kept)
	assert.Equal(t, 1, outcome.Refused)

	stored := reflections.stored
	require.Len(t, stored.Changes, 2)
	assert.Equal(t, agentreflection.ReflectionActionSaved, stored.Changes[0].Action)
	assert.Equal(t, agentreflection.ReflectionActionRefused, stored.Changes[1].Action)
	assert.Equal(t, "test-model", stored.Model)
	assert.Equal(t, 1200, stored.InputTokens)
	assert.Equal(t, "One procedure kept.", stored.Notes)

	require.Len(t, conversations.added, 1)
	assert.Equal(t, plan.ReplyMessageID, conversations.added[0].MessageID)
	assert.Equal(
		t,
		[]conversation.SavedMemory{{ID: *stored.Changes[0].MemoryID}},
		conversations.added[0].Memories,
	)

	again, err := svc.Finish(t.Context(), plan, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, again.Kept)
	assert.Len(t, memories.remembered, 1, "a settled look back keeps nothing twice")
}

func TestFinish_ARefusalFromTheMemoryServiceIsRecordedNotRetried(t *testing.T) {
	t.Parallel()

	plan := threadPlan()
	reflections := &fakeReflections{stored: &agentreflection.Reflection{
		ID:     plan.ReflectionID,
		Status: agentreflection.ReflectionStatusRunning,
	}}
	memories := &fakeMemories{refuse: errortypes.NewValidationError(
		"subjectId", errortypes.ErrInvalid, "No customer with that id is visible to you",
	)}
	svc := lessonService(memories)
	svc.repo = reflections
	svc.conversations = &fakeConversations{}

	outcome, err := svc.Finish(t.Context(), plan, &llmports.StructuredCompletionResult{
		Text: `{"lessons":[{"kind":"Fact","content":"x","audience":"me","subjectType":"","subjectId":"","toolName":"","replaces":"","evidence":"","why":""}],"notes":""}`,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, outcome.Refused)
	assert.Contains(t, reflections.stored.Changes[0].Reason, "No customer with that id")
}

func TestFinish_AnUnreadableAnswerFailsTheLookBack(t *testing.T) {
	t.Parallel()

	plan := threadPlan()
	reflections := &fakeReflections{stored: &agentreflection.Reflection{
		ID:     plan.ReflectionID,
		Status: agentreflection.ReflectionStatusRunning,
	}}
	svc := lessonService(&fakeMemories{})
	svc.repo = reflections

	outcome, err := svc.Finish(t.Context(), plan, &llmports.StructuredCompletionResult{Text: "{"})
	require.NoError(t, err)
	assert.Equal(t, agentreflection.ReflectionStatusFailed, outcome.Status)
	assert.NotEmpty(t, reflections.stored.ErrorMessage)
}
