package deskbench

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentsuggestion"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"gopkg.in/yaml.v3"
)

const (
	RoutingExpectCurrent = "current"
	RoutingExpectNone    = "none"
	RoutingExpectNothing = "nothing"

	routingReportFile  = "routing.json"
	routingSummaryFile = "routing.md"
)

var (
	ErrRoutingUnnamed  = errors.New("a routing case needs a name")
	ErrRoutingNoAsk    = errors.New("a routing case needs the question it asks")
	ErrRoutingNoExpect = errors.New("a routing case needs what it expects")
)

type RoutingCase struct {
	Name   string   `yaml:"name"   json:"name"`
	Ask    string   `yaml:"ask"    json:"ask"`
	From   string   `yaml:"from"   json:"from"`
	About  []string `yaml:"about"  json:"about,omitempty"`
	User   string   `yaml:"user"   json:"user,omitempty"`
	Expect string   `yaml:"expect" json:"expect"`
	Note   string   `yaml:"note"   json:"note,omitempty"`
	Source string   `yaml:"-"      json:"source"`
}

func (c *RoutingCase) Validate() error {
	switch {
	case strings.TrimSpace(c.Name) == "":
		return ErrRoutingUnnamed
	case strings.TrimSpace(c.Ask) == "":
		return fmt.Errorf("%s: %w", c.Name, ErrRoutingNoAsk)
	case strings.TrimSpace(c.Expect) == "":
		return fmt.Errorf("%s: %w", c.Name, ErrRoutingNoExpect)
	case len(c.About) > agentports.MaxAgentMatchContextKinds:
		return fmt.Errorf(
			"%s: a question is about at most %d kinds of record",
			c.Name,
			agentports.MaxAgentMatchContextKinds,
		)
	}

	return nil
}

func (c *RoutingCase) applyDefaults(defaults *RoutingCase) {
	if c.From == "" {
		c.From = defaults.From
	}
	if c.User == "" {
		c.User = defaults.User
	}
}

func LoadRoutingCases(paths []string) ([]*RoutingCase, error) {
	cases := make([]*RoutingCase, 0, 64)
	names := make(map[string]string, 64)
	for _, path := range paths {
		files, err := scenarioFiles(path)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			loaded, loadErr := loadRoutingFile(file)
			if loadErr != nil {
				return nil, loadErr
			}
			for _, routing := range loaded {
				if first, dup := names[routing.Name]; dup {
					return nil, fmt.Errorf(
						"%s: routing case %q is also in %s",
						file,
						routing.Name,
						first,
					)
				}
				names[routing.Name] = file
				cases = append(cases, routing)
			}
		}
	}

	return cases, nil
}

func loadRoutingFile(file string) ([]*RoutingCase, error) {
	raw, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", file, err)
	}

	var doc struct {
		Defaults RoutingCase    `yaml:"defaults"`
		Cases    []*RoutingCase `yaml:"cases"`
	}
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	if err = decoder.Decode(&doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", file, err)
	}

	for idx, routing := range doc.Cases {
		if routing == nil {
			return nil, fmt.Errorf("%s: routing case %d is empty", file, idx+1)
		}
		routing.Source = file
		routing.applyDefaults(&doc.Defaults)
		if err = routing.Validate(); err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
	}

	return doc.Cases, nil
}

type RoutingResult struct {
	Case     string   `json:"case"`
	Ask      string   `json:"ask"`
	From     string   `json:"from"`
	About    []string `json:"about,omitempty"`
	Expect   string   `json:"expect"`
	Outcome  string   `json:"outcome"`
	Agent    string   `json:"agent,omitempty"`
	Delegate string   `json:"delegate,omitempty"`
	Tools    []string `json:"tools,omitempty"`
	Passed   bool     `json:"passed"`
	Note     string   `json:"note,omitempty"`
	Error    string   `json:"error,omitempty"`
	Millis   int64    `json:"millis"`
}

func (r *RoutingResult) Got() string {
	switch {
	case r.Error != "":
		return "error"
	case r.Outcome == string(agentsuggestion.MatchNoAgent):
		return RoutingExpectNone
	case r.Outcome == string(agentsuggestion.MatchNone):
		return RoutingExpectNothing
	case r.Outcome == string(agentsuggestion.MatchCurrentAgent) && r.Delegate != "":
		return RoutingExpectCurrent + " (" + r.Agent + ", handing it to " + r.Delegate + ")"
	case r.Outcome == string(agentsuggestion.MatchCurrentAgent):
		return RoutingExpectCurrent + " (" + r.Agent + ")"
	default:
		return r.Agent
	}
}

type RouteOptions struct {
	User     string
	Progress func(*RoutingResult)
}

func (b *Bench) Route(
	ctx context.Context,
	cases []*RoutingCase,
	opts RouteOptions,
) ([]*RoutingResult, error) {
	sessions := make(map[string]*Session, 2)
	results := make([]*RoutingResult, 0, len(cases))
	for _, routing := range cases {
		user := routing.User
		if user == "" {
			user = opts.User
		}
		session, ok := sessions[user]
		if !ok {
			var err error
			session, err = b.SessionFor(ctx, user)
			if err != nil {
				return nil, err
			}
			sessions[user] = session
		}

		result := b.routeOne(ctx, session, routing)
		results = append(results, result)
		if opts.Progress != nil {
			opts.Progress(result)
		}
	}

	return results, nil
}

func (b *Bench) routeOne(
	ctx context.Context,
	session *Session,
	routing *RoutingCase,
) *RoutingResult {
	result := &RoutingResult{
		Case:   routing.Name,
		Ask:    routing.Ask,
		From:   routing.From,
		About:  routing.About,
		Expect: routing.Expect,
	}
	started := time.Now()
	defer func() { result.Millis = time.Since(started).Milliseconds() }()

	from, err := session.Agent(ctx, routing.From)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.From = from.Name

	found, err := b.Recommender.MatchAgent(session.Context(ctx), &agentports.AgentMatchRequest{
		Actor:          &session.Actor,
		Text:           routing.Ask,
		CurrentAgentID: from.ID,
		ContextKinds:   routing.About,
	})
	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.Outcome = string(found.Outcome)
	if found.Agent != nil {
		result.Agent = found.Agent.Name
	}
	if found.Delegate != nil {
		result.Delegate = found.Delegate.Name
	}
	for _, tool := range found.Tools {
		result.Tools = append(result.Tools, tool.Name)
	}
	result.Passed = RoutingPasses(routing.Expect, from, found)
	if !result.Passed {
		if agents, listErr := session.Agents(ctx); listErr == nil {
			result.Note = UnreachableExpectation(routing.Expect, agents)
		}
	}

	return result
}

func UnreachableExpectation(expect string, agents []*agentdefinition.Definition) string {
	named := make([]string, 0, 2)
	for _, option := range strings.Split(expect, "|") {
		option = strings.TrimSpace(option)
		switch strings.ToLower(option) {
		case RoutingExpectCurrent, RoutingExpectNone, RoutingExpectNothing:
			return ""
		}
		if slices.ContainsFunc(agents, func(definition *agentdefinition.Definition) bool {
			return agentNamed(definition, option)
		}) {
			return ""
		}
		named = append(named, option)
	}
	if len(named) == 0 {
		return ""
	}

	return "no enabled chat agent is " + strings.Join(named, " or ")
}

func RoutingPasses(
	expect string,
	from *agentdefinition.Definition,
	found *agentports.AgentMatch,
) bool {
	if found == nil {
		return false
	}
	for _, option := range strings.Split(expect, "|") {
		option = strings.TrimSpace(option)
		switch strings.ToLower(option) {
		case RoutingExpectCurrent:
			if found.Outcome == agentsuggestion.MatchCurrentAgent {
				return true
			}
		case RoutingExpectNone:
			if found.Outcome == agentsuggestion.MatchNoAgent {
				return true
			}
		case RoutingExpectNothing:
			if found.Outcome == agentsuggestion.MatchNone {
				return true
			}
		default:
			if slices.ContainsFunc(
				answeredBy(from, found),
				func(definition *agentdefinition.Definition) bool {
					return agentNamed(definition, option)
				},
			) {
				return true
			}
		}
	}

	return false
}

func answeredBy(
	from *agentdefinition.Definition,
	found *agentports.AgentMatch,
) []*agentdefinition.Definition {
	switch found.Outcome {
	case agentsuggestion.MatchBetterAgent:
		return []*agentdefinition.Definition{found.Agent}
	case agentsuggestion.MatchCurrentAgent:
		return []*agentdefinition.Definition{from, found.Delegate}
	case agentsuggestion.MatchNoAgent, agentsuggestion.MatchNone:
		return nil
	}

	return nil
}

func agentNamed(definition *agentdefinition.Definition, ref string) bool {
	return definition != nil && ref != "" &&
		(definition.ID.String() == ref ||
			strings.EqualFold(definition.Name, ref) ||
			strings.EqualFold(string(definition.Template), ref))
}

type RoutingReport struct {
	Commit    string           `json:"commit"`
	StartedAt time.Time        `json:"startedAt"`
	User      string           `json:"user"`
	Passed    int              `json:"passed"`
	Total     int              `json:"total"`
	Cases     []*RoutingResult `json:"cases"`
}

func BuildRoutingReport(
	commit, user string,
	startedAt time.Time,
	results []*RoutingResult,
) *RoutingReport {
	report := &RoutingReport{
		Commit:    commit,
		StartedAt: startedAt,
		User:      user,
		Total:     len(results),
		Cases:     results,
	}
	for _, result := range results {
		if result.Passed {
			report.Passed++
		}
	}

	return report
}

func (r *RoutingReport) Accuracy() float64 {
	if r == nil || r.Total == 0 {
		return 0
	}

	return float64(r.Passed) / float64(r.Total)
}

func (r *RoutingReport) Write(dir string, baseline *RoutingReport) error {
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	encoded, err := reportJSON.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("encode the routing report: %w", err)
	}
	if err = os.WriteFile(filepath.Join(dir, routingReportFile), encoded, fileMode); err != nil {
		return fmt.Errorf("write the routing report: %w", err)
	}
	if err = os.WriteFile(
		filepath.Join(dir, routingSummaryFile),
		[]byte(RenderRouting(r, baseline)),
		fileMode,
	); err != nil {
		return fmt.Errorf("write the routing summary: %w", err)
	}

	return nil
}

func LoadRoutingReport(path string) (*RoutingReport, error) {
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		path = filepath.Join(path, routingReportFile)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	report := new(RoutingReport)
	if err = sonic.Unmarshal(raw, report); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	return report, nil
}

func RenderRouting(r, baseline *RoutingReport) string {
	var out strings.Builder
	fmt.Fprintf(&out, "# Agent routing\n\n")
	fmt.Fprintf(&out, "%d of %d questions went where they should (%.0f%%)", r.Passed, r.Total,
		r.Accuracy()*100)
	if r.Commit != "" {
		fmt.Fprintf(&out, " at %s", r.Commit)
	}
	out.WriteString(".\n")
	if baseline != nil {
		fmt.Fprintf(&out, "\nThe run compared with: %d of %d (%.0f%%), %+.0f points.\n",
			baseline.Passed, baseline.Total, baseline.Accuracy()*100,
			(r.Accuracy()-baseline.Accuracy())*100)
		previous := make(map[string]bool, len(baseline.Cases))
		for _, result := range baseline.Cases {
			previous[result.Case] = result.Passed
		}
		changed := make([]string, 0, 8)
		for _, result := range r.Cases {
			passed, known := previous[result.Case]
			if !known || passed == result.Passed {
				continue
			}
			word := "now fails"
			if result.Passed {
				word = "now passes"
			}
			changed = append(changed, fmt.Sprintf("- %#q %s", result.Case, word))
		}
		if len(changed) > 0 {
			slices.Sort(changed)
			out.WriteString("\n" + strings.Join(changed, "\n") + "\n")
		}
	}

	out.WriteString("\n| Case | Asked from | Expected | Got | Tools |\n|---|---|---|---|---|\n")
	for _, result := range r.Cases {
		mark := "✓"
		if !result.Passed {
			mark = "✗"
		}
		got := result.Got()
		if result.Error != "" {
			got = "error: " + result.Error
		}
		if result.Note != "" {
			got += "; " + result.Note
		}
		fmt.Fprintf(&out, "| %s %s | %s | %s | %s | %s |\n",
			mark,
			markdownCell(result.Case),
			markdownCell(result.From),
			markdownCell(result.Expect),
			markdownCell(got),
			markdownCell(strings.Join(result.Tools, ", ")),
		)
	}

	return out.String()
}

func markdownCell(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "|", "\\|"), "\n", " ")
}
