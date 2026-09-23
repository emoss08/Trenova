package temporaltype

// The contract between a run's workflow, which hosts its Workflow Stream, and
// whoever reads that stream. Both sides live in different processes and
// different packages, so the names they agree on live here.
const (
	// StreamEventsTopic carries everything a reader of a run sees, in order.
	StreamEventsTopic = "events"

	// SignalStreamDrained tells a run's workflow that its reader has the last
	// event. The workflow waits for it before it closes, because a stream is
	// read by polling the workflow, and a workflow that has closed cannot be
	// polled for the events it published last.
	SignalStreamDrained = "stream-drained"
)

// StreamItem is one event on a run's stream, as the reader receives it.
type StreamItem struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}
