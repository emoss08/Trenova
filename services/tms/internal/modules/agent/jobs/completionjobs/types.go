// Package completionjobs runs the model calls a person waits on outside a
// conversation: composing a table's filters, writing or explaining a formula,
// rewriting today's briefing, and testing a provider. Each runs as a short
// workflow on the chat queue, and the request that asked waits for it.
package completionjobs

const (
	// callAttempts bounds Temporal's retries of a one-shot model call inside
	// the time the person is waiting.
	callAttempts = 3

	// writeAttempts bounds retries of a briefing write. Writing a day again
	// replaces it, so a retry is safe; two is enough for a lost worker.
	writeAttempts = 2
)
