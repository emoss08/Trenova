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
