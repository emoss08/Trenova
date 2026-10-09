package agentevalgate_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/services/agentevalgate"
	"github.com/stretchr/testify/require"
)

const (
	conversationSuite = "evals/conversation.yaml"
	minConversation   = 4
)

// Every scripted conversation holds: the runtime reads what a small model
// sends as the tool declares it, tells it what an enum allows, and records
// a reply without the ids and reprinted tables the prompt forbids.
func TestConversationsHoldTheRuntimesContract(t *testing.T) {
	t.Parallel()

	suite, err := agentevalgate.LoadConversationSuite(conversationSuite)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(suite.Cases), minConversation,
		"the conversation suite keeps at least %d cases", minConversation)

	for idx := range suite.Cases {
		c := &suite.Cases[idx]
		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()

			outcome, err := agentevalgate.RunConversation(t.Context(), c)
			require.NoError(t, err)
			failures := outcome.Failures()
			require.Emptyf(t, failures, "%s:\n%s\nreply:\n%s",
				c.Name, strings.Join(failures, "\n"), outcome.Result.Reply)
		})
	}
}
