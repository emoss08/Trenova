package lading

import (
	"github.com/emoss08/trenova/pkg/wasmmodule"
	"github.com/emoss08/trenova/pkg/x12"
	"github.com/emoss08/trenova/shared/jsonutils"
)

const (
	OutcomeReject  = "reject"
	OutcomeDecline = "decline"
	OutcomeHold    = "hold"
	OutcomeWarn    = "warn"
)

const (
	TransactionAccepted = "accepted"
	TransactionRejected = "rejected"
)

// RunInput is what a run reads: exactly one of its fields is set.
type RunInput struct {
	X12      *string           `json:"x12,omitempty"`
	CSV      *string           `json:"csv,omitempty"`
	JSON     jsonutils.RawJSON `json:"json,omitempty"`
	Document jsonutils.RawJSON `json:"document,omitempty"`
}

func X12Input(text string) RunInput {
	return RunInput{X12: &text}
}

func CSVInput(text string) RunInput {
	return RunInput{CSV: &text}
}

func JSONInput(document jsonutils.RawJSON) RunInput {
	return RunInput{JSON: document}
}

func DocumentInput(document jsonutils.RawJSON) RunInput {
	return RunInput{Document: document}
}

type Found struct {
	ID    string `json:"id"`
	Label string `json:"label,omitempty"`
	Zone  string `json:"zone,omitempty"`
}

type LookupAnswer struct {
	Kind  string `json:"kind"`
	Code  string `json:"code"`
	Found *Found `json:"found,omitempty"`
}

type Control struct {
	Interchange uint64 `json:"interchange"`
	Group       uint64 `json:"group"`
	Transaction uint64 `json:"transaction"`
}

type RunRequest struct {
	Map      string                       `json:"map"`
	Input    RunInput                     `json:"input"`
	Settings map[string]jsonutils.RawJSON `json:"settings,omitempty"`
	Lookups  []LookupAnswer               `json:"lookups,omitempty"`
	Now      string                       `json:"now,omitempty"`
	Control  Control                      `json:"control"`
	Rules    bool                         `json:"rules"`
}

type Source struct {
	Path     string `json:"path"`
	Segment  string `json:"segment,omitempty"`
	Position int    `json:"position,omitempty"`
	Element  int    `json:"element,omitempty"`
}

type Lineage struct {
	Path    string   `json:"path"`
	Sources []Source `json:"sources"`
	Steps   []string `json:"steps"`
	Value   string   `json:"value"`
}

type Outcome struct {
	Kind    string  `json:"kind"`
	Message string  `json:"message"`
	Rule    string  `json:"rule,omitempty"`
	Path    string  `json:"path,omitempty"`
	Element *Source `json:"element,omitempty"`
	Code    string  `json:"code,omitempty"`
}

type Fix struct {
	Rule string            `json:"rule"`
	Path string            `json:"path"`
	Old  jsonutils.RawJSON `json:"old"`
	New  jsonutils.RawJSON `json:"new"`
}

type Need struct {
	Kind string `json:"kind"`
	Code string `json:"code"`
}

type Unmapped struct {
	Kind string            `json:"kind"`
	Code string            `json:"code"`
	Hint jsonutils.RawJSON `json:"hint,omitempty"`
	Path string            `json:"path"`
}

type RunFinding struct {
	Severity string             `json:"severity"`
	Code     string             `json:"code"`
	Message  wasmmodule.Message `json:"message"`
	Path     string             `json:"path,omitempty"`
}

type TransactionResult struct {
	Index    int               `json:"index"`
	Control  string            `json:"control"`
	Status   string            `json:"status"`
	Findings []x12.Finding     `json:"findings"`
	Document jsonutils.RawJSON `json:"document"`
	Outcomes []Outcome         `json:"outcomes"`
	Fixes    []Fix             `json:"fixes"`
	Lineage  []Lineage         `json:"lineage"`
	Unread   []string          `json:"unread"`
}

func (t *TransactionResult) Rejected() bool {
	if t.Status == TransactionRejected {
		return true
	}
	for index := range t.Outcomes {
		if t.Outcomes[index].Kind == OutcomeReject {
			return true
		}
	}
	return false
}

type WriteResult struct {
	Text        string            `json:"text"`
	Transaction string            `json:"transaction"`
	Document    jsonutils.RawJSON `json:"document"`
	Findings    []RunFinding      `json:"findings"`
	Validation  []x12.Finding     `json:"validation"`
	Outcomes    []Outcome         `json:"outcomes"`
	Fixes       []Fix             `json:"fixes"`
	Lineage     []Lineage         `json:"lineage"`
}

// Blocked is whether the written message may not be sent: a finding or a validation error, or a
// rule that rejected or held the document.
func (w *WriteResult) Blocked() bool {
	for index := range w.Findings {
		if w.Findings[index].Severity == SeverityError {
			return true
		}
	}
	for index := range w.Validation {
		if w.Validation[index].IsError() {
			return true
		}
	}
	for index := range w.Outcomes {
		switch w.Outcomes[index].Kind {
		case OutcomeReject, OutcomeHold, OutcomeDecline:
			return true
		}
	}
	return false
}

type RunResult struct {
	Transactions []TransactionResult `json:"transactions"`
	Output       *WriteResult        `json:"output"`
	Needs        []Need              `json:"needs"`
	Unmapped     []Unmapped          `json:"unmapped"`
	Error        *wasmmodule.Message `json:"error"`
	Steps        uint64              `json:"steps"`
}
