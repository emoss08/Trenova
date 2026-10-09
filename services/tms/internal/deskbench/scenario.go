package deskbench

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/shared/jsonutils"
	"gopkg.in/yaml.v3"
)

type Scenario struct {
	Name        string       `yaml:"name"        json:"name"`
	Description string       `yaml:"description" json:"description,omitempty"`
	Tags        []string     `yaml:"tags"        json:"tags,omitempty"`
	Agent       string       `yaml:"agent"       json:"agent,omitempty"`
	User        string       `yaml:"user"        json:"user,omitempty"`
	Provider    string       `yaml:"provider"    json:"provider,omitempty"`
	Formula     *FormulaPage `yaml:"formula"     json:"formula,omitempty"`
	Exclusive   string       `yaml:"exclusive"   json:"exclusive,omitempty"`
	Steps       []Step       `yaml:"steps"       json:"steps"`
	Source      string       `yaml:"-"           json:"source,omitempty"`
}

type Step struct {
	Say           string           `yaml:"say"           json:"say,omitempty"`
	Agent         string           `yaml:"agent"         json:"agent,omitempty"`
	Page          map[string]any   `yaml:"page"          json:"page,omitempty"`
	Mentions      []map[string]any `yaml:"mentions"      json:"mentions,omitempty"`
	Surface       agent.Surface    `yaml:"surface"       json:"surface,omitempty"`
	Decide        DecisionAction   `yaml:"decide"        json:"decide,omitempty"`
	Note          string           `yaml:"note"          json:"note,omitempty"`
	Tool          string           `yaml:"tool"          json:"tool,omitempty"`
	Modifications map[string]any   `yaml:"modifications" json:"modifications,omitempty"`
	Expect        Expect           `yaml:"expect"        json:"expect"`
}

func (s *Step) Utterance() (Utterance, error) {
	utterance := Utterance{Content: s.Say, Surface: s.Surface}
	if len(s.Page) > 0 {
		utterance.Page = new(agent.PageContext)
		if err := jsonutils.Convert(s.Page, utterance.Page); err != nil {
			return Utterance{}, fmt.Errorf("page: %w", err)
		}
	}
	if len(s.Mentions) > 0 {
		utterance.Mentions = make([]agent.EntityRef, 0, len(s.Mentions))
		if err := jsonutils.Convert(s.Mentions, &utterance.Mentions); err != nil {
			return Utterance{}, fmt.Errorf("mentions: %w", err)
		}
	}

	return utterance, nil
}

func (s *Step) Label() string {
	if s.Decide != "" {
		label := string(s.Decide)
		if s.Tool != "" {
			label += " " + s.Tool
		}
		if s.Note != "" {
			label += ": " + s.Note
		}

		return label
	}

	return s.Say
}

type Expect struct {
	Calls          []string        `yaml:"calls"          json:"calls,omitempty"`
	NoCalls        []string        `yaml:"noCalls"        json:"noCalls,omitempty"`
	MaxCalls       *int            `yaml:"maxCalls"       json:"maxCalls,omitempty"`
	MaxFailedCalls *int            `yaml:"maxFailedCalls" json:"maxFailedCalls,omitempty"`
	ReplyIncludes  []string        `yaml:"replyIncludes"  json:"replyIncludes,omitempty"`
	ReplyExcludes  []string        `yaml:"replyExcludes"  json:"replyExcludes,omitempty"`
	NoRecordIDs    bool            `yaml:"noRecordIds"    json:"noRecordIds,omitempty"`
	Formula        []string        `yaml:"formula"        json:"formula,omitempty"`
	Refused        *bool           `yaml:"refused"        json:"refused,omitempty"`
	Proposals      []ExpectedWrite `yaml:"proposals"      json:"proposals,omitempty"`
	NoProposals    bool            `yaml:"noProposals"    json:"noProposals,omitempty"`
	Executed       *bool           `yaml:"executed"       json:"executed,omitempty"`
	Facts          []Fact          `yaml:"facts"          json:"facts,omitempty"`
	DB             []DBCheck       `yaml:"db"             json:"db,omitempty"`
	MaxSeconds     *float64        `yaml:"maxSeconds"     json:"maxSeconds,omitempty"`
	Rubric         string          `yaml:"rubric"         json:"rubric,omitempty"`
}

type ExpectedWrite struct {
	Tool string         `yaml:"tool" json:"tool"`
	Args map[string]any `yaml:"args" json:"args,omitempty"`
}

type DBCheck struct {
	SQL    string `yaml:"sql"    json:"sql"`
	Equals string `yaml:"equals" json:"equals"`
	Note   string `yaml:"note"   json:"note,omitempty"`
}

type Fact struct {
	SQL  string `yaml:"sql"  json:"sql"`
	Note string `yaml:"note" json:"note,omitempty"`
	Any  bool   `yaml:"any"  json:"any,omitempty"`
}

var (
	ErrScenarioUnnamed = errors.New("scenario has no name")
	ErrScenarioEmpty   = errors.New("scenario has no steps")
)

func LoadScenarios(paths []string) ([]*Scenario, error) {
	files := make([]string, 0, len(paths))
	for _, path := range paths {
		found, err := scenarioFiles(path)
		if err != nil {
			return nil, err
		}
		files = append(files, found...)
	}
	slices.Sort(files)
	files = slices.Compact(files)

	scenarios := make([]*Scenario, 0, len(files))
	names := make(map[string]string, len(files))
	for _, file := range files {
		loaded, err := loadScenarioFile(file)
		if err != nil {
			return nil, err
		}
		for _, scenario := range loaded {
			if previous, dup := names[scenario.Name]; dup {
				return nil, fmt.Errorf("scenario %q is defined in %s and %s", scenario.Name, previous, file)
			}
			names[scenario.Name] = file
			scenarios = append(scenarios, scenario)
		}
	}

	return scenarios, nil
}

func scenarioFiles(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("read scenarios at %s: %w", path, err)
	}
	if !info.IsDir() {
		return []string{path}, nil
	}

	files := make([]string, 0, 16)
	err = filepath.WalkDir(path, func(file string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		switch strings.ToLower(filepath.Ext(file)) {
		case ".yaml", ".yml":
			files = append(files, file)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk scenarios at %s: %w", path, err)
	}

	return files, nil
}

func loadScenarioFile(file string) ([]*Scenario, error) {
	raw, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", file, err)
	}

	var doc struct {
		Defaults  Scenario    `yaml:"defaults"`
		Scenarios []*Scenario `yaml:"scenarios"`
	}
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	if err = decoder.Decode(&doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", file, err)
	}

	for idx, scenario := range doc.Scenarios {
		if scenario == nil {
			return nil, fmt.Errorf("%s: scenario %d is empty", file, idx+1)
		}
		scenario.Source = file
		scenario.applyDefaults(&doc.Defaults)
		if err = scenario.Validate(); err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
	}

	return doc.Scenarios, nil
}

func (s *Scenario) applyDefaults(defaults *Scenario) {
	if s.Agent == "" {
		s.Agent = defaults.Agent
	}
	if s.User == "" {
		s.User = defaults.User
	}
	if s.Provider == "" {
		s.Provider = defaults.Provider
	}
	for _, tag := range defaults.Tags {
		if !slices.Contains(s.Tags, tag) {
			s.Tags = append(s.Tags, tag)
		}
	}
}

func (s *Scenario) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return ErrScenarioUnnamed
	}
	if len(s.Steps) == 0 {
		return fmt.Errorf("%s: %w", s.Name, ErrScenarioEmpty)
	}

	for idx := range s.Steps {
		step := &s.Steps[idx]
		where := fmt.Sprintf("%s step %d", s.Name, idx+1)
		hasSay := strings.TrimSpace(step.Say) != ""
		hasDecide := step.Decide != ""
		switch {
		case hasSay == hasDecide:
			return fmt.Errorf("%s: a step either says something or decides, not both or neither", where)
		case hasDecide && step.Decide != DecisionApprove && step.Decide != DecisionReject:
			return fmt.Errorf("%s: decide must be approve or reject, not %q", where, step.Decide)
		case hasDecide && idx == 0:
			return fmt.Errorf("%s: the first step must say something", where)
		case hasSay && (step.Note != "" || step.Tool != "" || len(step.Modifications) > 0):
			return fmt.Errorf("%s: note, tool and modifications belong on a decide step", where)
		case hasDecide && step.Agent != "":
			return fmt.Errorf("%s: agent belongs on a say step, the one that hands the message over", where)
		}
		if hasSay {
			if _, err := step.Utterance(); err != nil {
				return fmt.Errorf("%s: %w", where, err)
			}
		}
		for _, check := range step.Expect.DB {
			fact := Fact{SQL: check.SQL}
			if err := fact.validate(); err != nil {
				return fmt.Errorf("%s: %w", where, err)
			}
		}
		for _, fact := range step.Expect.Facts {
			if err := fact.validate(); err != nil {
				return fmt.Errorf("%s: %w", where, err)
			}
		}
	}

	return nil
}

func (f *Fact) validate() error {
	statement := strings.ToLower(strings.TrimSpace(f.SQL))
	if !strings.HasPrefix(statement, "select") && !strings.HasPrefix(statement, "with") {
		return fmt.Errorf("a fact is a SELECT: %q", f.SQL)
	}
	if strings.Contains(strings.TrimRight(statement, "; \n\t"), ";") {
		return fmt.Errorf("a fact is one statement: %q", f.SQL)
	}

	return nil
}

func (s *Scenario) HasTag(tags []string) bool {
	if len(tags) == 0 {
		return true
	}
	for _, tag := range tags {
		if slices.Contains(s.Tags, tag) {
			return true
		}
	}

	return false
}
