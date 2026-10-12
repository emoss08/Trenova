package modeladapter

import "strings"

// The reasons a provider gives for a prompt cache miss, in the Claude API's
// words, and the one this adds for a comparison that had not finished.
const (
	CacheMissModelChanged     = "model_changed"
	CacheMissSystemChanged    = "system_changed"
	CacheMissToolsChanged     = "tools_changed"
	CacheMissMessagesChanged  = "messages_changed"
	CacheMissPreviousNotFound = "previous_message_not_found"
	CacheMissUnavailable      = "unavailable"
	CacheMissPending          = "pending"
)

// CacheDiagnosis is what a provider said about where a call's prompt stopped
// matching the call named as the one before it.
type CacheDiagnosis struct {
	// Reason is the first place the two prompts differed, that there was no
	// earlier call to compare with, that no comparison was possible, or that
	// it was still running when the reply began.
	Reason string
	// MissedTokens estimates how much of the prompt came after the point
	// where it stopped matching. Only a change carries it, and it is a
	// magnitude rather than a billed count.
	MissedTokens int
}

// Changed reports a diagnosis that found the prompt itself had changed, which
// is the harness's to fix; the others say only that nothing was compared.
func (d *CacheDiagnosis) Changed() bool {
	return d != nil && strings.HasSuffix(d.Reason, "_changed")
}
