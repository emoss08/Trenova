package play

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/edi-partner-sim/internal/oracle"
	"github.com/emoss08/trenova/edi-partner-sim/internal/provision"
	"github.com/emoss08/trenova/edi-partner-sim/internal/runlog"
)

const (
	StatusPassed  = "passed"
	StatusFailed  = "failed"
	StatusSkipped = "skipped"
)

const (
	SourceHarness = "harness"
	SourceSim     = "simulator"
	SourceOracle  = "oracle"
	SourceTrenova = "trenova"
	SourceExpect  = "expectation"
	SourceSetup   = "provisioning"
)

type Report struct {
	Run       uint64           `json:"run"`
	API       string           `json:"api"`
	Sim       string           `json:"sim"`
	StartedAt time.Time        `json:"startedAt"`
	Took      Duration         `json:"took"`
	Partners  []*PartnerReport `json:"partners"`
	Failures  int              `json:"failures"`
}

type PartnerReport struct {
	Name      string            `json:"name"`
	Role      string            `json:"role"`
	Provision *provision.Result `json:"provision"`
	Scenarios []*ScenarioReport `json:"scenarios"`
	Failures  []Failure         `json:"failures,omitempty"`
}

type ScenarioReport struct {
	Name          string                `json:"name"`
	File          string                `json:"file"`
	Slug          string                `json:"slug"`
	Seed          uint64                `json:"seed"`
	Status        string                `json:"status"`
	StartedAt     time.Time             `json:"startedAt"`
	Took          Duration              `json:"took"`
	Steps         []*StepReport         `json:"steps"`
	Conversations []*ConversationReport `json:"conversations"`
	Failures      []Failure             `json:"failures,omitempty"`
	Replay        string                `json:"replay"`
}

type StepReport struct {
	Index   int            `json:"index"`
	Kind    string         `json:"kind"`
	Text    string         `json:"text"`
	Status  string         `json:"status"`
	Note    string         `json:"note,omitempty"`
	Why     string         `json:"why,omitempty"`
	At      Duration       `json:"at"`
	Took    Duration       `json:"took"`
	Ended   time.Time      `json:"ended"`
	Keys    []string       `json:"keys,omitempty"`
	Matches []runlog.Match `json:"matches,omitempty"`
	Events  []runlog.Event `json:"events,omitempty"`
	Gap     string         `json:"gap,omitempty"`
}

type ConversationReport struct {
	Name       string      `json:"name"`
	Key        string      `json:"key"`
	Seed       uint64      `json:"seed,omitempty"`
	InstanceID string      `json:"flowInstanceId,omitempty"`
	TransferID string      `json:"transferId,omitempty"`
	ShipmentID string      `json:"shipmentId,omitempty"`
	Milestones Milestones  `json:"milestones"`
	Wire       []WireEntry `json:"wire"`
	States     []Snapshot  `json:"states,omitempty"`
}

type Milestones struct {
	Ack     *Duration `json:"ack,omitempty"`
	Answer  *Duration `json:"answer,omitempty"`
	Status  *Duration `json:"status,omitempty"`
	Invoice *Duration `json:"invoice,omitempty"`
}

type WireEntry struct {
	MessageID string           `json:"messageId"`
	From      string           `json:"from"`
	Purpose   string           `json:"purpose"`
	Sets      []string         `json:"sets"`
	State     string           `json:"state"`
	At        time.Time        `json:"at"`
	Variation string           `json:"variation,omitempty"`
	Accepted  *bool            `json:"accepted,omitempty"`
	Ack       string           `json:"ack,omitempty"`
	AckCode   string           `json:"ackCode,omitempty"`
	Findings  []oracle.Finding `json:"findings,omitempty"`
	Error     string           `json:"error,omitempty"`
	X12       string           `json:"x12"`
}

type Snapshot struct {
	Step       int    `json:"step"`
	At         string `json:"at"`
	FlowStatus string `json:"flowStatus,omitempty"`
	FlowWhy    string `json:"flowWhy,omitempty"`
	Shipment   string `json:"shipment,omitempty"`
	Status     string `json:"shipmentStatus,omitempty"`
	Stops      int    `json:"stops,omitempty"`
	Total      string `json:"total,omitempty"`
	Error      string `json:"error,omitempty"`
}

type Failure struct {
	Step    int    `json:"step"`
	Source  string `json:"source"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
	Replay  string `json:"replay,omitempty"`
}

type Duration time.Duration

func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%.3f", time.Duration(d).Seconds())), nil
}

func (d *Duration) UnmarshalJSON(data []byte) error {
	var seconds float64
	if err := sonic.Unmarshal(data, &seconds); err != nil {
		return err
	}
	*d = Duration(seconds * float64(time.Second))
	return nil
}

func (d Duration) String() string {
	return time.Duration(d).Round(time.Millisecond).String()
}

func durationPtr(d time.Duration) *Duration {
	value := Duration(d)
	return &value
}

func (r *Report) Passed() bool {
	return r.Failures == 0
}

func (r *Report) tally() {
	r.Failures = 0
	for _, p := range r.Partners {
		r.Failures += len(p.Failures)
		for _, s := range p.Scenarios {
			r.Failures += len(s.Failures)
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

func (r *Report) WriteSummary(w io.Writer) {
	fmt.Fprintf(w, "run %d against %s (simulator %s), %s\n", r.Run, r.API, r.Sim, r.Took)
	for _, p := range r.Partners {
		writePartner(w, p)
	}
	if r.Passed() {
		fmt.Fprintln(w, "\nPASS")
		return
	}
	fmt.Fprintf(w, "\nFAIL: %d failures\n", r.Failures)
}

func writePartner(w io.Writer, p *PartnerReport) {
	passed, failed := 0, 0
	for _, s := range p.Scenarios {
		switch s.Status {
		case StatusPassed:
			passed++
		case StatusFailed:
			failed++
		}
	}
	verdict := "pass"
	if failed > 0 || len(p.Failures) > 0 {
		verdict = "FAIL"
	}
	fmt.Fprintf(w, "\n%s (%s): %s\n", p.Name, p.Role, verdict)
	if setup := p.Provision; setup != nil {
		fmt.Fprintf(w, "  provisioned %s over %s", setup.Code, firstNonEmpty(setup.Transport, "?"))
		if setup.VersionNumber > 0 {
			fmt.Fprintf(w, ", Lading version %d", setup.VersionNumber)
		}
		if setup.Unchanged {
			fmt.Fprint(w, " (unchanged)")
		}
		fmt.Fprintln(w)
		for _, test := range setup.FailingTests {
			fmt.Fprintf(w, "    failing test: %s\n", test)
		}
	}
	for _, f := range p.Failures {
		fmt.Fprintf(w, "  %s: %s\n", f.Source, f.Message)
	}
	if len(p.Scenarios) > 0 {
		fmt.Fprintf(w, "  scenarios: %d run, %d passed, %d failed\n",
			passed+failed, passed, failed)
	}
	for _, s := range p.Scenarios {
		mark := "ok  "
		switch s.Status {
		case StatusFailed:
			mark = "FAIL"
		case StatusSkipped:
			mark = "skip"
		}
		fmt.Fprintf(w, "  %s %-40s %8s  seed %d\n", mark, s.Slug, s.Took, s.Seed)
		for _, f := range s.Failures {
			text := ""
			if f.Step >= 0 && f.Step < len(s.Steps) {
				text = s.Steps[f.Step].Text
			}
			fmt.Fprintf(w, "       step %d %s\n       %s %s: %s\n", f.Step+1, text, f.Source,
				f.Rule, f.Message)
		}
		if s.Status == StatusFailed {
			fmt.Fprintf(w, "       replay: %s\n", s.Replay)
		}
	}
}

func (s *ScenarioReport) Write(w io.Writer) {
	fmt.Fprintf(w, "\n== %s (%s), seed %d: %s\n", s.Name, s.Slug, s.Seed, s.Status)
	for _, step := range s.Steps {
		fmt.Fprintf(w, "  %2d %-7s %-60s %-7s %8s", step.Index+1, step.Kind,
			truncate(step.Text, 60), step.Status, step.Took)
		if step.Note != "" {
			fmt.Fprintf(w, "  %s", step.Note)
		}
		fmt.Fprintln(w)
	}
	for _, c := range s.Conversations {
		fmt.Fprintf(w, "  -- %s key %s", c.Name, firstNonEmpty(c.Key, "(none)"))
		if c.InstanceID != "" {
			fmt.Fprintf(w, " flow %s", c.InstanceID)
		}
		fmt.Fprintln(w)
		for _, entry := range c.Wire {
			accepted := ""
			if entry.Accepted != nil && !*entry.Accepted {
				accepted = " REJECTED BY ORACLE"
			}
			fmt.Fprintf(w, "     %s %-8s %-8s %-10s %-9s ack %s %s%s\n",
				entry.At.Format("15:04:05.000"), entry.From, strings.Join(entry.Sets, ","),
				entry.Purpose, entry.State, firstNonEmpty(entry.Ack, "-"), entry.AckCode, accepted)
			for index := range entry.Findings {
				f := &entry.Findings[index]
				fmt.Fprintf(w, "        oracle %s %s %s: %s\n", f.Severity, f.SegmentID, f.Rule,
					f.Message)
			}
		}
	}
}

func truncate(text string, width int) string {
	if len(text) <= width {
		return text
	}
	return text[:width-1] + "…"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
