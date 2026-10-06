package agentreflectionservice

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeMemories struct {
	services.AgentMemoryService
	mode       agent.MemorySavingMode
	roles      []pulid.ID
	remembered []*services.RememberRequest
	refuse     error
}

func (f *fakeMemories) SavingMode(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (agent.MemorySavingMode, error) {
	if f.mode == "" {
		return agent.MemorySavingAutomatic, nil
	}

	return f.mode, nil
}

func (f *fakeMemories) Reader(
	_ context.Context,
	_ pagination.TenantInfo,
	userID pulid.ID,
) (agent.MemoryReader, error) {
	return agent.MemoryReader{UserID: userID, RoleIDs: f.roles}, nil
}

func (f *fakeMemories) Remember(
	_ context.Context,
	req *services.RememberRequest,
	_ *services.RequestActor,
) (*agent.Memory, error) {
	if f.refuse != nil {
		return nil, f.refuse
	}
	f.remembered = append(f.remembered, req)
	status := agent.MemoryStatusActive
	if req.Suggest {
		status = agent.MemoryStatusSuggested
	}

	return &agent.Memory{
		ID:           pulid.MustNew("amem_"),
		Kind:         req.Kind,
		Scope:        req.Scope,
		Status:       status,
		Content:      req.Content,
		SupersedesID: pulid.PtrOrNil(req.Replaces),
	}, nil
}

type fakePermissions struct {
	services.PermissionEngine
	allowed bool
}

func (f *fakePermissions) Check(
	context.Context,
	*services.PermissionCheckRequest,
) (*services.PermissionCheckResult, error) {
	return &services.PermissionCheckResult{Allowed: f.allowed}, nil
}

type fakeReflections struct {
	repositories.AgentReflectionRepository
	stored *agent.Reflection
}

func (f *fakeReflections) GetByID(
	context.Context,
	repositories.GetAgentReflectionRequest,
) (*agent.Reflection, error) {
	return f.stored, nil
}

func (f *fakeReflections) Update(
	_ context.Context,
	entity *agent.Reflection,
) (*agent.Reflection, error) {
	f.stored = entity

	return entity, nil
}

type fakeConversations struct {
	repositories.ConversationRepository
	added []repositories.AddSavedMemoriesRequest
}

func (f *fakeConversations) AddSavedMemories(
	_ context.Context,
	req repositories.AddSavedMemoriesRequest,
) error {
	f.added = append(f.added, req)

	return nil
}

func lessonService(memories *fakeMemories, allowed bool) *Service {
	return &Service{
		l:           zap.NewNop(),
		memories:    memories,
		permissions: &fakePermissions{allowed: allowed},
		now:         func() int64 { return 1_790_000_000 },
	}
}

func threadPlan() *services.ReflectionPlan {
	return &services.ReflectionPlan{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		ReflectionID:      pulid.MustNew("arfl_"),
		Subject:           agent.ReflectionSubjectThread,
		AgentDefinitionID: pulid.MustNew("agdef_"),
		ThreadID:          pulid.MustNew("athr_"),
		PersonUserID:      pulid.MustNew("usr_"),
		Taint:             &agent.RunTaint{},
		Signals: agent.ReflectionSignals{
			{Kind: agent.ReflectionSignalToolRecovered, Count: 1, Detail: "assign_move"},
		},
		ToolNames: []string{"assign_move"},
	}
}

func runPlan() *services.ReflectionPlan {
	plan := threadPlan()
	plan.Subject = agent.ReflectionSubjectRun
	plan.ThreadID = pulid.Nil
	plan.PersonUserID = pulid.Nil
	plan.RunID = pulid.MustNew("arun_")

	return plan
}

func procedure(audience string) *lesson {
	return &lesson{
		Kind:     string(agent.MemoryKindProcedure),
		Content:  "1. Read the move with get_shipment. 2. Pass its move id to assign_move.",
		Audience: audience,
		ToolName: "assign_move",
		Evidence: "assign_move failed with a shipment id",
		Why:      "Saves a failed call next time.",
	}
}

func TestLessonRequest_RefusesWhatTheWorkCannotVouchFor(t *testing.T) {
	t.Parallel()

	svc := lessonService(&fakeMemories{}, true)
	cases := []struct {
		name   string
		plan   *services.ReflectionPlan
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
				Content:  "Use send_invoice",
				Audience: audienceMe,
				ToolName: "send_invoice",
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

func TestLessonRequest_APersonWithoutPermissionKeepsSharedLessonsForThemselves(t *testing.T) {
	t.Parallel()

	plan := threadPlan()
	svc := lessonService(&fakeMemories{}, false)

	for _, audience := range []string{audienceAgent, audienceTeam, audienceOrganization} {
		req, reason, err := svc.lessonRequest(t.Context(), plan, procedure(audience))
		require.NoError(t, err)
		require.Empty(t, reason)
		assert.Equal(t, agent.MemoryScopeUser, req.Scope, audience)
		assert.Equal(t, plan.PersonUserID, req.OwnerUserID, audience)
		assert.False(t, req.Suggest, audience)
		assert.Equal(t, agent.MemorySourceReflection, req.Source)
		assert.Equal(t, plan.ReflectionID, req.ReflectionID)
		assert.Equal(t, "assign_move", req.ToolName)
	}
}

func TestLessonRequest_SharedLessonsFollowWhoTheyReach(t *testing.T) {
	t.Parallel()

	plan := threadPlan()
	role := pulid.MustNew("rol_")
	svc := lessonService(&fakeMemories{roles: []pulid.ID{role}}, true)

	agentWide, _, err := svc.lessonRequest(t.Context(), plan, procedure(audienceAgent))
	require.NoError(t, err)
	assert.Equal(t, agent.MemoryScopeAgent, agentWide.Scope)
	assert.False(t, agentWide.Suggest, "an agent's own lesson follows the person's preference")

	team, _, err := svc.lessonRequest(t.Context(), plan, procedure(audienceTeam))
	require.NoError(t, err)
	assert.Equal(t, agent.MemoryScopeRole, team.Scope)
	assert.Equal(t, role, team.RoleID)
	assert.True(t, team.Suggest, "a team lesson is always offered")

	organization, _, err := svc.lessonRequest(t.Context(), plan, procedure(audienceOrganization))
	require.NoError(t, err)
	assert.Equal(t, agent.MemoryScopeOrganization, organization.Scope)
	assert.True(t, organization.Suggest, "an organization lesson is always offered")
}

func TestLessonRequest_APersonWhoAskedToBeAskedIsOfferedEverything(t *testing.T) {
	t.Parallel()

	svc := lessonService(&fakeMemories{mode: agent.MemorySavingAskFirst}, true)

	req, _, err := svc.lessonRequest(t.Context(), threadPlan(), procedure(audienceMe))
	require.NoError(t, err)
	assert.Equal(t, agent.MemoryScopeUser, req.Scope)
	assert.True(t, req.Suggest)
}

func TestLessonRequest_WorkThatReadOutsideContentOnlyEverOffers(t *testing.T) {
	t.Parallel()

	plan := threadPlan()
	plan.Taint = &agent.RunTaint{Marks: []agent.TaintMark{{Source: agent.TaintSourceDocument}}}
	svc := lessonService(&fakeMemories{}, true)

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

func TestLessonRequest_ARunKeepsLessonsForItsAgent(t *testing.T) {
	t.Parallel()

	svc := lessonService(&fakeMemories{}, false)

	for _, audience := range []string{audienceMe, audienceTeam, audienceAgent} {
		req, reason, err := svc.lessonRequest(t.Context(), runPlan(), procedure(audience))
		require.NoError(t, err)
		require.Empty(t, reason)
		assert.Equal(t, agent.MemoryScopeAgent, req.Scope, audience)
		assert.False(t, req.Suggest, audience)
	}

	organization, _, err := svc.lessonRequest(
		t.Context(),
		runPlan(),
		procedure(audienceOrganization),
	)
	require.NoError(t, err)
	assert.Equal(t, agent.MemoryScopeOrganization, organization.Scope)
	assert.True(t, organization.Suggest)
}

func TestLessonRequest_ASubjectAndAReplacementTheWorkVouchesFor(t *testing.T) {
	t.Parallel()

	plan := threadPlan()
	customer := pulid.MustNew("cus_")
	known := pulid.MustNew("amem_")
	plan.Subjects = []services.ReflectionSubjectRef{
		{Type: agent.MemorySubjectCustomer, ID: customer},
	}
	plan.KnownMemoryIDs = []pulid.ID{known}
	svc := lessonService(&fakeMemories{}, false)

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
	assert.Equal(t, agent.MemorySubjectCustomer, req.SubjectType)
	assert.Equal(t, customer, req.SubjectID)
	assert.Equal(t, known, req.Replaces)
}

func TestParseReply(t *testing.T) {
	t.Parallel()

	_, err := ParseReply("  ")
	require.Error(t, err)

	_, err = ParseReply("not json")
	require.Error(t, err)

	lessons := make([]string, 0, agent.MaxReflectionLessons+2)
	for range agent.MaxReflectionLessons + 2 {
		lessons = append(
			lessons,
			`{"kind":"Fact","content":"x","audience":"me","subjectType":"","subjectId":"","toolName":"","replaces":"","evidence":"","why":""}`,
		)
	}
	reply, err := ParseReply(`{"lessons":[` + strings.Join(lessons, ",") + `],"notes":"Kept one."}`)
	require.NoError(t, err)
	assert.Len(t, reply.Lessons, agent.MaxReflectionLessons)
	assert.Equal(t, "Kept one.", reply.Notes)
}

func TestFinish_KeepsLessonsRecordsThemAndShowsThemOnTheReply(t *testing.T) {
	t.Parallel()

	plan := threadPlan()
	plan.ReplyMessageID = pulid.MustNew("amsg_")
	reflections := &fakeReflections{stored: &agent.Reflection{
		ID:     plan.ReflectionID,
		Status: agent.ReflectionStatusRunning,
	}}
	conversations := &fakeConversations{}
	memories := &fakeMemories{}
	svc := lessonService(memories, false)
	svc.repo = reflections
	svc.conversations = conversations

	outcome, err := svc.Finish(t.Context(), plan, &services.StructuredCompletionResult{
		Text: `{"lessons":[` +
			`{"kind":"Procedure","content":"Pass the move id to assign_move.","audience":"me","subjectType":"","subjectId":"","toolName":"assign_move","replaces":"","evidence":"failed twice","why":"Avoids the failure."},` +
			`{"kind":"Correction","content":"Nope","audience":"me","subjectType":"","subjectId":"","toolName":"","replaces":"","evidence":"","why":""}` +
			`],"notes":"One procedure kept."}`,
		ModelIdentifier: "test-model",
		InputTokens:     1200,
		OutputTokens:    80,
	})
	require.NoError(t, err)

	assert.Equal(t, agent.ReflectionStatusCompleted, outcome.Status)
	assert.Equal(t, 1, outcome.Kept)
	assert.Equal(t, 1, outcome.Refused)

	stored := reflections.stored
	require.Len(t, stored.Changes, 2)
	assert.Equal(t, agent.ReflectionActionSaved, stored.Changes[0].Action)
	assert.Equal(t, agent.ReflectionActionRefused, stored.Changes[1].Action)
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
	reflections := &fakeReflections{stored: &agent.Reflection{
		ID:     plan.ReflectionID,
		Status: agent.ReflectionStatusRunning,
	}}
	memories := &fakeMemories{refuse: errortypes.NewValidationError(
		"subjectId", errortypes.ErrInvalid, "No customer with that id is visible to you",
	)}
	svc := lessonService(memories, false)
	svc.repo = reflections
	svc.conversations = &fakeConversations{}

	outcome, err := svc.Finish(t.Context(), plan, &services.StructuredCompletionResult{
		Text: `{"lessons":[{"kind":"Fact","content":"x","audience":"me","subjectType":"","subjectId":"","toolName":"","replaces":"","evidence":"","why":""}],"notes":""}`,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, outcome.Refused)
	assert.Contains(t, reflections.stored.Changes[0].Reason, "No customer with that id")
}

func TestFinish_AnUnreadableAnswerFailsTheLookBack(t *testing.T) {
	t.Parallel()

	plan := threadPlan()
	reflections := &fakeReflections{stored: &agent.Reflection{
		ID:     plan.ReflectionID,
		Status: agent.ReflectionStatusRunning,
	}}
	svc := lessonService(&fakeMemories{}, false)
	svc.repo = reflections

	outcome, err := svc.Finish(t.Context(), plan, &services.StructuredCompletionResult{Text: "{"})
	require.NoError(t, err)
	assert.Equal(t, agent.ReflectionStatusFailed, outcome.Status)
	assert.NotEmpty(t, reflections.stored.ErrorMessage)
}
