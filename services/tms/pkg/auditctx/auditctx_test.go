package auditctx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFrom_IsEmptyWithoutASource(t *testing.T) {
	t.Parallel()

	assert.Equal(t, Source{}, From(t.Context()))
}

func TestWithAIRun_NamesTheRunAndHowTheAgentWasAsked(t *testing.T) {
	t.Parallel()

	ctx := WithAIRun(t.Context(), "arun_1", true)
	assert.Equal(t, Source{Channel: ChannelAgentConversation, AIRunID: "arun_1"}, From(ctx))

	ctx = WithAIRun(t.Context(), "arun_2", false)
	assert.Equal(t, Source{Channel: ChannelAgentRun, AIRunID: "arun_2"}, From(ctx))
}

func TestWithChannel_KeepsTheRunItWasGiven(t *testing.T) {
	t.Parallel()

	ctx := WithChannel(WithAIRun(t.Context(), "arun_1", false), ChannelSCIM)
	assert.Equal(t, Source{Channel: ChannelSCIM, AIRunID: "arun_1"}, From(ctx))
}

func TestWithAgent_NamesTheAgentBesideTheRunAndLeavesTheRestAlone(t *testing.T) {
	ctx := WithAgent(WithAIRun(t.Context(), "ar_1", false), "agdef_1")

	source := From(ctx)
	if source.AgentID != "agdef_1" || source.AIRunID != "ar_1" || source.Channel != ChannelAgentRun {
		t.Fatalf("source = %+v", source)
	}
	if got := From(WithAgent(t.Context(), "")); got != (Source{}) {
		t.Fatalf("an empty agent changed the source: %+v", got)
	}
}
