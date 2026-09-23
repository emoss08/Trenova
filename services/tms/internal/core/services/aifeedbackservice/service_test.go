package aifeedbackservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/aifeedback"
	"github.com/emoss08/trenova/internal/core/domain/briefing"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type conversationFixture struct {
	thread   *conversation.Thread
	question conversation.Message
	tool     conversation.Message
	answer   conversation.Message
	turn     *conversation.AssistantTurn
}

func newConversation(owner pulid.ID) conversationFixture {
	thread := &conversation.Thread{
		ID:                pulid.MustNew("athr_"),
		UserID:            owner,
		AgentDefinitionID: pulid.MustNew("agdef_"),
		SubjectType:       agent.SubjectType("Shipment"),
	}

	return conversationFixture{
		thread: thread,
		question: conversation.Message{
			ID:       pulid.MustNew("amsg_"),
			ThreadID: thread.ID,
			Role:     conversation.RoleUser,
			Kind:     conversation.MessageKindMessage,
			Content:  "Where is load 12?",
		},
		tool: conversation.Message{
			ID:          pulid.MustNew("amsg_"),
			ThreadID:    thread.ID,
			Role:        conversation.RoleTool,
			Kind:        conversation.MessageKindMessage,
			ToolName:    "get_shipment",
			ToolSummary: "Load 12",
			Content:     `{"proNumber":"12","status":"InTransit"}`,
		},
		answer: conversation.Message{
			ID:         pulid.MustNew("amsg_"),
			ThreadID:   thread.ID,
			Role:       conversation.RoleAssistant,
			Kind:       conversation.MessageKindMessage,
			Content:    "Load 12 is in Dallas.",
			Model:      "claude-opus",
			ProviderID: pulid.MustNew("aip_"),
		},
		turn: &conversation.AssistantTurn{
			ID:    pulid.MustNew("atrn_"),
			RunID: pulid.MustNew("arun_"),
		},
	}
}

func (c conversationFixture) source() *repositories.AIFeedbackMessageContext {
	answer := c.answer

	return &repositories.AIFeedbackMessageContext{
		Message:  &answer,
		Thread:   c.thread,
		Exchange: []conversation.Message{c.question, c.tool, c.answer},
		Turn:     c.turn,
	}
}

func thumbsDown(target repositories.AIFeedbackTargetRef) *services.SetAIFeedbackRequest {
	return &services.SetAIFeedbackRequest{
		Target:  target,
		Rating:  aifeedback.RatingNegative,
		Reasons: []aifeedback.Reason{aifeedback.ReasonInaccurate},
		Comment: "  It is in Houston.  ",
	}
}

func TestSetMine_AnAnswerIsRatedOnlyByTheThreadsOwner(t *testing.T) {
	t.Parallel()

	h := newHarness()
	tenant := testTenant()
	actor := person(tenant)
	fixture := newConversation(pulid.MustNew("usr_"))
	h.messages.context = fixture.source()

	req := thumbsDown(repositories.AIFeedbackTargetRef{
		TargetType: aifeedback.TargetAssistantMessage,
		TargetID:   fixture.answer.ID,
	})
	req.TenantInfo = tenant

	_, err := h.svc.SetMine(t.Context(), req, actor)
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err), "another person's answer reads as missing")
	assert.Empty(t, h.store.upserted)
}

func TestSetMine_CreditsTheAnswerAndKeepsWhatThePersonSaw(t *testing.T) {
	t.Parallel()

	h := newHarness()
	tenant := testTenant()
	actor := person(tenant)
	fixture := newConversation(actor.UserID)
	h.messages.context = fixture.source()
	h.definitions.definition = &agentdefinition.Definition{Version: 7}

	req := thumbsDown(repositories.AIFeedbackTargetRef{
		TargetType: aifeedback.TargetAssistantMessage,
		TargetID:   fixture.answer.ID,
	})
	req.TenantInfo = tenant

	saved, err := h.svc.SetMine(t.Context(), req, actor)
	require.NoError(t, err)
	require.Len(t, h.store.upserted, 1)

	assert.Equal(t, actor.UserID, saved.UserID)
	assert.Equal(t, fixture.thread.AgentDefinitionID, *saved.AgentDefinitionID)
	assert.Equal(t, int64(7), *saved.DefinitionVersion)
	assert.Equal(t, fixture.thread.ID, *saved.ThreadID)
	assert.Equal(t, fixture.turn.ID, *saved.TurnID)
	assert.Equal(t, fixture.turn.RunID, *saved.RunID)
	assert.Equal(t, "claude-opus", saved.Model)
	assert.Equal(t, aifeedback.FingerprintAtRating, saved.FingerprintSource)
	assert.Empty(t, saved.PromptHash, "turn-time fingerprints are recorded elsewhere")
	assert.Equal(t, "It is in Houston.", saved.Comment)
	assert.Equal(t,
		aifeedback.PatternKey([]string{"get_shipment"}, aifeedback.ReasonInaccurate, "Shipment"),
		saved.PatternKey,
	)

	require.NotNil(t, saved.TurnSnapshot)
	assert.Equal(t, "Where is load 12?", saved.TurnSnapshot.Question)
	assert.Equal(t, "Load 12 is in Dallas.", saved.TurnSnapshot.Answer)
	require.Len(t, saved.TurnSnapshot.Tools, 1)
	assert.Equal(t, "get_shipment", saved.TurnSnapshot.Tools[0].Name)
}

func TestSetMine_ReasonsMustMatchTheRating(t *testing.T) {
	t.Parallel()

	h := newHarness()
	tenant := testTenant()
	actor := person(tenant)
	fixture := newConversation(actor.UserID)
	h.messages.context = fixture.source()

	req := &services.SetAIFeedbackRequest{
		TenantInfo: tenant,
		Target: repositories.AIFeedbackTargetRef{
			TargetType: aifeedback.TargetAssistantMessage,
			TargetID:   fixture.answer.ID,
		},
		Rating:  aifeedback.RatingPositive,
		Reasons: []aifeedback.Reason{aifeedback.ReasonMadeUpNumbers},
	}

	_, err := h.svc.SetMine(t.Context(), req, actor)
	require.Error(t, err)
	assert.Empty(t, h.store.upserted)
}

func TestSetMine_ADelegatedAnswerIsCreditedToTheDelegate(t *testing.T) {
	t.Parallel()

	h := newHarness()
	tenant := testTenant()
	actor := person(tenant)
	fixture := newConversation(actor.UserID)
	delegate := pulid.MustNew("agdef_")
	fixture.answer.Kind = conversation.MessageKindDelegated
	fixture.answer.AgentDefinitionID = delegate
	h.messages.context = fixture.source()

	req := thumbsDown(repositories.AIFeedbackTargetRef{
		TargetType: aifeedback.TargetAssistantMessage,
		TargetID:   fixture.answer.ID,
	})
	req.TenantInfo = tenant
	_, err := h.svc.SetMine(t.Context(), req, actor)
	require.Error(t, err, "a delegated step is not the conversation agent's answer")

	req.Target.TargetType = aifeedback.TargetDelegatedAnswer
	saved, err := h.svc.SetMine(t.Context(), req, actor)
	require.NoError(t, err)
	assert.Equal(t, delegate, *saved.AgentDefinitionID)
}

func TestSetMine_OnlyAnAnswerCanBeRated(t *testing.T) {
	t.Parallel()

	h := newHarness()
	tenant := testTenant()
	actor := person(tenant)
	fixture := newConversation(actor.UserID)
	source := fixture.source()
	question := fixture.question
	source.Message = &question
	h.messages.context = source

	req := thumbsDown(repositories.AIFeedbackTargetRef{
		TargetType: aifeedback.TargetAssistantMessage,
		TargetID:   fixture.question.ID,
	})
	req.TenantInfo = tenant

	_, err := h.svc.SetMine(t.Context(), req, actor)
	require.Error(t, err)
}

func TestSetMine_AnInsightNeedsInsightRead(t *testing.T) {
	t.Parallel()

	h := newHarness()
	tenant := testTenant()
	actor := person(tenant)
	entity := &insight.Insight{
		ID:              pulid.MustNew("ins_"),
		DetectorKey:     "late_loads",
		Category:        insight.CategoryServiceQuality,
		Subject:         "Acme Freight",
		Headline:        "On-time fell to 81%",
		Narrative:       "Four late deliveries at the same dock.",
		ModelIdentifier: "claude-haiku",
	}
	h.insights.byID[entity.ID] = entity

	req := thumbsDown(repositories.AIFeedbackTargetRef{
		TargetType: aifeedback.TargetInsight,
		TargetID:   entity.ID,
	})
	req.TenantInfo = tenant

	_, err := h.svc.SetMine(t.Context(), req, actor)
	require.Error(t, err, "no insight:read, no rating")
	assert.Contains(t, h.permissions.asked, permission.ResourceInsight.String())

	h.permissions.allowed[permission.ResourceInsight] = true
	saved, err := h.svc.SetMine(t.Context(), req, actor)
	require.NoError(t, err)
	assert.Equal(t, "late_loads", saved.DetectorKey)
	assert.Nil(t, saved.AgentDefinitionID)
	assert.Equal(t, "claude-haiku", saved.Model)
	assert.Contains(t, saved.TurnSnapshot.Answer, "On-time fell to 81%")
}

func TestSetMine_ABriefingIsRatedByItsReader(t *testing.T) {
	t.Parallel()

	h := newHarness()
	tenant := testTenant()
	actor := person(tenant)
	someoneElse := pulid.MustNew("usr_")
	h.briefings.entity = &briefing.Briefing{
		ID:              pulid.MustNew("brf_"),
		RoleKey:         briefing.RoleGeneral,
		BriefingDate:    "2026-09-23",
		UserID:          &someoneElse,
		ModelIdentifier: "claude-sonnet",
		Sections: []briefing.Section{
			{Key: briefing.SectionKey("dispatch"), Title: "Dispatch", Summary: "Four uncovered"},
		},
	}

	req := thumbsDown(repositories.AIFeedbackTargetRef{
		TargetType: aifeedback.TargetBriefing,
		TargetID:   h.briefings.entity.ID,
	})
	req.TenantInfo = tenant

	_, err := h.svc.SetMine(t.Context(), req, actor)
	require.Error(t, err, "another person's briefing is not theirs to rate")
	assert.True(t, errortypes.IsNotFoundError(err))

	h.briefings.entity.UserID = nil
	_, err = h.svc.SetMine(t.Context(), req, actor)
	require.Error(t, err, "a shared briefing needs briefing:read")

	h.permissions.allowed[permission.ResourceBriefing] = true
	saved, err := h.svc.SetMine(t.Context(), req, actor)
	require.NoError(t, err)
	assert.Equal(t, "claude-sonnet", saved.Model)

	req.Target = repositories.AIFeedbackTargetRef{
		TargetType: aifeedback.TargetBriefingSection,
		TargetID:   h.briefings.entity.ID,
		TargetPart: "billing",
	}
	_, err = h.svc.SetMine(t.Context(), req, actor)
	require.Error(t, err, "a section that is not in the briefing cannot be rated")

	req.Target.TargetPart = "dispatch"
	saved, err = h.svc.SetMine(t.Context(), req, actor)
	require.NoError(t, err)
	assert.Equal(t, "dispatch", saved.TargetPart)
	assert.Contains(t, saved.TurnSnapshot.Answer, "Four uncovered")
}

func TestSetMine_AWatchtowerItemIsCreditedToTheAgentBehindIt(t *testing.T) {
	t.Parallel()

	h := newHarness()
	tenant := testTenant()
	actor := person(tenant)
	agentID := pulid.MustNew("agdef_")
	runID := pulid.MustNew("arun_")
	proposalID := pulid.MustNew("aprop_")
	h.watchtower.item = &watchtower.Item{
		ID:         pulid.MustNew("wt_"),
		SourceKind: watchtower.SourceAgentProposal,
		SourceID:   proposalID.String(),
		Title:      "Hold load 12",
		Summary:    "The agent proposes a hold",
	}
	h.proposals.proposal = &agent.AgentProposal{ID: proposalID, RunID: runID}
	h.runs.run = &agent.AgentRun{
		ID:                runID,
		AgentDefinitionID: agentID,
		ModelIdentifier:   "claude-opus",
	}

	req := thumbsDown(repositories.AIFeedbackTargetRef{
		TargetType: aifeedback.TargetWatchtowerItem,
		TargetID:   h.watchtower.item.ID,
	})
	req.TenantInfo = tenant

	_, err := h.svc.SetMine(t.Context(), req, actor)
	require.Error(t, err, "no watchtower:read, no rating")

	h.permissions.allowed[permission.ResourceWatchtower] = true
	_, err = h.svc.SetMine(t.Context(), req, actor)
	require.Error(t, err, "the item's source must be readable too")
	assert.Contains(t, h.permissions.asked, permission.ResourceAgentProposal.String())

	h.permissions.allowed[permission.ResourceAgentProposal] = true
	saved, err := h.svc.SetMine(t.Context(), req, actor)
	require.NoError(t, err)
	assert.Equal(t, agentID, *saved.AgentDefinitionID)
	assert.Equal(t, runID, *saved.RunID)
	assert.Equal(t, "claude-opus", saved.Model)
}

func TestSetMine_AWatchtowerItemFromAnInsightIsCreditedToItsDetector(t *testing.T) {
	t.Parallel()

	h := newHarness()
	tenant := testTenant()
	actor := person(tenant)
	source := &insight.Insight{ID: pulid.MustNew("ins_"), DetectorKey: "unbilled_detention"}
	h.insights.byID[source.ID] = source
	h.watchtower.item = &watchtower.Item{
		ID:         pulid.MustNew("wt_"),
		SourceKind: watchtower.SourceInsight,
		SourceID:   source.ID.String(),
		Title:      "Unbilled detention",
	}
	h.permissions.allowed[permission.ResourceWatchtower] = true
	h.permissions.allowed[permission.ResourceInsight] = true

	req := thumbsDown(repositories.AIFeedbackTargetRef{
		TargetType: aifeedback.TargetWatchtowerItem,
		TargetID:   h.watchtower.item.ID,
	})
	req.TenantInfo = tenant

	saved, err := h.svc.SetMine(t.Context(), req, actor)
	require.NoError(t, err)
	assert.Equal(t, "unbilled_detention", saved.DetectorKey)
}

func TestSetMine_AWatchtowerItemNotRaisedByAIIsRefused(t *testing.T) {
	t.Parallel()

	h := newHarness()
	tenant := testTenant()
	actor := person(tenant)
	h.watchtower.item = &watchtower.Item{
		ID:         pulid.MustNew("wt_"),
		SourceKind: watchtower.SourceWeatherAlert,
		SourceID:   pulid.MustNew("wal_").String(),
		Title:      "Ice storm",
	}
	h.permissions.allowed[permission.ResourceWatchtower] = true
	h.permissions.allowed[permission.ResourceShipment] = true

	req := thumbsDown(repositories.AIFeedbackTargetRef{
		TargetType: aifeedback.TargetWatchtowerItem,
		TargetID:   h.watchtower.item.ID,
	})
	req.TenantInfo = tenant

	_, err := h.svc.SetMine(t.Context(), req, actor)
	require.Error(t, err)
	assert.Empty(t, h.store.upserted)
}

func TestSetMine_OnlyAPersonRates(t *testing.T) {
	t.Parallel()

	h := newHarness()
	tenant := testTenant()
	actor := &services.RequestActor{
		PrincipalType: services.PrincipalTypeAPIKey,
		APIKeyID:      pulid.MustNew("key_"),
	}

	req := thumbsDown(repositories.AIFeedbackTargetRef{
		TargetType: aifeedback.TargetInsight,
		TargetID:   pulid.MustNew("ins_"),
	})
	req.TenantInfo = tenant

	_, err := h.svc.SetMine(t.Context(), req, actor)
	require.Error(t, err)

	_, err = h.svc.ClearMine(t.Context(), services.ClearAIFeedbackRequest{
		TenantInfo: tenant,
		Target:     req.Target,
	}, actor)
	require.Error(t, err)
}

func TestClearMine_RemovesOnlyTheCallersOwnRating(t *testing.T) {
	t.Parallel()

	h := newHarness()
	tenant := testTenant()
	actor := person(tenant)
	target := repositories.AIFeedbackTargetRef{
		TargetType: aifeedback.TargetBriefingSection,
		TargetID:   pulid.MustNew("brf_"),
		TargetPart: "  dispatch ",
	}

	removed, err := h.svc.ClearMine(t.Context(), services.ClearAIFeedbackRequest{
		TenantInfo: tenant,
		Target:     target,
	}, actor)
	require.NoError(t, err)
	assert.True(t, removed)
	require.Len(t, h.store.deleted, 1)
	assert.Equal(t, actor.UserID, h.store.deleted[0].UserID)
	assert.Equal(t, tenant, h.store.deleted[0].TenantInfo)
	assert.Equal(t, "dispatch", h.store.deleted[0].Target.TargetPart)

	_, err = h.svc.ClearMine(t.Context(), services.ClearAIFeedbackRequest{
		TenantInfo: tenant,
		Target: repositories.AIFeedbackTargetRef{
			TargetType: aifeedback.TargetBriefingSection,
			TargetID:   pulid.MustNew("brf_"),
		},
	}, actor)
	require.Error(t, err, "a section is named by its key")
}

func TestListMine_IsScopedToTheCallerAndBounded(t *testing.T) {
	t.Parallel()

	h := newHarness()
	tenant := testTenant()
	actor := person(tenant)

	targets := make([]repositories.AIFeedbackTargetRef, 0, repositories.MaxAIFeedbackTargetsPerRead+1)
	for range repositories.MaxAIFeedbackTargetsPerRead + 1 {
		targets = append(targets, repositories.AIFeedbackTargetRef{
			TargetType: aifeedback.TargetAssistantMessage,
			TargetID:   pulid.MustNew("amsg_"),
		})
	}

	_, err := h.svc.ListMine(t.Context(), services.ListMyAIFeedbackRequest{
		TenantInfo: tenant,
		Targets:    targets,
	}, actor)
	require.Error(t, err)

	_, err = h.svc.ListMine(t.Context(), services.ListMyAIFeedbackRequest{
		TenantInfo: tenant,
		Targets:    targets[:2],
	}, actor)
	require.NoError(t, err)
	require.Len(t, h.store.listed, 1)
	assert.Equal(t, actor.UserID, h.store.listed[0].UserID)
}

func TestSetMine_RedactsRestrictedToolValuesFromTheSnapshot(t *testing.T) {
	t.Parallel()

	h := newHarness()
	tenant := testTenant()
	actor := person(tenant)
	fixture := newConversation(actor.UserID)
	fixture.question.Content = "What is Dana Reyes's license number?"
	fixture.tool.ToolName = "get_worker"
	fixture.tool.ToolSummary = "Dana Reyes"
	fixture.tool.Content = "Result from get_worker:\n<untrusted_data>\n" +
		`{"name":"Dana Reyes","licenseNumber":"D1234567","payRate":0.62,` +
		`"status":"Active","nested":{"ssn":"000-11-2222"}}` +
		"\n</untrusted_data>"
	fixture.answer.Content = "Dana Reyes holds license D1234567, paid 0.62 a mile, SSN 000-11-2222."
	h.messages.context = fixture.source()
	h.svc.redactor = &redactor{
		tools: fakeToolResources{"get_worker": permission.ResourceWorker},
		registry: fakeSensitivity{
			"licenseNumber": permission.SensitivityConfidential,
			"payRate":       permission.SensitivityRestricted,
			"nested":        permission.SensitivityRestricted,
		},
	}

	req := thumbsDown(repositories.AIFeedbackTargetRef{
		TargetType: aifeedback.TargetAssistantMessage,
		TargetID:   fixture.answer.ID,
	})
	req.TenantInfo = tenant

	saved, err := h.svc.SetMine(t.Context(), req, actor)
	require.NoError(t, err)

	snapshot := saved.TurnSnapshot
	require.NotNil(t, snapshot)
	assert.True(t, snapshot.Redacted)
	assert.NotContains(t, snapshot.Answer, "D1234567")
	assert.NotContains(t, snapshot.Answer, "0.62")
	assert.NotContains(t, snapshot.Answer, "000-11-2222")
	assert.Contains(t, snapshot.Answer, "Dana Reyes", "an internal field is kept")
	assert.Contains(t, snapshot.Question, "Dana Reyes")
}

func TestRedactor_ReadsAResultThatWasCutOff(t *testing.T) {
	t.Parallel()

	r := &redactor{
		tools:    fakeToolResources{"list_workers": permission.ResourceWorker},
		registry: fakeSensitivity{"licenseNumber": permission.SensitivityRestricted},
	}
	cut := `[{"name":"A","licenseNumber":"L-998877"},{"name":"B","licenseNumber":"L-5544` +
		"\n\n[This result was cut off here]"

	values := r.restrictedValues([]toolResult{{name: "list_workers", content: cut}})

	assert.Contains(t, values, "L-998877")
	for _, value := range values {
		assert.False(t, strings.HasPrefix(value, "A"), "internal fields are not collected")
	}
}

func TestRedactor_IgnoresAToolItCannotPlace(t *testing.T) {
	t.Parallel()

	r := &redactor{tools: fakeToolResources{}, registry: fakeSensitivity{}}
	values := r.restrictedValues([]toolResult{
		{name: "retired_tool", content: `{"licenseNumber":"L-1"}`},
	})

	assert.Empty(t, values)
}

func TestPurgeExpired_UsesTheOrganizationsRetention(t *testing.T) {
	t.Parallel()

	h := newHarness()
	tenant := testTenant()
	h.store.purgeRows = []int64{purgeBatchSize, 3}

	purged, err := h.svc.PurgeExpired(t.Context(), services.PurgeExpiredAIFeedbackRequest{
		TenantInfo: tenant,
		Now:        1_760_000_000,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(purgeBatchSize+3), purged)
	require.Len(t, h.store.purges, 2, "purging continues while a batch comes back full")
	assert.Equal(t, int64(1_760_000_000-730*secondsPerDay), h.store.purges[0].Before,
		"an organization with no settings keeps two years")
	assert.Equal(t, tenant, h.store.purges[0].TenantInfo)
}

func TestAgentSummary_RefusesAWindowOutOfRange(t *testing.T) {
	t.Parallel()

	h := newHarness()
	_, err := h.svc.AgentSummary(t.Context(), services.AgentFeedbackSummaryRequest{
		TenantInfo:        pagination.TenantInfo{OrgID: pulid.MustNew("org_")},
		AgentDefinitionID: pulid.MustNew("agdef_"),
		WindowDays:        services.MaxAgentFeedbackWindowDays + 1,
	})
	require.Error(t, err)
}
