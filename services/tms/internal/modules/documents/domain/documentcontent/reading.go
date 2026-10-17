package documentcontent

const (
	structuredDiagnosticsKey     = "aiDiagnostics"
	diagnosticsAcceptanceKey     = "acceptanceStatus"
	diagnosticsAcceptancePending = "pending"
	// ModelReadingWindowSeconds bounds how long a model's reading of a
	// document is waited for once its text is in. A reading still marked
	// pending past it was lost along the way, and the text and the rule-based
	// findings are what there is.
	ModelReadingWindowSeconds int64 = 15 * 60
)

// ModelReadingPending reports a model's reading of the document under way: its
// text is in and its findings are the rule-based ones until the model's land.
func (c *Content) ModelReadingPending(now int64) bool {
	diagnostics, _ := c.StructuredData[structuredDiagnosticsKey].(map[string]any)
	status, _ := diagnostics[diagnosticsAcceptanceKey].(string)
	if status != diagnosticsAcceptancePending {
		return false
	}

	return c.LastExtractedAt == nil || now-*c.LastExtractedAt < ModelReadingWindowSeconds
}

// StillBeingRead reports a document whose reading has not finished: its text
// is not in, or a model's reading of it is under way. A document that could
// not be read is done being read.
func (c *Content) StillBeingRead(now int64) bool {
	if !c.Status.Settled() {
		return true
	}

	return c.Status != StatusFailed && c.ModelReadingPending(now)
}
