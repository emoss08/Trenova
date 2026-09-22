package agent

import "github.com/shopspring/decimal"

// ScorecardWindow is how far back a scorecard looks.
type ScorecardWindow string

const (
	ScorecardWindow7d  = ScorecardWindow("Last7Days")
	ScorecardWindow30d = ScorecardWindow("Last30Days")
	ScorecardWindow90d = ScorecardWindow("Last90Days")
)

func (w ScorecardWindow) IsValid() bool {
	switch w {
	case ScorecardWindow7d, ScorecardWindow30d, ScorecardWindow90d:
		return true
	default:
		return false
	}
}

// Days is the window's length. A window that is not one of the three is
// treated as thirty days rather than as zero, because a scorecard covering no
// time reads as an agent that has done nothing.
func (w ScorecardWindow) Days() int {
	switch w {
	case ScorecardWindow7d:
		return 7
	case ScorecardWindow90d:
		return 90
	default:
		return 30
	}
}

// ToolOutcomeCount is one tool's record over the window.
type ToolOutcomeCount struct {
	ToolName  string `json:"toolName"`
	Approved  int    `json:"approved"`
	Modified  int    `json:"modified"`
	Rejected  int    `json:"rejected"`
	Executed  int    `json:"executed"`
	Failed    int    `json:"failed"`
	Pending   int    `json:"pending"`
	Automatic int    `json:"automatic"`
}

// Decided is every proposal somebody answered. It is the denominator of the
// approval rate, and it deliberately leaves out what is still pending: an
// agent is not doing badly because nobody has got to its proposals yet.
func (c ToolOutcomeCount) Decided() int {
	return c.Approved + c.Modified + c.Rejected
}

// ScorecardPoint is one day of the trend line.
type ScorecardPoint struct {
	Day       int64 `json:"day"`
	Runs      int   `json:"runs"`
	Proposals int   `json:"proposals"`
	Approved  int   `json:"approved"`
}

// ScorecardTotals is what the repository counts.
type ScorecardTotals struct {
	Runs         int
	RunsFailed   int
	Exceptions   int
	InputTokens  int
	OutputTokens int
	CostUSD      decimal.Decimal
	ByTool       []ToolOutcomeCount
	Trend        []ScorecardPoint
}

// Scorecard answers "is this agent worth keeping".
//
// It is assembled rather than stored: every figure is counted from the runs,
// the proposals and the usage records at read time, so there is no second
// copy to fall out of step with the ledger a person can audit.
type Scorecard struct {
	AgentDefinitionID string             `json:"agentDefinitionId"`
	Window            ScorecardWindow    `json:"window"`
	Since             int64              `json:"since"`
	Runs              int                `json:"runs"`
	RunsFailed        int                `json:"runsFailed"`
	Exceptions        int                `json:"exceptions"`
	Proposals         int                `json:"proposals"`
	Approved          int                `json:"approved"`
	Modified          int                `json:"modified"`
	Rejected          int                `json:"rejected"`
	Pending           int                `json:"pending"`
	AutoExecuted      int                `json:"autoExecuted"`
	Executed          int                `json:"executed"`
	ExecutionFailures int                `json:"executionFailures"`
	InputTokens       int                `json:"inputTokens"`
	OutputTokens      int                `json:"outputTokens"`
	CostUSD           decimal.Decimal    `json:"costUsd"`
	ByTool            []ToolOutcomeCount `json:"byTool"`
	Trend             []ScorecardPoint   `json:"trend"`
	// EstimatedMinutesSaved is an estimate and is labelled as one wherever
	// it is shown. See timesaved.go for what it is counting.
	EstimatedMinutesSaved int `json:"estimatedMinutesSaved"`
}

// ApprovalRate is the share of answered proposals that were approved
// unchanged, between 0 and 1. Nothing answered yet is no rate rather than
// zero: an agent whose first proposal is still waiting has not been rejected.
func (s *Scorecard) ApprovalRate() *float64 {
	decided := s.Approved + s.Modified + s.Rejected
	if decided == 0 {
		return nil
	}
	rate := float64(s.Approved) / float64(decided)

	return &rate
}

// NewScorecard folds the counted totals into the shape a reader gets.
func NewScorecard(
	definitionID string,
	window ScorecardWindow,
	since int64,
	totals *ScorecardTotals,
) *Scorecard {
	card := &Scorecard{
		AgentDefinitionID: definitionID,
		Window:            window,
		Since:             since,
		Runs:              totals.Runs,
		RunsFailed:        totals.RunsFailed,
		Exceptions:        totals.Exceptions,
		InputTokens:       totals.InputTokens,
		OutputTokens:      totals.OutputTokens,
		CostUSD:           totals.CostUSD,
		ByTool:            totals.ByTool,
		Trend:             totals.Trend,
	}
	if card.ByTool == nil {
		card.ByTool = []ToolOutcomeCount{}
	}
	if card.Trend == nil {
		card.Trend = []ScorecardPoint{}
	}

	for _, tool := range card.ByTool {
		card.Approved += tool.Approved
		card.Modified += tool.Modified
		card.Rejected += tool.Rejected
		card.Pending += tool.Pending
		card.AutoExecuted += tool.Automatic
		card.Executed += tool.Executed
		card.ExecutionFailures += tool.Failed
		// Only what actually went through is counted as time saved. A
		// proposal approved but refused by the world saved nobody anything,
		// and counting approvals would let a broken tool look productive.
		card.EstimatedMinutesSaved += tool.Executed * MinutesSavedFor(tool.ToolName)
	}
	card.Proposals = card.Approved + card.Modified + card.Rejected + card.Pending

	return card
}
