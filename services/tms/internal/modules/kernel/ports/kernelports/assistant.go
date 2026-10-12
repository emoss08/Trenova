package kernelports

// AssistantStreamEmitter receives the events of one turn as they happen. It is
// called from the turn's own goroutine, in order, and never after the turn
// returns.
type AssistantStreamEmitter func(event StreamEvent)

// ThreadMessagesPage is one page of a thread in reading order, with what a
// client needs to ask for the page above it and to say how long the
// conversation has become.
type TranscriptFile struct {
	FileName string
	Body     string
}
