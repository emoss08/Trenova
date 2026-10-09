package deskbench

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/shared/numberguard"
	"github.com/emoss08/trenova/shared/recordids"
	"github.com/emoss08/trenova/shared/stringutils"
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
		proposals = append(proposals, turn.Proposals...)
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

	calls := evidence.calls()
	reply := evidence.reply()

	for _, want := range expect.Calls {
		checks = append(checks, calledCheck(want, calls))
	}
	for _, unwanted := range expect.NoCalls {
		checks = append(checks, notCalledCheck(unwanted, calls))
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
			Name:   "at most " + strconv.Itoa(*expect.MaxFailedCalls) + " refused or failed tool calls",
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
		checks = append(checks, b.factCheck(ctx, session, &expect.Facts[idx], reply))
	}
	for idx := range expect.DB {
		checks = append(checks, b.dbCheck(ctx, session, &expect.DB[idx]))
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

func formulaCheck(spec, formula string) CheckResult {
	options := alternatives(spec)
	check := CheckResult{Name: "the studio's formula has " + strings.Join(options, " or "), Detail: formula}
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

	problems := make([]string, 0, 1)
	for _, turn := range evidence.turns {
		if turn.Status.Terminal() && turn.Failure == "" && turn.CaptureError == "" {
			continue
		}
		problems = append(problems, fmt.Sprintf("%s %s: %s",
			turn.TurnID, turn.Status, joinErrors(turn.Failure, turn.CaptureError)))
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
			if mismatch := valueMismatch(fmt.Sprintf("%s[%d]", path, idx), want[idx], got[idx]); mismatch != "" {
				return mismatch
			}
		}

		return ""
	case string:
		if got, ok := actual.(string); ok && strings.EqualFold(strings.TrimSpace(got), strings.TrimSpace(want)) {
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

func (b *Bench) factCheck(ctx context.Context, session *Session, fact *Fact, reply string) CheckResult {
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
		return numberguard.Cites(reply, number)
	}

	return strings.Contains(strings.ToLower(reply), strings.ToLower(strings.TrimSpace(value)))
}

func (b *Bench) FactValues(ctx context.Context, session *Session, statement string) ([]string, error) {
	ctx = session.Context(ctx)

	return dbtx.Read(ctx, b.DB, func(ctx context.Context) ([]string, error) {
		db := b.DB.DBForContext(ctx)
		if _, err := db.ExecContext(ctx, "SET LOCAL statement_timeout = '"+factTimeout+"'"); err != nil {
			return nil, fmt.Errorf("bound the fact query: %w", err)
		}

		rows, err := db.QueryContext(ctx, statement)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		return firstColumn(rows)
	})
}

func firstColumn(rows *sql.Rows) ([]string, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	if len(columns) == 0 {
		return nil, nil
	}

	values := make([]string, 0, maxFactRows)
	cells := make([]any, len(columns))
	targets := make([]any, len(columns))
	for idx := range cells {
		targets[idx] = &cells[idx]
	}

	for rows.Next() && len(values) < maxFactRows {
		if err = rows.Scan(targets...); err != nil {
			return nil, err
		}
		values = append(values, cellText(cells[0]))
	}

	return values, rows.Err()
}

func cellText(cell any) string {
	switch v := cell.(type) {
	case nil:
		return ""
	case []byte:
		return string(v)
	case time.Time:
		return v.Format(time.DateOnly)
	default:
		return fmt.Sprint(v)
	}
}

func firstLine(text string) string {
	return stringutils.Ellipsize(stringutils.CollapseWhitespace(text), lineLimit)
}
