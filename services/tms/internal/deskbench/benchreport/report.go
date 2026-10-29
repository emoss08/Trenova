package benchreport

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/deskbench/benchrecord"
	"github.com/emoss08/trenova/internal/deskbench/benchresult"
	"github.com/emoss08/trenova/internal/deskbench/benchstats"
	"github.com/emoss08/trenova/internal/modules/agent/domain/evalsplit"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

const (
	reportFile     = "report.json"
	transcriptFile = "transcript.md"
	SummaryFile    = "summary.md"
	callsDir       = "calls"
	promptsDir     = "prompts"
	toolsDir       = "tools"
	sampleReasons  = 3
)

const reportVersion = 2

type ReportMeta struct {
	StartedAt  time.Time `json:"startedAt"`
	FinishedAt time.Time `json:"finishedAt"`
	Namespace  string    `json:"namespace"`
	Version    string    `json:"version,omitempty"`
	Commit     string    `json:"commit,omitempty"`
	Command    string    `json:"command,omitempty"`
	WorkerLog  string    `json:"workerLog,omitempty"`
	PassK      int       `json:"passK,omitempty"`
	Label      string    `json:"label,omitempty"`
	Threshold  *int      `json:"toolDisclosureThreshold,omitempty"`

	Data benchresult.DataProfile `json:"data"`
}

type Report struct {
	SchemaVersion  int                       `json:"schemaVersion,omitempty"`
	Meta           ReportMeta                `json:"meta"`
	Summary        Summary                   `json:"summary"`
	ToolHealth     []ToolHealth              `json:"toolHealth"`
	ModelNotes     []ModelNote               `json:"modelNotes,omitempty"`
	Cases          []*benchresult.CaseResult `json:"cases"`
	Comparison     []CompareEntry            `json:"comparison,omitempty"`
	ComparisonNote string                    `json:"comparisonNote,omitempty"`
	CellChanges    []CellChange              `json:"cellChanges,omitempty"`
}

type Summary struct {
	Cases        int               `json:"cases"`
	Passed       int               `json:"passed"`
	Skipped      int               `json:"skipped,omitempty"`
	SkippedCases []SkippedCase     `json:"skippedCases,omitempty"`
	PassK        int               `json:"passK,omitempty"`
	Usage        benchrecord.Usage `json:"usage"`
	Matrix       []MatrixRow       `json:"matrix"`
	Cells        []CellRow         `json:"cells,omitempty"`
	Failing      []Failing         `json:"failing,omitempty"`
}

type SkippedCase struct {
	Case   string `json:"case"`
	Trials int    `json:"trials"`
	Reason string `json:"reason"`
}

type Measures struct {
	Seconds            benchstats.Stat    `json:"seconds"`
	FirstTokenMs       benchstats.Stat    `json:"firstTokenMs"`
	ModelCalls         benchstats.Stat    `json:"modelCalls"`
	ToolCalls          benchstats.Stat    `json:"toolCalls"`
	Refused            benchstats.Stat    `json:"refused"`
	RefusedByVerdict   map[string]float64 `json:"refusedByVerdict,omitempty"`
	ReasksByGuard      map[string]float64 `json:"reasksByGuard,omitempty"`
	UncachedInput      benchstats.Stat    `json:"uncachedInput"`
	CacheRead          benchstats.Stat    `json:"cacheRead"`
	CacheWrite         benchstats.Stat    `json:"cacheWrite"`
	Output             benchstats.Stat    `json:"output"`
	PeakPrompt         benchstats.Stat    `json:"peakPrompt"`
	CacheReadShare     *float64           `json:"cacheReadShare,omitempty"`
	LaterCacheShare    *float64           `json:"laterCacheReadShare,omitempty"`
	FindFirstShare     *float64           `json:"findFirstShare,omitempty"`
	CostUSD            benchstats.Stat    `json:"costUsd"`
	CostEstimated      bool               `json:"costEstimated,omitempty"`
	CostUnpriced       bool               `json:"costUnpriced,omitempty"`
	CostPerPass        *float64           `json:"costPerCompletedTask,omitempty"`
	Precision          benchstats.Stat    `json:"precision"`
	Recall             benchstats.Stat    `json:"recall"`
	FirstCorrect       benchstats.Stat    `json:"firstCorrect"`
	ForbiddenCalls     benchstats.Stat    `json:"forbiddenCalls"`
	ArgumentValidity   benchstats.Stat    `json:"argumentValidity"`
	PreselectionRecall benchstats.Stat    `json:"preselectionRecall"`
	FindRecall         benchstats.Stat    `json:"findRecall"`
	FindHitRate        *float64           `json:"findHitRate,omitempty"`
	Unauthorized       int                `json:"unauthorizedEffects"`
	UnauthorizedRules  map[string]int     `json:"unauthorizedByRule,omitempty"`
	Unaudited          int                `json:"unauditedTrials,omitempty"`
}

func (m *Measures) Safe() bool {
	return m.Unauthorized == 0 && m.Unaudited == 0
}

type MatrixRow struct {
	Scenario string          `json:"scenario"`
	Provider string          `json:"provider"`
	Split    evalsplit.Split `json:"split,omitempty"`
	Runs     int             `json:"runs"`
	Passed   int             `json:"passed"`
	Pass     benchstats.Rate `json:"pass"`
	PassK    *float64        `json:"passK,omitempty"`
	Measures Measures        `json:"measures"`

	AvgSeconds    float64 `json:"avgSeconds"`
	AvgToolCalls  float64 `json:"avgToolCalls"`
	RefusedTotal  int     `json:"refusedCalls,omitempty"`
	InputTotal    int     `json:"inputTokens,omitempty"`
	OutputTotal   int     `json:"outputTokens,omitempty"`
	RefusedPerRun float64 `json:"refusedPerRun"`
	TokensPerRun  float64 `json:"tokensPerRun"`
}

func (r *MatrixRow) Key() string {
	return r.Scenario + " @ " + r.Provider
}

func (r *MatrixRow) PassRate() float64 {
	if r.Runs == 0 {
		return 0
	}

	return float64(r.Passed) / float64(r.Runs)
}

func (r *MatrixRow) refusedPerRun(schema int) float64 {
	if schema >= reportVersion || r.Runs == 0 {
		return r.RefusedPerRun
	}

	return float64(r.RefusedTotal) / float64(r.Runs)
}

func (r *MatrixRow) tokensPerRun(schema int) float64 {
	if schema >= reportVersion || r.Runs == 0 {
		return r.TokensPerRun
	}

	return float64(r.InputTotal+r.OutputTotal) / float64(r.Runs)
}

type CellRow struct {
	Provider  string          `json:"provider"`
	Split     evalsplit.Split `json:"split,omitempty"`
	Scenarios int             `json:"scenarios"`
	Trials    int             `json:"trials"`
	Passed    int             `json:"passed"`
	Pass      benchstats.Rate `json:"pass"`
	PassK     benchstats.Stat `json:"passK"`
	Measures  Measures        `json:"measures"`

	ScenarioPass benchstats.Stat `json:"scenarioPass"`
}

func (c *CellRow) Key() string {
	if c.Split == "" {
		return c.Provider
	}

	return c.Provider + " [" + string(c.Split) + "]"
}

type Failing struct {
	Case   string `json:"case"`
	Step   int    `json:"step"`
	Label  string `json:"label"`
	Check  string `json:"check"`
	Detail string `json:"detail,omitempty"`
}

type ToolHealth struct {
	Tool     string         `json:"tool"`
	Calls    int            `json:"calls"`
	Refused  int            `json:"refused"`
	Verdicts map[string]int `json:"verdicts"`
	Reasons  []string       `json:"reasons,omitempty"`
}

type ModelNote struct {
	Case   string `json:"case"`
	TurnID string `json:"turnId"`
	Seq    int    `json:"seq"`
	Model  string `json:"model,omitempty"`
	Note   string `json:"note"`
}

func BuildReport(meta *ReportMeta, cases []*benchresult.CaseResult) *Report {
	resolved := *meta
	if resolved.PassK <= 0 {
		resolved.PassK = benchstats.DefaultPassK
	}

	return &Report{
		SchemaVersion: reportVersion,
		Meta:          resolved,
		Summary:       summarize(cases, resolved.PassK),
		ToolHealth:    toolHealth(cases),
		ModelNotes:    modelNotes(cases),
		Cases:         cases,
	}
}

func (r *Report) Write(dir string) error {
	if err := os.MkdirAll(dir, benchrecord.DirMode); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}

	writer := &callWriter{dir: dir, prompts: map[string]bool{}, tools: map[string]bool{}}
	for _, result := range r.Cases {
		if err := writer.writeCase(result); err != nil {
			return err
		}
	}

	encoded, err := benchrecord.JSON.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("encode the report: %w", err)
	}
	if err = os.WriteFile(
		filepath.Join(dir, reportFile),
		encoded,
		benchrecord.FileMode,
	); err != nil {
		return fmt.Errorf("write the report: %w", err)
	}
	if err = os.WriteFile(
		filepath.Join(dir, transcriptFile),
		[]byte(renderTranscript(r)),
		benchrecord.FileMode,
	); err != nil {
		return fmt.Errorf("write the transcript: %w", err)
	}
	if err = os.WriteFile(
		filepath.Join(dir, SummaryFile),
		[]byte(RenderSummary(r)),
		benchrecord.FileMode,
	); err != nil {
		return fmt.Errorf("write the summary: %w", err)
	}

	return nil
}

func LoadReport(path string) (*Report, error) {
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		path = filepath.Join(path, reportFile)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	report := new(Report)
	if err = sonic.Unmarshal(raw, report); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	return report, nil
}

type callWriter struct {
	dir     string
	prompts map[string]bool
	tools   map[string]bool
}

type callFile struct {
	Seq              int                                   `json:"seq"`
	Kind             benchrecord.ModelCallKind             `json:"kind"`
	StartedAt        time.Time                             `json:"startedAt"`
	FinishedAt       time.Time                             `json:"finishedAt"`
	Attribution      aiusageports.AIUsageAttribution       `json:"attribution"`
	SystemFile       string                                `json:"systemFile,omitempty"`
	SystemStable     int                                   `json:"systemStable,omitempty"`
	ToolsFile        string                                `json:"toolsFile,omitempty"`
	ToolNames        []string                              `json:"toolNames,omitempty"`
	MaxTokens        int                                   `json:"maxTokens,omitempty"`
	Messages         []llmports.Message                    `json:"messages,omitempty"`
	Structured       *llmports.StructuredCompletionRequest `json:"structured,omitempty"`
	ChatResult       *llmports.ChatCompletionResult        `json:"chatResult,omitempty"`
	StructuredResult *llmports.StructuredCompletionResult  `json:"structuredResult,omitempty"`
	Retries          []llmports.ChatRetryNotice            `json:"retries,omitempty"`
	Error            string                                `json:"error,omitempty"`
}

func (w *callWriter) writeCase(result *benchresult.CaseResult) error {
	turnIndex := 0
	write := func(turn *benchrecord.TurnRecord) error {
		turnIndex++
		return w.writeTurn(result.ID(), turnIndex, turn)
	}

	for _, step := range result.Steps {
		for _, turn := range step.Turns {
			if err := write(turn); err != nil {
				return err
			}
		}
	}
	for _, turn := range result.Cleanup {
		if err := write(turn); err != nil {
			return err
		}
	}

	return nil
}

func (w *callWriter) writeTurn(caseID string, turnIndex int, turn *benchrecord.TurnRecord) error {
	dir := filepath.Join(w.dir, callsDir, caseID)
	if len(turn.ModelCalls) > 0 {
		if err := os.MkdirAll(dir, benchrecord.DirMode); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}

	for idx, call := range turn.ModelCalls {
		file := callFile{
			Seq:              call.Seq,
			Kind:             call.Kind,
			StartedAt:        call.StartedAt,
			FinishedAt:       call.FinishedAt,
			Attribution:      call.Attribution,
			Structured:       call.Structured,
			ChatResult:       call.ChatResult,
			StructuredResult: call.StructuredResult,
			Retries:          call.Retries,
			Error:            call.Error,
		}

		if req := call.Chat; req != nil {
			systemFile, err := w.prompt(req.System)
			if err != nil {
				return err
			}
			toolsFile, err := w.toolSpecs(req.Tools)
			if err != nil {
				return err
			}
			file.SystemFile = systemFile
			file.SystemStable = req.SystemStable
			file.ToolsFile = toolsFile
			file.ToolNames = toolNames(req.Tools)
			file.MaxTokens = req.MaxTokens
			file.Messages = req.Messages
		}
		if req := call.Structured; req != nil {
			systemFile, err := w.prompt(req.System)
			if err != nil {
				return err
			}
			file.SystemFile = systemFile
		}

		name := fmt.Sprintf("turn%02d-call%03d.json", turnIndex, call.Seq)
		relative := filepath.Join(callsDir, caseID, name)
		encoded, err := benchrecord.JSON.MarshalIndent(&file, "", "  ")
		if err != nil {
			return fmt.Errorf("encode model call %d: %w", call.Seq, err)
		}
		if err = os.WriteFile(
			filepath.Join(w.dir, relative),
			encoded,
			benchrecord.FileMode,
		); err != nil {
			return fmt.Errorf("write model call %d: %w", call.Seq, err)
		}
		if idx < len(turn.Calls) {
			turn.Calls[idx].File = relative
		}
	}

	return nil
}

func (w *callWriter) prompt(system string) (string, error) {
	if system == "" {
		return "", nil
	}

	digest := benchrecord.DigestOf(system)
	relative := filepath.Join(promptsDir, digest+".md")
	if w.prompts[digest] {
		return relative, nil
	}
	if err := os.MkdirAll(filepath.Join(w.dir, promptsDir), benchrecord.DirMode); err != nil {
		return "", fmt.Errorf("create prompts dir: %w", err)
	}
	if err := os.WriteFile(
		filepath.Join(w.dir, relative),
		[]byte(system),
		benchrecord.FileMode,
	); err != nil {
		return "", fmt.Errorf("write system prompt: %w", err)
	}
	w.prompts[digest] = true

	return relative, nil
}

func (w *callWriter) toolSpecs(tools []kernelports.ToolSpec) (string, error) {
	if len(tools) == 0 {
		return "", nil
	}

	digest := benchrecord.DigestOf(tools)
	relative := filepath.Join(toolsDir, digest+".json")
	if w.tools[digest] {
		return relative, nil
	}
	if err := os.MkdirAll(filepath.Join(w.dir, toolsDir), benchrecord.DirMode); err != nil {
		return "", fmt.Errorf("create tools dir: %w", err)
	}
	encoded, err := benchrecord.JSON.MarshalIndent(tools, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode tool specs: %w", err)
	}
	if err = os.WriteFile(
		filepath.Join(w.dir, relative),
		encoded,
		benchrecord.FileMode,
	); err != nil {
		return "", fmt.Errorf("write tool specs: %w", err)
	}
	w.tools[digest] = true

	return relative, nil
}

func toolNames(tools []kernelports.ToolSpec) []string {
	names := make([]string, 0, len(tools))
	for idx := range tools {
		names = append(names, tools[idx].Name)
	}

	return names
}

func allTurns(result *benchresult.CaseResult) []*benchrecord.TurnRecord {
	turns := make([]*benchrecord.TurnRecord, 0, len(result.Steps)+len(result.Cleanup))
	for _, step := range result.Steps {
		turns = append(turns, step.Turns...)
	}

	return append(turns, result.Cleanup...)
}

func summarize(cases []*benchresult.CaseResult, k int) Summary {
	summary := Summary{PassK: k}
	groups := make(map[string][]*benchresult.CaseResult, len(cases))
	order := make([]string, 0, len(cases))
	ran := make([]*benchresult.CaseResult, 0, len(cases))

	skipped := make(map[SkippedCase]int, 4)
	for _, result := range cases {
		if result.Skipped != "" {
			summary.Skipped++
			key := SkippedCase{Case: result.Key(), Reason: result.Skipped}
			if _, seen := skipped[key]; !seen {
				skipped[key] = len(summary.SkippedCases)
				summary.SkippedCases = append(summary.SkippedCases, key)
			}
			summary.SkippedCases[skipped[key]].Trials++
			continue
		}
		ran = append(ran, result)
		summary.Cases++
		if result.Passed {
			summary.Passed++
		}
		summary.Usage.Add(&result.Usage)

		key := result.Key()
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], result)
		summary.Failing = append(summary.Failing, failingOf(result)...)
	}

	summary.Matrix = make([]MatrixRow, 0, len(order))
	for _, key := range order {
		summary.Matrix = append(summary.Matrix, matrixRow(groups[key], k))
	}
	summary.Cells = cellRows(ran, summary.Matrix)

	return summary
}

func failingOf(result *benchresult.CaseResult) []Failing {
	label := result.Key() + " #" + strconv.Itoa(result.Repeat)
	failing := make([]Failing, 0, 2)
	for _, step := range result.Steps {
		for _, check := range step.Checks {
			if check.Passed {
				continue
			}
			failing = append(failing, Failing{
				Case:   label,
				Step:   step.Index,
				Label:  benchrecord.FirstLine(step.Label),
				Check:  check.Name,
				Detail: benchrecord.FirstLine(check.Detail),
			})
		}
		if step.Error != "" {
			failing = append(failing, Failing{
				Case:   label,
				Step:   step.Index,
				Label:  benchrecord.FirstLine(step.Label),
				Check:  "step ran",
				Detail: benchrecord.FirstLine(step.Error),
			})
		}
	}
	if result.Error != "" {
		failing = append(failing, Failing{
			Case:   label,
			Check:  "case ran",
			Detail: benchrecord.FirstLine(result.Error),
		})
	}

	return failing
}

func matrixRow(results []*benchresult.CaseResult, k int) MatrixRow {
	first := results[0]
	row := MatrixRow{
		Scenario: first.Scenario.Name,
		Provider: first.Provider,
		Split:    first.Scenario.Split,
		Runs:     len(results),
	}
	for _, result := range results {
		if result.Passed {
			row.Passed++
		}
	}
	row.Pass = benchstats.RateOf(row.Passed, row.Runs)
	row.PassK = benchstats.PassHatK(row.Passed, row.Runs, k)
	row.Measures = MeasuresOf(results)
	row.AvgSeconds = row.Measures.Seconds.Mean
	row.AvgToolCalls = row.Measures.ToolCalls.Mean
	row.RefusedPerRun = row.Measures.Refused.Mean
	row.TokensPerRun = row.Measures.UncachedInput.Mean + row.Measures.CacheRead.Mean +
		row.Measures.CacheWrite.Mean + row.Measures.Output.Mean

	return row
}

func cellRows(cases []*benchresult.CaseResult, matrix []MatrixRow) []CellRow {
	type cellKey struct {
		provider string
		split    evalsplit.Split
	}
	byCell := make(map[cellKey][]*benchresult.CaseResult, 8)
	rowsByCell := make(map[cellKey][]MatrixRow, 8)
	order := make([]cellKey, 0, 8)
	add := func(key cellKey, result *benchresult.CaseResult) {
		if _, ok := byCell[key]; !ok {
			order = append(order, key)
		}
		byCell[key] = append(byCell[key], result)
	}
	for _, result := range cases {
		add(cellKey{provider: result.Provider}, result)
		if result.Scenario.Split != "" {
			add(cellKey{provider: result.Provider, split: result.Scenario.Split}, result)
		}
	}
	for idx := range matrix {
		row := matrix[idx]
		rowsByCell[cellKey{provider: row.Provider}] = append(
			rowsByCell[cellKey{provider: row.Provider}], row)
		if row.Split != "" {
			key := cellKey{provider: row.Provider, split: row.Split}
			rowsByCell[key] = append(rowsByCell[key], row)
		}
	}

	cells := make([]CellRow, 0, len(order))
	for _, key := range order {
		results := byCell[key]
		cell := CellRow{
			Provider: key.provider,
			Split:    key.split,
			Trials:   len(results),
			Measures: MeasuresOf(results),
		}
		rows := rowsByCell[key]
		passK := make([]float64, 0, len(rows))
		scenarioPass := make([]float64, 0, len(rows))
		for idx := range rows {
			row := &rows[idx]
			cell.Scenarios++
			cell.Passed += row.Passed
			if row.Runs > 0 {
				scenarioPass = append(scenarioPass, float64(row.Passed)/float64(row.Runs))
			}
			if row.PassK != nil {
				passK = append(passK, *row.PassK)
			}
		}
		cell.Pass = benchstats.RateOf(cell.Passed, cell.Trials)
		cell.ScenarioPass = benchstats.UnitStatOf(scenarioPass)
		cell.PassK = benchstats.UnitStatOf(passK)
		cells = append(cells, cell)
	}
	slices.SortStableFunc(cells, func(a, b CellRow) int {
		return cmp.Or(cmp.Compare(a.Provider, b.Provider), cmp.Compare(a.Split, b.Split))
	})

	return cells
}

type measureSeries struct {
	seconds, firstToken, modelCalls, toolCalls, refused []float64
	uncached, read, write, output, peak, cost           []float64
	precision, recall, firstCorrect, forbidden          []float64
	preselection, find, argumentValidity                []float64
	findAnswers, findHits                               int
	laterPrompt, laterRead, toolTurns, findFirst        int
	verdicts, guards                                    map[string]float64
	estimated, unpriced                                 bool
	passed, unauthorized, unaudited                     int
	unauthorizedRules                                   map[string]int
}

func MeasuresOf(results []*benchresult.CaseResult) Measures {
	series := measureSeries{
		verdicts:          map[string]float64{},
		guards:            map[string]float64{},
		unauthorizedRules: map[string]int{},
	}
	for _, result := range results {
		series.add(result)
	}

	return series.measures(len(results))
}

func (m *measureSeries) add(result *benchresult.CaseResult) {
	metrics := &result.Metrics
	m.addActivity(metrics)
	m.addUsage(metrics)
	m.addOutcome(result)
	m.addSelection(metrics)
}

func (m *measureSeries) addActivity(metrics *benchresult.CaseMetrics) {
	m.seconds = append(m.seconds, metrics.Seconds)
	if metrics.FirstTokenMs != nil {
		m.firstToken = append(m.firstToken, *metrics.FirstTokenMs)
	}
	m.modelCalls = append(m.modelCalls, float64(metrics.ModelCalls))
	m.toolCalls = append(m.toolCalls, float64(metrics.ToolCalls))
	m.refused = append(m.refused, float64(metrics.RefusedCalls))
	if metrics.ToolCalls > 0 {
		invalid := float64(metrics.RefusedByVerdict[benchrecord.VerdictInvalid])
		m.argumentValidity = append(m.argumentValidity,
			1-invalid/float64(metrics.ToolCalls))
	}
	for verdict, count := range metrics.RefusedByVerdict {
		m.verdicts[verdict] += float64(count)
	}
	for guard, count := range metrics.ReasksByGuard {
		m.guards[guard] += float64(count)
	}
}

func (m *measureSeries) addUsage(metrics *benchresult.CaseMetrics) {
	usage := &metrics.Usage
	m.uncached = append(m.uncached, float64(usage.InputTokens))
	m.read = append(m.read, float64(usage.CacheReadTokens))
	m.write = append(m.write, float64(usage.CacheWriteTokens))
	m.output = append(m.output, float64(usage.OutputTokens))
	m.peak = append(m.peak, float64(metrics.PeakPromptTokens))
	m.laterPrompt += metrics.LaterPrompt
	m.laterRead += metrics.LaterCacheRead
	m.toolTurns += metrics.ToolTurns
	m.findFirst += metrics.FindFirstTurns
	switch {
	case usage.UnpricedCalls > 0 || (usage.CostUSD == nil && usage.ModelCalls > 0):
		m.unpriced = true
	case usage.CostUSD != nil:
		cost, _ := usage.CostUSD.Float64()
		m.cost = append(m.cost, cost)
	}
	if usage.EstimatedCalls > 0 {
		m.estimated = true
	}
}

func (m *measureSeries) addOutcome(result *benchresult.CaseResult) {
	if result.Passed {
		m.passed++
	}
	if result.AuditError != "" {
		m.unaudited++
	}
	m.unauthorized += len(result.Unauthorized)
	for idx := range result.Unauthorized {
		for _, rule := range result.Unauthorized[idx].Rules {
			m.unauthorizedRules[string(rule)]++
		}
	}
}

func (m *measureSeries) addSelection(metrics *benchresult.CaseMetrics) {
	if selection := metrics.Selection; selection != nil {
		if selection.Precision != nil {
			m.precision = append(m.precision, *selection.Precision)
		}
		m.recall = append(m.recall, selection.Recall)
		if selection.FirstCorrect > 0 {
			m.firstCorrect = append(m.firstCorrect, float64(selection.FirstCorrect))
		}
		m.forbidden = append(m.forbidden, float64(selection.ForbiddenCalls))
	}
	if retrieval := metrics.Retrieval; retrieval != nil {
		if retrieval.PreselectionRecall != nil {
			m.preselection = append(m.preselection, *retrieval.PreselectionRecall)
		}
		if retrieval.FindRecall != nil {
			m.find = append(m.find, *retrieval.FindRecall)
		}
		m.findAnswers += retrieval.FindAnswers
		m.findHits += retrieval.FindHits
	}
}

func (m *measureSeries) measures(trials int) Measures {
	measures := Measures{
		Seconds:            benchstats.StatOf(m.seconds),
		FirstTokenMs:       benchstats.StatOf(m.firstToken),
		ModelCalls:         benchstats.StatOf(m.modelCalls),
		ToolCalls:          benchstats.StatOf(m.toolCalls),
		Refused:            benchstats.StatOf(m.refused),
		UncachedInput:      benchstats.StatOf(m.uncached),
		CacheRead:          benchstats.StatOf(m.read),
		CacheWrite:         benchstats.StatOf(m.write),
		Output:             benchstats.StatOf(m.output),
		PeakPrompt:         benchstats.StatOf(m.peak),
		CostUSD:            benchstats.StatOf(m.cost),
		CostEstimated:      m.estimated,
		CostUnpriced:       m.unpriced,
		Precision:          benchstats.StatOf(m.precision),
		Recall:             benchstats.StatOf(m.recall),
		FirstCorrect:       benchstats.StatOf(m.firstCorrect),
		ForbiddenCalls:     benchstats.StatOf(m.forbidden),
		ArgumentValidity:   benchstats.StatOf(m.argumentValidity),
		PreselectionRecall: benchstats.StatOf(m.preselection),
		FindRecall:         benchstats.StatOf(m.find),
		RefusedByVerdict:   perTrial(m.verdicts, trials),
		ReasksByGuard:      perTrial(m.guards, trials),
		Unauthorized:       m.unauthorized,
		Unaudited:          m.unaudited,
	}
	if len(m.unauthorizedRules) > 0 {
		measures.UnauthorizedRules = m.unauthorizedRules
	}
	prompt := measures.UncachedInput.Mean + measures.CacheRead.Mean + measures.CacheWrite.Mean
	if prompt > 0 {
		share := measures.CacheRead.Mean / prompt
		measures.CacheReadShare = &share
	}
	if m.passed > 0 && !m.unpriced && len(m.cost) > 0 {
		total := 0.0
		for _, cost := range m.cost {
			total += cost
		}
		perPass := total / float64(m.passed)
		measures.CostPerPass = &perPass
	}
	if m.findAnswers > 0 {
		rate := float64(m.findHits) / float64(m.findAnswers)
		measures.FindHitRate = &rate
	}
	if m.laterPrompt > 0 {
		share := float64(m.laterRead) / float64(m.laterPrompt)
		measures.LaterCacheShare = &share
	}
	if m.toolTurns > 0 {
		share := float64(m.findFirst) / float64(m.toolTurns)
		measures.FindFirstShare = &share
	}

	return measures
}

func perTrial(totals map[string]float64, trials int) map[string]float64 {
	if len(totals) == 0 || trials == 0 {
		return nil
	}
	out := make(map[string]float64, len(totals))
	for key, total := range totals {
		out[key] = total / float64(trials)
	}

	return out
}

func toolHealth(cases []*benchresult.CaseResult) []ToolHealth {
	byTool := make(map[string]*ToolHealth, 32)
	for _, result := range cases {
		for _, turn := range allTurns(result) {
			for _, call := range turn.Tools {
				health, ok := byTool[call.Name]
				if !ok {
					health = &ToolHealth{Tool: call.Name, Verdicts: map[string]int{}}
					byTool[call.Name] = health
				}
				health.Calls++
				health.Verdicts[benchrecord.VerdictLabel(call)]++
				if !call.Refused() {
					continue
				}
				health.Refused++
				reason := benchrecord.FirstLine(call.Result)
				if len(health.Reasons) < sampleReasons && !slices.Contains(health.Reasons, reason) {
					health.Reasons = append(health.Reasons, reason)
				}
			}
		}
	}

	out := make([]ToolHealth, 0, len(byTool))
	for _, health := range byTool {
		out = append(out, *health)
	}
	slices.SortFunc(out, func(a, b ToolHealth) int {
		if a.Refused != b.Refused {
			return b.Refused - a.Refused
		}
		if a.Calls != b.Calls {
			return b.Calls - a.Calls
		}

		return cmp.Compare(a.Tool, b.Tool)
	})

	return out
}

func modelNotes(cases []*benchresult.CaseResult) []ModelNote {
	notes := make([]ModelNote, 0, 4)
	for _, result := range cases {
		for _, turn := range allTurns(result) {
			for idx := range turn.Calls {
				call := &turn.Calls[idx]
				for _, note := range callNotes(call) {
					notes = append(notes, ModelNote{
						Case:   result.Key() + " #" + strconv.Itoa(result.Repeat),
						TurnID: turn.TurnID.String(),
						Seq:    call.Seq,
						Model:  call.Model,
						Note:   note,
					})
				}
			}
		}
	}

	return notes
}

func callNotes(call *benchrecord.CallSummary) []string {
	notes := make([]string, 0, 2)
	if call.Error != "" {
		notes = append(notes, "failed: "+benchrecord.FirstLine(call.Error))
	}
	if call.Truncated {
		notes = append(notes, "reply cut off at the output limit")
	}
	if call.CutOffCall != "" {
		notes = append(notes, "cut off inside a call to "+call.CutOffCall)
	}
	if call.FallbackFrom != nil {
		notes = append(notes, fmt.Sprintf("fell back from %s (%s): %s",
			call.FallbackFrom.Name, call.FallbackFrom.Status, call.FallbackFrom.Detail))
	}
	for _, retry := range call.Retries {
		notes = append(notes, fmt.Sprintf("retried (%s) on %s: %s",
			retry.Kind, retry.Provider, retry.Reason))
	}
	for idx := range call.ToolCalls {
		if call.ToolCalls[idx].ArgumentsError != "" {
			notes = append(notes, fmt.Sprintf("arguments to %s did not parse: %s",
				call.ToolCalls[idx].Name, call.ToolCalls[idx].ArgumentsError))
		}
	}

	return notes
}

func (r *Report) WriteIndex(dir string) error {
	if err := os.MkdirAll(dir, benchrecord.DirMode); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	if err := benchrecord.WriteJSON(dir, reportFile, r); err != nil {
		return fmt.Errorf("write the report: %w", err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, SummaryFile),
		[]byte(RenderSummary(r)),
		benchrecord.FileMode,
	); err != nil {
		return fmt.Errorf("write the summary: %w", err)
	}

	return nil
}
