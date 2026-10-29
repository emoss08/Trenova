package anthropicadapter

import (
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
)

// anthropicMessageIDPrefix opens every id the Claude API gives a message.
const anthropicMessageIDPrefix = "msg_"

// applyDiagnostics asks the Claude API to compare this call's prompt with the
// one before it, when the server's configuration says to and the provider is
// the Claude API itself: Claude Platform on AWS or Google Cloud, the cloud
// platforms and gateways speak the same protocol and do not take the field.
// The API keeps a fingerprint only of a call that asked, so every call asks,
// the first of a conversation with nothing to compare.
func applyDiagnostics(params *anthropic.BetaMessageNewParams, call *modeladapter.Call) {
	if !call.CacheDiagnostics || !call.Provider.OnClaudeAPI() {
		return
	}

	previous := param.Null[string]()
	if id := previousReply(call); id != "" {
		previous = anthropic.String(id)
	}
	params.Diagnostics = anthropic.BetaDiagnosticsParam{PreviousMessageID: previous}
}

// previousReply is the id of the last reply this provider gave in the
// conversation, which names the call before this one. A reply from another
// provider is passed over, and one from this provider with no id, kept
// before ids were, means there is nothing to name.
func previousReply(call *modeladapter.Call) string {
	messages := call.Request.Messages
	for idx := len(messages) - 1; idx >= 0; idx-- {
		msg := &messages[idx]
		if msg.Role != modeladapter.RoleAssistant || msg.ReplyProvider != call.Provider.ID {
			continue
		}
		if !strings.HasPrefix(msg.ReplyID, anthropicMessageIDPrefix) {
			return ""
		}

		return msg.ReplyID
	}

	return ""
}

// anthropicDiagnosis reads the diagnostics a message carries. Null means the
// call did not ask, had nothing to compare with, or found no difference; a
// reason left null means the comparison was still running.
func anthropicDiagnosis(message *anthropic.BetaMessage) *modeladapter.CacheDiagnosis {
	if !message.JSON.Diagnostics.Valid() {
		return nil
	}
	if !message.Diagnostics.JSON.CacheMissReason.Valid() {
		return &modeladapter.CacheDiagnosis{Reason: modeladapter.CacheMissPending}
	}

	reason := message.Diagnostics.CacheMissReason
	if reason.Type == "" {
		return nil
	}

	return &modeladapter.CacheDiagnosis{
		Reason:       reason.Type,
		MissedTokens: int(reason.CacheMissedInputTokens),
	}
}
