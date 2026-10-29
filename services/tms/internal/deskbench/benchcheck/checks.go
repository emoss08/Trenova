package benchcheck

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/deskbench"
	"github.com/emoss08/trenova/internal/deskbench/benchrecord"
	"github.com/emoss08/trenova/internal/deskbench/benchresult"
	"github.com/emoss08/trenova/internal/deskbench/benchscenario"
	"github.com/emoss08/trenova/internal/deskbench/benchsql"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/replycheck"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/numberguard"
	"github.com/emoss08/trenova/shared/recordids"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

const (
	floatTolerance = 1e-9
)

type Checker struct {
	facts benchsql.FactSource
}

func New(facts benchsql.FactSource) *Checker {
	return &Checker{facts: facts}
}

type Evidence struct {
	Turns     []*benchrecord.TurnRecord
	Decisions []*benchrecord.DecisionRecord
	Formula   string
	Run       *RunOutcome
}

func (e *Evidence) calls() []*benchrecord.ToolCallRecord {
	calls := make([]*benchrecord.ToolCallRecord, 0, 8)
	for _, turn := range e.Turns {
		calls = append(calls, turn.Tools...)
	}

	return calls
}

func (e *Evidence) reply() string {
	replies := make([]string, 0, len(e.Turns))
	for _, turn := range e.Turns {
		if strings.TrimSpace(turn.Reply) != "" {
			replies = append(replies, turn.Reply)
		}
	}

	return strings.Join(replies, "\n\n")
}

func (e *Evidence) proposals() []*benchrecord.ProposalRecord {
	proposals := make([]*benchrecord.ProposalRecord, 0, 2)
	for _, turn := range e.Turns {
		for _, proposal := range turn.Proposals {
			if proposal.Tier != agent.TierAutoExecute {
				proposals = append(proposals, proposal)
			}
		}
	}

	return proposals
}

func (e *Evidence) duration() time.Duration {
	var total time.Duration
	for _, turn := range e.Turns {
		total += turn.Duration()
	}

	return total
}

func (c *Checker) Evaluate(
	ctx context.Context,
	session *deskbench.Session,
	expect *benchscenario.Expect,
	evidence *Evidence,
) []benchresult.CheckResult {
	checks := make([]benchresult.CheckResult, 0, 8)
	checks = append(checks, turnsCompleted(evidence))
	if evidence.Run == nil {
		checks = append(checks, artifactLinkChecks(evidence.Turns)...)
		checks = append(checks, groundingCheck(evidence.Turns))
	}
	checks = append(checks, RunChecks(expect, evidence.Run)...)

	calls := evidence.calls()
	reply := evidence.reply()

	for _, want := range expect.Calls {
		checks = append(checks, calledCheck(want, calls))
	}
	for _, unwanted := range expect.NoCalls {
		checks = append(checks, notCalledCheck(unwanted, calls))
	}
	if len(expect.NoDelegateCalls) > 0 {
		delegated := delegateCalls(calls)
		for _, unwanted := range expect.NoDelegateCalls {
			check := notCalledCheck(unwanted, delegated)
			check.Name = "no delegate called " + strings.Join(
				benchscenario.Alternatives(unwanted),
				" or ",
			)
			checks = append(checks, check)
		}
	}
	if expect.MaxCalls != nil {
		checks = append(checks, benchresult.CheckResult{
			Name:   "at most " + strconv.Itoa(*expect.MaxCalls) + " tool calls",
			Passed: len(calls) <= *expect.MaxCalls,
			Detail: strconv.Itoa(len(calls)) + " made",
		})
	}
	if expect.MaxFailedCalls != nil {
		failed := refusedCalls(calls)
		checks = append(checks, benchresult.CheckResult{
			Name: "at most " + strconv.Itoa(
				*expect.MaxFailedCalls,
			) + " refused or failed tool calls",
			Passed: len(failed) <= *expect.MaxFailedCalls,
			Detail: describeCalls(failed),
		})
	}
	for _, want := range expect.ReplyIncludes {
		checks = append(checks, replyCheck(want, reply, true))
	}
	for _, unwanted := range expect.ReplyExcludes {
		checks = append(checks, replyCheck(unwanted, reply, false))
	}
	for _, want := range expect.Formula {
		checks = append(checks, formulaCheck(want, evidence.Formula))
	}
	if expect.NoRecordIDs {
		checks = append(checks, benchresult.CheckResult{
			Name:   "reply shows no internal record ids",
			Passed: !recordids.Contains(reply),
		})
	}
	if expect.Refused != nil {
		checks = append(checks, refusedCheck(*expect.Refused, evidence.Turns))
	}
	for idx := range expect.Proposals {
		checks = append(checks, proposedCheck(&expect.Proposals[idx], evidence.proposals()))
	}
	if expect.NoProposals {
		proposals := evidence.proposals()
		names := make([]string, 0, len(proposals))
		for _, proposal := range proposals {
			names = append(names, proposal.Tool)
		}
		checks = append(checks, benchresult.CheckResult{
			Name:   "no proposals",
			Passed: len(proposals) == 0,
			Detail: strings.Join(names, ", "),
		})
	}
	if expect.Executed != nil {
		checks = append(checks, executedCheck(*expect.Executed, evidence.Decisions))
	}
	for idx := range expect.Facts {
		fact := expect.Facts[idx]
		fact.SQL = evidence.Run.substitute(fact.SQL)
		checks = append(checks, c.factCheck(ctx, session, &fact, reply))
	}
	for idx := range expect.DB {
		check := expect.DB[idx]
		check.SQL = evidence.Run.substitute(check.SQL)
		checks = append(checks, c.dbCheck(ctx, session, &check))
	}
	if expect.MaxSeconds != nil {
		elapsed := evidence.duration().Seconds()
		checks = append(checks, benchresult.CheckResult{
			Name:   fmt.Sprintf("answered within %.0fs", *expect.MaxSeconds),
			Passed: elapsed <= *expect.MaxSeconds,
			Detail: fmt.Sprintf("%.1fs", elapsed),
		})
	}

	return checks
}

func artifactLinkChecks(turns []*benchrecord.TurnRecord) []benchresult.CheckResult {
	var stray, misnamed, corrected []string
	named := false
	for _, turn := range turns {
		if len(turn.Artifacts) == 0 && len(replycheck.ArtifactRefIDs(turn.Reply)) == 0 {
			continue
		}
		named = true
		stray = append(stray, replycheck.StrayArtifactLinks(turn.Reply)...)
		titles := make(map[string]string, len(turn.Artifacts))
		for idx := range turn.Artifacts {
			titles[turn.Artifacts[idx].ID.String()] = turn.Artifacts[idx].Title
		}
		titled := replycheck.NameArtifactLinks(turn.Reply, func(id, words string) (string, bool) {
			if title, ok := titles[id]; ok {
				return title, true
			}

			return words, true
		})
		if titled != turn.Reply {
			misnamed = append(misnamed, benchrecord.FirstLine(turn.Reply))
		}
		corrected = append(corrected, artifactCorrections(turn.Events)...)
	}
	if !named {
		return nil
	}

	return []benchresult.CheckResult{
		{
			Name:   "artifacts are named inside sentences",
			Passed: len(stray) == 0,
			Detail: strings.Join(stray, ", "),
		},
		{
			Name:   "artifact links read as their artifacts' titles",
			Passed: len(misnamed) == 0,
			Detail: strings.Join(misnamed, " | "),
		},
		{
			Name:   "artifacts named without a correction",
			Passed: len(corrected) == 0,
			Detail: strings.Join(corrected, ", "),
		},
	}
}

// artifactCorrections is the link corrections that cost the person a restart:
// a reply asked again, and a reply placed after that. A link the runtime mends
// without asking again (a left-out artifact, or one standing apart from any
// sentence moved into the reply's own) is a place_artifacts alone, and is not
// one.
func artifactCorrections(events []benchrecord.Event) []string {
	if !slices.ContainsFunc(events, func(event benchrecord.Event) bool {
		data, ok := event.Data.(map[string]any)
		return ok && event.Name == assistantstreamports.AssistantEventReplyRegrounded &&
			data["action"] == string(assistantstreamports.RegroundRelinkArtifacts)
	}) {
		return nil
	}

	var actions []string
	for _, event := range events {
		if event.Name != assistantstreamports.AssistantEventReplyRegrounded {
			continue
		}
		data, ok := event.Data.(map[string]any)
		if !ok {
			continue
		}
		action, _ := data["action"].(string)
		if reground := assistantstreamports.RegroundAction(
			action,
		); reground == assistantstreamports.RegroundRelinkArtifacts ||
			reground == assistantstreamports.RegroundPlaceArtifacts {
			actions = append(actions, action)
		}
	}

	return actions
}

// groundingCheck fails a step whose reply the grounding guard had to rewrite or
// annotate: one that named what a filed change does not hold, or stated an
// amount nothing the conversation read. Either is the model drifting, and a
// guard that fires where it should not shows here too.
func groundingCheck(turns []*benchrecord.TurnRecord) benchresult.CheckResult {
	var corrected []string
	for _, turn := range turns {
		for _, event := range turn.Events {
			if event.Name != assistantstreamports.AssistantEventReplyRegrounded {
				continue
			}
			data, ok := event.Data.(map[string]any)
			if !ok {
				continue
			}
			action, _ := data["action"].(string)
			reground := assistantstreamports.RegroundAction(action)
			if reground != assistantstreamports.RegroundRewrite &&
				reground != assistantstreamports.RegroundNote {
				continue
			}
			corrected = append(corrected, action+": "+strings.Join(
				append(eventStrings(data["figures"]), eventStrings(data["fields"])...), ", "))
		}
	}

	return benchresult.CheckResult{
		Name:   "reply grounded without a correction",
		Passed: len(corrected) == 0,
		Detail: strings.Join(corrected, " | "),
	}
}

func eventStrings(value any) []string {
	items, _ := value.([]any)
	out := make([]string, 0, len(items))
	for _, item := range items {
		if text, ok := item.(string); ok {
			out = append(out, text)
		}
	}

	return out
}

func formulaCheck(spec, formula string) benchresult.CheckResult {
	options := benchscenario.Alternatives(spec)
	check := benchresult.CheckResult{
		Name:   "the studio's formula has " + strings.Join(options, " or "),
		Detail: formula,
	}
	for _, option := range options {
		ok, err := replyMatches(option, formula)
		if err != nil {
			check.Detail = err.Error()
			return check
		}
		if ok {
			check.Passed = true
			return check
		}
	}
	if formula == "" {
		check.Detail = "the studio's draft is empty"
	}

	return check
}

func turnsCompleted(evidence *Evidence) benchresult.CheckResult {
	if len(evidence.Turns) == 0 {
		return benchresult.CheckResult{Name: "a turn ran", Passed: false, Detail: "no turn started"}
	}

	stopped := evidence.Run != nil && evidence.Run.Stopped
	problems := make([]string, 0, 1)
	for _, turn := range evidence.Turns {
		failure := turn.Failure
		if stopped {
			failure = ""
		}
		if turn.Status.Terminal() && failure == "" && turn.CaptureError == "" {
			continue
		}
		problems = append(problems, fmt.Sprintf("%s %s: %s",
			turn.TurnID, turn.Status, benchrecord.JoinErrors(failure, turn.CaptureError)))
	}

	return benchresult.CheckResult{
		Name:   "every turn finished",
		Passed: len(problems) == 0,
		Detail: strings.Join(problems, "; "),
	}
}

func calledCheck(spec string, calls []*benchrecord.ToolCallRecord) benchresult.CheckResult {
	names := benchscenario.Alternatives(spec)
	check := benchresult.CheckResult{Name: "called " + strings.Join(names, " or ")}

	for _, call := range calls {
		for _, name := range names {
			if call.Name != name {
				continue
			}
			if !call.Refused() {
				check.Passed = true
				return check
			}
			check.Detail = benchrecord.JoinErrors(
				check.Detail,
				call.Name+" was "+benchrecord.VerdictLabel(call),
			)
		}
	}
	if check.Detail == "" {
		check.Detail = "called: " + calledNames(calls)
	}

	return check
}

func notCalledCheck(spec string, calls []*benchrecord.ToolCallRecord) benchresult.CheckResult {
	names := benchscenario.Alternatives(spec)
	check := benchresult.CheckResult{
		Name:   "did not call " + strings.Join(names, " or "),
		Passed: true,
	}

	for _, call := range calls {
		for _, name := range names {
			if call.Name == name {
				check.Passed = false
				check.Detail = benchrecord.JoinErrors(
					check.Detail,
					call.Name+" ("+benchrecord.VerdictLabel(call)+")",
				)
			}
		}
	}

	return check
}

func delegateCalls(calls []*benchrecord.ToolCallRecord) []*benchrecord.ToolCallRecord {
	delegated := make([]*benchrecord.ToolCallRecord, 0, len(calls))
	for _, call := range calls {
		if call.DelegateCallID != "" {
			delegated = append(delegated, call)
		}
	}

	return delegated
}

func calledNames(calls []*benchrecord.ToolCallRecord) string {
	if len(calls) == 0 {
		return "nothing"
	}

	names := make([]string, 0, len(calls))
	for _, call := range calls {
		names = append(names, call.Name+"("+benchrecord.VerdictLabel(call)+")")
	}

	return strings.Join(names, ", ")
}

func refusedCalls(calls []*benchrecord.ToolCallRecord) []*benchrecord.ToolCallRecord {
	out := make([]*benchrecord.ToolCallRecord, 0, 2)
	for _, call := range calls {
		if call.Refused() {
			out = append(out, call)
		}
	}

	return out
}

func describeCalls(calls []*benchrecord.ToolCallRecord) string {
	parts := make([]string, 0, len(calls))
	for _, call := range calls {
		parts = append(
			parts,
			call.Name+" "+benchrecord.VerdictLabel(call)+": "+benchrecord.FirstLine(call.Result),
		)
	}

	return strings.Join(parts, "; ")
}

func replyCheck(spec, reply string, include bool) benchresult.CheckResult {
	options := benchscenario.Alternatives(spec)
	verb := "reply mentions "
	if !include {
		verb = "reply does not mention "
	}
	check := benchresult.CheckResult{Name: verb + strings.Join(options, " or ")}

	matched := ""
	for _, option := range options {
		ok, err := replyMatches(option, reply)
		if err != nil {
			check.Detail = err.Error()
			return check
		}
		if ok {
			matched = option
			break
		}
	}

	check.Passed = (matched != "") == include
	if !include && matched != "" {
		check.Detail = "said " + matched
	}

	return check
}

func replyMatches(option, reply string) (bool, error) {
	if len(option) > 1 && strings.HasPrefix(option, "/") && strings.HasSuffix(option, "/") {
		pattern, err := regexp.Compile("(?i)" + option[1:len(option)-1])
		if err != nil {
			return false, fmt.Errorf("bad pattern %s: %w", option, err)
		}

		return pattern.MatchString(reply), nil
	}

	return strings.Contains(strings.ToLower(reply), strings.ToLower(option)), nil
}

func refusedCheck(want bool, turns []*benchrecord.TurnRecord) benchresult.CheckResult {
	refused := false
	for _, turn := range turns {
		refused = refused || turn.Refused
	}

	name := "was answered"
	if want {
		name = "was refused"
	}

	return benchresult.CheckResult{Name: name, Passed: refused == want}
}

func proposedCheck(
	want *benchscenario.ExpectedWrite,
	proposals []*benchrecord.ProposalRecord,
) benchresult.CheckResult {
	tools := benchscenario.Alternatives(want.Tool)
	check := benchresult.CheckResult{Name: "proposed " + strings.Join(tools, " or ")}
	tried := make([]string, 0, len(proposals))
	for _, proposal := range proposals {
		if !slices.Contains(tools, proposal.Tool) {
			tried = append(tried, proposal.Tool)
			continue
		}
		mismatch := argsMismatch("", want.Args, proposal.Arguments)
		if mismatch == "" {
			check.Passed = true
			return check
		}
		tried = append(tried, proposal.Tool+" with "+mismatch)
	}

	if len(tried) == 0 {
		check.Detail = "nothing was proposed"
	} else {
		check.Detail = "proposed: " + strings.Join(tried, "; ")
	}

	return check
}

func argsMismatch(path string, want, got map[string]any) string {
	for key, expected := range want {
		at := key
		if path != "" {
			at = path + "." + key
		}
		actual, ok := got[key]
		if !ok {
			return at + " missing"
		}
		if mismatch := valueMismatch(at, expected, actual); mismatch != "" {
			return mismatch
		}
	}

	return ""
}

func valueMismatch(path string, expected, actual any) string {
	switch want := expected.(type) {
	case map[string]any:
		got, ok := actual.(map[string]any)
		if !ok {
			return fmt.Sprintf("%s is %v, not an object", path, actual)
		}

		return argsMismatch(path, want, got)
	case []any:
		got, ok := actual.([]any)
		if !ok || len(got) != len(want) {
			return fmt.Sprintf("%s is %v, want %v", path, actual, want)
		}
		for idx := range want {
			if mismatch := valueMismatch(
				fmt.Sprintf("%s[%d]", path, idx),
				want[idx],
				got[idx],
			); mismatch != "" {
				return mismatch
			}
		}

		return ""
	case string:
		if got, ok := actual.(string); ok &&
			strings.EqualFold(strings.TrimSpace(got), strings.TrimSpace(want)) {
			return ""
		}
	case bool:
		if got, ok := actual.(bool); ok && got == want {
			return ""
		}
	default:
		if numbersEqual(expected, actual) {
			return ""
		}
	}

	return fmt.Sprintf("%s is %v, want %v", path, actual, expected)
}

func numbersEqual(a, b any) bool {
	left, okLeft := asFloat(a)
	right, okRight := asFloat(b)

	return okLeft && okRight && math.Abs(left-right) <= floatTolerance
}

func asFloat(value any) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case float64:
		return v, true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return parsed, err == nil
	}

	return 0, false
}

func executedCheck(want bool, decisions []*benchrecord.DecisionRecord) benchresult.CheckResult {
	name := "approved changes ran"
	if !want {
		name = "approved changes did not run"
	}
	if len(decisions) == 0 {
		return benchresult.CheckResult{Name: name, Passed: !want, Detail: "nothing was decided"}
	}

	problems := make([]string, 0, 1)
	for _, decision := range decisions {
		label := decision.Tool
		ran := decision.Executed && decision.Error == ""
		if ran == want {
			continue
		}
		detail := string(decision.StatusAfter)
		if decision.Error != "" {
			detail = benchrecord.JoinErrors(detail, decision.Error)
		}
		if decision.ExecutionError != "" {
			detail = benchrecord.JoinErrors(detail, decision.ExecutionError)
		}
		problems = append(problems, label+": "+detail)
	}

	return benchresult.CheckResult{
		Name:   name,
		Passed: len(problems) == 0,
		Detail: strings.Join(problems, "; "),
	}
}

func (c *Checker) factCheck(
	ctx context.Context,
	session *deskbench.Session,
	fact *benchscenario.Fact,
	reply string,
) benchresult.CheckResult {
	name := "states " + fact.Note
	if fact.Note == "" {
		name = "states " + benchrecord.FirstLine(fact.SQL)
	}
	check := benchresult.CheckResult{Name: name}

	values, err := c.facts(ctx, session, fact.SQL)
	if err != nil {
		check.Detail = "query failed: " + err.Error()
		return check
	}
	if len(values) == 0 {
		check.Detail = "the query returned no rows"
		return check
	}

	missing := make([]string, 0, len(values))
	found := 0
	for _, value := range values {
		if replyStates(reply, value) {
			found++
			continue
		}
		missing = append(missing, value)
	}

	if fact.Any {
		check.Passed = found > 0
	} else {
		check.Passed = len(missing) == 0
	}
	check.Detail = fmt.Sprintf("truth %s; missing from reply: %s",
		strings.Join(values, ", "), strings.Join(missing, ", "))

	return check
}

func (c *Checker) dbCheck(
	ctx context.Context,
	session *deskbench.Session,
	check *benchscenario.DBCheck,
) benchresult.CheckResult {
	name := "the database has " + check.Note
	if check.Note == "" {
		name = "the database answers " + check.Equals + " to " + benchrecord.FirstLine(check.SQL)
	}
	result := benchresult.CheckResult{Name: name}

	values, err := c.facts(ctx, session, check.SQL)
	switch {
	case err != nil:
		result.Detail = "query failed: " + err.Error()
	case len(values) == 0:
		result.Detail = "the query returned no rows"
	default:
		result.Passed = sameValue(values[0], check.Equals)
		result.Detail = "got " + values[0] + ", want " + check.Equals
	}

	return result
}

func sameValue(got, want string) bool {
	if strings.EqualFold(strings.TrimSpace(got), strings.TrimSpace(want)) {
		return true
	}

	return numbersEqual(got, want)
}

func replyStates(reply, value string) bool {
	if number, err := decimal.NewFromString(strings.ReplaceAll(value, ",", "")); err == nil {
		return numberguard.Cites(reply, number) || statesInWords(reply, number)
	}

	if strings.Contains(strings.ToLower(reply), strings.ToLower(strings.TrimSpace(value))) {
		return true
	}
	folded := stringutils.NormalizeIdentifier(value)

	return folded != "" && strings.Contains(stringutils.NormalizeIdentifier(reply), folded)
}

var (
	zeroInWords = regexp.MustCompile(`(?i)\b(zero|none|nothing)\b`)
	noBefore    = regexp.MustCompile(`(?i)\bno\s+([a-z][a-z-]*)`)
	notCounted  = map[string]struct{}{
		"big": {}, "doubt": {}, "idea": {}, "later": {}, "less": {}, "longer": {},
		"matter": {}, "more": {}, "need": {}, "one": {}, "problem": {}, "problems": {},
		"rush": {}, "sooner": {}, "surprise": {}, "thanks": {}, "way": {}, "wonder": {},
		"worries": {}, "worry": {},
	}
)

func statesZero(reply string) bool {
	if zeroInWords.MatchString(reply) {
		return true
	}
	for _, match := range noBefore.FindAllStringSubmatch(reply, -1) {
		if _, idiom := notCounted[strings.ToLower(match[1])]; !idiom {
			return true
		}
	}

	return false
}

func statesInWords(reply string, number decimal.Decimal) bool {
	if !number.IsInteger() || number.IsNegative() {
		return false
	}
	count := int(number.IntPart())
	if count == 0 {
		return statesZero(reply)
	}
	word := intutils.SmallNumberWord(count)
	if word == strconv.Itoa(count) {
		return false
	}

	return regexp.MustCompile(`(?i)\b` + word + `\b`).MatchString(reply)
}
