package deskbench

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/numberguard"
	"github.com/emoss08/trenova/shared/recordids"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

const (
	maxFactRows      = 50
	factTimeout      = "10s"
	floatTolerance   = 1e-9
	alternativeSplit = "|"
	lineLimit        = 240
)

type CheckResult struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail,omitempty"`
}

type stepEvidence struct {
	turns     []*TurnRecord
	decisions []*DecisionRecord
	formula   string
	run       *runOutcome
}

func (e *stepEvidence) calls() []*ToolCallRecord {
	calls := make([]*ToolCallRecord, 0, 8)
	for _, turn := range e.turns {
		calls = append(calls, turn.Tools...)
	}

	return calls
}

func (e *stepEvidence) reply() string {
	replies := make([]string, 0, len(e.turns))
	for _, turn := range e.turns {
		if strings.TrimSpace(turn.Reply) != "" {
			replies = append(replies, turn.Reply)
		}
	}

	return strings.Join(replies, "\n\n")
}

func (e *stepEvidence) proposals() []*ProposalRecord {
	proposals := make([]*ProposalRecord, 0, 2)
	for _, turn := range e.turns {
		for _, proposal := range turn.Proposals {
			if proposal.Tier != agent.TierAutoExecute {
				proposals = append(proposals, proposal)
			}
		}
	}

	return proposals
}

func (e *stepEvidence) duration() time.Duration {
	var total time.Duration
	for _, turn := range e.turns {
		total += turn.Duration()
	}

	return total
}

func (b *Bench) evaluate(
	ctx context.Context,
	session *Session,
	expect *Expect,
	evidence *stepEvidence,
) []CheckResult {
	checks := make([]CheckResult, 0, 8)
	checks = append(checks, turnsCompleted(evidence))
	if evidence.run == nil {
		checks = append(checks, artifactLinkChecks(evidence.turns)...)
		checks = append(checks, groundingCheck(evidence.turns))
	}
	checks = append(checks, runChecks(expect, evidence.run)...)

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
			check.Name = "no delegate called " + strings.Join(alternatives(unwanted), " or ")
			checks = append(checks, check)
		}
	}
	if expect.MaxCalls != nil {
		checks = append(checks, CheckResult{
			Name:   "at most " + strconv.Itoa(*expect.MaxCalls) + " tool calls",
			Passed: len(calls) <= *expect.MaxCalls,
			Detail: strconv.Itoa(len(calls)) + " made",
		})
	}
	if expect.MaxFailedCalls != nil {
		failed := refusedCalls(calls)
		checks = append(checks, CheckResult{
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
		checks = append(checks, formulaCheck(want, evidence.formula))
	}
	if expect.NoRecordIDs {
		checks = append(checks, CheckResult{
			Name:   "reply shows no internal record ids",
			Passed: !recordids.Contains(reply),
		})
	}
	if expect.Refused != nil {
		checks = append(checks, refusedCheck(*expect.Refused, evidence.turns))
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
		checks = append(checks, CheckResult{
			Name:   "no proposals",
			Passed: len(proposals) == 0,
			Detail: strings.Join(names, ", "),
		})
	}
	if expect.Executed != nil {
		checks = append(checks, executedCheck(*expect.Executed, evidence.decisions))
	}
	for idx := range expect.Facts {
		fact := expect.Facts[idx]
		fact.SQL = evidence.run.substitute(fact.SQL)
		checks = append(checks, b.factCheck(ctx, session, &fact, reply))
	}
	for idx := range expect.DB {
		check := expect.DB[idx]
		check.SQL = evidence.run.substitute(check.SQL)
		checks = append(checks, b.dbCheck(ctx, session, &check))
	}
	if expect.MaxSeconds != nil {
		elapsed := evidence.duration().Seconds()
		checks = append(checks, CheckResult{
			Name:   fmt.Sprintf("answered within %.0fs", *expect.MaxSeconds),
			Passed: elapsed <= *expect.MaxSeconds,
			Detail: fmt.Sprintf("%.1fs", elapsed),
		})
	}

	return checks
}

func artifactLinkChecks(turns []*TurnRecord) []CheckResult {
	var stray, misnamed, corrected []string
	named := false
	for _, turn := range turns {
		if len(turn.Artifacts) == 0 && len(agentruntime.ArtifactRefIDs(turn.Reply)) == 0 {
			continue
		}
		named = true
		stray = append(stray, agentruntime.StrayArtifactLinks(turn.Reply)...)
		titles := make(map[string]string, len(turn.Artifacts))
		for idx := range turn.Artifacts {
			titles[turn.Artifacts[idx].ID.String()] = turn.Artifacts[idx].Title
		}
		titled := agentruntime.NameArtifactLinks(turn.Reply, func(id, words string) (string, bool) {
			if title, ok := titles[id]; ok {
				return title, true
			}

			return words, true
		})
		if titled != turn.Reply {
			misnamed = append(misnamed, firstLine(turn.Reply))
		}
		corrected = append(corrected, artifactCorrections(turn.Events)...)
	}
	if !named {
		return nil
	}

	return []CheckResult{
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
// a reply asked again, and a reply placed after that. A left-out artifact
// named by a closing sentence, with nothing asked again, is not one.
func artifactCorrections(events []Event) []string {
	if !slices.ContainsFunc(events, func(event Event) bool {
		data, ok := event.Data.(map[string]any)
		return ok && event.Name == agentports.AssistantEventReplyRegrounded &&
			data["action"] == string(agentports.RegroundRelinkArtifacts)
	}) {
		return nil
	}

	var actions []string
	for _, event := range events {
		if event.Name != agentports.AssistantEventReplyRegrounded {
			continue
		}
		data, ok := event.Data.(map[string]any)
		if !ok {
			continue
		}
		action, _ := data["action"].(string)
		if reground := agentports.RegroundAction(
			action,
		); reground == agentports.RegroundRelinkArtifacts ||
			reground == agentports.RegroundPlaceArtifacts {
			actions = append(actions, action)
		}
	}

	return actions
}

// groundingCheck fails a step whose reply the grounding guard had to rewrite or
// annotate: one that named what a filed change does not hold, or stated an
// amount nothing the conversation read. Either is the model drifting, and a
// guard that fires where it should not shows here too.
func groundingCheck(turns []*TurnRecord) CheckResult {
	var corrected []string
	for _, turn := range turns {
		for _, event := range turn.Events {
			if event.Name != agentports.AssistantEventReplyRegrounded {
				continue
			}
			data, ok := event.Data.(map[string]any)
			if !ok {
				continue
			}
			action, _ := data["action"].(string)
			reground := agentports.RegroundAction(action)
			if reground != agentports.RegroundRewrite && reground != agentports.RegroundNote {
				continue
			}
			corrected = append(corrected, action+": "+strings.Join(
				append(eventStrings(data["figures"]), eventStrings(data["fields"])...), ", "))
		}
	}

	return CheckResult{
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

func formulaCheck(spec, formula string) CheckResult {
	options := alternatives(spec)
	check := CheckResult{
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

func turnsCompleted(evidence *stepEvidence) CheckResult {
	if len(evidence.turns) == 0 {
		return CheckResult{Name: "a turn ran", Passed: false, Detail: "no turn started"}
	}

	stopped := evidence.run != nil && evidence.run.Stopped
	problems := make([]string, 0, 1)
	for _, turn := range evidence.turns {
		failure := turn.Failure
		if stopped {
			failure = ""
		}
		if turn.Status.Terminal() && failure == "" && turn.CaptureError == "" {
			continue
		}
		problems = append(problems, fmt.Sprintf("%s %s: %s",
			turn.TurnID, turn.Status, joinErrors(failure, turn.CaptureError)))
	}

	return CheckResult{
		Name:   "every turn finished",
		Passed: len(problems) == 0,
		Detail: strings.Join(problems, "; "),
	}
}

func alternatives(spec string) []string {
	parts := strings.Split(spec, alternativeSplit)
	out := make([]string, 0, len(parts))
	for idx := 0; idx < len(parts); idx++ {
		part := strings.TrimSpace(parts[idx])
		if strings.HasPrefix(part, "/") && !closesPattern(part) {
			for idx+1 < len(parts) && !closesPattern(part) {
				idx++
				part += alternativeSplit + parts[idx]
			}
			part = strings.TrimSpace(part)
		}
		if part != "" {
			out = append(out, part)
		}
	}

	return out
}

func closesPattern(part string) bool {
	return len(part) > 1 && strings.HasSuffix(part, "/")
}

func calledCheck(spec string, calls []*ToolCallRecord) CheckResult {
	names := alternatives(spec)
	check := CheckResult{Name: "called " + strings.Join(names, " or ")}

	for _, call := range calls {
		for _, name := range names {
			if call.Name != name {
				continue
			}
			if !call.Refused() {
				check.Passed = true
				return check
			}
			check.Detail = joinErrors(check.Detail, call.Name+" was "+verdictLabel(call))
		}
	}
	if check.Detail == "" {
		check.Detail = "called: " + calledNames(calls)
	}

	return check
}

func notCalledCheck(spec string, calls []*ToolCallRecord) CheckResult {
	names := alternatives(spec)
	check := CheckResult{Name: "did not call " + strings.Join(names, " or "), Passed: true}

	for _, call := range calls {
		for _, name := range names {
			if call.Name == name {
				check.Passed = false
				check.Detail = joinErrors(check.Detail, call.Name+" ("+verdictLabel(call)+")")
			}
		}
	}

	return check
}

func delegateCalls(calls []*ToolCallRecord) []*ToolCallRecord {
	delegated := make([]*ToolCallRecord, 0, len(calls))
	for _, call := range calls {
		if call.DelegateCallID != "" {
			delegated = append(delegated, call)
		}
	}

	return delegated
}

func verdictLabel(call *ToolCallRecord) string {
	if call.Verdict != "" {
		return call.Verdict
	}
	if call.Failed {
		return verdictFailed
	}
	if !call.Finished {
		return "unfinished"
	}

	return verdictRan
}

func calledNames(calls []*ToolCallRecord) string {
	if len(calls) == 0 {
		return "nothing"
	}

	names := make([]string, 0, len(calls))
	for _, call := range calls {
		names = append(names, call.Name+"("+verdictLabel(call)+")")
	}

	return strings.Join(names, ", ")
}

func refusedCalls(calls []*ToolCallRecord) []*ToolCallRecord {
	out := make([]*ToolCallRecord, 0, 2)
	for _, call := range calls {
		if call.Refused() {
			out = append(out, call)
		}
	}

	return out
}

func describeCalls(calls []*ToolCallRecord) string {
	parts := make([]string, 0, len(calls))
	for _, call := range calls {
		parts = append(parts, call.Name+" "+verdictLabel(call)+": "+firstLine(call.Result))
	}

	return strings.Join(parts, "; ")
}

func replyCheck(spec, reply string, include bool) CheckResult {
	options := alternatives(spec)
	verb := "reply mentions "
	if !include {
		verb = "reply does not mention "
	}
	check := CheckResult{Name: verb + strings.Join(options, " or ")}

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

func refusedCheck(want bool, turns []*TurnRecord) CheckResult {
	refused := false
	for _, turn := range turns {
		refused = refused || turn.Refused
	}

	name := "was answered"
	if want {
		name = "was refused"
	}

	return CheckResult{Name: name, Passed: refused == want}
}

func proposedCheck(want *ExpectedWrite, proposals []*ProposalRecord) CheckResult {
	tools := alternatives(want.Tool)
	check := CheckResult{Name: "proposed " + strings.Join(tools, " or ")}
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

func executedCheck(want bool, decisions []*DecisionRecord) CheckResult {
	name := "approved changes ran"
	if !want {
		name = "approved changes did not run"
	}
	if len(decisions) == 0 {
		return CheckResult{Name: name, Passed: !want, Detail: "nothing was decided"}
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
			detail = joinErrors(detail, decision.Error)
		}
		if decision.ExecutionError != "" {
			detail = joinErrors(detail, decision.ExecutionError)
		}
		problems = append(problems, label+": "+detail)
	}

	return CheckResult{Name: name, Passed: len(problems) == 0, Detail: strings.Join(problems, "; ")}
}

func (b *Bench) factCheck(
	ctx context.Context,
	session *Session,
	fact *Fact,
	reply string,
) CheckResult {
	name := "states " + fact.Note
	if fact.Note == "" {
		name = "states " + firstLine(fact.SQL)
	}
	check := CheckResult{Name: name}

	values, err := b.FactValues(ctx, session, fact.SQL)
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

func (b *Bench) dbCheck(ctx context.Context, session *Session, check *DBCheck) CheckResult {
	name := "the database has " + check.Note
	if check.Note == "" {
		name = "the database answers " + check.Equals + " to " + firstLine(check.SQL)
	}
	result := CheckResult{Name: name}

	values, err := b.FactValues(ctx, session, check.SQL)
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

func (b *Bench) FactValues(
	ctx context.Context,
	session *Session,
	statement string,
) ([]string, error) {
	if b.facts != nil {
		return b.facts(ctx, session, statement)
	}

	return b.queryFacts(ctx, session, statement)
}

func (b *Bench) queryFacts(
	ctx context.Context,
	session *Session,
	statement string,
) ([]string, error) {
	ctx = session.Context(ctx)

	return dbtx.Read(ctx, b.DB, func(ctx context.Context) ([]string, error) {
		db := b.DB.Keel(ctx)
		if _, err := keel.Exec(ctx, db, boundStatementTimeoutSQL); err != nil {
			return nil, fmt.Errorf("bound the fact query: %w", err)
		}

		values := make([]string, 0, maxFactRows)
		err := keel.QueryRows(ctx, db, func(row pgx.CollectableRow) error {
			if len(values) >= maxFactRows || len(row.FieldDescriptions()) == 0 {
				return nil
			}
			cells, err := row.Values()
			if err != nil {
				return err
			}
			values = append(values, cellText(cells[0]))
			return nil
		}, statement)
		if err != nil {
			return nil, err
		}

		return values, nil
	})
}

const boundStatementTimeoutSQL = "SET LOCAL statement_timeout = '" + factTimeout + "'"

// benchLeftovers cancels the shipments earlier bench conversations created and
// left New, before a run starts. A load a scenario booked and never ran made
// "this one" and "the acme load" ambiguous for every run after it. Only a load a
// bench thread's create or duplicate made is touched: an edit's result names
// the seed load it changed, and reading every executed proposal canceled
// SEED-SHP-001 after a re-rate. Loads people entered are never touched.
//
// It also retires what earlier bench conversations taught the agents, and
// dismisses what they suggested. A rejected hold taught "this person does not
// want a hold for missing paperwork", and the case that asks for that hold
// failed on every run after. Memories from people's own conversations are
// never touched.
func benchLeftovers() []string {
	return []string{
		`CREATE TEMP TABLE bench_leftovers ON COMMIT DROP AS
		SELECT s.id FROM shipments s
		WHERE s.status = 'New' AND s.pro_number NOT LIKE 'SEED-%' AND s.id IN (
			SELECT p.execution_result->'record'->>'id'
			FROM agent_proposals p
			JOIN agent_runs r ON r.id = p.run_id
			JOIN assistant_threads t ON t.id = r.subject_id
			WHERE t.title LIKE '[bench]%'
				AND p.status = 'Executed'
				AND p.tool_name IN ('shipment_create', 'shipment_duplicate')
				AND p.execution_result->'record'->>'entityType' = 'shipment')`,
		`UPDATE assignments a SET status = 'Canceled' FROM shipment_moves sm
		WHERE a.shipment_move_id = sm.id AND sm.shipment_id IN (SELECT id FROM bench_leftovers)`,
		`UPDATE shipment_moves SET status = 'Canceled'
		WHERE shipment_id IN (SELECT id FROM bench_leftovers)`,
		`UPDATE shipments SET status = 'Canceled' WHERE id IN (SELECT id FROM bench_leftovers)`,
		`CREATE TEMP TABLE bench_lessons ON COMMIT DROP AS
		SELECT m.id FROM agent_memories m
		WHERE m.status IN ('Active', 'Paused', 'Suggested') AND (
			m.reflection_id IN (
				SELECT rf.id FROM agent_reflections rf
				LEFT JOIN agent_runs r ON r.id = rf.run_id
				JOIN assistant_threads t ON t.id = COALESCE(rf.thread_id, r.subject_id)
				WHERE t.title LIKE '[bench]%')
			OR m.source_run_id IN (
				SELECT r.id FROM agent_runs r
				JOIN assistant_threads t ON t.id = r.subject_id
				WHERE t.title LIKE '[bench]%'))`,
		`UPDATE agent_memories SET
			status = CASE WHEN status = 'Suggested' THEN 'Dismissed' ELSE 'Retired' END,
			retired_at = CASE WHEN status = 'Suggested' THEN retired_at
				ELSE ` + nowEpoch + ` END,
			version = version + 1,
			updated_at = ` + nowEpoch + `
		WHERE id IN (SELECT id FROM bench_lessons)`,
	}
}

const nowEpoch = "extract(epoch from current_timestamp)::bigint"

func (b *Bench) runSetup(ctx context.Context, session *Session, statements []string) error {
	if len(statements) == 0 {
		return nil
	}

	return b.runStatements(ctx, session, statements)
}

func (b *Bench) runStatements(ctx context.Context, session *Session, statements []string) error {
	ctx = session.Context(ctx)

	return dbtx.WriteErr(ctx, b.DB, func(ctx context.Context) error {
		db := b.DB.Keel(ctx)
		if _, err := keel.Exec(ctx, db, boundStatementTimeoutSQL); err != nil {
			return fmt.Errorf("bound the setup: %w", err)
		}
		confined, err := readConfinement(ctx, db)
		if err != nil {
			return err
		}
		if err = confined.err(); err != nil {
			return err
		}
		for idx, statement := range statements {
			if _, err := keel.Exec(ctx, db, statement); err != nil {
				return fmt.Errorf("setup %d: %w", idx+1, err)
			}
		}

		return nil
	})
}

func cellText(cell any) string {
	switch v := cell.(type) {
	case nil:
		return ""
	case string:
		return v
	case []byte:
		return string(v)
	case time.Time:
		return v.Format(time.DateOnly)
	case driver.Valuer:
		value, err := v.Value()
		if err != nil {
			return fmt.Sprint(v)
		}
		return cellText(value)
	case map[string]any, []any:
		text, err := sonic.MarshalString(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return text
	default:
		return fmt.Sprint(v)
	}
}

func firstLine(text string) string {
	return stringutils.Ellipsize(stringutils.CollapseWhitespace(text), lineLimit)
}

var ErrSetupUnconfined = errors.New("setup refused: the bench's database role is not confined to " +
	"the bench user's organization")

type confinement struct {
	Bypass   bool   `db:"bypass"`
	Security string `db:"security"`
}

const confinementSQL = `SELECT r.rolsuper OR r.rolbypassrls AS bypass,
	current_setting('row_security') AS security
	FROM pg_roles r WHERE r.rolname = current_user`

func readConfinement(ctx context.Context, db *keel.DB) (confinement, error) {
	rows, err := keel.QueryAll(ctx, db, pgx.RowToStructByName[confinement], confinementSQL)
	if err != nil {
		return confinement{}, fmt.Errorf("read the role's row-level security: %w", err)
	}
	if len(rows) == 0 {
		return confinement{}, errors.New("read the role's row-level security: the role has no pg_roles row")
	}

	return rows[0], nil
}

func (c confinement) err() error {
	switch {
	case c.Bypass:
		return fmt.Errorf("%w: it is a superuser or bypasses row-level security",
			ErrSetupUnconfined)
	case c.Security != "on":
		return fmt.Errorf("%w: row_security is %q", ErrSetupUnconfined, c.Security)
	default:
		return nil
	}
}
