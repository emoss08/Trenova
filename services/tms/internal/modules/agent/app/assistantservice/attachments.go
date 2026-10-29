package assistantservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/pkg/errortypes"
)

// turnContext is what the person handed over with a message besides the
// words: the page, the files, and the records they named. It is stored on
// the user turn and rendered into the prompt, and nowhere else.
type turnContext struct {
	page        *agentpagecontext.PageContext
	attachments []conversation.MessageAttachment
	runtime     []agentturn.RuntimeAttachment
	mentions    []agentpagecontext.EntityRef
}

// validateMentions bounds and checks the records a person named.
func validateMentions(
	mentions []agentpagecontext.EntityRef,
	multiErr *errortypes.MultiError,
) []agentpagecontext.EntityRef {
	normalized := agentpagecontext.NormalizeEntityRefs(mentions)
	agentpagecontext.ValidateEntityRefs("mentions", normalized, multiErr)

	return normalized
}

// attachTurnContext records what the person handed over on their turn only:
// the assistant and tool messages that follow are about the same things, but
// the fact worth keeping is what the question came with.
func attachTurnContext(messages []conversation.Message, tc turnContext) {
	for i := range messages {
		if messages[i].Role != conversation.RoleUser {
			continue
		}
		messages[i].PageContext = tc.page
		if len(tc.attachments) > 0 {
			messages[i].Attachments = tc.attachments
		}
		if len(tc.mentions) > 0 {
			messages[i].Mentions = tc.mentions
		}

		return
	}
}
