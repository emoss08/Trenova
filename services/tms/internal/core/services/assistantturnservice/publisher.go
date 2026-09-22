package assistantturnservice

import (
	"context"
	"strings"
	"time"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"go.uber.org/zap"
)

const (
	// flushInterval is how long text may accumulate before it is published.
	//
	// A model emits twenty to sixty deltas a second, and one redis round trip
	// each would make the stream cost more than the reply. Forty milliseconds
	// is below what a reader perceives as delay and cuts the traffic by an
	// order of magnitude.
	flushInterval = 40 * time.Millisecond
	// flushBytes publishes early when the model is producing faster than the
	// interval, so a burst does not arrive as one wall of text.
	flushBytes = 2048
)

// publisher writes a turn's events to its stream, coalescing the text.
//
// Everything that is not text is published as it happens: a tool starting, a
// proposal, a refusal. Those are few, and each one is a thing a reader reacts
// to rather than watches accumulate. Text is the only high-frequency event and
// the only one that concatenates cleanly, which is why it is the only one
// buffered.
//
// Nothing here is allowed to fail the turn. A stream that cannot be written is
// a reader who cannot reattach; the reply itself still arrives on the
// connection that asked for it and is still saved. Failures are logged once
// and the turn carries on.
type publisher struct {
	stream serviceports.TurnStreamPublisher
	ref    serviceports.TurnStreamRef
	logger *zap.Logger

	// buffered is the text waiting to go out, and kind is which of the two
	// text events it belongs to — a reply and the thinking behind it must not
	// be concatenated into each other.
	buffered strings.Builder
	kind     string
	lastSent time.Time
	// failed stops a broken stream from logging once per delta.
	failed bool
}

func newPublisher(
	stream serviceports.TurnStreamPublisher,
	ref serviceports.TurnStreamRef,
	logger *zap.Logger,
) *publisher {
	return &publisher{stream: stream, ref: ref, logger: logger, lastSent: time.Now()}
}

// emit takes one event from the turn. It is called from the turn's own
// goroutine, in order, which is what lets the buffer be a plain field.
func (p *publisher) emit(ctx context.Context, event serviceports.StreamEvent) {
	text, isText := textOf(event)
	if !isText {
		p.flush(ctx)
		p.publish(ctx, event)

		return
	}

	// A change of voice flushes: the assistant's thinking and its reply are
	// two streams of text that happen to arrive interleaved.
	if p.kind != "" && p.kind != event.Event {
		p.flush(ctx)
	}
	p.kind = event.Event
	p.buffered.WriteString(text)

	if p.buffered.Len() >= flushBytes || time.Since(p.lastSent) >= flushInterval {
		p.flush(ctx)
	}
}

// close flushes what is buffered and publishes the frame that ends the turn.
// A stream with no terminal frame is how a relay learns its writer died, so
// this is the one publish whose absence means something.
func (p *publisher) close(ctx context.Context, event serviceports.StreamEvent) {
	p.flush(ctx)

	if err := p.stream.Close(ctx, p.ref, event); err != nil {
		p.report("could not close a turn's event stream", err)
	}
}

func (p *publisher) flush(ctx context.Context) {
	if p.buffered.Len() == 0 {
		return
	}

	text := p.buffered.String()
	kind := p.kind
	p.buffered.Reset()
	p.kind = ""
	p.lastSent = time.Now()

	p.publish(ctx, serviceports.StreamEvent{Event: kind, Data: textEvent(kind, text)})
}

func (p *publisher) publish(ctx context.Context, event serviceports.StreamEvent) {
	if err := p.stream.Publish(ctx, p.ref, event); err != nil {
		p.report("could not publish a turn event", err)
	}
}

func (p *publisher) report(message string, err error) {
	if p.failed {
		return
	}
	p.failed = true

	p.logger.Error(message,
		zap.String("turn", p.ref.TurnID.String()),
		zap.Error(err),
	)
}

// textOf reports the text an event carries, for the two events that carry
// only text and can therefore be joined.
func textOf(event serviceports.StreamEvent) (string, bool) {
	switch event.Event {
	case serviceports.AssistantEventDelta:
		if data, ok := event.Data.(serviceports.AssistantDeltaEvent); ok {
			return data.Text, true
		}
	case serviceports.AssistantEventReasoning:
		if data, ok := event.Data.(serviceports.AssistantReasoningEvent); ok {
			return data.Text, true
		}
	}

	return "", false
}

func textEvent(kind, text string) any {
	if kind == serviceports.AssistantEventReasoning {
		return serviceports.AssistantReasoningEvent{Text: text}
	}

	return serviceports.AssistantDeltaEvent{Text: text}
}
