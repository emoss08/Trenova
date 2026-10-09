package deskbench

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	liveArgsLimit    = 400
	liveSummaryLimit = 200
	liveEventLimit   = 300
	liveTimeLayout   = "15:04:05"
)

type LiveFeed struct {
	mu      sync.Mutex
	writers []io.Writer
	files   []*os.File
}

func OpenLiveFeed(paths ...string) (*LiveFeed, error) {
	feed := &LiveFeed{}
	for _, path := range paths {
		if path == "" {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), dirMode); err != nil {
			feed.Close()
			return nil, fmt.Errorf("create %s: %w", filepath.Dir(path), err)
		}
		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fileMode)
		if err != nil {
			feed.Close()
			return nil, fmt.Errorf("open the live feed %s: %w", path, err)
		}
		feed.files = append(feed.files, file)
		feed.writers = append(feed.writers, file)
	}

	return feed, nil
}

func (f *LiveFeed) Close() {
	if f == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, file := range f.files {
		_ = file.Close()
	}
	f.files = nil
	f.writers = nil
}

func (f *LiveFeed) Line(label, format string, args ...any) {
	if f == nil {
		return
	}
	f.write(label, fmt.Sprintf(format, args...))
}

func (f *LiveFeed) write(label, text string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	stamp := time.Now().Format(liveTimeLayout)
	var b strings.Builder
	for line := range strings.SplitSeq(strings.TrimRight(text, "\n"), "\n") {
		b.WriteString(stamp)
		b.WriteString(" [")
		b.WriteString(label)
		b.WriteString("] ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	encoded := []byte(b.String())
	for _, writer := range f.writers {
		_, _ = writer.Write(encoded)
	}
}

type liveTurn struct {
	feed    *LiveFeed
	label   string
	mode    string
	pending strings.Builder
}

func (f *LiveFeed) turn(label string) *liveTurn {
	if f == nil {
		return nil
	}

	return &liveTurn{feed: f, label: label}
}

const (
	liveModeText     = "reply"
	liveModeThinking = "thinking"
)

func (t *liveTurn) line(format string, args ...any) {
	if t == nil {
		return
	}
	t.flush()
	t.feed.Line(t.label, format, args...)
}

func (t *liveTurn) stream(mode, chunk string) {
	if t == nil || chunk == "" {
		return
	}
	if t.mode != mode {
		t.flush()
		t.mode = mode
		t.feed.Line(t.label, "%s:", mode)
	}

	t.pending.WriteString(chunk)
	text := t.pending.String()
	cut := strings.LastIndexByte(text, '\n')
	if cut < 0 {
		return
	}
	t.feed.write(t.label, "  "+strings.ReplaceAll(text[:cut], "\n", "\n  "))
	t.pending.Reset()
	t.pending.WriteString(text[cut+1:])
}

func (t *liveTurn) flush() {
	if t == nil {
		return
	}
	if rest := strings.TrimSpace(t.pending.String()); rest != "" {
		t.feed.write(t.label, "  "+strings.ReplaceAll(rest, "\n", "\n  "))
	}
	t.pending.Reset()
	t.mode = ""
}

func (t *liveTurn) frame(frame serviceports.TurnStreamFrame) {
	if t == nil {
		return
	}

	switch frame.Event {
	case serviceports.AssistantEventDelta, serviceports.AssistantEventDelegateDelta:
		t.stream(liveModeText, frameText(frame.Data))
	case serviceports.AssistantEventReasoning, serviceports.AssistantEventDelegateReasoning:
		t.stream(liveModeThinking, frameText(frame.Data))
	case serviceports.AssistantEventToolStarted:
		var started serviceports.AssistantToolStartedEvent
		if sonic.Unmarshal(frame.Data, &started) == nil {
			t.line("-> %s%s %s", started.Name, delegateTag(started.DelegateCallID),
				stringutils.Ellipsize(compactJSON(started.Arguments, liveArgsLimit), liveArgsLimit))
		}
	case serviceports.AssistantEventToolFinished:
		var finished serviceports.AssistantToolFinishedEvent
		if sonic.Unmarshal(frame.Data, &finished) == nil {
			record := ToolCallRecord{Verdict: finished.Verdict, Failed: finished.Failed, Finished: true}
			detail := finished.Summary
			if record.Refused() {
				detail = firstLine(finished.Content)
			}
			t.line("<- %s%s %s · %s", finished.Name, delegateTag(finished.DelegateCallID),
				verdictLabel(&record), stringutils.Ellipsize(detail, liveSummaryLimit))
		}
	case serviceports.AssistantEventMessage, serviceports.AssistantEventDone,
		serviceports.AssistantEventAccepted, serviceports.AssistantEventTurn,
		serviceports.AssistantEventThread, serviceports.AssistantEventContext,
		serviceports.AssistantEventArtifact:
		t.flush()
	default:
		t.line("event %s %s", frame.Event, stringutils.Ellipsize(string(frame.Data), liveEventLimit))
	}
}

func frameText(data []byte) string {
	var payload struct {
		Text string `json:"text"`
	}
	if sonic.Unmarshal(data, &payload) != nil {
		return ""
	}

	return payload.Text
}

func delegateTag(delegateCallID string) string {
	if delegateCallID == "" {
		return ""
	}

	return " (delegate)"
}
