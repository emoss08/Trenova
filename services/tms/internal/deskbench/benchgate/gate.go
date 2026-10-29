package benchgate

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/deskbench/benchrecord"
	"github.com/emoss08/trenova/internal/deskbench/benchreport"
	"github.com/emoss08/trenova/internal/deskbench/benchresult"
	"github.com/emoss08/trenova/internal/deskbench/benchscenario"
	"github.com/emoss08/trenova/internal/deskbench/benchstats"
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
)

type GateSpec struct {
	Repeat  int         `yaml:"repeat"  json:"repeat"`
	PassK   int         `yaml:"passK"   json:"passK"`
	Before  string      `yaml:"before"  json:"before"`
	Budget  GateBudget  `yaml:"budget"  json:"budget"`
	Groups  []GateGroup `yaml:"groups"  json:"groups"`
	Effort  EffortRule  `yaml:"effort"  json:"effort"`
	Advisor AdvisorRule `yaml:"advisor" json:"advisor"`
	HeldOut HeldOutRule `yaml:"heldOut" json:"heldOut"`

	CodeMode CodeModeRule `yaml:"codeMode" json:"codeMode"`
}

type GateBudget struct {
	MaxUSD     float64 `yaml:"maxUsd"     json:"maxUsd"`
	MaxHours   float64 `yaml:"maxHours"   json:"maxHours"`
	Overrun    float64 `yaml:"overrun"    json:"overrun"`
	ProbeEvery int     `yaml:"probeEvery" json:"probeEvery"`
}

type GateGroup struct {
	Name          string                `yaml:"name"          json:"name"`
	Cells         []string              `yaml:"cells"         json:"cells"`
	Tags          []string              `yaml:"tags"          json:"tags,omitempty"`
	Routes        []benchscenario.Route `yaml:"routes"        json:"routes,omitempty"`
	Splits        []evalsplit.Split     `yaml:"splits"        json:"splits,omitempty"`
	AdvisorRoutes bool                  `yaml:"advisorRoutes" json:"advisorRoutes,omitempty"`
}

func (g *GateGroup) Selects(scenario *benchscenario.Scenario) bool {
	if !scenario.HasTag(g.Tags) {
		return false
	}
	if len(g.Routes) > 0 && !slices.Contains(g.Routes, benchscenario.RouteOf(scenario)) {
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
	Cell      string  `yaml:"cell"      json:"cell,omitempty"`
	Reference string  `yaml:"reference" json:"reference,omitempty"`
	Gap       float64 `yaml:"gap"       json:"gap"`
	Gain      float64 `yaml:"gain"      json:"gain"`
}

type HeldOutRule struct {
	Tolerance      float64 `yaml:"tolerance"      json:"tolerance"`
	SelectionSlack float64 `yaml:"selectionSlack" json:"selectionSlack"`
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
		s.Repeat = benchstats.DefaultPassK
	}
	if s.PassK <= 0 {
		s.PassK = benchstats.DefaultPassK
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

func (s *GateSpec) advisorGrouped() bool {
	return slices.ContainsFunc(s.Groups, func(g GateGroup) bool {
		return g.AdvisorRoutes && slices.Contains(g.Cells, s.Advisor.Cell)
	})
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
	case s.Advisor.Cell != "" && s.Advisor.Reference == "":
		return fmt.Errorf("%w: the advisor needs a reference cell", ErrGateSpec)
	case s.Advisor.Cell != "" && !s.advisorGrouped():
		return fmt.Errorf("%w: the advisor cell runs in a group with advisorRoutes", ErrGateSpec)
	}

	checks := []struct{ what, cell string }{
		{"before", s.Before},
		{"advisor cell", s.Advisor.Cell},
		{"advisor reference", s.Advisor.Reference},
	}
	for _, cell := range s.Effort.Candidates {
		checks = append(checks, struct{ what, cell string }{"effort candidate", cell})
	}
	for _, check := range checks {
		if err := need(check.what, check.cell); err != nil {
			return err
		}
	}

	return s.CodeMode.validate(cells)
}

type scoreTable struct {
	passK  int
	byCell map[string][]*benchresult.CaseResult
}

func newScoreTable(cases []*benchresult.CaseResult, passK int) *scoreTable {
	table := &scoreTable{passK: passK, byCell: map[string][]*benchresult.CaseResult{}}
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

func (t *scoreTable) scenarios(
	cell string,
	keep func(*benchscenario.Scenario) bool,
) map[string]*scenarioScore {
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
		score.p4 = benchstats.PassHatK(score.passed, score.trials, t.passK)
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

func (t *scoreTable) figure(cell string, keep func(*benchscenario.Scenario) bool) GateFigure {
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
		mean := benchstats.MeanOf(values)
		figure.PassK = &mean
	}
	if passed > 0 && priced {
		perPass := cost / float64(passed)
		figure.CostPerPass = &perPass
	}

	return figure
}

type mixedSelection map[benchscenario.Route]string

func (t *scoreTable) mixedFigure(
	cells mixedSelection,
	keep func(*benchscenario.Scenario) bool,
) GateFigure {
	figure := GateFigure{Cell: cells.label()}
	values := make([]float64, 0, 32)
	cost, passed, priced := 0.0, 0, true
	for _, route := range benchscenario.AllRoutes() {
		cell, ok := cells[route]
		if !ok {
			continue
		}
		for _, score := range t.scenarios(cell, func(s *benchscenario.Scenario) bool {
			return benchscenario.RouteOf(s) == route && keep(s)
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
		mean := benchstats.MeanOf(values)
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
	for _, route := range benchscenario.AllRoutes() {
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

func heldOutOnly(s *benchscenario.Scenario) bool {
	return s.Split == evalsplit.HeldOut && !scriptHalf(s)
}

func onRoute(route benchscenario.Route, split evalsplit.Split) func(*benchscenario.Scenario) bool {
	return func(s *benchscenario.Scenario) bool {
		return s.Split == split && benchscenario.RouteOf(s) == route && !scriptHalf(s)
	}
}

type EffortDecision struct {
	Route      benchscenario.Route `json:"route"`
	Chosen     string              `json:"chosen"`
	Inherited  bool                `json:"inheritedFromChat,omitempty"`
	Candidates []GateFigure        `json:"candidates"`
	Reason     string              `json:"reason"`
}

type AdvisorDecision struct {
	Route     benchscenario.Route `json:"route"`
	Offered   bool                `json:"offered"`
	Inherited bool                `json:"inheritedFromChat,omitempty"`
	Advisor   GateFigure          `json:"advisor"`
	Chosen    GateFigure          `json:"chosenEffort"`
	Reference GateFigure          `json:"reference"`
	Eligible  bool                `json:"eligible"`
	Safety    cellSafety          `json:"safety"`
	Reason    string              `json:"reason"`
	Checks    []GateCheck         `json:"checks"`
}

type GateCheck struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

type HeldOutBefore struct {
	Before     GateFigure                         `json:"before"`
	BeforeByRt map[benchscenario.Route]GateFigure `json:"beforeByRoute"`
	Chosen     GateFigure                         `json:"chosen"`
	ChosenByRt map[benchscenario.Route]GateFigure `json:"chosenByRoute"`
	Selection  map[string]selectionFigure         `json:"selection"`
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
	Configuration mixedSelection        `json:"configuration"`
	HeldOut       HeldOutBefore         `json:"heldOutBefore"`
	Safety        map[string]cellSafety `json:"safety"`
	Models        map[string]CellModel  `json:"models"`
	CodeMode      []CodeModeDecision    `json:"codeMode,omitempty"`
}

func chooseEfforts(spec *GateSpec, table *scoreTable) []EffortDecision {
	decisions := make([]EffortDecision, 0, len(benchscenario.AllRoutes()))
	chat := ""
	for _, route := range benchscenario.AllRoutes() {
		decision := decideEffort(spec, table, route, chat)
		if route == benchscenario.RouteChat {
			chat = decision.Chosen
		}
		decisions = append(decisions, decision)
	}

	return decisions
}

func AdvisorRoutes(spec *GateSpec, cases []*benchresult.CaseResult) []benchscenario.Route {
	if spec.Advisor.Cell == "" {
		return nil
	}
	table := newScoreTable(cases, spec.PassK)
	routes := make([]benchscenario.Route, 0, len(benchscenario.AllRoutes()))
	chatEligible := false
	for _, decision := range chooseEfforts(spec, table) {
		eligible, _ := advisorEligible(spec, table, decision.Route, decision.Chosen)
		if decision.Inherited {
			eligible = chatEligible
		}
		if decision.Route == benchscenario.RouteChat {
			chatEligible = eligible
		}
		if eligible {
			routes = append(routes, decision.Route)
		}
	}

	return routes
}

func advisorEligible(
	spec *GateSpec,
	table *scoreTable,
	route benchscenario.Route,
	chosen string,
) (bool, GateCheck) {
	keep := onRoute(route, evalsplit.Development)
	haiku := table.figure(chosen, keep)
	reference := table.figure(spec.Advisor.Reference, keep)
	check := GateCheck{Name: fmt.Sprintf("%s at least %.2f below %s", chosen, spec.Advisor.Gap,
		spec.Advisor.Reference)}
	if haiku.PassK == nil || reference.PassK == nil {
		check.Detail = "a pass^k is missing"
		return false, check
	}
	gap := *reference.PassK - *haiku.PassK
	check.Passed = gap >= spec.Advisor.Gap-1e-9
	check.Detail = fmt.Sprintf("%.3f against %.3f (gap %.3f)", *haiku.PassK, *reference.PassK, gap)

	return check.Passed, check
}

func EvaluateGate(spec *GateSpec, cases []*benchresult.CaseResult) *GateResult {
	table := newScoreTable(cases, spec.PassK)
	result := &GateResult{
		Spec:          spec,
		Configuration: mixedSelection{},
		Safety:        map[string]cellSafety{},
		Models:        map[string]CellModel{},
	}
	for _, cell := range spec.Cells() {
		result.Safety[cell] = table.safety(cell)
		result.Models[cell] = table.model(cell)
	}
	result.CodeMode = evaluateCodeMode(spec, table)

	efforts := map[benchscenario.Route]string{}
	result.Efforts = chooseEfforts(spec, table)
	for _, decision := range result.Efforts {
		efforts[decision.Route] = decision.Chosen
	}

	advised := map[benchscenario.Route]bool{}
	if spec.Advisor.Cell != "" {
		for _, route := range benchscenario.AllRoutes() {
			decision := decideAdvisor(
				spec,
				table,
				route,
				efforts[route],
				advised[benchscenario.RouteChat],
			)
			advised[route] = decision.Offered
			result.Advisors = append(result.Advisors, decision)
		}
	}
	for _, route := range benchscenario.AllRoutes() {
		result.Configuration[route] = efforts[route]
		if advised[route] {
			result.Configuration[route] = spec.Advisor.Cell
		}
	}

	result.HeldOut = heldOutBefore(spec, table, result.Configuration)

	return result
}

func decideEffort(
	spec *GateSpec,
	table *scoreTable,
	route benchscenario.Route,
	chat string,
) EffortDecision {
	decision := EffortDecision{Route: route}
	best := -1.0
	scenarios := 0
	for _, cell := range spec.Effort.Candidates {
		figure := table.figure(cell, onRoute(route, evalsplit.Development))
		decision.Candidates = append(decision.Candidates, figure)
		scenarios = max(scenarios, figure.Scenarios)
		if figure.PassK != nil && *figure.PassK > best {
			best = *figure.PassK
		}
	}
	if route != benchscenario.RouteChat && scenarios < spec.Effort.MinScenarios && chat != "" {
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

	for idx := range decision.Candidates {
		figure := &decision.Candidates[idx]
		if figure.PassK != nil && *figure.PassK >= best-spec.Effort.Within-1e-9 {
			decision.Chosen = figure.Cell
			break
		}
	}
	decision.Reason = fmt.Sprintf("best pass^%d %.3f; %s is the lowest effort within %.2f of it",
		spec.PassK, best, decision.Chosen, spec.Effort.Within)

	return decision
}

func decideAdvisor(
	spec *GateSpec,
	table *scoreTable,
	route benchscenario.Route,
	chosen string,
	chatOffered bool,
) AdvisorDecision {
	keep := onRoute(route, evalsplit.Development)
	decision := AdvisorDecision{
		Route:     route,
		Advisor:   table.figure(spec.Advisor.Cell, keep),
		Chosen:    table.figure(chosen, keep),
		Reference: table.figure(spec.Advisor.Reference, keep),
		Safety:    table.safety(spec.Advisor.Cell),
	}
	if route != benchscenario.RouteChat && decision.Chosen.Scenarios < spec.Effort.MinScenarios {
		decision.Offered = chatOffered
		decision.Inherited = true
		decision.Reason = "too few development scenarios: takes chat's decision"
		return decision
	}

	eligible, gap := advisorEligible(spec, table, route, chosen)
	decision.Eligible = eligible
	if !eligible {
		decision.Checks = []GateCheck{gap}
		decision.Reason = "not tried: " + gap.Name + " (" + gap.Detail + ")"
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
	cost := GateCheck{Name: "cost per completed task no higher than " + spec.Advisor.Reference}
	if decision.Advisor.CostPerPass != nil && decision.Reference.CostPerPass != nil {
		cost.Passed = *decision.Advisor.CostPerPass <= *decision.Reference.CostPerPass
		cost.Detail = fmt.Sprintf("$%.4f against $%.4f",
			*decision.Advisor.CostPerPass, *decision.Reference.CostPerPass)
	} else {
		cost.Detail = "a cost per completed task is missing"
	}
	safe := GateCheck{
		Name:   "no unauthorized effect",
		Passed: decision.Safety.Safe(),
		Detail: safetyDetail(decision.Safety),
	}
	decision.Checks = []GateCheck{gap, gain, cost, safe}
	decision.Offered = gain.Passed && cost.Passed && safe.Passed
	decision.Reason = checksReason(decision.Checks)

	return decision
}

func heldOutBefore(spec *GateSpec, table *scoreTable, chosen mixedSelection) HeldOutBefore {
	before := mixedSelection{}
	for _, route := range benchscenario.AllRoutes() {
		before[route] = spec.Before
	}
	held := HeldOutBefore{
		Before:     table.mixedFigure(before, heldOutOnly),
		Chosen:     table.mixedFigure(chosen, heldOutOnly),
		BeforeByRt: map[benchscenario.Route]GateFigure{},
		ChosenByRt: map[benchscenario.Route]GateFigure{},
		Selection:  map[string]selectionFigure{},
	}
	for _, route := range benchscenario.AllRoutes() {
		held.BeforeByRt[route] = table.figure(spec.Before, onRoute(route, evalsplit.HeldOut))
		held.ChosenByRt[route] = table.figure(chosen[route], onRoute(route, evalsplit.HeldOut))
	}
	held.Selection[chosen.label()] = table.mixedSelectionFigure(chosen, heldOutOnly)

	return held
}

func (t *scoreTable) mixedSelectionFigure(
	cells mixedSelection,
	keep func(*benchscenario.Scenario) bool,
) selectionFigure {
	picked := make([]*benchresult.CaseResult, 0, 64)
	for _, route := range benchscenario.AllRoutes() {
		for _, result := range t.byCell[cells[route]] {
			if benchscenario.RouteOf(result.Scenario) == route && keep(result.Scenario) {
				picked = append(picked, result)
			}
		}
	}
	measures := benchreport.MeasuresOf(picked)
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
	byCell := map[string][]benchscenario.Route{}
	order := make([]string, 0, 4)
	add := func(cell string, route benchscenario.Route) {
		if _, ok := byCell[cell]; !ok {
			order = append(order, cell)
		}
		if !slices.Contains(byCell[cell], route) {
			byCell[cell] = append(byCell[cell], route)
		}
	}
	for _, route := range benchscenario.AllRoutes() {
		add(r.Configuration[route], route)
		if r.Configuration[route] != r.Spec.Before {
			add(r.Spec.Before, route)
		}
	}
	groups := make([]GateGroup, 0, len(order))
	for _, cell := range order {
		groups = append(groups, GateGroup{
			Name:   "confirm " + cell,
			Cells:  []string{cell},
			Routes: byCell[cell],
			Splits: []evalsplit.Split{evalsplit.HeldOut},
		})
	}

	return append(groups, r.codeModeConfirmGroups(byCell)...)
}

type RouteConfirmation struct {
	Route  benchscenario.Route `json:"route"`
	Cell   string              `json:"cell"`
	Before GateFigure          `json:"before"`
	After  GateFigure          `json:"after"`
	Holds  bool                `json:"holds"`
	Detail string              `json:"detail"`
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
	Gate          string              `json:"gate"`
	Run           string              `json:"run"`
	Commit        string              `json:"commit,omitempty"`
	StartedAt     time.Time           `json:"startedAt"`
	FinishedAt    time.Time           `json:"finishedAt"`
	Routes        []RouteConfirmation `json:"routes"`
	Descriptions  GateCheck           `json:"descriptions"`
	Configuration mixedSelection      `json:"configuration"`
	Before        HeadlineFigures     `json:"before"`
	After         HeadlineFigures     `json:"after"`
	CodeMode      []CodeModeRecheck   `json:"codeMode,omitempty"`
}

func ConfirmGate(gate *GateResult, run1, run2 []*benchresult.CaseResult) *GateConfirmation {
	spec := gate.Spec
	first := newScoreTable(run1, spec.PassK)
	second := newScoreTable(run2, spec.PassK)
	confirmation := &GateConfirmation{Configuration: mixedSelection{}}

	for _, route := range benchscenario.AllRoutes() {
		check := confirmRoute(spec, first, second, route, gate.Configuration[route])
		confirmation.Configuration[route] = spec.Before
		if check.Holds {
			confirmation.Configuration[route] = check.Cell
		}
		confirmation.Routes = append(confirmation.Routes, check)
	}
	confirmation.Descriptions = confirmDescriptions(spec, first, second, gate.Configuration)

	beforeSelection := mixedSelection{}
	for _, route := range benchscenario.AllRoutes() {
		beforeSelection[route] = spec.Before
	}
	confirmation.Before = headline(first, beforeSelection)
	confirmation.After = headline(second, confirmation.Configuration)
	confirmation.CodeMode = recheckCodeMode(gate, second)

	return confirmation
}

func confirmRoute(
	spec *GateSpec,
	first, second *scoreTable,
	route benchscenario.Route,
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

func confirmDescriptions(
	spec *GateSpec,
	first, second *scoreTable,
	chosen mixedSelection,
) GateCheck {
	keep := sharedHeldOut(first, second, chosen)
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

func sharedHeldOut(
	first, second *scoreTable,
	chosen mixedSelection,
) func(*benchscenario.Scenario) bool {
	shared := map[benchscenario.Route]map[string]bool{}
	for _, route := range benchscenario.AllRoutes() {
		cell, ok := chosen[route]
		if !ok {
			continue
		}
		later := second.scenarios(cell, onRoute(route, evalsplit.HeldOut))
		names := map[string]bool{}
		for name := range first.scenarios(cell, onRoute(route, evalsplit.HeldOut)) {
			if _, both := later[name]; both {
				names[name] = true
			}
		}
		shared[route] = names
	}

	return func(s *benchscenario.Scenario) bool {
		return heldOutOnly(s) && shared[benchscenario.RouteOf(s)][s.Name]
	}
}

func headline(table *scoreTable, cells mixedSelection) HeadlineFigures {
	keep := heldOutOnly
	figures := HeadlineFigures{Figure: table.mixedFigure(cells, keep)}
	picked := make([]*benchresult.CaseResult, 0, 64)
	for _, route := range benchscenario.AllRoutes() {
		for _, result := range table.byCell[cells[route]] {
			if benchscenario.RouteOf(result.Scenario) == route && keep(result.Scenario) {
				picked = append(picked, result)
			}
		}
	}
	measures := benchreport.MeasuresOf(picked)
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

func ProbeScenarios(
	spec *GateSpec,
	group *GateGroup,
	scenarios []*benchscenario.Scenario,
) []*benchscenario.Scenario {
	selected := make([]*benchscenario.Scenario, 0, len(scenarios))
	for _, scenario := range scenarios {
		if scenario.Split == evalsplit.Development && group.Selects(scenario) {
			selected = append(selected, scenario)
		}
	}
	slices.SortFunc(
		selected,
		func(a, b *benchscenario.Scenario) int { return strings.Compare(a.Name, b.Name) },
	)
	sampled := make([]*benchscenario.Scenario, 0, len(selected)/spec.Budget.ProbeEvery+1)
	for idx := 0; idx < len(selected); idx += spec.Budget.ProbeEvery {
		sampled = append(sampled, selected[idx])
	}

	return sampled
}

func PlannedScenarios(
	group *GateGroup,
	scenarios []*benchscenario.Scenario,
) []*benchscenario.Scenario {
	selected := make([]*benchscenario.Scenario, 0, len(scenarios))
	for _, scenario := range scenarios {
		if group.Selects(scenario) {
			selected = append(selected, scenario)
		}
	}

	return selected
}

func Project(
	spec *GateSpec,
	probe []*benchresult.CaseResult,
	scenarios []*benchscenario.Scenario,
	parallel int,
) *Projection {
	projection := &Projection{
		Parallel: max(parallel, 1),
		MaxUSD:   spec.Budget.MaxUSD,
		MaxHours: spec.Budget.MaxHours,
	}
	byCell := map[string][]*benchresult.CaseResult{}
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
			entry.CostPerTrial = benchstats.MeanOf(costs)
			entry.SecondsPerRun = benchstats.MeanOf(seconds)
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

func (r *GateResult) Write(dir string) error {
	if err := benchrecord.WriteJSON(dir, gateJSON, r); err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, gateMarkdown), []byte(r.Render()), benchrecord.FileMode)
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
	if err := benchrecord.WriteJSON(dir, confirmJSON, c); err != nil {
		return err
	}

	return os.WriteFile(
		filepath.Join(dir, confirmMarkdown),
		[]byte(c.Render()),
		benchrecord.FileMode,
	)
}

func (p *Projection) Write(dir string) error {
	if err := benchrecord.WriteJSON(dir, probeJSON, p); err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, probeMarkdown), []byte(p.Render()), benchrecord.FileMode)
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

func SplitReports(
	meta *benchreport.ReportMeta,
	cases []*benchresult.CaseResult,
) map[evalsplit.Split]*benchreport.Report {
	bySplit := map[evalsplit.Split][]*benchresult.CaseResult{}
	for _, result := range cases {
		split := evalsplit.Development
		if result.Scenario != nil && result.Scenario.Split != "" {
			split = result.Scenario.Split
		}
		bySplit[split] = append(bySplit[split], result)
	}
	reports := make(map[evalsplit.Split]*benchreport.Report, len(bySplit))
	for split, picked := range bySplit {
		reports[split] = benchreport.BuildReport(*meta, picked)
	}

	return reports
}

const (
	sealedDir      = "sealed"
	groupsDir      = "groups"
	developmentDir = string(evalsplit.Development)
)

func SealedDir(root string) string {
	return filepath.Join(root, sealedDir)
}

func GroupDir(root, group string) string {
	return filepath.Join(root, sealedDir, groupsDir, group)
}

func HandoverPaths(root string) []string {
	return []string{
		filepath.Join(root, developmentDir),
		filepath.Join(root, gateMarkdown),
	}
}

func WriteGateRun(
	root string,
	meta *benchreport.ReportMeta,
	cases []*benchresult.CaseResult,
) error {
	report := benchreport.BuildReport(*meta, cases)
	sealed := SealedDir(root)
	if err := report.WriteIndex(sealed); err != nil {
		return err
	}
	if err := os.WriteFile(
		filepath.Join(root, benchreport.SummaryFile),
		[]byte(RenderFigures(report)),
		benchrecord.FileMode,
	); err != nil {
		return fmt.Errorf("write the summary: %w", err)
	}
	for split, splitReport := range SplitReports(meta, cases) {
		dir := filepath.Join(sealed, string(split))
		if split == evalsplit.Development {
			dir = filepath.Join(root, developmentDir)
		}
		if err := splitReport.Write(dir); err != nil {
			return err
		}
	}

	return nil
}

func LoadGateRunReport(root string) (*benchreport.Report, error) {
	return benchreport.LoadReport(SealedDir(root))
}
