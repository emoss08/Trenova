package agentquerytoolservice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/analysis"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/analysisports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/scriptports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	paramScriptTitle     = codemode.ParamScriptTitle
	paramScriptPurpose   = codemode.ParamScriptPurpose
	paramScriptCode      = codemode.ParamScriptCode
	paramScriptInputs    = "inputs"
	paramScriptContinues = codemode.ParamScriptContinues
	paramInputName       = "name"
	paramInputValue      = "value"
	maxInputNameRunes    = 64
	maxContinuesRunes    = 200

	maxScriptTitleRunes   = 80
	maxScriptPurposeRunes = 500
	maxScriptTraceback    = 3000
	maxAnswerTableRows    = 25

	scriptDoneNote = "Each read the script made ran as the person, exactly as the tool would " +
		"have run on its own. Answer from output and result; say what a refused read means " +
		"rather than leaving it out."
	scriptErrorNote = "The script raised an error. Fix it and run it again; do not run the " +
		"same script unchanged. calls lists what each read returned or why it was refused."
	scriptStoppedNote = "The script was stopped at a limit. Read less, or split the work, " +
		"and run it again; or answer with what it returned."
	scriptBatchFiledNote = " The changes it prepared are proposed as one batch: nothing has " +
		"run, and a person approves or rejects them as a whole. Say that they wait for approval; " +
		"never say they were made."
	scriptBatchDroppedNote = " The script did not finish, so none of the changes it prepared " +
		"was proposed. Fix it and run it again."
	scriptBatchRefusedNote = " None of the changes it prepared was proposed: "
	scriptShownNote        = " What it showed with show_figure, show_table and show_chart is " +
		"beside your reply under the script's title; say what it means and what to do about " +
		"it, without listing it again or describing how it was drawn."
	scriptNotShownNote = " Some of what it showed was past what a script may show and is " +
		"not beside your reply: notShown counts it."
	scriptPlainNote = " Nothing was shown with show_figure, show_table or show_chart, so " +
		"the person sees nothing of this script; show what they should see when the answer " +
		"is figures, a table or a chart."
	scriptUnshownAnalysisNote = " It read its datasets and showed nothing, so no analysis " +
		"was kept and the person sees none of it. Before you answer, show it: call " +
		"analysis_run with continues set to scriptId and code holding only the " +
		"show_figure, show_table and show_chart lines, which run after this script with " +
		"every name it left, so the headline numbers, the rows that matter and any " +
		"comparison or trend are shown. Then say what they mean."
)

var errScriptBrokerMissing = errors.New("analysis_run runs only through the agent runtime")

type runScriptTool struct {
	runner   scriptports.Runner
	datasets *scriptDatasets
	analyses analysisports.RunRecorder
}

type runScriptToolParams struct {
	Runner      scriptports.Runner
	Reports     reportRunner
	Files       documentports.ConversationFileReader
	Permissions permissionports.PermissionEngine
	Analyses    analysisports.RunRecorder
}

func newRunScriptTool(p *runScriptToolParams) agentports.AgentQueryTool {
	return &runScriptTool{
		runner: p.Runner,
		datasets: &scriptDatasets{
			reports:     p.Reports,
			files:       p.Files,
			permissions: p.Permissions,
		},
		analyses: p.Analyses,
	}
}

func (t *runScriptTool) Name() string { return codemode.ToolRunScript }

func (t *runScriptTool) Description() string {
	return "Run a short Python script that calls the reads you hold and totals, compares " +
		"or charts what they return. Use it for every record of a list, two " +
		"sources reconciled, a report run or attached spreadsheet analyzed, or any figure no " +
		"read gives; for one or two reads in a row, call the reads themselves. Call a read as " +
		"trenova.<namespace>.<the rest of its name>(**parameters), as " +
		"trenova.shipment.list(status=[\"InTransit\"]) for shipment_list; parameters and " +
		"results are exactly the tool's. trenova.tools() lists what you may call, " +
		"trenova.gather(a.defer(...), b.defer(...)) runs several at once, and a refused call " +
		"raises trenova.Refused. The person sees only what it shows with show_figure, " +
		"show_table and show_chart, kept as an analysis; what it prints and " +
		"trenova.result(value) reach only you. A script changes nothing: a change it calls is " +
		"proposed as one batch for a person to approve."
}

func (t *runScriptTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			paramScriptTitle: map[string]any{
				toolschema.KeyType: toolschema.TypeString,
				toolschema.KeyDescription: "A short name for what the script works out, three " +
					"to eight words, such as \"Customers past terms\".",
				toolschema.KeyMaxLength: maxScriptTitleRunes,
			},
			paramScriptPurpose: map[string]any{
				toolschema.KeyType: toolschema.TypeString,
				toolschema.KeyDescription: "What the script answers, in one plain sentence the " +
					"person will read.",
				toolschema.KeyMaxLength: maxScriptPurposeRunes,
			},
			paramScriptCode: map[string]any{
				toolschema.KeyType: toolschema.TypeString,
				toolschema.KeyDescription: "The Python to run, in a sandboxed interpreter: " +
					"trenova is already there; json, datetime, math, re, collections and " +
					"itertools import as usual; date.today() is the organization's today; " +
					"classes cannot inherit; there are no files, packages or network. Show " +
					"the person the answer with show_figure(label, value, kind), " +
					"show_table(rows, title) and show_chart(kind, x, y, title, value_kind), " +
					"kind bar, line, area, scatter or pie and y a list or {series: list}; " +
					"value kinds are money, count, percent (a fraction) and number. mean, " +
					"median, percentile, stdev and group_by are built in. print() and " +
					"trenova.result() are for you: the person sees only what is shown.",
				toolschema.KeyMaxLength: codemode.MaxCodeChars,
			},
			paramScriptInputs: map[string]any{
				toolschema.KeyType: toolschema.TypeArray,
				toolschema.KeyDescription: "Values the script reads from trenova.inputs by " +
					"name, such as a date or a list of ids, so the same script can run again " +
					"with others.",
				toolschema.KeyMaxItems: codemode.MaxInputKeys,
				toolschema.KeyItems: map[string]any{
					toolschema.KeyType: toolschema.TypeObject,
					toolschema.KeyProperties: map[string]any{
						paramInputName: map[string]any{
							toolschema.KeyType:        toolschema.TypeString,
							toolschema.KeyDescription: "The name the script reads it by.",
							toolschema.KeyMaxLength:   maxInputNameRunes,
						},
						paramInputValue: map[string]any{
							toolschema.KeyType:        toolschema.TypeString,
							toolschema.KeyDescription: "The value, as text.",
							toolschema.KeyMaxLength:   codemode.MaxInputBytes,
						},
					},
					toolschema.KeyRequired:             []string{paramInputName, paramInputValue},
					toolschema.KeyAdditionalProperties: false,
				},
			},
			paramScriptDatasets: datasetsSchema(),
			paramScriptContinues: map[string]any{
				toolschema.KeyType: toolschema.TypeString,
				toolschema.KeyDescription: "Only a scriptId an analysis_run answer of " +
					"this turn gave you, never a dataset's name; leave it out otherwise. This " +
					"code then runs after that script, with every name it left and over its " +
					"datasets, so send only what to add or show, without datasets or inputs; " +
					"it keeps that script's title and purpose unless you give others.",
				toolschema.KeyMaxLength: maxContinuesRunes,
			},
		},
		toolschema.KeyRequired: []string{
			paramScriptTitle, paramScriptPurpose, paramScriptCode,
		},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *runScriptTool) Policy() agentports.ToolPolicy {
	policy := readPolicy(t.Name(), readSpec{
		resource: permission.ResourceCodeMode,
		reads:    agent.ExternalReadMarked,
		source:   agent.TaintSourceRecordNote,
		rationale: "Runs a short script in an interpreter that cannot reach the network or a " +
			"file; every read it makes is one of the agent's own reads, run as the person under " +
			"the same checks, " +
			"and the call carries the taint of every read that brought outside text in. It " +
			"changes nothing itself: a change it prepares is proposed with the rest of its " +
			"batch for a person to approve.",
	})
	policy.Script = agentports.ScriptRunner
	policy.Sources = scriptTaintSources

	return policy
}

func (t *runScriptTool) SearchTerms() []string {
	return []string{
		"for each", "every one of", "each of them", "reconcile", "cross-check", "across all",
		"script", "batch read", "go through all", "analyze", "analysis", "analyst",
		"run analyst", "chart", "graph", "plot", "trend", "python", "spreadsheet", "calculate",
		"statistics", "compare periods",
	}
}

func datasetsSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeArray,
		toolschema.KeyDescription: "Tables the script reads in full, at most five, each " +
			"bound to its name as a list of row dicts keyed by its column labels, numbers as " +
			"numbers and times as dates; a text file is its text. Give each a name and " +
			"exactly one of reportRunId or documentId.",
		toolschema.KeyMaxItems: analysis.MaxDatasets,
		toolschema.KeyItems: map[string]any{
			toolschema.KeyType: toolschema.TypeObject,
			toolschema.KeyProperties: map[string]any{
				paramDatasetName: map[string]any{
					toolschema.KeyType: toolschema.TypeString,
					toolschema.KeyDescription: "The name the script reads it by: lower-case " +
						"letters, digits and underscores, starting with a letter, such as lanes.",
					toolschema.KeyMaxLength: analysis.MaxDatasetName,
				},
				paramDatasetRun: agenttoolschema.OfKinds(map[string]any{
					toolschema.KeyType: toolschema.TypeString,
					toolschema.KeyDescription: "A finished report run you ran for this person, " +
						"from report_run.",
				}, permission.KindReportRun),
				paramDatasetDocument: agenttoolschema.RecordIDText(
					permission.ResourceDocument,
					"A CSV, XLSX or text file the person attached to this conversation, "+
						"from agent_read_attachment.",
				),
			},
			toolschema.KeyRequired:             []string{paramDatasetName},
			toolschema.KeyAdditionalProperties: false,
		},
	}
}

type scriptCallAnswer struct {
	Seq      int    `json:"seq"`
	Tool     string `json:"tool"`
	Status   string `json:"status"`
	Rows     int    `json:"rows,omitempty"`
	Bytes    int    `json:"bytes,omitempty"`
	BrokerMs int64  `json:"brokerMs"`
	ReplyMs  int64  `json:"replyMs,omitempty"`
	Message  string `json:"message,omitempty"`
}

type scriptLimitsAnswer struct {
	Calls     int    `json:"calls"`
	MaxCalls  int    `json:"maxCalls"`
	Rows      int    `json:"rows"`
	MaxRows   int    `json:"maxRows"`
	Bytes     int    `json:"bytes"`
	MaxBytes  int    `json:"maxBytes"`
	Seconds   int    `json:"seconds"`
	Hit       string `json:"hit,omitempty"`
	Truncated bool   `json:"outputTruncated,omitempty"`
}

type scriptAnswer struct {
	Title            string                    `json:"title"`
	Status           string                    `json:"status"`
	Output           string                    `json:"output"`
	Result           sonic.NoCopyRawMessage    `json:"result,omitempty"`
	Calls            []scriptCallAnswer        `json:"calls"`
	Refusals         []scriptCallAnswer        `json:"refusals,omitempty"`
	Limits           scriptLimitsAnswer        `json:"limits"`
	Error            *codemode.ScriptError     `json:"error,omitempty"`
	Failure          string                    `json:"failure,omitempty"`
	DurationMs       int64                     `json:"durationMs"`
	WithheldByAccess []string                  `json:"withheldByAccess,omitempty"`
	Changes          []codemode.PreparedChange `json:"changes,omitempty"`
	AnalysisRunID    string                    `json:"analysisRunId,omitempty"`
	Code             string                    `json:"code,omitempty"`
	ScriptID         string                    `json:"scriptId,omitempty"`
	Purpose          string                    `json:"purpose,omitempty"`
	Datasets         []scriptDatasetAnswer     `json:"datasets,omitempty"`
	Figures          []analysis.Figure         `json:"figures,omitempty"`
	Tables           []analysis.Table          `json:"tables,omitempty"`
	Charts           []analysis.Chart          `json:"charts,omitempty"`
	Layout           []analysis.Block          `json:"layout,omitempty"`
	NotShown         int                       `json:"notShown,omitempty"`
	Note             string                    `json:"note"`

	attachments  []agent.RecordRef
	continuation codemode.Continuation
}

type scriptDatasetAnswer struct {
	Name    string                      `json:"name"`
	Source  string                      `json:"source"`
	Label   string                      `json:"label,omitempty"`
	Rows    int                         `json:"rows"`
	Columns []scriptports.DatasetColumn `json:"columns,omitempty"`
	Text    bool                        `json:"text,omitempty"`
}

// TaintedRecords names the attached files the script read: what the person
// attached may have been written outside the organization.
func (a *scriptAnswer) TaintedRecords() []agent.RecordRef { return a.attachments }

// ScriptCompleted reports a script that ran to its end, the only one whose
// prepared changes are proposed.
func (a *scriptAnswer) ScriptCompleted() bool {
	return a.Status == string(scriptports.Succeeded)
}

func (a *scriptAnswer) ScriptContinuation() codemode.Continuation { return a.continuation }

func (t *runScriptTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if params.Broker == nil {
		return nil, errScriptBrokerMissing
	}

	run, err := scriptRunOf(params)
	if err != nil {
		return nil, err
	}
	title, purpose, code := run.title, run.purpose, run.code
	requests, err := datasetsFrom(run.datasets)
	if err != nil {
		return nil, err
	}
	resolved, err := t.datasets.resolve(ctx, params, requests)
	if err != nil {
		return nil, err
	}

	started := time.Now()
	outcome, err := t.runner.Run(ctx, &scriptports.Request{
		TenantInfo: pagination.TenantInfo{
			OrgID:  params.OrganizationID,
			BuID:   params.BusinessUnitID,
			UserID: params.Actor.UserID,
		},
		Title:    stringutils.TruncateRunes(title, maxScriptTitleRunes),
		Purpose:  stringutils.TruncateRunes(purpose, maxScriptPurposeRunes),
		Code:     code,
		Earlier:  run.earlier,
		Inputs:   scriptInputs(run.inputs),
		Datasets: resolved.datasets,
		Timezone: params.Timezone,
		Broker:   params.Broker,
		StepKey:  params.StepKey,
	})
	summary := params.Broker.Summary()
	if err != nil {
		markScriptSpan(ctx, &summary, nil)
		return nil, err
	}
	markScriptSpan(ctx, &summary, outcome)

	answer := scriptAnswerOf(title, outcome, &summary)
	answer.attachments = resolved.attachments
	answer.Datasets = datasetAnswers(resolved.datasets)
	if err = t.keepAnalysis(ctx, &analysisRecord{
		params:   params,
		answer:   answer,
		outcome:  outcome,
		purpose:  purpose,
		code:     run.wholeCode(),
		datasets: resolved.datasets,
		started:  started,
	}); err != nil {
		return nil, err
	}
	markShowing(ctx, &showing{
		unshown: len(resolved.datasets) > 0 && outcome.Status == scriptports.Succeeded &&
			outcome.Shown.Empty(),
		continues: run.earlier != "",
	})

	return answer, nil
}

// showing is whether a script that read a dataset showed nothing, and whether
// it continued one that had: how often a model forgets to show, and how often
// the continuation recovers it.
type showing struct {
	unshown   bool
	continues bool
}

func markShowing(ctx context.Context, s *showing) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}
	span.SetAttributes(
		aitrace.AICodeModeUnshown.Bool(s.unshown),
		aitrace.AICodeModeContinues.Bool(s.continues),
	)
}

// continuable names a script a later call of the turn may continue, and how:
// any that ran to the end, to add to what it showed, and one that read a
// dataset and showed nothing, to show it, at no cost to the turn's scripts. A
// script that prepared changes is never continued: running it again would
// propose them twice.
func continuable(r *analysisRecord) {
	if r.outcome.Status != scriptports.Succeeded || r.params.CallID == "" ||
		len(r.answer.Changes) > 0 {
		return
	}
	r.answer.ScriptID = r.params.CallID
	r.answer.continuation = codemode.ContinueCounted
	if r.outcome.Shown.Empty() && len(r.datasets) > 0 && r.params.Continues == nil {
		r.answer.continuation = codemode.ContinueFree
	}
}

// scriptRun is what a call runs: its own code, or for a continuation its code
// after the earlier script's, over that script's datasets and inputs.
type scriptRun struct {
	title    string
	purpose  string
	code     string
	earlier  string
	datasets any
	inputs   any
}

var errContinuationReadsItsOwn = errors.New(
	"a continuation runs over the datasets and inputs of the script it continues; " +
		"leave datasets and inputs out",
)

func scriptRunOf(params *agentports.QueryToolParams) (*scriptRun, error) {
	code, _ := params.Params[paramScriptCode].(string)
	run := &scriptRun{
		title:    optionalString(params.Params, paramScriptTitle),
		purpose:  optionalString(params.Params, paramScriptPurpose),
		code:     code,
		datasets: params.Params[paramScriptDatasets],
		inputs:   params.Params[paramScriptInputs],
	}
	earlier := params.Continues
	if earlier == nil {
		return run, nil
	}
	if hasItems(run.datasets) || hasItems(run.inputs) {
		return nil, errContinuationReadsItsOwn
	}
	run.earlier, _ = earlier[paramScriptCode].(string)
	run.datasets = earlier[paramScriptDatasets]
	run.inputs = earlier[paramScriptInputs]
	if run.title == "" {
		run.title = optionalString(earlier, paramScriptTitle)
	}
	if run.purpose == "" {
		run.purpose = optionalString(earlier, paramScriptPurpose)
	}

	return run, nil
}

// wholeCode is the code an analysis keeps: a continuation's after the
// earlier script's, so the card shows everything that ran.
func (r *scriptRun) wholeCode() string {
	if r.earlier == "" {
		return r.code
	}

	return strings.TrimRight(r.earlier, "\n") + "\n\n" + r.code
}

func hasItems(value any) bool {
	items, _ := value.([]any)

	return len(items) > 0
}

type analysisRecord struct {
	params   *agentports.QueryToolParams
	answer   *scriptAnswer
	outcome  *scriptports.Outcome
	purpose  string
	code     string
	datasets []scriptports.Dataset
	started  time.Time
}

// keepAnalysis records a script that showed the person something, so the
// Desk's card can read its tables past the rows the agent is shown, and tells
// the agent what the person sees. A script that only printed is the agent
// looking at the data and is not kept; one that read a dataset and only
// printed is an analysis the person never saw, so the agent is told to show it
// by continuing it, which the turn does not count as another script.
func (t *runScriptTool) keepAnalysis(ctx context.Context, r *analysisRecord) error {
	continuable(r)
	shown := r.outcome.Shown
	if shown.Empty() {
		if r.outcome.Status != scriptports.Succeeded {
			return nil
		}
		if r.answer.continuation == codemode.ContinueFree {
			r.answer.Note += scriptUnshownAnalysisNote
		} else {
			r.answer.Note += scriptPlainNote
		}
		return nil
	}

	run := &analysis.Run{
		BusinessUnitID:    r.params.BusinessUnitID,
		OrganizationID:    r.params.OrganizationID,
		Status:            analysis.Status(r.outcome.Status),
		Title:             r.answer.Title,
		Purpose:           r.purpose,
		Code:              r.code,
		Datasets:          datasetRecords(r.datasets),
		Output:            r.outcome.Output,
		Tables:            shown.Tables,
		Figures:           shown.Figures,
		Charts:            shown.Charts,
		Layout:            shown.Layout,
		Failure:           r.outcome.Failure,
		DurationMs:        r.outcome.DurationMs,
		AgentDefinitionID: r.params.AgentDefinitionID,
		AgentRunID:        r.params.RunID,
		ThreadID:          r.params.ThreadID,
		RequestedByID:     personOf(r.params.Actor),
		StartedAt:         r.started.Unix(),
		FinishedAt:        time.Now().Unix(),
	}
	if r.outcome.Error != nil {
		run.ErrorName = r.outcome.Error.Name
		run.ErrorValue = r.outcome.Error.Value
		run.Traceback = r.outcome.Error.Traceback
	}
	recorded, err := t.analyses.Record(ctx, run)
	if err != nil {
		return fmt.Errorf("record the analysis: %w", err)
	}

	r.answer.AnalysisRunID = recorded.ID.String()
	if r.params.Continues != nil {
		r.answer.Code = r.code
	}
	r.answer.Purpose = recorded.Purpose
	r.answer.Figures = recorded.Figures
	r.answer.Tables = answerTables(recorded.Tables)
	r.answer.Charts = recorded.Charts
	r.answer.Layout = recorded.Layout
	r.answer.NotShown = r.outcome.NotShown
	r.answer.Note += scriptShownNote
	if r.outcome.NotShown > 0 {
		r.answer.Note += scriptNotShownNote
	}

	return nil
}

// answerTables are the tables as the agent reads them: each with its first
// rows, and how many it has in all.
func answerTables(tables []analysis.Table) []analysis.Table {
	answers := make([]analysis.Table, 0, len(tables))
	for idx := range tables {
		table := tables[idx]
		table.Rows = table.Rows[:min(len(table.Rows), maxAnswerTableRows)]
		answers = append(answers, table)
	}

	return answers
}

func datasetAnswers(datasets []scriptports.Dataset) []scriptDatasetAnswer {
	answers := make([]scriptDatasetAnswer, 0, len(datasets))
	for idx := range datasets {
		dataset := &datasets[idx]
		answers = append(answers, scriptDatasetAnswer{
			Name:    dataset.Name,
			Source:  dataset.Source.String(),
			Label:   dataset.Label,
			Rows:    len(dataset.Rows),
			Columns: dataset.Columns,
			Text:    dataset.Text != nil,
		})
	}

	return answers
}

func datasetRecords(datasets []scriptports.Dataset) []analysis.Dataset {
	records := make([]analysis.Dataset, 0, len(datasets))
	for idx := range datasets {
		dataset := &datasets[idx]
		records = append(records, analysis.Dataset{
			Name:     dataset.Name,
			Source:   dataset.Source,
			SourceID: dataset.SourceID,
			Label:    dataset.Label,
			Rows:     len(dataset.Rows),
		})
	}

	return records
}

func scriptInputs(raw any) map[string]any {
	list, _ := raw.([]any)
	inputs := make(map[string]any, len(list))
	for _, entry := range list {
		item, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		name, _ := item[paramInputName].(string)
		value, _ := item[paramInputValue].(string)
		if name = strings.TrimSpace(name); name != "" {
			inputs[name] = value
		}
	}

	return inputs
}

var scriptTaintSources = []agent.TaintSource{
	agent.TaintSourceInboundMessage,
	agent.TaintSourceDocument,
	agent.TaintSourceEDI,
	agent.TaintSourceBankReceipt,
	agent.TaintSourceWeather,
	agent.TaintSourceAttachment,
	agent.TaintSourceMemory,
	agent.TaintSourceRunRecord,
}

func scriptAnswerOf(
	title string,
	outcome *scriptports.Outcome,
	summary *codemode.Summary,
) *scriptAnswer {
	answer := &scriptAnswer{
		Title:            title,
		Status:           string(outcome.Status),
		Output:           outcome.Output,
		Result:           outcome.Result,
		Calls:            make([]scriptCallAnswer, 0, len(summary.Calls)),
		Error:            boundedError(outcome.Error),
		Failure:          outcome.Failure,
		DurationMs:       outcome.DurationMs,
		WithheldByAccess: summary.Withheld,
		Limits: scriptLimitsAnswer{
			Calls:     summary.Usage.Calls,
			MaxCalls:  outcome.Limits.Calls,
			Rows:      summary.Usage.Rows,
			MaxRows:   outcome.Limits.Rows,
			Bytes:     summary.Usage.Bytes,
			MaxBytes:  outcome.Limits.Bytes,
			Seconds:   int(outcome.Limits.Wall.Seconds()),
			Hit:       string(hitOf(outcome, summary)),
			Truncated: outcome.OutputTruncated,
		},
	}
	for idx := range summary.Calls {
		call := summary.Calls[idx]
		entry := scriptCallAnswer{
			Seq:      call.Seq,
			Tool:     call.Tool,
			Status:   string(call.Status),
			Rows:     call.Rows,
			Bytes:    call.Bytes,
			BrokerMs: call.BrokerMs,
			ReplyMs:  call.ReplyMs,
		}
		if call.Status != codemode.CallOK {
			entry.Message = call.Message
			answer.Refusals = append(answer.Refusals, entry)
		}
		answer.Calls = append(answer.Calls, entry)
	}

	switch outcome.Status {
	case scriptports.Succeeded:
		answer.Note = scriptDoneNote
	case scriptports.Errored:
		answer.Note = scriptErrorNote
	case scriptports.TimedOut, scriptports.Stopped,
		scriptports.Failed:
		answer.Note = scriptStoppedNote
	}
	answer.Note += batchNote(answer, &summary.Batch)

	return answer
}

func hitOf(outcome *scriptports.Outcome, summary *codemode.Summary) codemode.Limit {
	if outcome.Limit != codemode.LimitNone {
		return outcome.Limit
	}

	return summary.Usage.Hit
}

func boundedError(err *codemode.ScriptError) *codemode.ScriptError {
	if err == nil {
		return nil
	}
	bounded := *err
	bounded.Value = stringutils.TruncateRunes(bounded.Value, maxScriptTraceback)
	if len(bounded.Traceback) > maxScriptTraceback {
		bounded.Traceback = "…" + bounded.Traceback[len(bounded.Traceback)-maxScriptTraceback:]
	}

	return &bounded
}

func markScriptSpan(
	ctx context.Context,
	summary *codemode.Summary,
	outcome *scriptports.Outcome,
) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}
	attrs := []attribute.KeyValue{
		aitrace.AICodeModeCalls.Int(summary.Usage.Calls),
		aitrace.AICodeModeRows.Int(summary.Usage.Rows),
		aitrace.AICodeModeBytes.Int(summary.Usage.Bytes),
	}
	broker := make([]int64, 0, len(summary.Calls))
	reply := make([]int64, 0, len(summary.Calls))
	for idx := range summary.Calls {
		broker = append(broker, summary.Calls[idx].BrokerMs)
		if summary.Calls[idx].ReplyMs > 0 {
			reply = append(reply, summary.Calls[idx].ReplyMs)
		}
	}
	attrs = append(attrs,
		aitrace.AICodeModeBrokerMs.Int64Slice(broker),
		aitrace.AICodeModeReplyMs.Int64Slice(reply),
	)
	if outcome != nil {
		attrs = append(attrs,
			aitrace.AICodeModeStatus.String(string(outcome.Status)),
		)
		if hit := hitOf(outcome, summary); hit != codemode.LimitNone {
			attrs = append(attrs, aitrace.AICodeModeLimitHit.String(string(hit)))
		}
	}
	span.SetAttributes(attrs...)
}

// batchNote says what became of the changes a script prepared: proposed as
// one batch for a person, or none of them, and why.
func batchNote(answer *scriptAnswer, batch *codemode.Batch) string {
	if len(batch.Changes) == 0 && batch.Refusal == "" {
		return ""
	}
	answer.Changes = batch.Changes
	switch {
	case batch.Refusal != "":
		return scriptBatchRefusedNote + batch.Refusal + "."
	case !answer.ScriptCompleted():
		return scriptBatchDroppedNote
	default:
		return scriptBatchFiledNote
	}
}
