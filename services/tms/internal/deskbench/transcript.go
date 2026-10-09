package deskbench

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	resultLimit   = 4000
	thinkingLimit = 2500
	textLimit     = 4000
	eventLimit    = 800
	argsLimit     = 2000
	passMark      = "PASS"
	failMark      = "FAIL"
	checkPass     = "[x]"
	checkFail     = "[ ]"
)

var quietEvents = map[string]bool{
	"accepted":      true,
	"done":          true,
	"turn":          true,
	"thread":        true,
	"context":       true,
	"message":       true,
	"tool_started":  true,
	"tool_finished": true,
	"artifact":      true,
	"next_turn":     true,
}

func renderTranscript(r *Report) string {
	var b strings.Builder
	b.Grow(64 * 1024)

	fmt.Fprintf(&b, "# Desk bench transcript\n\n")
	fmt.Fprintf(&b, "Started %s, %d cases, %d passed. Worker log: `%s`.\n\n",
		r.Meta.StartedAt.Format(time.RFC3339), r.Summary.Cases, r.Summary.Passed, r.Meta.WorkerLog)

	for _, result := range r.Cases {
		renderCase(&b, result)
	}

	return b.String()
}

func renderCase(b *strings.Builder, result *CaseResult) {
	mark := passMark
	if !result.Passed {
		mark = failMark
	}
	fmt.Fprintf(b, "---\n\n## %s %s #%d\n\n", mark, result.Key(), result.Repeat)
	if result.Scenario.Description != "" {
		fmt.Fprintf(b, "%s\n\n", strings.TrimSpace(result.Scenario.Description))
	}
	fmt.Fprintf(b, "- Agent: %s\n- Thread: `%s`\n- Took: %s\n- %s\n",
		result.Agent, result.ThreadID,
		result.FinishedAt.Sub(result.StartedAt).Round(100*time.Millisecond), usageLine(result.Usage))
	if result.Scenario.Source != "" {
		fmt.Fprintf(b, "- Scenario: `%s`\n", result.Scenario.Source)
	}
	if result.Error != "" {
		fmt.Fprintf(b, "- **Error:** %s\n", result.Error)
	}
	b.WriteString("\n")

	for _, step := range result.Steps {
		renderStep(b, step)
	}
	if len(result.Cleanup) > 0 {
		b.WriteString("### Cleanup (leftover proposals rejected)\n\n")
		for _, turn := range result.Cleanup {
			renderTurn(b, turn)
		}
	}
}

func renderStep(b *strings.Builder, step *StepResult) {
	mark := passMark
	if !step.Passed {
		mark = failMark
	}
	fmt.Fprintf(b, "### Step %d %s: %s %s\n\n", step.Index, mark, step.Kind, quoteLine(step.Label))
	if step.Error != "" {
		fmt.Fprintf(b, "**Step error:** %s\n\n", step.Error)
	}

	for _, check := range step.Checks {
		box := checkPass
		if !check.Passed {
			box = checkFail
		}
		fmt.Fprintf(b, "- %s %s", box, check.Name)
		if check.Detail != "" {
			fmt.Fprintf(b, " — %s", check.Detail)
		}
		b.WriteString("\n")
	}
	if step.Formula != "" {
		fmt.Fprintf(b, "- Formula studio draft after this step: `%s`\n", step.Formula)
	}
	if step.Rubric != "" {
		fmt.Fprintf(b, "- Rubric (judge by reading): %s\n", strings.TrimSpace(step.Rubric))
	}
	b.WriteString("\n")

	for _, decision := range step.Decisions {
		renderDecision(b, decision)
	}
	for _, turn := range step.Turns {
		renderTurn(b, turn)
	}
}

func renderDecision(b *strings.Builder, decision *DecisionRecord) {
	target := "proposal `" + decision.ProposalID.String() + "`"
	if decision.PlanID.IsNotNil() {
		target = "plan `" + decision.PlanID.String() + "`"
	}
	fmt.Fprintf(b, "- Decided %s (%s): %s", target, decision.Tool, decision.Decision)
	if decision.Note != "" {
		fmt.Fprintf(b, ", note %q", decision.Note)
	}
	fmt.Fprintf(b, " → %s", decision.StatusAfter)
	if decision.Executed {
		b.WriteString(", ran")
	}
	if decision.Error != "" {
		fmt.Fprintf(b, ", **error:** %s", decision.Error)
	}
	if decision.ExecutionError != "" {
		fmt.Fprintf(b, ", **execution error:** %s", decision.ExecutionError)
	}
	b.WriteString("\n")
	if decision.Result != nil {
		writeFenced(b, "json", compactJSON(decision.Result, resultLimit))
	}
	b.WriteString("\n")
}

func renderTurn(b *strings.Builder, turn *TurnRecord) {
	fmt.Fprintf(b, "#### Turn `%s` · %s · %s · %s\n\n",
		turn.TurnID, turn.Origin, turn.Status, turn.Duration().Round(100*time.Millisecond))
	fmt.Fprintf(b, "%s · models: %s", usageLine(turn.Usage), strings.Join(turn.Models, ", "))
	if turn.TraceID != "" {
		fmt.Fprintf(b, " · trace `%s`", turn.TraceID)
	}
	b.WriteString("\n\n")

	if turn.Input != "" {
		fmt.Fprintf(b, "**Person:** %s\n\n", turn.Input)
	}
	if turn.Refused {
		b.WriteString("**Refused.**\n\n")
	}
	if turn.Failure != "" {
		fmt.Fprintf(b, "**Failure:** %s\n\n", turn.Failure)
	}
	if turn.CaptureError != "" {
		fmt.Fprintf(b, "**Bench capture problem:** %s\n\n", turn.CaptureError)
	}

	for _, item := range timeline(turn) {
		item.render(b)
	}

	for _, proposal := range turn.Proposals {
		renderProposal(b, proposal)
	}

	if len(turn.Unattributed) > 0 {
		fmt.Fprintf(b, "_%d model calls in this window carried no thread (guards, classifiers):_ ",
			len(turn.Unattributed))
		parts := make([]string, 0, len(turn.Unattributed))
		for _, call := range turn.Unattributed {
			parts = append(parts, fmt.Sprintf("#%d %s %s", call.Seq, call.Kind, call.Model))
		}
		b.WriteString(strings.Join(parts, ", ") + "\n\n")
	}

	b.WriteString("**Reply:**\n\n")
	if strings.TrimSpace(turn.Reply) == "" {
		b.WriteString("_(none)_\n\n")
	} else {
		writeQuoted(b, turn.Reply)
	}
}

type timelineItem struct {
	at     time.Time
	order  int
	render func(*strings.Builder)
}

func timeline(turn *TurnRecord) []timelineItem {
	items := make([]timelineItem, 0, len(turn.Calls)+len(turn.Tools)+len(turn.Events))

	starts := make(map[int]time.Time, len(turn.ModelCalls))
	for _, call := range turn.ModelCalls {
		starts[call.Seq] = call.StartedAt
	}
	for idx := range turn.Calls {
		call := turn.Calls[idx]
		items = append(items, timelineItem{
			at:     starts[call.Seq],
			order:  0,
			render: func(b *strings.Builder) { renderCall(b, &call) },
		})
	}
	for _, tool := range turn.Tools {
		items = append(items, timelineItem{
			at:     tool.StartedAt,
			order:  1,
			render: func(b *strings.Builder) { renderTool(b, tool) },
		})
	}
	for _, event := range turn.Events {
		if quietEvents[event.Name] {
			continue
		}
		items = append(items, timelineItem{
			at:     event.At,
			order:  2,
			render: func(b *strings.Builder) { renderEvent(b, event) },
		})
	}

	slices.SortStableFunc(items, func(a, b timelineItem) int {
		if c := a.at.Compare(b.at); c != 0 {
			return c
		}

		return a.order - b.order
	})

	return items
}

func renderCall(b *strings.Builder, call *CallSummary) {
	fmt.Fprintf(b, "**Model call #%d** · %s · %s · %d in / %d out", call.Seq, call.Kind,
		stringutils.WithDefault(call.Model, "?"), call.InputTokens, call.OutputTokens)
	if call.ReasoningTokens > 0 {
		fmt.Fprintf(b, " (%d thinking)", call.ReasoningTokens)
	}
	fmt.Fprintf(b, " · %dms · %d messages, %d tools offered", call.LatencyMs, call.Messages, call.Tools)
	if call.Delegate != "" {
		fmt.Fprintf(b, " · delegate `%s`", call.Delegate)
	}
	if call.File != "" {
		fmt.Fprintf(b, " · `%s`", call.File)
	}
	b.WriteString("\n\n")

	for _, note := range callNotes(call) {
		fmt.Fprintf(b, "- **%s**\n", note)
	}
	if strings.TrimSpace(call.Thinking) != "" {
		b.WriteString("<details><summary>thinking</summary>\n\n")
		writeQuoted(b, stringutils.Ellipsize(call.Thinking, thinkingLimit))
		b.WriteString("</details>\n\n")
	}
	if strings.TrimSpace(call.Text) != "" {
		writeQuoted(b, stringutils.Ellipsize(call.Text, textLimit))
	}
	for idx := range call.ToolCalls {
		requested := &call.ToolCalls[idx]
		fmt.Fprintf(b, "- asks for `%s` `%s`\n", requested.Name,
			compactJSON(requested.Arguments, argsLimit))
		if requested.Why != nil {
			fmt.Fprintf(b, "  - why: saw %q, because %q", requested.Why.Saw, requested.Why.Because)
			if requested.Why.InsteadOf != "" {
				fmt.Fprintf(b, ", instead of %q", requested.Why.InsteadOf)
			}
			b.WriteString("\n")
		}
	}
	if len(call.ToolCalls) > 0 {
		b.WriteString("\n")
	}
}

func renderTool(b *strings.Builder, tool *ToolCallRecord) {
	elapsed := ""
	if tool.Finished {
		elapsed = " · " + tool.FinishedAt.Sub(tool.StartedAt).Round(time.Millisecond).String()
	}
	who := ""
	if tool.DelegateCallID != "" {
		who = " · delegate `" + tool.DelegateCallID + "`"
	}
	flag := ""
	if tool.Refused() {
		flag = " **REFUSED/FAILED**"
	}
	fmt.Fprintf(b, "**Tool `%s`** · %s%s%s%s\n\n", tool.Name, verdictLabel(tool), elapsed, who, flag)
	if tool.Summary != "" {
		fmt.Fprintf(b, "%s\n\n", tool.Summary)
	}
	fmt.Fprintf(b, "args: `%s`\n\n", compactJSON(tool.Arguments, argsLimit))
	if tool.Result != "" {
		writeFenced(b, "", stringutils.Ellipsize(tool.Result, resultLimit))
	}
}

func renderEvent(b *strings.Builder, event Event) {
	fmt.Fprintf(b, "_event `%s`_ `%s`\n\n", event.Name, compactJSON(event.Data, eventLimit))
}

func renderProposal(b *strings.Builder, proposal *ProposalRecord) {
	fmt.Fprintf(b, "**Proposal** `%s` · `%s` · %s", proposal.ID, proposal.Tool, proposal.Status)
	if proposal.Tier != "" {
		fmt.Fprintf(b, " · tier %s", proposal.Tier)
	}
	if len(proposal.HeldBy) > 0 {
		fmt.Fprintf(b, " · held by %s", strings.Join(proposal.HeldBy, ", "))
	}
	if proposal.PlanID.IsNotNil() {
		fmt.Fprintf(b, " · plan `%s` step %d", proposal.PlanID, proposal.PlanStep)
	}
	b.WriteString("\n\n")
	if proposal.Rationale != "" {
		fmt.Fprintf(b, "rationale: %s\n\n", proposal.Rationale)
	}
	fmt.Fprintf(b, "args: `%s`\n\n", compactJSON(proposal.Arguments, argsLimit))
	if proposal.Error != "" {
		fmt.Fprintf(b, "**Bench could not read it fully:** %s\n\n", proposal.Error)
	}
	if preview := proposal.Preview; preview != nil {
		fmt.Fprintf(b, "preview: %s (coverage %s, %d changes, %d warnings)\n\n",
			preview.Summary, preview.Coverage, len(preview.Changes), len(preview.Warnings))
		if len(preview.Warnings) > 0 || len(preview.Changes) > 0 {
			writeFenced(b, "json", compactJSON(map[string]any{
				"warnings": preview.Warnings,
				"changes":  preview.Changes,
			}, resultLimit))
		}
	}
	if proposal.Simulated != nil {
		writeFenced(b, "json", compactJSON(proposal.Simulated, resultLimit))
	}
}

func usageLine(usage Usage) string {
	line := fmt.Sprintf("%d model calls, %d in / %d out tokens", usage.ModelCalls,
		usage.InputTokens, usage.OutputTokens)
	if usage.ReasoningTokens > 0 {
		line += " (" + strconv.Itoa(usage.ReasoningTokens) + " thinking)"
	}
	if usage.CostUSD != nil {
		line += ", $" + usage.CostUSD.StringFixed(4)
	}

	return line
}

func compactJSON(value any, limit int) string {
	if value == nil {
		return "{}"
	}
	encoded, err := reportJSON.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}

	return stringutils.Ellipsize(string(encoded), limit)
}

func writeFenced(b *strings.Builder, lang, text string) {
	fence := "```"
	for strings.Contains(text, fence) {
		fence += "`"
	}
	fmt.Fprintf(b, "%s%s\n%s\n%s\n\n", fence, lang, strings.TrimRight(text, "\n"), fence)
}

func writeQuoted(b *strings.Builder, text string) {
	for line := range strings.SplitSeq(strings.TrimRight(text, "\n"), "\n") {
		b.WriteString("> ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
}

func quoteLine(text string) string {
	quoted, err := sonic.MarshalString(firstLine(text))
	if err != nil {
		return text
	}

	return quoted
}
