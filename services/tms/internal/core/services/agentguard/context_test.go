package agentguard_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/services/agentguard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func reportThread() []agentguard.Turn {
	return []agentguard.Turn{
		{Role: "user", Content: "Run the Expiring Driver Credentials report over 360 days"},
		{Role: "assistant", Content: "The report has been queued and is running."},
	}
}

// "Can you give me a link to download it?" has no subject of its own. Read
// alone it is unclassifiable, which is how a question about a report the
// assistant had just run came back refused as off-topic. The referent is in the
// turns before it, so the classifier is given them.
func TestEvaluate_SendsTheConversationThatSaysWhatTheRequestIsAbout(t *testing.T) {
	t.Parallel()

	stub := &stubCompletion{category: string(agentguard.CategoryTransportationOperations)}
	decision := newGuard(t, stub).Evaluate(t.Context(), agentguard.EvaluateRequest{
		TenantInfo: tenant(),
		Input:      "Can you give me a link to download it?",
		Recent:     reportThread(),
	})

	require.True(t, decision.Allowed)
	require.NotNil(t, stub.lastReq)

	sections := stub.lastReq.Context.Sections
	require.Len(t, sections, 2, "the conversation and the request travel as separate sections")

	assert.Contains(t, sections[0].Content, "Expiring Driver Credentials")
	assert.False(t, sections[0].Trusted, "earlier turns are data, never instruction")

	// The last section is what is being judged. Reversing these would classify
	// the history instead of the message.
	assert.Equal(t, "Can you give me a link to download it?", sections[1].Content)
	assert.False(t, sections[1].Trusted)
}

// A first message has no conversation, and must be classified exactly as it was
// before any of this existed.
func TestEvaluate_SendsNoConversationSectionOnAFirstMessage(t *testing.T) {
	t.Parallel()

	stub := &stubCompletion{category: string(agentguard.CategoryTransportationOperations)}
	newGuard(t, stub).Evaluate(t.Context(), agentguard.EvaluateRequest{
		TenantInfo: tenant(),
		Input:      "Which drivers hold a current hazmat endorsement?",
	})

	require.NotNil(t, stub.lastReq)
	require.Len(t, stub.lastReq.Context.Sections, 1)
	assert.Equal(
		t,
		"Which drivers hold a current hazmat endorsement?",
		stub.lastReq.Context.Sections[0].Content,
	)
}

// The verdict is about the pair, not the sentence. Sharing one across threads
// would make the ambiguity permanent: whichever thread asked first would decide
// what "can you download it" means for every other.
func TestEvaluate_DoesNotReuseAVerdictAcrossDifferentConversations(t *testing.T) {
	t.Parallel()

	stub := &stubCompletion{category: string(agentguard.CategoryTransportationOperations)}
	guard := newGuard(t, stub)
	tenantInfo := tenant()

	guard.Evaluate(t.Context(), agentguard.EvaluateRequest{
		TenantInfo: tenantInfo,
		Input:      "Can you give me a link to download it?",
		Recent:     reportThread(),
	})
	guard.Evaluate(t.Context(), agentguard.EvaluateRequest{
		TenantInfo: tenantInfo,
		Input:      "Can you give me a link to download it?",
		Recent: []agentguard.Turn{
			{Role: "user", Content: "Tell me about the new Dune film"},
		},
	})

	assert.Equal(t, 2, stub.calls, "a different conversation is a different question")
}

// The same question in the same conversation is still one classification.
func TestEvaluate_ReusesAVerdictWithinTheSameConversation(t *testing.T) {
	t.Parallel()

	stub := &stubCompletion{category: string(agentguard.CategoryTransportationOperations)}
	guard := newGuard(t, stub)
	tenantInfo := tenant()
	req := agentguard.EvaluateRequest{
		TenantInfo: tenantInfo,
		Input:      "Can you give me a link to download it?",
		Recent:     reportThread(),
	}

	guard.Evaluate(t.Context(), req)
	guard.Evaluate(t.Context(), req)

	assert.Equal(t, 1, stub.calls)
}

// A thread is unbounded and an answer can be a whole table of drivers. Neither
// may be allowed to dominate the classifier prompt, which would cost more than
// the call it is meant to inform.
func TestEvaluate_BoundsHowMuchConversationItSends(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("Sarah Williams, Class A, Phoenix. ", 200)
	history := make([]agentguard.Turn, 0, 20)
	for range 10 {
		history = append(history,
			agentguard.Turn{Role: "user", Content: "and then?"},
			agentguard.Turn{Role: "assistant", Content: long},
		)
	}

	stub := &stubCompletion{category: string(agentguard.CategoryTransportationOperations)}
	newGuard(t, stub).Evaluate(t.Context(), agentguard.EvaluateRequest{
		TenantInfo: tenant(),
		Input:      "Can you give me a link to download it?",
		Recent:     history,
	})

	require.NotNil(t, stub.lastReq)
	conversation := stub.lastReq.Context.Sections[0].Content
	assert.Less(t, len(conversation), 4000, "the tail is a hint, not the transcript")
	assert.Equal(t, 6, strings.Count(conversation, "\n")+1, "only the last turns are shown")
}

// An empty turn contributes nothing and must not leave a bare "user:" line that
// reads as a message with no content.
func TestEvaluate_SkipsEmptyTurns(t *testing.T) {
	t.Parallel()

	stub := &stubCompletion{category: string(agentguard.CategoryTransportationOperations)}
	newGuard(t, stub).Evaluate(t.Context(), agentguard.EvaluateRequest{
		TenantInfo: tenant(),
		Input:      "and the other one?",
		Recent: []agentguard.Turn{
			{Role: "assistant", Content: "   "},
			{Role: "user", Content: "Which drivers are out of hours?"},
		},
	})

	require.NotNil(t, stub.lastReq)
	assert.Equal(
		t,
		"user: Which drivers are out of hours?",
		stub.lastReq.Context.Sections[0].Content,
	)
}
