package agentquerytoolservice

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	paramScriptTitle   = "title"
	paramScriptPurpose = "purpose"
	paramScriptCode    = "code"
	paramScriptInputs  = "inputs"
	paramInputName     = "name"
	paramInputValue    = "value"
	maxInputNameRunes  = 64

	maxScriptTitleRunes   = 80
	maxScriptPurposeRunes = 500
	maxScriptTraceback    = 3000

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
)

var errScriptBrokerMissing = errors.New("agent_run_script runs only through the agent runtime")

type runScriptTool struct {
	runner agentextensionports.ScriptRunner
}

func newRunScriptTool(runner agentextensionports.ScriptRunner) agentports.AgentQueryTool {
	return &runScriptTool{runner: runner}
}

func (t *runScriptTool) Name() string { return agentextension.ToolRunScript }

func (t *runScriptTool) Description() string {
	return "Run a short read-only Python script that calls the reads you hold, for questions " +
		"that need many reads joined, compared or totalled. Use it to go over every record of " +
		"a list (each customer past terms with their last payment), to reconcile two sources, " +
		"or for a report across many records; for one or two reads in a row, call the reads " +
		"themselves. Call a read as trenova.<namespace>.<the rest of its name>(**parameters), " +
		"such as trenova.shipment.list(status=[\"InTransit\"]) for shipment_list; parameters " +
		"and results are exactly the tool's. trenova.tools() lists the reads, " +
		"help(trenova.shipment.list) gives one's parameters, trenova.gather(a.defer(...), " +
		"b.defer(...)) runs several at once, and a refused read raises trenova.Refused. " +
		"Only what you print and one trenova.result(value) come back. A script changes " +
		"nothing and cannot reach the internet."
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
				toolschema.KeyDescription: "The Python to run. import trenova is done for you; " +
					"print() what you need to answer and pass one summary to trenova.result().",
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
		rationale: "Runs a short script in a sandbox that cannot reach the network; every read " +
			"it makes is one of the agent's own reads, run as the person under the same checks, " +
			"and the call carries the taint of every read that brought outside text in. It " +
			"changes nothing.",
	})
	policy.Script = agentports.ScriptRunner
	policy.Sources = scriptTaintSources

	return policy
}

func (t *runScriptTool) SearchTerms() []string {
	return []string{
		"for each", "every one of", "each of them", "reconcile", "cross-check", "across all",
		"script", "batch read", "go through all",
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
	Note             string                    `json:"note"`
}

// ScriptCompleted reports a script that ran to its end, the only one whose
// prepared changes are proposed.
func (a *scriptAnswer) ScriptCompleted() bool {
	return a.Status == string(agentextensionports.ScriptSucceeded)
}

func (t *runScriptTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if params.Broker == nil {
		return nil, errScriptBrokerMissing
	}

	title := optionalString(params.Params, paramScriptTitle)
	purpose := optionalString(params.Params, paramScriptPurpose)
	code, _ := params.Params[paramScriptCode].(string)
	inputs := scriptInputs(params.Params[paramScriptInputs])

	outcome, err := t.runner.RunScript(ctx, &agentextensionports.ScriptRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  params.OrganizationID,
			BuID:   params.BusinessUnitID,
			UserID: params.Actor.UserID,
		},
		Title:   stringutils.TruncateRunes(title, maxScriptTitleRunes),
		Purpose: stringutils.TruncateRunes(purpose, maxScriptPurposeRunes),
		Code:    code,
		Inputs:  inputs,
		Broker:  params.Broker,
		StepKey: params.StepKey,
	})
	summary := params.Broker.Summary()
	if err != nil {
		markScriptSpan(ctx, &summary, nil)
		return nil, err
	}
	markScriptSpan(ctx, &summary, outcome)

	return scriptAnswerOf(title, outcome, &summary), nil
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
	outcome *agentextensionports.ScriptOutcome,
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
	case agentextensionports.ScriptSucceeded:
		answer.Note = scriptDoneNote
	case agentextensionports.ScriptErrored:
		answer.Note = scriptErrorNote
	case agentextensionports.ScriptTimedOut, agentextensionports.ScriptStopped,
		agentextensionports.ScriptFailed:
		answer.Note = scriptStoppedNote
	}
	answer.Note += batchNote(answer, &summary.Batch)

	return answer
}

func hitOf(outcome *agentextensionports.ScriptOutcome, summary *codemode.Summary) codemode.Limit {
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
	outcome *agentextensionports.ScriptOutcome,
) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}
	attrs := []attribute.KeyValue{
		aitrace.AICodeModeCalls.Int(summary.Usage.Calls),
		aitrace.AICodeModeRows.Int(summary.Usage.Rows),
		aitrace.AICodeModeBytes.Int(summary.Usage.Bytes),
		aitrace.AICodeModePollMs.Int64(codemode.PollInterval.Milliseconds()),
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
			aitrace.AICodeModeRoundTripMs.Int64Slice(slices.Clone(outcome.RoundTripsMs)),
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
