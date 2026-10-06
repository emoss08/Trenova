package assistanthandoffservice

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeConversations struct {
	repositories.ConversationRepository

	threads  map[pulid.ID]*conversation.Thread
	history  []conversation.Message
	appended map[pulid.ID][]conversation.Message
}

func (f *fakeConversations) GetThread(
	_ context.Context,
	req repositories.GetThreadRequest,
) (*conversation.Thread, error) {
	thread, ok := f.threads[req.ID]
	if !ok || thread.UserID != req.UserID {
		return nil, errortypes.NewNotFoundError("Conversation not found")
	}

	return thread, nil
}

func (f *fakeConversations) ListMessages(
	_ context.Context,
	req repositories.ListMessagesRequest,
) ([]conversation.Message, error) {
	return f.history, nil
}

func (f *fakeConversations) AppendTurn(
	_ context.Context,
	req repositories.AppendTurnRequest,
) ([]conversation.Message, error) {
	f.appended[req.ThreadID] = append(f.appended[req.ThreadID], req.Messages...)

	return req.Messages, nil
}

type fakeDefinitions struct {
	repositories.AgentDefinitionRepository

	byID map[pulid.ID]*agentdefinition.Definition
}

func (f *fakeDefinitions) GetByID(
	_ context.Context,
	req repositories.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	definition, ok := f.byID[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Agent not found")
	}

	return definition, nil
}

type fakeArtifacts struct {
	repositories.AssistantArtifactRepository

	listed []*assistantartifact.Artifact
	copied []*assistantartifact.Artifact
}

func (f *fakeArtifacts) ListPage(
	_ context.Context,
	req repositories.ListArtifactsRequest,
) (*repositories.ArtifactPage, error) {
	out := make([]*assistantartifact.Artifact, 0, len(f.listed))
	for _, artifact := range f.listed {
		if !req.PinnedOnly || artifact.Pinned {
			out = append(out, artifact)
		}
	}

	return &repositories.ArtifactPage{Artifacts: out, Total: len(out)}, nil
}

func (f *fakeArtifacts) Upsert(
	_ context.Context,
	artifact *assistantartifact.Artifact,
) (*assistantartifact.Artifact, error) {
	artifact.ID = pulid.MustNew("art_")
	f.copied = append(f.copied, artifact)

	return artifact, nil
}

type fakeAssistant struct {
	services.AssistantService

	started *services.StartThreadRequest
	refuse  error
}

func (f *fakeAssistant) StartThread(
	_ context.Context,
	req *services.StartThreadRequest,
	actor *services.RequestActor,
) (*conversation.Thread, error) {
	if f.refuse != nil {
		return nil, f.refuse
	}
	f.started = req

	return &conversation.Thread{
		ID:                 pulid.MustNew("athr_"),
		UserID:             actor.UserID,
		AgentDefinitionID:  req.AgentDefinitionID,
		HandedFromThreadID: req.HandedFromThreadID,
		Taint:              req.Taint,
	}, nil
}

type fakeCompleter struct {
	text string
	err  error
	req  *services.StructuredCompletionRequest
}

func (f *fakeCompleter) CompleteStructured(
	_ context.Context,
	req *services.StructuredCompletionRequest,
) (*services.StructuredCompletionResult, error) {
	f.req = req
	if f.err != nil {
		return nil, f.err
	}

	return &services.StructuredCompletionResult{Text: f.text}, nil
}

type handoffFixture struct {
	svc           *Service
	user          pulid.ID
	origin        *conversation.Thread
	target        *agentdefinition.Definition
	conversations *fakeConversations
	artifacts     *fakeArtifacts
	assistant     *fakeAssistant
	completer     *fakeCompleter
}

func newHandoffFixture() *handoffFixture {
	user := pulid.MustNew("usr_")
	source := &agentdefinition.Definition{ID: pulid.MustNew("agdef_"), Name: "Billing Specialist"}
	target := &agentdefinition.Definition{ID: pulid.MustNew("agdef_"), Name: "Cash Application"}
	origin := &conversation.Thread{
		ID:                pulid.MustNew("athr_"),
		UserID:            user,
		AgentDefinitionID: source.ID,
		Title:             "Acme payments",
		Taint:             &agent.RunTaint{},
		PinnedFacts:       []string{"Net 30"},
	}
	conversations := &fakeConversations{
		threads: map[pulid.ID]*conversation.Thread{origin.ID: origin},
		history: []conversation.Message{
			{Role: conversation.RoleUser, Content: "Which Acme invoices are unpaid?"},
			{Role: conversation.RoleTool, Content: "raw rows"},
			{Role: conversation.RoleAssistant, Content: "INV-1 and INV-2, $9,835 together."},
		},
		appended: map[pulid.ID][]conversation.Message{},
	}
	artifacts := &fakeArtifacts{listed: []*assistantartifact.Artifact{
		{ID: pulid.MustNew("art_"), Pinned: true, Kind: "table_view", Title: "Unpaid invoices",
			Payload: map[string]any{"rows": 2}},
		{ID: pulid.MustNew("art_"), Pinned: true, Kind: "email_draft", Title: "Reminder",
			ProposalID: pulid.MustNew("aprop_")},
		{ID: pulid.MustNew("art_"), Kind: "record", Title: "INV-1"},
	}}
	assistant := &fakeAssistant{}
	completer := &fakeCompleter{text: `{"summary":"The person is chasing two unpaid Acme invoices."}`}

	svc := New(Params{
		Logger:        zap.NewNop(),
		Conversations: conversations,
		Definitions: &fakeDefinitions{byID: map[pulid.ID]*agentdefinition.Definition{
			source.ID: source, target.ID: target,
		}},
		Artifacts:  artifacts,
		Assistant:  assistant,
		Completion: completer,
	}).(*Service)

	return &handoffFixture{
		svc: svc, user: user, origin: origin, target: target,
		conversations: conversations, artifacts: artifacts,
		assistant: assistant, completer: completer,
	}
}

func (f *handoffFixture) handoff(t *testing.T, facts ...string) (*services.HandoffThreadResult, error) {
	t.Helper()

	return f.svc.Handoff(t.Context(), &services.HandoffThreadRequest{
		ThreadID:          f.origin.ID,
		AgentDefinitionID: f.target.ID,
		Facts:             facts,
	}, &services.RequestActor{PrincipalType: services.PrincipalTypeUser, UserID: f.user})
}

// The new conversation opens with what was carried over, and the one handed
// off keeps a card saying where it went.
func TestHandoff_SeedsTheNewConversationAndLeavesACard(t *testing.T) {
	t.Parallel()

	f := newHandoffFixture()
	result, err := f.handoff(t, "Invoice date is Oct 3", " Invoice date is Oct 3 ", "")
	require.NoError(t, err)

	newThread := result.Thread
	assert.Equal(t, f.origin.ID, newThread.HandedFromThreadID, "the new conversation links back")
	assert.Equal(t, f.origin.Taint, f.assistant.started.Taint,
		"what the old conversation read comes along with its summary")
	assert.Equal(t, conversation.ThreadOriginDesk, f.assistant.started.Origin)

	brief := f.conversations.appended[newThread.ID]
	require.Len(t, brief, 1)
	assert.Equal(t, conversation.MessageKindHandoffBrief, brief[0].Kind)
	assert.Equal(t, conversation.RoleUser, brief[0].Role, "the brief is replayed to the new agent")
	assert.Contains(t, brief[0].Content, "two unpaid Acme invoices")
	assert.Contains(t, brief[0].Content, "Invoice date is Oct 3")
	assert.Contains(t, brief[0].Content, "Unpaid invoices")

	card := f.conversations.appended[f.origin.ID]
	require.Len(t, card, 1)
	assert.Equal(t, conversation.MessageKindHandoff, card[0].Kind)
	assert.Contains(t, conversation.ModelHiddenKinds(), card[0].Kind,
		"the old agent is never sent the card")
	handoff := card[0].Handoff
	require.NotNil(t, handoff)
	assert.Equal(t, newThread.ID, handoff.ToThreadID)
	assert.Equal(t, "Cash Application", handoff.ToAgentName)
	assert.Equal(t, []string{"Net 30", "Invoice date is Oct 3"}, handoff.Facts,
		"the conversation's own facts come first; the rest are cleaned and deduplicated")
	assert.Equal(t, handoff.Facts, f.assistant.started.PinnedFacts,
		"the new conversation keeps the facts pinned")
	require.Len(t, handoff.Artifacts, 1, "only standalone pinned artifacts are carried")
	assert.Equal(t, "Unpaid invoices", handoff.Artifacts[0].Title)

	require.Len(t, f.artifacts.copied, 1)
	copied := f.artifacts.copied[0]
	assert.Equal(t, newThread.ID, copied.ThreadID)
	assert.True(t, copied.Pinned)
	assert.True(t, strings.HasPrefix(copied.SourceToolCallID, handoffCallPrefix))

	transcript := f.completer.req.Context.Sections[0].Content
	assert.NotContains(t, transcript, "raw rows", "tool traffic is not summarized")
	assert.Equal(t, 700, f.completer.req.MaxTokens, "the summary call is bounded")
}

func TestHandoff_FallsBackToTheLastExchangesWithoutAModel(t *testing.T) {
	t.Parallel()

	f := newHandoffFixture()
	f.completer.err = errors.New("provider down")

	result, err := f.handoff(t)
	require.NoError(t, err, "a summary that could not be written does not stop the hand-off")

	brief := f.conversations.appended[result.Thread.ID][0]
	assert.Contains(t, brief.Handoff.Summary, "Person: Which Acme invoices are unpaid?")
	assert.Contains(t, brief.Handoff.Summary, "Assistant: INV-1 and INV-2")
}

func TestHandoff_RefusedWritesNothing(t *testing.T) {
	t.Parallel()

	t.Run("someone else's conversation", func(t *testing.T) {
		t.Parallel()

		f := newHandoffFixture()
		_, err := f.svc.Handoff(t.Context(), &services.HandoffThreadRequest{
			ThreadID:          f.origin.ID,
			AgentDefinitionID: f.target.ID,
		}, &services.RequestActor{PrincipalType: services.PrincipalTypeUser, UserID: pulid.MustNew("usr_")})
		require.Error(t, err)
		assert.Empty(t, f.conversations.appended)
	})

	t.Run("an agent the person may not use", func(t *testing.T) {
		t.Parallel()

		f := newHandoffFixture()
		f.assistant.refuse = errortypes.NewAuthorizationError("You do not have access")
		_, err := f.handoff(t)
		require.Error(t, err)
		assert.Empty(t, f.conversations.appended)
		assert.Empty(t, f.artifacts.copied)
		assert.Nil(t, f.completer.req, "no model is asked for a refused hand-off")
	})

	t.Run("the same agent", func(t *testing.T) {
		t.Parallel()

		f := newHandoffFixture()
		_, err := f.svc.Handoff(t.Context(), &services.HandoffThreadRequest{
			ThreadID:          f.origin.ID,
			AgentDefinitionID: f.origin.AgentDefinitionID,
		}, &services.RequestActor{PrincipalType: services.PrincipalTypeUser, UserID: f.user})
		require.Error(t, err)
	})
}

func TestTranscript_KeepsTheNewestWithinBudget(t *testing.T) {
	t.Parallel()

	history := []conversation.Message{
		{Role: conversation.RoleUser, Content: strings.Repeat("a", 50)},
		{Role: conversation.RoleAssistant, Content: "newest answer"},
	}

	got := Transcript(history, 30)
	assert.Equal(t, "Assistant: newest answer", got)
}

func TestCleanFacts_BoundsTheList(t *testing.T) {
	t.Parallel()

	facts := make([]string, 0, 30)
	for i := range 30 {
		facts = append(facts, strings.Repeat("x", i+1))
	}
	facts = append(facts, strings.Repeat("y", 500))

	got := CleanFacts(facts)
	assert.Len(t, got, conversation.MaxHandoffFacts)
}

// A pinned artifact with versions goes over once, as its newest version says.
func TestLatestVersions_KeepsTheNewestOfEachLineage(t *testing.T) {
	t.Parallel()

	lineage := pulid.MustNew("art_")
	first := &assistantartifact.Artifact{ID: lineage, LineageID: lineage, LineageSeq: 1, Title: "Brief"}
	second := &assistantartifact.Artifact{ID: pulid.MustNew("art_"), LineageID: lineage, LineageSeq: 2, Title: "Brief, revised"}
	other := &assistantartifact.Artifact{ID: pulid.MustNew("art_"), Title: "Table"}

	got := latestVersions([]*assistantartifact.Artifact{first, other, second})

	require.Len(t, got, 2)
	assert.Equal(t, "Brief, revised", got[0].Title)
	assert.Equal(t, "Table", got[1].Title)
}
