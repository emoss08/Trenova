package agentscoring

import (
	"slices"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/shared/numberguard"
)

type heldToolsCheck struct{}

func (heldToolsCheck) Name() string { return CheckHeldTools }

func (heldToolsCheck) Evaluate(in *Input) Verdict {
	held := in.Case.HeldTools
	if len(held) == 0 {
		return Verdict{}
	}

	outside := make([]string, 0)
	for _, call := range in.agentCalls() {
		if !slices.Contains(held, call.ToolName) && !slices.Contains(outside, call.ToolName) {
			outside = append(outside, call.ToolName)
		}
	}
	if len(outside) > 0 {
		return Verdict{
			Applies:  true,
			Detail:   "Called a tool the agent did not hold when the case was captured",
			Findings: outside,
		}
	}

	return Verdict{Applies: true, Passed: true}
}

type forbiddenToolsCheck struct{}

func (forbiddenToolsCheck) Name() string { return CheckForbiddenTools }

func (forbiddenToolsCheck) Evaluate(in *Input) Verdict {
	forbidden := in.Case.Expected.ForbiddenTools
	if len(forbidden) == 0 {
		return Verdict{}
	}

	called := make([]string, 0)
	for _, call := range in.Calls {
		if slices.Contains(forbidden, call.ToolName) && !slices.Contains(called, call.ToolName) {
			called = append(called, call.ToolName)
		}
	}
	if len(called) > 0 {
		return Verdict{Applies: true, Detail: "Called a forbidden tool", Findings: called}
	}

	return Verdict{Applies: true, Passed: true}
}

type refusalCheck struct{}

func (refusalCheck) Name() string { return CheckRefusal }

func (refusalCheck) Evaluate(in *Input) Verdict {
	expected := in.Case.Expected.ExpectRefusal
	switch {
	case expected && !in.Refused:
		return Verdict{Applies: true, Detail: "A refusal was expected and the agent answered"}
	case !expected && in.Refused:
		return Verdict{Applies: true, Detail: "The agent refused a question it should answer"}
	default:
		return Verdict{Applies: true, Passed: true}
	}
}

func toolChoice(in *Input) softResult {
	expected := in.Case.Expected.Tools
	if len(expected) == 0 {
		return softResult{}
	}

	want := make([]string, 0, len(expected))
	for _, tool := range expected {
		want = append(want, tool.Name)
	}
	calls := in.agentCalls()
	got := make([]string, 0, len(calls))
	for _, call := range calls {
		got = append(got, call.ToolName)
	}

	switch in.Case.Expected.ToolMode {
	case agentquality.ToolMatchOrdered:
		matched := longestCommonSubsequence(want, got)
		return softResult{
			applies:  true,
			score:    float64(matched) / float64(len(want)),
			detail:   "Expected tools called in order",
			findings: missing(want, got),
		}
	case agentquality.ToolMatchSubset:
		if len(got) == 0 {
			return softResult{applies: true, score: 1, detail: "No tool outside the expected set"}
		}
		outside := make([]string, 0)
		inside := 0
		for _, name := range got {
			if slices.Contains(want, name) {
				inside++
				continue
			}
			outside = append(outside, name)
		}

		return softResult{
			applies:  true,
			score:    float64(inside) / float64(len(got)),
			detail:   "Calls kept within the expected tools",
			findings: outside,
		}
	default:
		absent := missing(want, got)
		return softResult{
			applies:  true,
			score:    float64(len(want)-len(absent)) / float64(len(want)),
			detail:   "Expected tools called in any order",
			findings: absent,
		}
	}
}

func arguments(in *Input) softResult {
	calls := in.Calls
	used := make([]bool, len(calls))
	checked, passed := 0, 0
	findings := make([]string, 0)

	for _, tool := range in.Case.Expected.Tools {
		keys := tool.Checked()
		if len(keys) == 0 {
			continue
		}
		checked += len(keys)

		best, bestPassed := -1, -1
		var bestFailed []string
		for i, call := range calls {
			if used[i] || call.ToolName != tool.Name {
				continue
			}
			ok, failed := 0, make([]string, 0)
			for _, key := range keys {
				value, present := call.Arguments[key]
				if Match(tool.RuleFor(key), tool.Args[key], value, present) {
					ok++
					continue
				}
				failed = append(failed, key)
			}
			if ok > bestPassed {
				best, bestPassed, bestFailed = i, ok, failed
			}
		}
		if best < 0 {
			findings = append(findings, tool.Name+": not called")
			continue
		}
		used[best] = true
		passed += bestPassed
		for _, key := range bestFailed {
			findings = append(findings, tool.Name+"."+key)
		}
	}

	if checked == 0 {
		return softResult{}
	}

	return softResult{
		applies:  true,
		score:    float64(passed) / float64(checked),
		detail:   "Arguments within their tolerance rules",
		findings: findings,
	}
}

func proposals(in *Input) softResult {
	expected := in.Case.Expected.Proposals
	if len(expected) == 0 {
		return softResult{}
	}

	originals := make([]agent.OriginalProposal, 0, len(expected))
	for _, proposal := range expected {
		original := agent.OriginalProposal{
			ToolName: proposal.ToolName,
			Params:   proposal.Params,
			Status:   agent.ProposalStatusAccepted,
			Decision: agent.DecisionAccepted,
		}
		if proposal.Rejected {
			original.Status = agent.ProposalStatusRejected
			original.Decision = agent.DecisionRejected
		}
		originals = append(originals, original)
	}

	comparison := agent.CompareReplay(originals, in.Actions)
	passed := 0
	findings := make([]string, 0)
	for i, proposal := range expected {
		match := comparison.Matches[i]
		if proposalPasses(proposal, match) {
			passed++
			continue
		}
		findings = append(findings, proposal.ToolName+": "+string(match.Verdict))
	}

	return softResult{
		applies:  true,
		score:    float64(passed) / float64(len(expected)),
		detail:   "Proposals against what people approved and rejected",
		findings: findings,
	}
}

func proposalPasses(proposal agentquality.ExpectedProposal, match agent.ReplayMatch) bool {
	switch match.Verdict {
	case agent.VerdictAgreed, agent.VerdictImproved:
		return true
	case agent.VerdictChanged:
		within, _ := argumentsMatch(proposal.Params, proposal.Rules, match.ReplayParams)
		return within != proposal.Rejected
	default:
		return false
	}
}

func factGuard(in *Input) softResult {
	if strings.TrimSpace(in.Reply) == "" {
		return softResult{}
	}

	texts := make([]string, 0, 4+len(in.ObservedText)+len(in.Actions))
	texts = append(texts,
		in.Case.FixtureText(),
		in.Case.Input,
		in.Case.HistoryText(),
		in.Case.PageText(),
	)
	texts = append(texts, in.ObservedText...)
	for _, action := range in.Actions {
		if encoded, err := sonic.MarshalString(action.Arguments); err == nil {
			texts = append(texts, encoded)
		}
	}

	check := numberguard.CheckNumbers(in.Reply, numberguard.SupportedFromText(texts...))
	if !check.OK {
		return softResult{
			applies:  true,
			detail:   "The reply cites figures nothing it was given supports",
			findings: check.Unsupported,
		}
	}

	return softResult{applies: true, score: 1, detail: "Every figure in the reply is supported"}
}

func mentions(in *Input) softResult {
	required := in.Case.Expected.MustMention
	forbidden := in.Case.Expected.MustNotMention
	total := len(required) + len(forbidden)
	if total == 0 {
		return softResult{}
	}

	reply := strings.ToLower(in.Reply)
	satisfied := 0
	findings := make([]string, 0)
	for _, phrase := range required {
		if strings.Contains(reply, strings.ToLower(phrase)) {
			satisfied++
			continue
		}
		findings = append(findings, "missing: "+phrase)
	}
	for _, phrase := range forbidden {
		if !strings.Contains(reply, strings.ToLower(phrase)) {
			satisfied++
			continue
		}
		findings = append(findings, "mentioned: "+phrase)
	}

	return softResult{
		applies:  true,
		score:    float64(satisfied) / float64(total),
		detail:   "Phrases the reply must and must not mention",
		findings: findings,
	}
}

func missing(want, got []string) []string {
	remaining := slices.Clone(got)
	absent := make([]string, 0)
	for _, name := range want {
		index := slices.Index(remaining, name)
		if index < 0 {
			absent = append(absent, name)
			continue
		}
		remaining = slices.Delete(remaining, index, index+1)
	}

	return absent
}

func longestCommonSubsequence(want, got []string) int {
	previous := make([]int, len(got)+1)
	current := make([]int, len(got)+1)
	for i := 1; i <= len(want); i++ {
		for j := 1; j <= len(got); j++ {
			switch {
			case want[i-1] == got[j-1]:
				current[j] = previous[j-1] + 1
			case previous[j] >= current[j-1]:
				current[j] = previous[j]
			default:
				current[j] = current[j-1]
			}
		}
		previous, current = current, previous
	}

	return previous[len(got)]
}
