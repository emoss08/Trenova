package benchlive

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/deskbench/benchrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	liveArgsLimit    = 400
	liveSummaryLimit = 200
	liveEventLimit   = 300
	liveTimeLayout   = "15:04:05"
)

type Feed struct {
	mu      sync.Mutex
	writers []io.Writer
	files   []*os.File
}

func Open(paths ...string) (*Feed, error) {
	feed := &Feed{}
	for _, path := range paths {
		if path == "" {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), benchrecord.DirMode); err != nil {
			feed.Close()
			return nil, fmt.Errorf("create %s: %w", filepath.Dir(path), err)
		}
		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, benchrecord.FileMode)
		if err != nil {
			feed.Close()
			return nil, fmt.Errorf("open the live feed %s: %w", path, err)
		}
		feed.files = append(feed.files, file)
		feed.writers = append(feed.writers, file)
	}

	return feed, nil
}

func (f *Feed) Close() {
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

func (f *Feed) Linef(label, format string, args ...any) {
	if f == nil {
		return
	}
	f.write(label, fmt.Sprintf(format, args...))
}

func (f *Feed) write(label, text string) {
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

type Turn struct {
	feed    *Feed
	label   string
	mode    string
	pending strings.Builder
}

func (f *Feed) Turn(label string) *Turn {
	if f == nil {
		return nil
	}

	return &Turn{feed: f, label: label}
}

const (
	liveModeText     = "reply"
	liveModeThinking = "thinking"
)

func (t *Turn) Linef(format string, args ...any) {
	if t == nil {
		return
	}
	t.flush()
	t.feed.Linef(t.label, format, args...)
}

func (t *Turn) stream(mode, chunk string) {
	if t == nil || chunk == "" {
		return
	}
	if t.mode != mode {
		t.flush()
		t.mode = mode
		t.feed.Linef(t.label, "%s:", mode)
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

func (t *Turn) flush() {
	if t == nil {
		return
	}
	if rest := strings.TrimSpace(t.pending.String()); rest != "" {
		t.feed.write(t.label, "  "+strings.ReplaceAll(rest, "\n", "\n  "))
	}
	t.pending.Reset()
	t.mode = ""
}

func (t *Turn) Frame(frame assistantstreamports.TurnStreamFrame) {
	if t == nil {
		return
	}

	switch frame.Event {
	case assistantstreamports.AssistantEventDelta, assistantstreamports.AssistantEventDelegateDelta:
		t.stream(liveModeText, frameText(frame.Data))
	case assistantstreamports.AssistantEventReasoning,
		assistantstreamports.AssistantEventDelegateReasoning:
		t.stream(liveModeThinking, frameText(frame.Data))
	case assistantstreamports.AssistantEventToolStarted:
		var started assistantstreamports.AssistantToolStartedEvent
		if sonic.Unmarshal(frame.Data, &started) == nil {
			t.Linef(
				"-> %s%s %s",
				started.Name,
				delegateTag(started.DelegateCallID),
				stringutils.Ellipsize(
					benchrecord.CompactJSON(started.Arguments, liveArgsLimit),
					liveArgsLimit,
				),
			)
		}
	case assistantstreamports.AssistantEventToolFinished:
		var finished assistantstreamports.AssistantToolFinishedEvent
		if sonic.Unmarshal(frame.Data, &finished) == nil {
			record := benchrecord.ToolCallRecord{
				Verdict:  finished.Verdict,
				Failed:   finished.Failed,
				Finished: true,
			}
			detail := finished.Summary
			if record.Refused() {
				detail = benchrecord.FirstLine(finished.Content)
			}
			t.Linef("<- %s%s %s · %s", finished.Name, delegateTag(finished.DelegateCallID),
				benchrecord.VerdictLabel(&record), stringutils.Ellipsize(detail, liveSummaryLimit))
		}
	case assistantstreamports.AssistantEventMessage, assistantstreamports.AssistantEventDone,
		assistantstreamports.AssistantEventAccepted, assistantstreamports.AssistantEventTurn,
		assistantstreamports.AssistantEventThread, assistantstreamports.AssistantEventContext,
		assistantstreamports.AssistantEventArtifact:
		t.flush()
	default:
		t.Linef(
			"event %s %s",
			frame.Event,
			stringutils.Ellipsize(string(frame.Data), liveEventLimit),
		)
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

func AnnounceTurnEnd(live *Turn, record *benchrecord.TurnRecord) {
	live.flush()
	live.Linef(
		"== %s · %s · %d tool calls · %s ==",
		record.Status,
		record.Duration().
			Round(100*time.Millisecond),
		len(record.Tools),
		benchrecord.UsageLine(&record.Usage),
	)
	for _, proposal := range record.Proposals {
		live.Linef(
			"proposal %s %s (%s)",
			proposal.Tool,
			stringutils.Ellipsize(
				benchrecord.CompactJSON(proposal.Arguments, liveArgsLimit),
				liveArgsLimit,
			),
			proposal.Status,
		)
	}
	if record.Failure != "" {
		live.Linef("failure: %s", record.Failure)
	}
}
