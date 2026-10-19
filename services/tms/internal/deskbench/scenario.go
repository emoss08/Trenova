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

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/evalsplit"
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
	Palette     bool         `yaml:"palette"     json:"palette,omitempty"`
	Exclusive   string       `yaml:"exclusive"   json:"exclusive,omitempty"`
	Setup       []string     `yaml:"setup"       json:"setup,omitempty"`
	Steps       []Step       `yaml:"steps"       json:"steps"`
	Source      string       `yaml:"-"           json:"source,omitempty"`

	Split          evalsplit.Split `yaml:"split"          json:"split,omitempty"`
	ExpectedTools  []string        `yaml:"expectedTools"  json:"expectedTools,omitempty"`
	ForbiddenTools []string        `yaml:"forbiddenTools" json:"forbiddenTools,omitempty"`
	LabelsDerived  bool            `yaml:"-"              json:"labelsDerived,omitempty"`
}

const generalAssistantDesk = "General assistant"

func (s *Scenario) Stratum() evalsplit.Stratum {
	if s.Unattended() {
		return evalsplit.Stratum{
			Desk: string(s.Steps[0].Fire.Template()),
			Mode: evalsplit.Unattended,
		}
	}

	desk := s.Agent
	switch {
	case s.Palette:
		desk = "Palette"
	case s.Formula != nil:
		desk = "Formula studio"
	case desk == "":
		for idx := range s.Steps {
			if s.Steps[idx].Agent != "" {
				desk = s.Steps[idx].Agent
				break
			}
		}
	}
	desk = strings.TrimSpace(strings.TrimPrefix(desk, BenchAgentPrefix))
	if desk == "" {
		desk = generalAssistantDesk
	}

	return evalsplit.Stratum{Desk: desk, Mode: evalsplit.Chat}
}

func (s *Scenario) deriveLabels() {
	if s.ExpectedTools != nil && s.ForbiddenTools != nil {
		return
	}

	expected := make([]string, 0, 4)
	forbidden := make([]string, 0, 2)
	for idx := range s.Steps {
		want := &s.Steps[idx].Expect
		for _, spec := range want.Calls {
			if !slices.Contains(expected, spec) {
				expected = append(expected, spec)
			}
		}
		for _, proposal := range want.Proposals {
			if !slices.Contains(expected, proposal.Tool) {
				expected = append(expected, proposal.Tool)
			}
		}
		for _, spec := range want.NoCalls {
			for _, name := range alternatives(spec) {
				if !slices.Contains(forbidden, name) {
					forbidden = append(forbidden, name)
				}
			}
		}
	}
	if s.ExpectedTools == nil {
		s.ExpectedTools = expected
		s.LabelsDerived = true
	}
	if s.ForbiddenTools == nil {
		s.ForbiddenTools = slices.DeleteFunc(forbidden, func(name string) bool {
			return slices.ContainsFunc(s.ExpectedTools, func(spec string) bool {
				return slices.Contains(alternatives(spec), name)
			})
		})
		s.LabelsDerived = true
	}
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
	Attach        []string         `yaml:"attach"        json:"attach,omitempty"`
	Fire          *Fire            `yaml:"fire"          json:"fire,omitempty"`
	Expect        Expect           `yaml:"expect"        json:"expect"`
}

func (s *Step) Utterance() (Utterance, error) {
	utterance := Utterance{Content: s.Say, Surface: s.Surface, Attachments: s.Attach}
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

func (s *Step) mentionFinds() map[int]string {
	finds := make(map[int]string, len(s.Mentions))
	for idx, mention := range s.Mentions {
		if query, ok := mention["find"].(string); ok {
			finds[idx] = query
		}
	}

	return finds
}

func (s *Step) validateMentionFinds(where string) error {
	for idx, query := range s.mentionFinds() {
		mention := s.Mentions[idx]
		if _, hasID := mention["id"]; hasID {
			return fmt.Errorf("%s mention %d: give an id or a find query, not both", where, idx+1)
		}
		if kind, _ := mention["type"].(string); strings.TrimSpace(kind) == "" {
			return fmt.Errorf("%s mention %d: a found mention still names its type", where, idx+1)
		}
		fact := Fact{SQL: query}
		if err := fact.validate(); err != nil {
			return fmt.Errorf("%s mention %d: %w", where, idx+1, err)
		}
	}

	return nil
}

func (s *Step) validateAttach(where string, decides bool) error {
	switch {
	case decides && len(s.Attach) > 0:
		return fmt.Errorf("%s: attach belongs on a say step, the message the files go with", where)
	case len(s.Attach) > conversation.MaxMessageAttachments:
		return fmt.Errorf("%s: a message carries at most %d files",
			where, conversation.MaxMessageAttachments)
	default:
		return nil
	}
}

func (s *Step) Label() string {
	if s.Fire != nil {
		return s.Fire.Label()
	}
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
	Run            string          `yaml:"run"            json:"run,omitempty"`
	Flagged        *bool           `yaml:"flagged"        json:"flagged,omitempty"`
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
				return nil, fmt.Errorf(
					"scenario %q is defined in %s and %s",
					scenario.Name,
					previous,
					file,
				)
			}
			names[scenario.Name] = file
			scenarios = append(scenarios, scenario)
		}
	}

	if err := assignSplits(scenarios); err != nil {
		return nil, err
	}

	return scenarios, nil
}

func assignSplits(scenarios []*Scenario) error {
	population, err := splitPopulation(scenarios)
	if err != nil {
		return err
	}

	items := make([]evalsplit.Item, 0, len(population))
	for _, scenario := range population {
		items = append(items, scenario.splitItem())
	}
	splits := evalsplit.Assign(items)
	assigned := make(map[string]evalsplit.Split, len(population))
	for idx := range items {
		assigned[items[idx].Key] = splits[idx]
	}
	for _, scenario := range scenarios {
		scenario.Split = assigned[scenario.Name]
	}

	return nil
}

func (s *Scenario) splitItem() evalsplit.Item {
	return evalsplit.Item{
		Key:         s.Name,
		Stratum:     s.Stratum(),
		Adversarial: s.Split == evalsplit.Adversarial,
	}
}

func splitPopulation(scenarios []*Scenario) ([]*Scenario, error) {
	loaded := make(map[string]struct{}, len(scenarios))
	dirs := make([]string, 0, 2)
	for _, scenario := range scenarios {
		loaded[filepath.Clean(scenario.Source)] = struct{}{}
		if dir := filepath.Dir(scenario.Source); !slices.Contains(dirs, dir) {
			dirs = append(dirs, dir)
		}
	}

	population := slices.Clone(scenarios)
	seen := make(map[string]struct{}, len(scenarios))
	for _, scenario := range scenarios {
		seen[scenario.Name] = struct{}{}
	}
	for _, dir := range dirs {
		files, err := scenarioFiles(dir)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			if _, ok := loaded[filepath.Clean(file)]; ok {
				continue
			}
			siblings, loadErr := loadScenarioFile(file)
			if loadErr != nil {
				return nil, loadErr
			}
			for _, sibling := range siblings {
				if _, ok := seen[sibling.Name]; ok {
					continue
				}
				seen[sibling.Name] = struct{}{}
				population = append(population, sibling)
			}
		}
	}

	return population, nil
}

func SplitLedger(scenarios []*Scenario) evalsplit.Ledger {
	items := make([]evalsplit.Item, 0, len(scenarios))
	splits := make([]evalsplit.Split, 0, len(scenarios))
	for _, scenario := range scenarios {
		items = append(items, scenario.splitItem())
		splits = append(splits, scenario.Split)
	}

	return evalsplit.LedgerOf(items, splits)
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
		if scenario.Split != "" && scenario.Split != evalsplit.Adversarial {
			return nil, fmt.Errorf("%s: %s: split %q is assigned by rule; only adversarial is "+
				"written by hand (docs/engineering/desk-bench.md, Labels and splits)",
				file, scenario.Name, scenario.Split)
		}
		scenario.resolveAttachments(filepath.Dir(file))
		scenario.deriveLabels()
		if err = scenario.Validate(); err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
	}

	return doc.Scenarios, nil
}

func (s *Scenario) resolveAttachments(dir string) {
	for idx := range s.Steps {
		for pos, ref := range s.Steps[idx].Attach {
			if strings.HasPrefix(ref, documentIDPrefix) || filepath.IsAbs(ref) {
				continue
			}
			s.Steps[idx].Attach[pos] = filepath.Join(dir, ref)
		}
	}
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
	if s.Split == "" {
		s.Split = defaults.Split
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
	for idx, statement := range s.Setup {
		if err := validSetup(statement); err != nil {
			return fmt.Errorf("%s setup %d: %w", s.Name, idx+1, err)
		}
	}
	if err := s.validatePalette(); err != nil {
		return err
	}
	if err := s.validateFire(); err != nil {
		return err
	}
	if s.Split != "" && !s.Split.IsValid() {
		return fmt.Errorf("%s: split must be development, held-out or adversarial, not %q",
			s.Name, s.Split)
	}
	for _, spec := range s.ForbiddenTools {
		for _, name := range alternatives(spec) {
			if slices.ContainsFunc(s.ExpectedTools, func(expected string) bool {
				return slices.Contains(alternatives(expected), name)
			}) {
				return fmt.Errorf("%s: %s is both expected and forbidden", s.Name, name)
			}
		}
	}

	for idx := range s.Steps {
		step := &s.Steps[idx]
		where := fmt.Sprintf("%s step %d", s.Name, idx+1)
		if step.Fire != nil {
			if err := validateChecks(where, &step.Expect); err != nil {
				return err
			}
			continue
		}
		if step.Expect.Run != "" || step.Expect.Flagged != nil {
			return fmt.Errorf(
				"%s: run and flagged grade an unattended run; they belong on a fire step",
				where,
			)
		}
		hasSay := strings.TrimSpace(step.Say) != ""
		hasDecide := step.Decide != ""
		switch {
		case hasSay == hasDecide:
			return fmt.Errorf(
				"%s: a step either says something or decides, not both or neither",
				where,
			)
		case hasDecide && step.Decide != DecisionApprove && step.Decide != DecisionReject:
			return fmt.Errorf("%s: decide must be approve or reject, not %q", where, step.Decide)
		case hasDecide && idx == 0:
			return fmt.Errorf("%s: the first step must say something", where)
		case hasSay && (step.Note != "" || step.Tool != "" || len(step.Modifications) > 0):
			return fmt.Errorf("%s: note, tool and modifications belong on a decide step", where)
		case hasDecide && step.Agent != "":
			return fmt.Errorf(
				"%s: agent belongs on a say step, the one that hands the message over",
				where,
			)
		}
		if err := step.validateAttach(where, hasDecide); err != nil {
			return err
		}
		if hasSay {
			if _, err := step.Utterance(); err != nil {
				return fmt.Errorf("%s: %w", where, err)
			}
			if err := step.validateMentionFinds(where); err != nil {
				return err
			}
		}
		if err := validateChecks(where, &step.Expect); err != nil {
			return err
		}
	}

	return nil
}

func validateChecks(where string, expect *Expect) error {
	for _, check := range expect.DB {
		fact := Fact{SQL: check.SQL}
		if err := fact.validate(); err != nil {
			return fmt.Errorf("%s: %w", where, err)
		}
	}
	for _, fact := range expect.Facts {
		if err := fact.validate(); err != nil {
			return fmt.Errorf("%s: %w", where, err)
		}
	}
	if expect.NoProposals && len(expect.Proposals) > 0 {
		return fmt.Errorf("%s: noProposals and proposals cannot both be expected", where)
	}

	return nil
}

func (s *Scenario) Unattended() bool {
	return len(s.Steps) > 0 && s.Steps[0].Fire != nil
}

func (s *Scenario) validateFire() error {
	fired := slices.ContainsFunc(s.Steps, func(step Step) bool { return step.Fire != nil })
	if !fired {
		return nil
	}
	switch {
	case len(s.Steps) != 1:
		return fmt.Errorf("%s: an unattended scenario is one fire step; the run is graded on "+
			"what it left behind", s.Name)
	case s.Palette || s.Formula != nil || s.Agent != "":
		return fmt.Errorf("%s: a fire step names its agent's template on the step itself", s.Name)
	}
	step := &s.Steps[0]
	where := s.Name + " step 1"
	if step.Say != "" || step.Decide != "" || len(step.Attach) > 0 || len(step.Mentions) > 0 ||
		len(step.Page) > 0 {
		return fmt.Errorf("%s: a fire step says nothing and decides nothing", where)
	}

	return step.Fire.validate(where)
}

func (s *Scenario) validatePalette() error {
	if !s.Palette {
		return nil
	}
	switch {
	case s.Formula != nil:
		return fmt.Errorf("%s: a scenario asks from the palette or the formula studio, not both",
			s.Name)
	case s.Agent != "":
		return fmt.Errorf("%s: the palette picks who answers; leave agent out", s.Name)
	}
	for idx := range s.Steps {
		step := &s.Steps[idx]
		if step.Agent != "" || (step.Surface != "" && step.Surface != agent.SurfacePalette) {
			return fmt.Errorf(
				"%s step %d: every message in a palette scenario is asked from the palette",
				s.Name, idx+1,
			)
		}
	}

	return nil
}

var setupVerbs = []string{"update", "insert", "delete", "with"}

var setupForbidden = []string{"drop ", "alter ", "truncate ", "grant ", "revoke ", "create "}

func validSetup(statement string) error {
	text := strings.ToLower(strings.TrimSpace(statement))
	if !slices.ContainsFunc(
		setupVerbs,
		func(verb string) bool { return strings.HasPrefix(text, verb) },
	) {
		return fmt.Errorf("setup is an UPDATE, INSERT or DELETE: %q", statement)
	}
	if strings.Contains(strings.TrimRight(text, "; \n\t"), ";") {
		return fmt.Errorf("setup is one statement each: %q", statement)
	}
	for _, word := range setupForbidden {
		if strings.Contains(text, word) {
			return fmt.Errorf("setup changes rows, not the schema or grants: %q", statement)
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
