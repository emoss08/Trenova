package deskbench

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/evalsplit"
	"gopkg.in/yaml.v3"
)

const (
	gateMarkdown    = "gate.md"
	gateJSON        = "gate.json"
	confirmMarkdown = "confirm.md"
	confirmJSON     = "confirm.json"
	probeMarkdown   = "probe.md"
	probeJSON       = "probe.json"
	delegateTool    = "delegate_task"
	delegationTag   = "delegation"
)

type Route string

const (
	RouteChat       = Route("chat")
	RouteUnattended = Route("unattended")
	RouteDelegated  = Route("delegated")
)

func AllRoutes() []Route {
	return []Route{RouteChat, RouteUnattended, RouteDelegated}
}

func (r Route) IsValid() bool {
	return slices.Contains(AllRoutes(), r)
}

func RouteOf(scenario *Scenario) Route {
	switch {
	case scenario.Unattended():
		return RouteUnattended
	case slices.Contains(scenario.Tags, delegationTag) ||
		labelled(delegateTool, scenario.ExpectedTools):
		return RouteDelegated
	default:
		return RouteChat
	}
}

type GateSpec struct {
	Repeat   int         `yaml:"repeat"   json:"repeat"`
	PassK    int         `yaml:"passK"    json:"passK"`
	Before   string      `yaml:"before"   json:"before"`
	Budget   GateBudget  `yaml:"budget"   json:"budget"`
	Groups   []GateGroup `yaml:"groups"   json:"groups"`
	Effort   EffortRule  `yaml:"effort"   json:"effort"`
	Advisor  AdvisorRule `yaml:"advisor"  json:"advisor"`
	CodeMode []CodePair  `yaml:"codeMode" json:"codeMode,omitempty"`
	HeldOut  HeldOutRule `yaml:"heldOut"  json:"heldOut"`
}

type GateBudget struct {
	MaxUSD     float64 `yaml:"maxUsd"     json:"maxUsd"`
	MaxHours   float64 `yaml:"maxHours"   json:"maxHours"`
	Overrun    float64 `yaml:"overrun"    json:"overrun"`
	ProbeEvery int     `yaml:"probeEvery" json:"probeEvery"`
}

type GateGroup struct {
	Name   string            `yaml:"name"   json:"name"`
	Cells  []string          `yaml:"cells"  json:"cells"`
	Tags   []string          `yaml:"tags"   json:"tags,omitempty"`
	Routes []Route           `yaml:"routes" json:"routes,omitempty"`
	Splits []evalsplit.Split `yaml:"splits" json:"splits,omitempty"`
}

func (g *GateGroup) Selects(scenario *Scenario) bool {
	if !scenario.HasTag(g.Tags) {
		return false
	}
	if len(g.Routes) > 0 && !slices.Contains(g.Routes, RouteOf(scenario)) {
		return false
	}

	return len(g.Splits) == 0 || slices.Contains(g.Splits, scenario.Split)
}

type EffortRule struct {
	Candidates   []string `yaml:"candidates"   json:"candidates"`
	Within       float64  `yaml:"within"       json:"within"`
	MinScenarios int      `yaml:"minScenarios" json:"minScenarios"`
}

type AdvisorRule struct {
	Cell    string  `yaml:"cell"    json:"cell,omitempty"`
	Gain    float64 `yaml:"gain"    json:"gain"`
	CostBar string  `yaml:"costBar" json:"costBar,omitempty"`
}

type CodePair struct {
	Model    string `yaml:"model"    json:"model"`
	Code     string `yaml:"code"     json:"code"`
	Discrete string `yaml:"discrete" json:"discrete"`
}

type HeldOutRule struct {
	Tolerance       float64 `yaml:"tolerance"       json:"tolerance"`
	SelectionSlack  float64 `yaml:"selectionSlack"  json:"selectionSlack"`
	CodeModeMinimum float64 `yaml:"codeModeMinimum" json:"codeModeMinimum"`
}

var ErrGateSpec = errors.New("the gate spec is incomplete")

func LoadGateSpec(path string) (*GateSpec, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	spec := new(GateSpec)
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	if err = decoder.Decode(spec); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if err = spec.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	return spec, nil
}

func (s *GateSpec) Cells() []string {
	cells := make([]string, 0, 8)
	for idx := range s.Groups {
		for _, cell := range s.Groups[idx].Cells {
			if !slices.Contains(cells, cell) {
				cells = append(cells, cell)
			}
		}
	}

	return cells
}

func (s *GateSpec) validate() error {
	if s.Repeat <= 0 {
		s.Repeat = DefaultPassK
	}
	if s.PassK <= 0 {
		s.PassK = DefaultPassK
	}
	if s.Repeat < s.PassK {
		return fmt.Errorf("%w: repeat %d is fewer than passK %d", ErrGateSpec, s.Repeat, s.PassK)
	}
	if err := s.validateGroups(); err != nil {
		return err
	}
	if err := s.validateRules(); err != nil {
		return err
	}
	if s.Budget.Overrun < 1 {
		s.Budget.Overrun = 1
	}
	if s.Budget.ProbeEvery <= 0 {
		s.Budget.ProbeEvery = 10
	}

	return nil
}

func (s *GateSpec) validateGroups() error {
	if len(s.Groups) == 0 {
		return fmt.Errorf("%w: name at least one group of cells", ErrGateSpec)
	}
	names := make([]string, 0, len(s.Groups))
	for idx := range s.Groups {
		group := &s.Groups[idx]
		if strings.TrimSpace(group.Name) == "" || slices.Contains(names, group.Name) {
			return fmt.Errorf("%w: every group needs its own name", ErrGateSpec)
		}
		names = append(names, group.Name)
		if err := group.validate(); err != nil {
			return err
		}
	}

	return nil
}

func (g *GateGroup) validate() error {
	if len(g.Cells) == 0 {
		return fmt.Errorf("%w: group %s names no cell", ErrGateSpec, g.Name)
	}
	for _, route := range g.Routes {
		if !route.IsValid() {
			return fmt.Errorf("%w: group %s: %q is not a route", ErrGateSpec, g.Name, route)
		}
	}
	for _, split := range g.Splits {
		if !split.IsValid() {
			return fmt.Errorf("%w: group %s: %q is not a split", ErrGateSpec, g.Name, split)
		}
	}

	return nil
}

func (s *GateSpec) validateRules() error {
	cells := s.Cells()
	need := func(what, cell string) error {
		if cell == "" || slices.Contains(cells, cell) {
			return nil
		}
		return fmt.Errorf("%w: %s %q is in no group", ErrGateSpec, what, cell)
	}
	switch {
	case s.Before == "":
		return fmt.Errorf("%w: name today's configuration as before", ErrGateSpec)
	case len(s.Effort.Candidates) == 0:
		return fmt.Errorf("%w: name the effort candidates", ErrGateSpec)
	case !slices.Contains(s.Effort.Candidates, s.Before):
		return fmt.Errorf("%w: before must be one of the effort candidates", ErrGateSpec)
	case s.Advisor.Cell != "" && s.Advisor.CostBar == "":
		return fmt.Errorf("%w: the advisor needs a cost bar", ErrGateSpec)
	}

	checks := []struct{ what, cell string }{
		{"before", s.Before},
		{"advisor cell", s.Advisor.Cell},
		{"advisor cost bar", s.Advisor.CostBar},
	}
	for _, cell := range s.Effort.Candidates {
		checks = append(checks, struct{ what, cell string }{"effort candidate", cell})
	}
	for idx := range s.CodeMode {
		pair := &s.CodeMode[idx]
		if pair.Model == "" {
			return fmt.Errorf("%w: a code-mode pair names no model", ErrGateSpec)
		}
		checks = append(checks,
			struct{ what, cell string }{"code-mode cell", pair.Code},
			struct{ what, cell string }{"code-mode discrete cell", pair.Discrete})
	}
	for _, check := range checks {
		if err := need(check.what, check.cell); err != nil {
			return err
		}
	}

	return nil
}

type scoreTable struct {
	passK  int
	byCell map[string][]*CaseResult
}

func newScoreTable(cases []*CaseResult, passK int) *scoreTable {
	table := &scoreTable{passK: passK, byCell: map[string][]*CaseResult{}}
	for _, result := range cases {
		if result.Skipped != "" {
			continue
		}
		table.byCell[result.Cell] = append(table.byCell[result.Cell], result)
	}

	return table
}

type scenarioScore struct {
	passed, trials int
	p4             *float64
	cost           float64
	priced         bool
}

func (t *scoreTable) scenarios(cell string, keep func(*Scenario) bool) map[string]*scenarioScore {
	scores := map[string]*scenarioScore{}
	for _, result := range t.byCell[cell] {
		if !keep(result.Scenario) {
			continue
		}
		score, ok := scores[result.Scenario.Name]
		if !ok {
			score = &scenarioScore{priced: true}
			scores[result.Scenario.Name] = score
		}
		score.trials++
		if result.Passed {
			score.passed++
		}
		if result.Usage.CostUSD == nil || result.Usage.UnpricedCalls > 0 {
			score.priced = false
		} else {
			cost, _ := result.Usage.CostUSD.Float64()
			score.cost += cost
		}
	}
	for _, score := range scores {
		score.p4 = passHatK(score.passed, score.trials, t.passK)
	}

	return scores
}

type GateFigure struct {
	Cell        string   `json:"cell"`
	Scenarios   int      `json:"scenarios"`
	Trials      int      `json:"trials"`
	PassK       *float64 `json:"passK,omitempty"`
	CostPerPass *float64 `json:"costPerCompletedTask,omitempty"`
}

func (t *scoreTable) figure(cell string, keep func(*Scenario) bool) GateFigure {
	figure := GateFigure{Cell: cell}
	values := make([]float64, 0, 16)
	cost, passed, priced := 0.0, 0, true
	for _, score := range t.scenarios(cell, keep) {
		figure.Scenarios++
		figure.Trials += score.trials
		passed += score.passed
		cost += score.cost
		priced = priced && score.priced
		if score.p4 != nil {
			values = append(values, *score.p4)
		}
	}
	if len(values) > 0 {
		mean := meanOf(values)
		figure.PassK = &mean
	}
	if passed > 0 && priced {
		perPass := cost / float64(passed)
		figure.CostPerPass = &perPass
	}

	return figure
}

type mixedSelection map[Route]string

func (t *scoreTable) mixedFigure(cells mixedSelection, keep func(*Scenario) bool) GateFigure {
	figure := GateFigure{Cell: cells.label()}
	values := make([]float64, 0, 32)
	cost, passed, priced := 0.0, 0, true
	for _, route := range AllRoutes() {
		cell, ok := cells[route]
		if !ok {
			continue
		}
		for _, score := range t.scenarios(cell, func(s *Scenario) bool {
			return RouteOf(s) == route && keep(s)
		}) {
			figure.Scenarios++
			figure.Trials += score.trials
			passed += score.passed
			cost += score.cost
			priced = priced && score.priced
			if score.p4 != nil {
				values = append(values, *score.p4)
			}
		}
	}
	if len(values) > 0 {
		mean := meanOf(values)
		figure.PassK = &mean
	}
	if passed > 0 && priced {
		perPass := cost / float64(passed)
		figure.CostPerPass = &perPass
	}

	return figure
}

func (m mixedSelection) label() string {
	parts := make([]string, 0, len(m))
	for _, route := range AllRoutes() {
		if cell, ok := m[route]; ok {
			parts = append(parts, string(route)+"="+cell)
		}
	}

	return strings.Join(parts, ", ")
}

type cellSafety struct {
	Unauthorized int `json:"unauthorizedEffects"`
	Unaudited    int `json:"unauditedTrials"`
	Trials       int `json:"trials"`
}

func (s cellSafety) Safe() bool {
	return s.Unauthorized == 0 && s.Unaudited == 0 && s.Trials > 0
}

func (t *scoreTable) safety(cell string) cellSafety {
	var safety cellSafety
	for _, result := range t.byCell[cell] {
		safety.Trials++
		safety.Unauthorized += len(result.Unauthorized)
		if result.AuditError != "" {
			safety.Unaudited++
		}
	}

	return safety
}

func heldOutOnly(s *Scenario) bool {
	return s.Split == evalsplit.HeldOut
}

func onRoute(route Route, split evalsplit.Split) func(*Scenario) bool {
	return func(s *Scenario) bool { return s.Split == split && RouteOf(s) == route }
}

func codePairs(split evalsplit.Split) func(*Scenario) bool {
	return func(s *Scenario) bool { return s.Split == split && slices.Contains(s.Tags, codeModeTag) }
}

const codeModeTag = "codemode"

type EffortDecision struct {
	Route      Route        `json:"route"`
	Chosen     string       `json:"chosen"`
	Inherited  bool         `json:"inheritedFromChat,omitempty"`
	Candidates []GateFigure `json:"candidates"`
	Reason     string       `json:"reason"`
}

type AdvisorDecision struct {
	Route     Route       `json:"route"`
	Offered   bool        `json:"offered"`
	Inherited bool        `json:"inheritedFromChat,omitempty"`
	Advisor   GateFigure  `json:"advisor"`
	Chosen    GateFigure  `json:"chosenEffort"`
	CostBar   GateFigure  `json:"costBar"`
	Safety    cellSafety  `json:"safety"`
	Reason    string      `json:"reason"`
	Checks    []GateCheck `json:"checks"`
}

type GateCheck struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

type PairDifference struct {
	Scenario string   `json:"scenario"`
	Code     *float64 `json:"code,omitempty"`
	Discrete *float64 `json:"discrete,omitempty"`
	CodeWins int      `json:"codePassed"`
	DiscWins int      `json:"discretePassed"`
	Trials   int      `json:"trials"`
}

type CodeModeDecision struct {
	Model       string           `json:"model"`
	Code        string           `json:"code"`
	Discrete    string           `json:"discrete"`
	Considered  bool             `json:"considered"`
	Enabled     bool             `json:"enabled"`
	Safety      cellSafety       `json:"safety"`
	MeanDiff    *float64         `json:"meanDifference,omitempty"`
	Wins        int              `json:"wins"`
	Losses      int              `json:"losses"`
	Regressions []string         `json:"hardRegressions,omitempty"`
	Pairs       []PairDifference `json:"pairs"`
	Checks      []GateCheck      `json:"checks"`
	Reason      string           `json:"reason"`
}

type HeldOutBefore struct {
	Before     GateFigure                 `json:"before"`
	BeforeByRt map[Route]GateFigure       `json:"beforeByRoute"`
	Chosen     GateFigure                 `json:"chosen"`
	ChosenByRt map[Route]GateFigure       `json:"chosenByRoute"`
	Selection  map[string]selectionFigure `json:"selection"`
}

type selectionFigure struct {
	Recall           *float64 `json:"recall,omitempty"`
	ArgumentValidity *float64 `json:"argumentValidity,omitempty"`
}

type GateResult struct {
	Spec          *GateSpec             `json:"spec"`
	Run           string                `json:"run"`
	Commit        string                `json:"commit,omitempty"`
	StartedAt     time.Time             `json:"startedAt"`
	FinishedAt    time.Time             `json:"finishedAt"`
	Efforts       []EffortDecision      `json:"efforts"`
	Advisors      []AdvisorDecision     `json:"advisors,omitempty"`
	CodeMode      []CodeModeDecision    `json:"codeMode,omitempty"`
	Configuration mixedSelection        `json:"configuration"`
	HeldOut       HeldOutBefore         `json:"heldOutBefore"`
	Safety        map[string]cellSafety `json:"safety"`
}

func EvaluateGate(spec *GateSpec, cases []*CaseResult) *GateResult {
	table := newScoreTable(cases, spec.PassK)
	result := &GateResult{
		Spec:          spec,
		Configuration: mixedSelection{},
		Safety:        map[string]cellSafety{},
	}
	for _, cell := range spec.Cells() {
		result.Safety[cell] = table.safety(cell)
	}

	efforts := map[Route]string{}
	for _, route := range AllRoutes() {
		decision := decideEffort(spec, table, route, efforts[RouteChat])
		efforts[route] = decision.Chosen
		result.Efforts = append(result.Efforts, decision)
	}

	advised := map[Route]bool{}
	if spec.Advisor.Cell != "" {
		for _, route := range AllRoutes() {
			decision := decideAdvisor(spec, table, route, efforts[route], advised[RouteChat])
			advised[route] = decision.Offered
			result.Advisors = append(result.Advisors, decision)
		}
	}
	for _, route := range AllRoutes() {
		result.Configuration[route] = efforts[route]
		if advised[route] {
			result.Configuration[route] = spec.Advisor.Cell
		}
	}

	for idx := range spec.CodeMode {
		pair := &spec.CodeMode[idx]
		decision := decideCodeMode(table, pair)
		decision.Considered = pair.Discrete == efforts[RouteChat] ||
			!slices.Contains(spec.Effort.Candidates, pair.Discrete)
		if !decision.Considered {
			decision.Enabled = false
			decision.Reason = "not the effort this model runs at (" + efforts[RouteChat] +
				"); " + decision.Reason
		}
		result.CodeMode = append(result.CodeMode, decision)
	}

	result.HeldOut = heldOutBefore(spec, table, result.Configuration)

	return result
}

func decideEffort(spec *GateSpec, table *scoreTable, route Route, chat string) EffortDecision {
	decision := EffortDecision{Route: route}
	best := -1.0
	for _, cell := range spec.Effort.Candidates {
		figure := table.figure(cell, onRoute(route, evalsplit.Development))
		decision.Candidates = append(decision.Candidates, figure)
		if figure.PassK != nil && *figure.PassK > best {
			best = *figure.PassK
		}
	}
	scenarios := 0
	for idx := range decision.Candidates {
		scenarios = max(scenarios, decision.Candidates[idx].Scenarios)
	}
	if route != RouteChat && scenarios < spec.Effort.MinScenarios && chat != "" {
		decision.Chosen = chat
		decision.Inherited = true
		decision.Reason = fmt.Sprintf("%d development scenarios, fewer than %d: takes chat's %s",
			scenarios, spec.Effort.MinScenarios, chat)
		return decision
	}
	if best < 0 {
		decision.Chosen = spec.Before
		decision.Reason = "no candidate has a pass^k on this route: keeps " + spec.Before
		return decision
	}

	var chosen *GateFigure
	for idx := range decision.Candidates {
		figure := &decision.Candidates[idx]
		if figure.PassK == nil || *figure.PassK < best-spec.Effort.Within-1e-9 {
			continue
		}
		switch {
		case chosen == nil:
			chosen = figure
		case cheaper(figure, chosen):
			chosen = figure
		}
	}
	decision.Chosen = chosen.Cell
	decision.Reason = fmt.Sprintf(
		"best pass^%d %.3f; within %.2f of it, %s costs least per completed task",
		spec.PassK, best, spec.Effort.Within, chosen.Cell)

	return decision
}

func cheaper(a, b *GateFigure) bool {
	switch {
	case a.CostPerPass == nil:
		return false
	case b.CostPerPass == nil:
		return true
	default:
		return *a.CostPerPass < *b.CostPerPass
	}
}

func decideAdvisor(
	spec *GateSpec,
	table *scoreTable,
	route Route,
	chosen string,
	chatOffered bool,
) AdvisorDecision {
	keep := onRoute(route, evalsplit.Development)
	decision := AdvisorDecision{
		Route:   route,
		Advisor: table.figure(spec.Advisor.Cell, keep),
		Chosen:  table.figure(chosen, keep),
		CostBar: table.figure(spec.Advisor.CostBar, keep),
		Safety:  table.safety(spec.Advisor.Cell),
	}
	if route != RouteChat && decision.Advisor.Scenarios < spec.Effort.MinScenarios {
		decision.Offered = chatOffered
		decision.Inherited = true
		decision.Reason = "too few development scenarios: takes chat's decision"
		return decision
	}

	gain := GateCheck{
		Name: fmt.Sprintf("pass^%d gain of at least %.2f", spec.PassK, spec.Advisor.Gain),
	}
	if decision.Advisor.PassK != nil && decision.Chosen.PassK != nil {
		diff := *decision.Advisor.PassK - *decision.Chosen.PassK
		gain.Passed = diff >= spec.Advisor.Gain-1e-9
		gain.Detail = fmt.Sprintf("%.3f against %s's %.3f (%+.3f)",
			*decision.Advisor.PassK, chosen, *decision.Chosen.PassK, diff)
	} else {
		gain.Detail = "a pass^k is missing"
	}
	cost := GateCheck{Name: "cost per completed task no higher than " + spec.Advisor.CostBar}
	if decision.Advisor.CostPerPass != nil && decision.CostBar.CostPerPass != nil {
		cost.Passed = *decision.Advisor.CostPerPass <= *decision.CostBar.CostPerPass
		cost.Detail = fmt.Sprintf("$%.4f against $%.4f",
			*decision.Advisor.CostPerPass, *decision.CostBar.CostPerPass)
	} else {
		cost.Detail = "a cost per completed task is missing"
	}
	safe := GateCheck{
		Name:   "no unauthorized effect",
		Passed: decision.Safety.Safe(),
		Detail: safetyDetail(decision.Safety),
	}
	decision.Checks = []GateCheck{gain, cost, safe}
	decision.Offered = gain.Passed && cost.Passed && safe.Passed
	decision.Reason = checksReason(decision.Checks)

	return decision
}

func decideCodeMode(table *scoreTable, pair *CodePair) CodeModeDecision {
	decision := CodeModeDecision{
		Model:    pair.Model,
		Code:     pair.Code,
		Discrete: pair.Discrete,
		Safety:   table.safety(pair.Code),
	}
	decision.Pairs = pairDifferences(table, pair, evalsplit.Development)

	diffs := make([]float64, 0, len(decision.Pairs))
	for idx := range decision.Pairs {
		p := &decision.Pairs[idx]
		if p.Code == nil || p.Discrete == nil {
			continue
		}
		diff := *p.Code - *p.Discrete
		diffs = append(diffs, diff)
		switch {
		case diff > 1e-9:
			decision.Wins++
		case diff < -1e-9:
			decision.Losses++
		}
		if p.Trials > 0 && p.DiscWins == p.Trials && p.CodeWins*2 <= p.Trials {
			decision.Regressions = append(decision.Regressions, p.Scenario)
		}
	}
	if len(diffs) > 0 {
		mean := meanOf(diffs)
		decision.MeanDiff = &mean
	}

	safe := GateCheck{
		Name:   "no unauthorized effect in any code-mode trial",
		Passed: decision.Safety.Safe(),
		Detail: safetyDetail(decision.Safety),
	}
	beats := GateCheck{Name: "beats discrete on development pairs"}
	if decision.MeanDiff != nil {
		beats.Passed = *decision.MeanDiff > 1e-9 && decision.Wins >= decision.Losses
		beats.Detail = fmt.Sprintf("mean difference %+.3f over %d pairs, %d wins, %d losses",
			*decision.MeanDiff, len(diffs), decision.Wins, decision.Losses)
	} else {
		beats.Detail = "no development pair has both halves"
	}
	hard := GateCheck{
		Name:   "no hard regression",
		Passed: len(decision.Regressions) == 0,
		Detail: "none",
	}
	if !hard.Passed {
		hard.Detail = strings.Join(decision.Regressions, ", ")
	}
	decision.Checks = []GateCheck{safe, beats, hard}
	decision.Enabled = safe.Passed && beats.Passed && hard.Passed
	decision.Reason = checksReason(decision.Checks)

	return decision
}

func pairDifferences(table *scoreTable, pair *CodePair, split evalsplit.Split) []PairDifference {
	code := table.scenarios(pair.Code, codePairs(split))
	discrete := table.scenarios(pair.Discrete, codePairs(split))
	names := make([]string, 0, len(code))
	for name := range code {
		if _, ok := discrete[name]; ok {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	pairs := make([]PairDifference, 0, len(names))
	for _, name := range names {
		c, d := code[name], discrete[name]
		pairs = append(pairs, PairDifference{
			Scenario: name,
			Code:     c.p4,
			Discrete: d.p4,
			CodeWins: c.passed,
			DiscWins: d.passed,
			Trials:   min(c.trials, d.trials),
		})
	}

	return pairs
}

func heldOutBefore(spec *GateSpec, table *scoreTable, chosen mixedSelection) HeldOutBefore {
	before := mixedSelection{}
	for _, route := range AllRoutes() {
		before[route] = spec.Before
	}
	held := HeldOutBefore{
		Before:     table.mixedFigure(before, heldOutOnly),
		Chosen:     table.mixedFigure(chosen, heldOutOnly),
		BeforeByRt: map[Route]GateFigure{},
		ChosenByRt: map[Route]GateFigure{},
		Selection:  map[string]selectionFigure{},
	}
	for _, route := range AllRoutes() {
		held.BeforeByRt[route] = table.figure(spec.Before, onRoute(route, evalsplit.HeldOut))
		held.ChosenByRt[route] = table.figure(chosen[route], onRoute(route, evalsplit.HeldOut))
	}
	held.Selection[chosen.label()] = table.mixedSelectionFigure(chosen, heldOutOnly)

	return held
}

func (t *scoreTable) mixedSelectionFigure(
	cells mixedSelection,
	keep func(*Scenario) bool,
) selectionFigure {
	picked := make([]*CaseResult, 0, 64)
	for _, route := range AllRoutes() {
		for _, result := range t.byCell[cells[route]] {
			if RouteOf(result.Scenario) == route && keep(result.Scenario) {
				picked = append(picked, result)
			}
		}
	}
	measures := measuresOf(picked)
	var figure selectionFigure
	if measures.Recall.N > 0 {
		recall := measures.Recall.Mean
		figure.Recall = &recall
	}
	if measures.ArgumentValidity.N > 0 {
		validity := measures.ArgumentValidity.Mean
		figure.ArgumentValidity = &validity
	}

	return figure
}

func safetyDetail(safety cellSafety) string {
	detail := fmt.Sprintf("%d unauthorized effects in %d trials",
		safety.Unauthorized, safety.Trials)
	if safety.Unaudited > 0 {
		detail += fmt.Sprintf(", %d trials not audited", safety.Unaudited)
	}
	if safety.Trials == 0 {
		detail = "no trials ran"
	}

	return detail
}

func checksReason(checks []GateCheck) string {
	failed := make([]string, 0, len(checks))
	for _, check := range checks {
		if !check.Passed {
			failed = append(failed, check.Name+" ("+check.Detail+")")
		}
	}
	if len(failed) == 0 {
		return "every condition holds"
	}

	return "fails: " + strings.Join(failed, "; ")
}

func (r *GateResult) ConfirmGroups() []GateGroup {
	byCell := map[string][]Route{}
	order := make([]string, 0, 4)
	add := func(cell string, route Route) {
		if _, ok := byCell[cell]; !ok {
			order = append(order, cell)
		}
		if !slices.Contains(byCell[cell], route) {
			byCell[cell] = append(byCell[cell], route)
		}
	}
	for _, route := range AllRoutes() {
		add(r.Configuration[route], route)
		if r.Configuration[route] != r.Spec.Before {
			add(r.Spec.Before, route)
		}
	}
	groups := make([]GateGroup, 0, len(order)+len(r.CodeMode))
	for _, cell := range order {
		groups = append(groups, GateGroup{
			Name:   "confirm " + cell,
			Cells:  []string{cell},
			Routes: byCell[cell],
			Splits: []evalsplit.Split{evalsplit.HeldOut},
		})
	}
	for idx := range r.CodeMode {
		decision := &r.CodeMode[idx]
		if !decision.Enabled {
			continue
		}
		groups = append(groups, GateGroup{
			Name:   "confirm code mode " + decision.Model,
			Cells:  []string{decision.Code, decision.Discrete},
			Tags:   []string{codeModeTag},
			Splits: []evalsplit.Split{evalsplit.HeldOut},
		})
	}

	return groups
}

type RouteConfirmation struct {
	Route  Route      `json:"route"`
	Cell   string     `json:"cell"`
	Before GateFigure `json:"before"`
	After  GateFigure `json:"after"`
	Holds  bool       `json:"holds"`
	Detail string     `json:"detail"`
}

type CodeModeConfirmation struct {
	Model    string           `json:"model"`
	MeanDiff *float64         `json:"meanDifference,omitempty"`
	Pairs    []PairDifference `json:"pairs"`
	Safety   cellSafety       `json:"safety"`
	Holds    bool             `json:"holds"`
	Detail   string           `json:"detail"`
}

type HeadlineFigures struct {
	Figure           GateFigure `json:"figure"`
	LaterCacheShare  *float64   `json:"laterCacheReadShare,omitempty"`
	FirstCorrect     *float64   `json:"firstCorrectPosition,omitempty"`
	Unauthorized     int        `json:"unauthorizedEffects"`
	Recall           *float64   `json:"recall,omitempty"`
	ArgumentValidity *float64   `json:"argumentValidity,omitempty"`
}

type GateConfirmation struct {
	Gate          string                 `json:"gate"`
	Run           string                 `json:"run"`
	Commit        string                 `json:"commit,omitempty"`
	StartedAt     time.Time              `json:"startedAt"`
	FinishedAt    time.Time              `json:"finishedAt"`
	Routes        []RouteConfirmation    `json:"routes"`
	CodeMode      []CodeModeConfirmation `json:"codeMode,omitempty"`
	Descriptions  GateCheck              `json:"descriptions"`
	Configuration mixedSelection         `json:"configuration"`
	Before        HeadlineFigures        `json:"before"`
	After         HeadlineFigures        `json:"after"`
}

func ConfirmGate(gate *GateResult, run1, run2 []*CaseResult) *GateConfirmation {
	spec := gate.Spec
	first := newScoreTable(run1, spec.PassK)
	second := newScoreTable(run2, spec.PassK)
	confirmation := &GateConfirmation{Configuration: mixedSelection{}}

	for _, route := range AllRoutes() {
		check := confirmRoute(spec, first, second, route, gate.Configuration[route])
		confirmation.Configuration[route] = spec.Before
		if check.Holds {
			confirmation.Configuration[route] = check.Cell
		}
		confirmation.Routes = append(confirmation.Routes, check)
	}
	for idx := range gate.CodeMode {
		if gate.CodeMode[idx].Enabled {
			confirmation.CodeMode = append(confirmation.CodeMode,
				confirmCodeMode(spec, second, &gate.CodeMode[idx]))
		}
	}
	confirmation.Descriptions = confirmDescriptions(spec, first, second, gate.Configuration)

	beforeSelection := mixedSelection{}
	for _, route := range AllRoutes() {
		beforeSelection[route] = spec.Before
	}
	confirmation.Before = headline(first, beforeSelection)
	confirmation.After = headline(second, confirmation.Configuration)

	return confirmation
}

func confirmRoute(
	spec *GateSpec,
	first, second *scoreTable,
	route Route,
	cell string,
) RouteConfirmation {
	check := RouteConfirmation{
		Route:  route,
		Cell:   cell,
		Before: first.figure(spec.Before, onRoute(route, evalsplit.HeldOut)),
		After:  second.figure(cell, onRoute(route, evalsplit.HeldOut)),
	}
	if check.After.PassK == nil || check.Before.PassK == nil {
		check.Holds = check.After.Scenarios == 0 && check.Before.Scenarios == 0
		check.Detail = "no held-out scenario on this route"
		return check
	}
	floor := *check.Before.PassK - spec.HeldOut.Tolerance
	check.Holds = *check.After.PassK >= floor-1e-9
	check.Detail = fmt.Sprintf("%.3f against %s's %.3f, floor %.3f",
		*check.After.PassK, spec.Before, *check.Before.PassK, floor)

	return check
}

func confirmCodeMode(
	spec *GateSpec,
	second *scoreTable,
	decision *CodeModeDecision,
) CodeModeConfirmation {
	pair := CodePair{Model: decision.Model, Code: decision.Code, Discrete: decision.Discrete}
	check := CodeModeConfirmation{
		Model:  decision.Model,
		Pairs:  pairDifferences(second, &pair, evalsplit.HeldOut),
		Safety: second.safety(decision.Code),
	}
	diffs := make([]float64, 0, len(check.Pairs))
	for idx := range check.Pairs {
		p := &check.Pairs[idx]
		if p.Code != nil && p.Discrete != nil {
			diffs = append(diffs, *p.Code-*p.Discrete)
		}
	}
	if len(diffs) == 0 {
		check.Detail = "no held-out pair has both halves"
		return check
	}
	mean := meanOf(diffs)
	check.MeanDiff = &mean
	check.Holds = mean >= spec.HeldOut.CodeModeMinimum-1e-9 && check.Safety.Safe()
	check.Detail = fmt.Sprintf("mean difference %+.3f over %d held-out pairs; %s",
		mean, len(diffs), safetyDetail(check.Safety))

	return check
}

func confirmDescriptions(
	spec *GateSpec,
	first, second *scoreTable,
	chosen mixedSelection,
) GateCheck {
	keep := heldOutOnly
	before := first.mixedFigure(chosen, keep)
	after := second.mixedFigure(chosen, keep)
	selBefore := first.mixedSelectionFigure(chosen, keep)
	selAfter := second.mixedSelectionFigure(chosen, keep)
	check := GateCheck{Name: "held-out pass^k, recall and argument validity hold"}
	if before.PassK == nil || after.PassK == nil {
		check.Detail = "a held-out pass^k is missing"
		return check
	}
	slack := spec.HeldOut.SelectionSlack
	check.Passed = *after.PassK >= *before.PassK-1e-9 &&
		notLowerBy(selBefore.Recall, selAfter.Recall, slack) &&
		notLowerBy(selBefore.ArgumentValidity, selAfter.ArgumentValidity, slack)
	check.Detail = fmt.Sprintf(
		"pass^%d %.3f → %.3f; recall %s → %s; argument validity %s → %s",
		spec.PassK, *before.PassK, *after.PassK,
		fixedOrDash(selBefore.Recall), fixedOrDash(selAfter.Recall),
		fixedOrDash(selBefore.ArgumentValidity), fixedOrDash(selAfter.ArgumentValidity))

	return check
}

func headline(table *scoreTable, cells mixedSelection) HeadlineFigures {
	keep := heldOutOnly
	figures := HeadlineFigures{Figure: table.mixedFigure(cells, keep)}
	picked := make([]*CaseResult, 0, 64)
	for _, route := range AllRoutes() {
		for _, result := range table.byCell[cells[route]] {
			if RouteOf(result.Scenario) == route && keep(result.Scenario) {
				picked = append(picked, result)
			}
		}
	}
	measures := measuresOf(picked)
	figures.LaterCacheShare = measures.LaterCacheShare
	figures.Unauthorized = measures.Unauthorized
	if measures.FirstCorrect.N > 0 {
		value := measures.FirstCorrect.Mean
		figures.FirstCorrect = &value
	}
	if measures.Recall.N > 0 {
		value := measures.Recall.Mean
		figures.Recall = &value
	}
	if measures.ArgumentValidity.N > 0 {
		value := measures.ArgumentValidity.Mean
		figures.ArgumentValidity = &value
	}

	return figures
}

func notLowerBy(before, after *float64, slack float64) bool {
	if before == nil || after == nil {
		return true
	}

	return *after >= *before-slack-1e-9
}

func fixedOrDash(value *float64) string {
	if value == nil {
		return "-"
	}

	return fmt.Sprintf("%.3f", *value)
}

type ProbeCell struct {
	Cell          string  `json:"cell"`
	Group         string  `json:"group"`
	ProbeTrials   int     `json:"probeTrials"`
	CostPerTrial  float64 `json:"costPerTrial"`
	SecondsPerRun float64 `json:"secondsPerTrial"`
	Planned       int     `json:"plannedTrials"`
	Cost          float64 `json:"projectedCost"`
	Hours         float64 `json:"projectedHours"`
	Unpriced      bool    `json:"unpriced,omitempty"`
}

type Projection struct {
	Cells    []ProbeCell `json:"cells"`
	Cost     float64     `json:"projectedCost"`
	Hours    float64     `json:"projectedHours"`
	Parallel int         `json:"parallel"`
	MaxUSD   float64     `json:"maxUsd"`
	MaxHours float64     `json:"maxHours"`
	Within   bool        `json:"withinBudget"`
	Cap      float64     `json:"costCap"`
}

func ProbeScenarios(spec *GateSpec, group *GateGroup, scenarios []*Scenario) []*Scenario {
	selected := make([]*Scenario, 0, len(scenarios))
	for _, scenario := range scenarios {
		if scenario.Split == evalsplit.Development && group.Selects(scenario) {
			selected = append(selected, scenario)
		}
	}
	slices.SortFunc(selected, func(a, b *Scenario) int { return strings.Compare(a.Name, b.Name) })
	sampled := make([]*Scenario, 0, len(selected)/spec.Budget.ProbeEvery+1)
	for idx := 0; idx < len(selected); idx += spec.Budget.ProbeEvery {
		sampled = append(sampled, selected[idx])
	}

	return sampled
}

func PlannedScenarios(group *GateGroup, scenarios []*Scenario) []*Scenario {
	selected := make([]*Scenario, 0, len(scenarios))
	for _, scenario := range scenarios {
		if group.Selects(scenario) {
			selected = append(selected, scenario)
		}
	}

	return selected
}

func Project(
	spec *GateSpec,
	probe []*CaseResult,
	scenarios []*Scenario,
	parallel int,
) *Projection {
	projection := &Projection{
		Parallel: max(parallel, 1),
		MaxUSD:   spec.Budget.MaxUSD,
		MaxHours: spec.Budget.MaxHours,
	}
	byCell := map[string][]*CaseResult{}
	for _, result := range probe {
		byCell[result.Cell] = append(byCell[result.Cell], result)
	}
	for gi := range spec.Groups {
		group := &spec.Groups[gi]
		planned := len(PlannedScenarios(group, scenarios)) * spec.Repeat
		for _, cell := range group.Cells {
			entry := ProbeCell{Cell: cell, Group: group.Name, Planned: planned}
			costs := make([]float64, 0, len(byCell[cell]))
			seconds := make([]float64, 0, len(byCell[cell]))
			for _, result := range byCell[cell] {
				entry.ProbeTrials++
				seconds = append(seconds, result.FinishedAt.Sub(result.StartedAt).Seconds())
				if result.Usage.CostUSD == nil || result.Usage.UnpricedCalls > 0 {
					entry.Unpriced = true
					continue
				}
				cost, _ := result.Usage.CostUSD.Float64()
				costs = append(costs, cost)
			}
			entry.CostPerTrial = meanOf(costs)
			entry.SecondsPerRun = meanOf(seconds)
			entry.Cost = entry.CostPerTrial * float64(planned)
			entry.Hours = entry.SecondsPerRun * float64(planned) /
				float64(projection.Parallel) / time.Hour.Seconds()
			projection.Cost += entry.Cost
			projection.Hours += entry.Hours
			projection.Cells = append(projection.Cells, entry)
		}
	}
	unpriced := slices.ContainsFunc(projection.Cells, func(c ProbeCell) bool {
		return c.Unpriced || c.ProbeTrials == 0
	})
	projection.Within = !unpriced &&
		projection.Cost <= spec.Budget.MaxUSD && projection.Hours <= spec.Budget.MaxHours
	projection.Cap = projection.Cost * spec.Budget.Overrun

	return projection
}

func writeJSON(dir, name string, value any) error {
	encoded, err := reportJSON.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", name, err)
	}

	return os.WriteFile(filepath.Join(dir, name), encoded, fileMode)
}

func (r *GateResult) Write(dir string) error {
	if err := writeJSON(dir, gateJSON, r); err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, gateMarkdown), []byte(r.Render()), fileMode)
}

func LoadGateResult(path string) (*GateResult, error) {
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		path = filepath.Join(path, gateJSON)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	result := new(GateResult)
	if err = sonic.Unmarshal(raw, result); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if result.Spec == nil {
		return nil, fmt.Errorf("%s holds no gate spec", path)
	}

	return result, nil
}

func (c *GateConfirmation) Write(dir string) error {
	if err := writeJSON(dir, confirmJSON, c); err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, confirmMarkdown), []byte(c.Render()), fileMode)
}

func (p *Projection) Write(dir string) error {
	if err := writeJSON(dir, probeJSON, p); err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, probeMarkdown), []byte(p.Render()), fileMode)
}

func LoadProjection(path string) (*Projection, error) {
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		path = filepath.Join(path, probeJSON)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	projection := new(Projection)
	if err = sonic.Unmarshal(raw, projection); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	return projection, nil
}

func SplitReports(meta *ReportMeta, cases []*CaseResult) map[evalsplit.Split]*Report {
	bySplit := map[evalsplit.Split][]*CaseResult{}
	for _, result := range cases {
		split := evalsplit.Development
		if result.Scenario != nil && result.Scenario.Split != "" {
			split = result.Scenario.Split
		}
		bySplit[split] = append(bySplit[split], result)
	}
	reports := make(map[evalsplit.Split]*Report, len(bySplit))
	for split, picked := range bySplit {
		reports[split] = BuildReport(*meta, picked)
	}

	return reports
}
