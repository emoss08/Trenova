package play

import (
	"fmt"
	"os"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/edi-partner-sim/internal/oracle"
	"github.com/emoss08/trenova/edi-partner-sim/internal/provision"
	"github.com/emoss08/trenova/edi-partner-sim/internal/runlog"
	"github.com/emoss08/trenova/edi-partner-sim/internal/scenario"
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
	Scale     float64          `json:"deadlineScale"`
	Waiting   Waiting          `json:"waiting"`
}

type Waiting struct {
	Expectations Duration `json:"expectations"`
	Expected     int      `json:"expected"`
	Actions      Duration `json:"actions"`
	Acted        int      `json:"acted"`
	Failed       Duration `json:"failed"`
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
	r.Failures, r.Waiting = 0, Waiting{}
	for _, p := range r.Partners {
		r.Failures += len(p.Failures)
		for _, s := range p.Scenarios {
			r.Failures += len(s.Failures)
			for _, step := range s.Steps {
				r.Waiting.add(step)
			}
		}
	}
}

func (w *Waiting) add(step *StepReport) {
	switch step.Kind {
	case scenario.KindExpect:
		w.Expectations += step.Took
		w.Expected++
	case scenario.KindTrenova, scenario.KindAdvance:
		w.Actions += step.Took
		w.Acted++
	default:
		return
	}
	if step.Status == StatusFailed {
		w.Failed += step.Took
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
