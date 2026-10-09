package deskbench

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"github.com/bytedance/sonic"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

const (
	reportFile     = "report.json"
	transcriptFile = "transcript.md"
	summaryFile    = "summary.md"
	callsDir       = "calls"
	promptsDir     = "prompts"
	toolsDir       = "tools"
	dirMode        = 0o750
	fileMode       = 0o640
	sampleReasons  = 3
)

var reportJSON = sonic.Config{SortMapKeys: true, EscapeHTML: false}.Froze()

type ReportMeta struct {
	StartedAt  time.Time `json:"startedAt"`
	FinishedAt time.Time `json:"finishedAt"`
	Namespace  string    `json:"namespace"`
	Version    string    `json:"version,omitempty"`
	Commit     string    `json:"commit,omitempty"`
	Command    string    `json:"command,omitempty"`
	WorkerLog  string    `json:"workerLog,omitempty"`
}

type Report struct {
	Meta       ReportMeta     `json:"meta"`
	Summary    Summary        `json:"summary"`
	ToolHealth []ToolHealth   `json:"toolHealth"`
	ModelNotes []ModelNote    `json:"modelNotes,omitempty"`
	Cases      []*CaseResult  `json:"cases"`
	Comparison []CompareEntry `json:"comparison,omitempty"`
}

type Summary struct {
	Cases   int         `json:"cases"`
	Passed  int         `json:"passed"`
	Usage   Usage       `json:"usage"`
	Matrix  []MatrixRow `json:"matrix"`
	Failing []Failing   `json:"failing,omitempty"`
}

type MatrixRow struct {
	Scenario     string  `json:"scenario"`
	Provider     string  `json:"provider"`
	Runs         int     `json:"runs"`
	Passed       int     `json:"passed"`
	AvgSeconds   float64 `json:"avgSeconds"`
	AvgToolCalls float64 `json:"avgToolCalls"`
	RefusedCalls int     `json:"refusedCalls"`
	InputTokens  int     `json:"inputTokens"`
	OutputTokens int     `json:"outputTokens"`
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

func BuildReport(meta ReportMeta, cases []*CaseResult) *Report {
	return &Report{
		Meta:       meta,
		Summary:    summarize(cases),
		ToolHealth: toolHealth(cases),
		ModelNotes: modelNotes(cases),
		Cases:      cases,
	}
}

func (r *Report) Write(dir string) error {
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}

	writer := &callWriter{dir: dir, prompts: map[string]bool{}, tools: map[string]bool{}}
	for _, result := range r.Cases {
		if err := writer.writeCase(result); err != nil {
			return err
		}
	}

	encoded, err := reportJSON.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("encode the report: %w", err)
	}
	if err = os.WriteFile(filepath.Join(dir, reportFile), encoded, fileMode); err != nil {
		return fmt.Errorf("write the report: %w", err)
	}
	if err = os.WriteFile(filepath.Join(dir, transcriptFile), []byte(renderTranscript(r)), fileMode); err != nil {
		return fmt.Errorf("write the transcript: %w", err)
	}
	if err = os.WriteFile(filepath.Join(dir, summaryFile), []byte(RenderSummary(r)), fileMode); err != nil {
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
	Seq              int                                       `json:"seq"`
	Kind             ModelCallKind                             `json:"kind"`
	StartedAt        time.Time                                 `json:"startedAt"`
	FinishedAt       time.Time                                 `json:"finishedAt"`
	Attribution      serviceports.AIUsageAttribution           `json:"attribution"`
	SystemFile       string                                    `json:"systemFile,omitempty"`
	SystemStable     int                                       `json:"systemStable,omitempty"`
	ToolsFile        string                                    `json:"toolsFile,omitempty"`
	ToolNames        []string                                  `json:"toolNames,omitempty"`
	MaxTokens        int                                       `json:"maxTokens,omitempty"`
	Messages         []serviceports.Message                    `json:"messages,omitempty"`
	Structured       *serviceports.StructuredCompletionRequest `json:"structured,omitempty"`
	ChatResult       *serviceports.ChatCompletionResult        `json:"chatResult,omitempty"`
	StructuredResult *serviceports.StructuredCompletionResult  `json:"structuredResult,omitempty"`
	Retries          []serviceports.ChatRetryNotice            `json:"retries,omitempty"`
	Error            string                                    `json:"error,omitempty"`
}

func (w *callWriter) writeCase(result *CaseResult) error {
	turnIndex := 0
	write := func(turn *TurnRecord) error {
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

func (w *callWriter) writeTurn(caseID string, turnIndex int, turn *TurnRecord) error {
	dir := filepath.Join(w.dir, callsDir, caseID)
	if len(turn.ModelCalls) > 0 {
		if err := os.MkdirAll(dir, dirMode); err != nil {
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
		encoded, err := reportJSON.MarshalIndent(&file, "", "  ")
		if err != nil {
			return fmt.Errorf("encode model call %d: %w", call.Seq, err)
		}
		if err = os.WriteFile(filepath.Join(w.dir, relative), encoded, fileMode); err != nil {
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

	digest := digestOf(system)
	relative := filepath.Join(promptsDir, digest+".md")
	if w.prompts[digest] {
		return relative, nil
	}
	if err := os.MkdirAll(filepath.Join(w.dir, promptsDir), dirMode); err != nil {
		return "", fmt.Errorf("create prompts dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(w.dir, relative), []byte(system), fileMode); err != nil {
		return "", fmt.Errorf("write system prompt: %w", err)
	}
	w.prompts[digest] = true

	return relative, nil
}

func (w *callWriter) toolSpecs(tools []serviceports.ToolSpec) (string, error) {
	if len(tools) == 0 {
		return "", nil
	}

	digest := digestOf(tools)
	relative := filepath.Join(toolsDir, digest+".json")
	if w.tools[digest] {
		return relative, nil
	}
	if err := os.MkdirAll(filepath.Join(w.dir, toolsDir), dirMode); err != nil {
		return "", fmt.Errorf("create tools dir: %w", err)
	}
	encoded, err := reportJSON.MarshalIndent(tools, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode tool specs: %w", err)
	}
	if err = os.WriteFile(filepath.Join(w.dir, relative), encoded, fileMode); err != nil {
		return "", fmt.Errorf("write tool specs: %w", err)
	}
	w.tools[digest] = true

	return relative, nil
}

func toolNames(tools []serviceports.ToolSpec) []string {
	names := make([]string, 0, len(tools))
	for idx := range tools {
		names = append(names, tools[idx].Name)
	}

	return names
}

func allTurns(result *CaseResult) []*TurnRecord {
	turns := make([]*TurnRecord, 0, len(result.Steps)+len(result.Cleanup))
	for _, step := range result.Steps {
		turns = append(turns, step.Turns...)
	}

	return append(turns, result.Cleanup...)
}

func summarize(cases []*CaseResult) Summary {
	summary := Summary{Cases: len(cases)}
	rows := make(map[string]*MatrixRow, len(cases))
	order := make([]string, 0, len(cases))
	durations := make(map[string]float64, len(cases))
	toolCalls := make(map[string]int, len(cases))

	for _, result := range cases {
		if result.Passed {
			summary.Passed++
		}
		summary.Usage.add(result.Usage)

		key := result.Key()
		row, ok := rows[key]
		if !ok {
			row = &MatrixRow{Scenario: result.Scenario.Name, Provider: result.Provider}
			rows[key] = row
			order = append(order, key)
		}
		row.Runs++
		if result.Passed {
			row.Passed++
		}
		row.InputTokens += result.Usage.InputTokens
		row.OutputTokens += result.Usage.OutputTokens
		durations[key] += result.FinishedAt.Sub(result.StartedAt).Seconds()

		for _, step := range result.Steps {
			for _, turn := range step.Turns {
				toolCalls[key] += len(turn.Tools)
				row.RefusedCalls += len(refusedCalls(turn.Tools))
			}
			for _, check := range step.Checks {
				if check.Passed {
					continue
				}
				summary.Failing = append(summary.Failing, Failing{
					Case:   result.Key() + " #" + strconv.Itoa(result.Repeat),
					Step:   step.Index,
					Label:  firstLine(step.Label),
					Check:  check.Name,
					Detail: firstLine(check.Detail),
				})
			}
			if step.Error != "" {
				summary.Failing = append(summary.Failing, Failing{
					Case:   result.Key() + " #" + strconv.Itoa(result.Repeat),
					Step:   step.Index,
					Label:  firstLine(step.Label),
					Check:  "step ran",
					Detail: firstLine(step.Error),
				})
			}
		}
		if result.Error != "" {
			summary.Failing = append(summary.Failing, Failing{
				Case:   result.Key() + " #" + strconv.Itoa(result.Repeat),
				Check:  "case ran",
				Detail: firstLine(result.Error),
			})
		}
	}

	summary.Matrix = make([]MatrixRow, 0, len(order))
	for _, key := range order {
		row := rows[key]
		row.AvgSeconds = durations[key] / float64(row.Runs)
		row.AvgToolCalls = float64(toolCalls[key]) / float64(row.Runs)
		summary.Matrix = append(summary.Matrix, *row)
	}

	return summary
}

func toolHealth(cases []*CaseResult) []ToolHealth {
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
				health.Verdicts[verdictLabel(call)]++
				if !call.Refused() {
					continue
				}
				health.Refused++
				reason := firstLine(call.Result)
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

func modelNotes(cases []*CaseResult) []ModelNote {
	notes := make([]ModelNote, 0, 4)
	for _, result := range cases {
		for _, turn := range allTurns(result) {
			for _, call := range turn.Calls {
				for _, note := range callNotes(&call) {
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

func callNotes(call *CallSummary) []string {
	notes := make([]string, 0, 2)
	if call.Error != "" {
		notes = append(notes, "failed: "+firstLine(call.Error))
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
