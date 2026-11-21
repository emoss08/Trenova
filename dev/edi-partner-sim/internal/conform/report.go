package conform

import (
	"fmt"
	"os"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/edi-partner-sim/internal/acknowledge"
	"github.com/emoss08/trenova/edi-partner-sim/internal/oracle"
	"github.com/emoss08/trenova/shared/jsonutils"
)

const (
	StatusRead     = "read"
	StatusSent     = "sent"
	StatusHeld     = "held"
	StatusDeclined = "declined"
	StatusRejected = "rejected"
	StatusFailed   = "failed"
	StatusSkipped  = "skipped"
	StatusNotSent  = "not sent"
)

const (
	SourceProject  = "project"
	SourceLading   = "lading"
	SourceOracle   = "oracle"
	SourceEnvelope = "envelope"
	SourceAck      = "ack"
	SourceHarness  = "harness"
)

type Report struct {
	Seed     uint64           `json:"seed"`
	Count    uint64           `json:"count"`
	Workers  int              `json:"workers"`
	Partners []*PartnerReport `json:"partners"`
	Failures int              `json:"failures"`
}

type PartnerReport struct {
	Name     string        `json:"name"`
	Role     string        `json:"role"`
	Project  ProjectReport `json:"project"`
	Failures []Failure     `json:"failures,omitempty"`
	Cases    []*Case       `json:"cases"`
}

type ProjectReport struct {
	Errors      int      `json:"errors"`
	Warnings    int      `json:"warnings"`
	Diagnostics []string `json:"diagnostics,omitempty"`
	Tests       int      `json:"tests"`
	FailedTests []string `json:"failedTests,omitempty"`
	Ready       bool     `json:"ready"`
}

type Case struct {
	Partner   string    `json:"partner"`
	Seed      uint64    `json:"seed"`
	Variation string    `json:"variation"`
	Title     string    `json:"title"`
	Steps     []*Step   `json:"steps"`
	Failures  []Failure `json:"failures,omitempty"`
}

type Step struct {
	Name      string            `json:"name"`
	Text      string            `json:"text"`
	Direction string            `json:"direction"`
	Set       string            `json:"set"`
	Map       string            `json:"map,omitempty"`
	Status    string            `json:"status"`
	Note      string            `json:"note,omitempty"`
	Document  jsonutils.RawJSON `json:"document,omitempty"`
	X12       string            `json:"x12,omitempty"`
	Lading    []LadingFinding   `json:"lading,omitempty"`
	Oracle    []oracle.Finding  `json:"oracle,omitempty"`
	Ack       *Ack              `json:"ack,omitempty"`
}

type LadingFinding struct {
	Kind     string `json:"kind"`
	Severity string `json:"severity,omitempty"`
	Code     string `json:"code,omitempty"`
	Rule     string `json:"rule,omitempty"`
	Path     string `json:"path,omitempty"`
	Message  string `json:"message"`
}

type Ack struct {
	Set      string    `json:"set"`
	Code     string    `json:"code"`
	Expected string    `json:"expected"`
	Errors   int       `json:"errors"`
	Warnings int       `json:"warnings"`
	Notes    []AckNote `json:"notes,omitempty"`
}

type AckNote struct {
	Segment  string           `json:"segment"`
	Position int              `json:"position"`
	Loop     string           `json:"loop,omitempty"`
	Code     string           `json:"code"`
	Elements []AckElementNote `json:"elements,omitempty"`
}

type AckElementNote struct {
	Position  int    `json:"position"`
	Component int    `json:"component,omitempty"`
	Code      string `json:"code"`
	Copy      string `json:"copy,omitempty"`
}

func ackNotesOf(notes []acknowledge.SegmentNote) []AckNote {
	out := make([]AckNote, 0, len(notes))
	for _, note := range notes {
		written := AckNote{
			Segment:  note.ID,
			Position: note.Position,
			Loop:     note.Loop,
			Code:     note.Code,
		}
		for _, element := range note.Elements {
			written.Elements = append(written.Elements, AckElementNote{
				Position:  element.Position,
				Component: element.Component,
				Code:      element.Code,
				Copy:      element.Copy,
			})
		}
		out = append(out, written)
	}
	return out
}

type Failure struct {
	Step    string `json:"step"`
	Set     string `json:"set,omitempty"`
	Source  string `json:"source"`
	Rule    string `json:"rule"`
	X12Code string `json:"x12Code,omitempty"`
	Segment string `json:"segment,omitempty"`
	Message string `json:"message"`
	Replay  string `json:"replay"`
}

func (r *Report) Passed() bool {
	return r.Failures == 0
}

func (r *Report) tally() {
	r.Failures = 0
	for _, p := range r.Partners {
		r.Failures += len(p.Failures)
		for _, c := range p.Cases {
			r.Failures += len(c.Failures)
		}
	}
}

func (r *Report) WriteJSON(path string) error {
	data, err := sonic.ConfigStd.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("encode the report: %w", err)
	}
	if err = os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func (c *Case) fail(step *Step, f *Failure) {
	f.Step, f.Set = step.Name, step.Set
	f.Replay = replayOf(c.Partner, c.Seed)
	step.Status = StatusFailed
	c.Failures = append(c.Failures, *f)
}

func (c *Case) step(name, text, direction, set string) *Step {
	s := &Step{Name: name, Text: text, Direction: direction, Set: set}
	c.Steps = append(c.Steps, s)
	return s
}

func replayOf(name string, seed uint64) string {
	return fmt.Sprintf("%s:%d", name, seed)
}

func segmentOf(f *oracle.Finding) string {
	if f.SegmentID == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString(f.SegmentID)
	if f.ElementPos > 0 {
		fmt.Fprintf(&b, "-%02d", f.ElementPos)
	}
	if f.ComponentPos > 0 {
		fmt.Fprintf(&b, "-%d", f.ComponentPos)
	}
	if f.SegmentIndex > 0 {
		fmt.Fprintf(&b, " (segment %d", f.SegmentIndex)
		if f.LoopID != "" {
			b.WriteString(", loop " + f.LoopID)
		}
		b.WriteString(")")
	}
	return b.String()
}
