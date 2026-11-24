package agentevalgate

import (
	"fmt"
	"math"
	"os"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentcapability"
	"gopkg.in/yaml.v3"
)

const (
	CapabilitySuitePath   = "evals/capabilities.yaml"
	CapabilityResultsPath = "evals/capabilities.results.json"
	CapabilityTop         = 3

	StatusLoaded          = "loaded"
	StatusAvailable       = "available"
	StatusNotPermitted    = "not_permitted"
	StatusNotInThisAgent  = "not_in_this_agent"
	ShapeProposal         = "proposal"
	ShapeRefusal          = "refusal"
	ShapeHandoff          = "handoff"
	AgentUniversal        = "universal"
	minimumCapabilityCase = 150
)

type CapabilitySuite struct {
	Cases []CapabilityCase `yaml:"cases"`
}

type CapabilityCase struct {
	Request  string         `yaml:"request"`
	Kind     string         `yaml:"kind"`
	Agent    string         `yaml:"agent"`
	Expect   string         `yaml:"expect"`
	Args     map[string]any `yaml:"args"`
	Shape    string         `yaml:"shape"`
	Includes []string       `yaml:"includes"`
	Chain    []string       `yaml:"chain"`
	Denied   []string       `yaml:"denied"`
	Status   string         `yaml:"status"`
}

func (c *CapabilityCase) AgentName() string {
	if c.Agent == "" {
		return AgentUniversal
	}

	return c.Agent
}

func (c *CapabilityCase) ExpectedStatus() string {
	if c.Status == "" {
		return StatusAvailable
	}

	return c.Status
}

func (c *CapabilityCase) WantsRead() bool {
	switch c.Shape {
	case ShapeProposal, ShapeRefusal, ShapeHandoff:
		return false
	}

	return len(c.Chain) == 0 || !isWrite(c.Expect)
}

func isWrite(id string) bool {
	_, _, entityRead := agentcapability.ParseCapabilityID(id)
	if entityRead {
		return false
	}
	_, verb, _ := strings.Cut(id, "_")
	verb, _, _ = strings.Cut(verb, "_")

	return verb != "get" && verb != "list" && verb != "run" && verb != "shop" && verb != "rank"
}

func LoadCapabilitySuite(path string) (*CapabilitySuite, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	suite := &CapabilitySuite{}
	if err = yaml.Unmarshal(raw, suite); err != nil {
		return nil, err
	}
	if len(suite.Cases) < minimumCapabilityCase {
		return nil, fmt.Errorf("the capability suite has %d cases; it needs at least %d",
			len(suite.Cases), minimumCapabilityCase)
	}

	return suite, nil
}

type Candidate struct {
	ID     string
	Status string
}

type SearchAnswer struct {
	Candidates    []Candidate
	LowConfidence bool
	Tokens        int
}

type TurnObservation struct {
	SchemaTokens     []int
	CapabilityTokens []int
	Opened           []string
}

var conversationTools = []string{
	"ask_user", "withdraw_proposal", "name_plan", "publish_artifact", "show_answer", "show_board",
	"request_decision", "delegate_task", "search_tools", "find_tools",
}

func IsConversationTool(name string) bool { return slices.Contains(conversationTools, name) }

type ReadPlan struct {
	Calls     int
	Proposals int
}

type CapabilitySystem struct {
	Name   string
	Search func(c *CapabilityCase) (SearchAnswer, error)
	Turn   func(c *CapabilityCase) (TurnObservation, error)
	Read   func(c *CapabilityCase) ReadPlan
}

type CapabilityOutcome struct {
	Case      *CapabilityCase
	Answer    SearchAnswer
	Rank      int
	Status    string
	Calls     int
	Tokens    float64
	CapTokens float64
	Opened    bool
	Proposal  bool
}

type CapabilityMetrics struct {
	System             string  `json:"system"`
	Cases              int     `json:"cases"`
	Top1               float64 `json:"top1"`
	Top3               float64 `json:"top3"`
	StatusCorrect      float64 `json:"statusCorrect"`
	CallsPerRequest    float64 `json:"callsPerRequest"`
	SchemaTokensTurn   float64 `json:"schemaTokensPerTurn"`
	CapabilityTokens   float64 `json:"capabilitySchemaTokensPerTurn"`
	SearchTokensMax    int     `json:"searchTokensMax"`
	FalseNotAvailable  int     `json:"falseNotAvailable"`
	ProposalsForReads  int     `json:"proposalsForReads"`
	VagueTop1          float64 `json:"vagueTop1"`
	DeniedOrHandoffHit float64 `json:"deniedOrHandoffCorrect"`
}

type CapabilityReport struct {
	Metrics  CapabilityMetrics
	Outcomes []CapabilityOutcome
}

func EvaluateCapabilities(suite *CapabilitySuite, system *CapabilitySystem) (*CapabilityReport, error) {
	report := &CapabilityReport{Outcomes: make([]CapabilityOutcome, 0, len(suite.Cases))}
	for idx := range suite.Cases {
		outcome, err := evaluateCapability(&suite.Cases[idx], system)
		if err != nil {
			return nil, fmt.Errorf("%q: %w", suite.Cases[idx].Request, err)
		}
		report.Outcomes = append(report.Outcomes, outcome)
	}
	report.Metrics = summarize(system.Name, report.Outcomes)

	return report, nil
}

func evaluateCapability(c *CapabilityCase, system *CapabilitySystem) (CapabilityOutcome, error) {
	outcome := CapabilityOutcome{Case: c}
	turn, err := system.Turn(c)
	if err != nil {
		return outcome, err
	}
	outcome.Tokens = mean(turn.SchemaTokens)
	outcome.CapTokens = mean(turn.CapabilityTokens)
	outcome.Opened = slices.Contains(turn.Opened, c.Expect)

	answer, err := system.Search(c)
	if err != nil {
		return outcome, err
	}
	outcome.Answer = answer
	for idx, candidate := range answer.Candidates {
		if candidate.ID == c.Expect {
			outcome.Rank = idx + 1
			outcome.Status = candidate.Status
			break
		}
	}

	plan := system.Read(c)
	searches := 0
	switch {
	case outcome.Opened:
	case outcome.Rank > 0 && outcome.Rank <= CapabilityTop:
		searches = 1
	default:
		searches = 2
	}
	outcome.Calls = searches + plan.Calls
	if len(answer.Candidates) > 0 && c.WantsRead() && isWrite(answer.Candidates[0].ID) && !outcome.Opened {
		outcome.Proposal = true
	}

	return outcome, nil
}

func (o *CapabilityOutcome) Found() bool {
	return o.Opened || (o.Rank > 0 && o.Rank <= CapabilityTop)
}

func (o *CapabilityOutcome) StatusRight() bool {
	want := o.Case.ExpectedStatus()
	if want == StatusAvailable {
		return o.Opened || o.Status == StatusAvailable || o.Status == StatusLoaded
	}

	return o.Rank > 0 && o.Rank <= CapabilityTop && o.Status == want
}

func (o *CapabilityOutcome) FalseNotAvailable() bool {
	if o.Case.ExpectedStatus() != StatusAvailable {
		return !o.StatusRight() && !o.Answer.LowConfidence
	}

	return !o.Found() && !o.Answer.LowConfidence
}

func summarize(system string, outcomes []CapabilityOutcome) CapabilityMetrics {
	metrics := CapabilityMetrics{System: system, Cases: len(outcomes)}
	var top1, top3, status, vague, vagueTop1, special, specialHit int
	var calls, tokens, capTokens float64
	for idx := range outcomes {
		outcome := &outcomes[idx]
		if outcome.Opened || outcome.Rank == 1 {
			top1++
		}
		if outcome.Found() {
			top3++
		}
		if outcome.StatusRight() {
			status++
		}
		if outcome.FalseNotAvailable() {
			metrics.FalseNotAvailable++
		}
		if outcome.Proposal {
			metrics.ProposalsForReads++
		}
		if outcome.Case.Kind == "vague" {
			vague++
			if outcome.Opened || outcome.Rank == 1 {
				vagueTop1++
			}
		}
		if outcome.Case.ExpectedStatus() != StatusAvailable {
			special++
			if outcome.StatusRight() {
				specialHit++
			}
		}
		calls += float64(outcome.Calls)
		tokens += outcome.Tokens
		capTokens += outcome.CapTokens
		metrics.SearchTokensMax = max(metrics.SearchTokensMax, outcome.Answer.Tokens)
	}
	total := float64(len(outcomes))
	metrics.Top1 = round3(float64(top1) / total)
	metrics.Top3 = round3(float64(top3) / total)
	metrics.StatusCorrect = round3(float64(status) / total)
	metrics.CallsPerRequest = round3(calls / total)
	metrics.SchemaTokensTurn = math.Round(tokens / total)
	metrics.CapabilityTokens = math.Round(capTokens / total)
	if vague > 0 {
		metrics.VagueTop1 = round3(float64(vagueTop1) / float64(vague))
	}
	if special > 0 {
		metrics.DeniedOrHandoffHit = round3(float64(specialHit) / float64(special))
	}

	return metrics
}

func (r *CapabilityReport) Misses() []string {
	out := make([]string, 0, 16)
	for idx := range r.Outcomes {
		outcome := &r.Outcomes[idx]
		if outcome.Found() && outcome.StatusRight() {
			continue
		}
		top := make([]string, 0, CapabilityTop)
		for i, candidate := range outcome.Answer.Candidates {
			if i == CapabilityTop {
				break
			}
			top = append(top, candidate.ID+"("+candidate.Status+")")
		}
		out = append(out, fmt.Sprintf("%q [%s/%s] want %s %s, got %s",
			outcome.Case.Request, outcome.Case.Kind, outcome.Case.AgentName(),
			outcome.Case.Expect, outcome.Case.ExpectedStatus(), strings.Join(top, " ")))
	}

	return out
}

func MetricsTable(rows ...CapabilityMetrics) string {
	var b strings.Builder
	b.WriteString("| System | Cases | Top 1 | Top 3 | Vague top 1 | Status right | Denied/hand-off right | " +
		"Calls per request | Schema tokens per turn | Capability schema tokens per turn | Largest search answer | False \"not available\" | " +
		"Proposals where a read was expected |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	for _, m := range rows {
		fmt.Fprintf(&b, "| %s | %d | %.1f%% | %.1f%% | %.1f%% | %.1f%% | %.1f%% | %.2f | %.0f | %.0f | %d | %d | %d |\n",
			m.System, m.Cases, m.Top1*100, m.Top3*100, m.VagueTop1*100, m.StatusCorrect*100,
			m.DeniedOrHandoffHit*100, m.CallsPerRequest, m.SchemaTokensTurn, m.CapabilityTokens, m.SearchTokensMax,
			m.FalseNotAvailable, m.ProposalsForReads)
	}

	return b.String()
}

func mean(values []int) float64 {
	if len(values) == 0 {
		return 0
	}
	total := 0
	for _, value := range values {
		total += value
	}

	return float64(total) / float64(len(values))
}

func round3(value float64) float64 {
	return math.Round(value*1000) / 1000
}

func SchemaTokens(name, description string, parameters map[string]any) int {
	encoded, err := MarshalJSON(map[string]any{
		"name":         name,
		"description":  description,
		"input_schema": parameters,
	})
	if err != nil {
		return 0
	}

	return TextTokens(string(encoded))
}

func TextTokens(text string) int {
	return (len(text) + 3) / 4
}
