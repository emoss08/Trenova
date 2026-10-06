package assistantservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type runTranscriptFixture struct {
	svc       *Service
	runs      *stubRunRepo
	proposals *stubProposalRepo
	run       *agent.AgentRun
	tenant    pagination.TenantInfo
}

func newRunTranscriptFixture(transcript *agent.RunTranscript) runTranscriptFixture {
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	definitionID := pulid.MustNew("agdef_")
	completedAt := int64(1_790_000_090)
	run := &agent.AgentRun{
		ID:                pulid.ID("ar_01J000000000000000000RUN0001"),
		AgentDefinitionID: definitionID,
		Trigger:           agent.RunTriggerScheduled,
		Status:            agent.RunStatusCompleted,
		ModelIdentifier:   "nvidia/nemotron-3-super",
		StartedAt:         1_790_000_000,
		CompletedAt:       &completedAt,
	}
	runs := &stubRunRepo{
		byID:        map[pulid.ID]*agent.AgentRun{run.ID: run},
		transcripts: map[pulid.ID]*agent.RunTranscript{},
	}
	if transcript != nil {
		runs.transcripts[run.ID] = transcript
	}
	proposals := &stubProposalRepo{}

	return runTranscriptFixture{
		svc: &Service{
			logger: zap.NewNop(),
			runs:   runs,
			definitions: &delegateDefinitions{byID: map[pulid.ID]*agentdefinition.Definition{
				definitionID:       {ID: definitionID, Name: "Overnight check"},
				parityDelegateID(): {ID: parityDelegateID(), Name: "Pricing desk"},
			}},
			proposals: proposals,
		},
		runs:      runs,
		proposals: proposals,
		run:       run,
		tenant:    tenant,
	}
}

func (f runTranscriptFixture) request() repositories.GetAgentRunByIDRequest {
	tenant := f.tenant

	return repositories.GetAgentRunByIDRequest{ID: f.run.ID, TenantInfo: &tenant}
}

func roundTripped(messages []conversation.Message) []conversation.Message {
	transcript := conversation.RunTranscriptOf(messages)
	out := make([]conversation.Message, len(transcript.Messages))
	for idx := range transcript.Messages {
		out[idx] = conversation.MessageOfTranscript(&transcript.Messages[idx])
	}

	return out
}

func afterHeader(t *testing.T, body string) string {
	t.Helper()

	idx := strings.Index(body, "\n---\n")
	require.Positive(t, idx)

	return body[idx:]
}

func TestRunTranscript_RendersTheRunWithTheConversationsRenderer(t *testing.T) {
	t.Parallel()

	messages := parityMessages()
	fixture := newRunTranscriptFixture(conversation.RunTranscriptOf(messages))
	fixture.proposals.byRun = parityInput().Proposals

	file, err := fixture.svc.RunTranscript(t.Context(), fixture.request())
	require.NoError(t, err)

	thread := parityInput()
	thread.Messages = roundTripped(messages)
	thread.AgentName = "Overnight check"
	assert.Equal(t, afterHeader(t, renderTranscript(thread)), afterHeader(t, file.Body))

	assert.Equal(t, "overnight-check-run-0run0001.md", file.FileName)
	assert.True(t, strings.HasPrefix(file.Body, "# Overnight check run\n\n"+
		"- **Agent:** Overnight check\n"+
		"- **Trigger:** Scheduled\n"+
		"- **Status:** Completed\n"+
		"- **Started:** Sep 21, 2026 2:13:20 PM UTC\n"+
		"- **Finished:** Sep 21, 2026 2:14:50 PM UTC\n"+
		"- **Model:** nvidia/nemotron-3-super\n"+
		"- **Messages:** 14\n"))
	assert.Contains(t, file.Body, "- **Run id:** `ar_01J000000000000000000RUN0001`\n")
	assert.Contains(t, file.Body, "## Handed to Pricing desk")
	assert.NotContains(t, file.Body, "left out")
}

func TestRunTranscript_ReadsUnderTheRequestsTenant(t *testing.T) {
	t.Parallel()

	fixture := newRunTranscriptFixture(conversation.RunTranscriptOf(transcriptMessages()))

	_, err := fixture.svc.RunTranscript(t.Context(), fixture.request())
	require.NoError(t, err)

	require.Len(t, fixture.runs.transcriptReads, 1)
	assert.Equal(t, fixture.tenant, fixture.runs.transcriptReads[0].TenantInfo)
	assert.Equal(t, []pulid.ID{fixture.run.ID}, fixture.runs.transcriptReads[0].IDs)
	assert.Equal(t, fixture.tenant, fixture.proposals.lastRun.TenantInfo)
	assert.Equal(t, fixture.run.ID, fixture.proposals.lastRun.RunID)
}

func TestRunTranscript_StatesWhatTheRunLeftOut(t *testing.T) {
	t.Parallel()

	transcript := &agent.RunTranscript{
		Messages: []agent.TranscriptMessage{
			{Role: "User", Content: "Check overnight loads.", CreatedAt: 1_790_000_000},
			{
				Role:      "Assistant",
				Omitted:   true,
				ToolCalls: []agent.TranscriptToolCall{{ID: "c1", Name: "list_shipments"}},
				CreatedAt: 1_790_000_001,
			},
			{Role: "Assistant", Content: "All loads are on time.", CreatedAt: 1_790_000_080},
		},
		OmittedMessages: 12,
		OmittedAt:       2,
	}
	fixture := newRunTranscriptFixture(transcript)

	file, err := fixture.svc.RunTranscript(t.Context(), fixture.request())
	require.NoError(t, err)

	body := file.Body
	assert.Contains(t, body, "- **Messages:** 3 kept, 12 left out\n")
	assert.Contains(t, body, "**Called `list_shipments`**\n\n---")
	assert.Contains(t, body, "## Overnight check · Sep 21, 2026 2:13:21 PM UTC\n\n"+
		transcriptTrimmedNote)
	assert.Contains(t, body, "\n---\n\n_12 messages left out here: the run kept its opening "+
		"and its end within its size limit._\n\n---\n\n## Overnight check · "+
		"Sep 21, 2026 2:14:40 PM UTC")
	assert.Less(t, strings.Index(body, "list_shipments"), strings.Index(body, "12 messages"))
	assert.Less(t, strings.Index(body, "12 messages"), strings.Index(body, "All loads"))
}

func TestRunTranscript_AGapNeverSplitsADelegationIntoOne(t *testing.T) {
	t.Parallel()

	delegated := func(content string, at int64) agent.TranscriptMessage {
		return agent.TranscriptMessage{
			Role:              "Assistant",
			Kind:              string(conversation.MessageKindDelegated),
			AgentDefinitionID: parityDelegateID(),
			DelegateCallID:    "call_d",
			Content:           content,
			CreatedAt:         at,
		}
	}
	fixture := newRunTranscriptFixture(&agent.RunTranscript{
		Messages:        []agent.TranscriptMessage{delegated("before", 1), delegated("after", 2)},
		OmittedMessages: 1,
		OmittedAt:       1,
	})

	file, err := fixture.svc.RunTranscript(t.Context(), fixture.request())
	require.NoError(t, err)

	assert.Equal(t, 2, strings.Count(file.Body, "## Handed to Pricing desk"))
	assert.Less(t, strings.Index(file.Body, "before"), strings.Index(file.Body, "1 message left"))
	assert.Less(t, strings.Index(file.Body, "1 message left"), strings.Index(file.Body, "after"))
}

func TestRunTranscript_ARunWithoutATranscriptSaysSo(t *testing.T) {
	t.Parallel()

	fixture := newRunTranscriptFixture(nil)

	file, err := fixture.svc.RunTranscript(t.Context(), fixture.request())
	require.NoError(t, err)

	assert.Contains(t, file.Body, "- **Messages:** 0\n")
	assert.Contains(t, file.Body, "\n---\n\n_This run kept no transcript._\n")
}

func TestRunTranscript_NamesARunWhoseAgentIsGone(t *testing.T) {
	t.Parallel()

	fixture := newRunTranscriptFixture(nil)
	fixture.run.AgentDefinitionID = pulid.MustNew("agdef_")

	file, err := fixture.svc.RunTranscript(t.Context(), fixture.request())
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(file.Body, "# Agent run\n"))
	assert.Equal(t, "agent-run-0run0001.md", file.FileName)
}

func TestRunTranscript_ARunOutsideTheTenantIsNotFound(t *testing.T) {
	t.Parallel()

	fixture := newRunTranscriptFixture(nil)
	request := fixture.request()
	request.ID = pulid.MustNew("ar_")

	_, err := fixture.svc.RunTranscript(t.Context(), request)

	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
	assert.Empty(t, fixture.runs.transcriptReads)
}

func TestRunTranscript_RefusesARequestWithoutATenant(t *testing.T) {
	t.Parallel()

	fixture := newRunTranscriptFixture(nil)

	_, err := fixture.svc.RunTranscript(
		t.Context(),
		repositories.GetAgentRunByIDRequest{ID: fixture.run.ID},
	)

	require.Error(t, err)
	assert.Zero(t, fixture.runs.reads)
}
