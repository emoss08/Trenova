package formulatemplateservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/rating/app/formula/contextvariablecache"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetablecache"
	"github.com/emoss08/trenova/shared/decimalutils"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const (
	recordedDraftLabel = "the recorded draft"
	savedVersionLabel  = "the saved version"
	boundMinimum       = "minimum"
	boundMaximum       = "maximum"
)

var traceNearTolerance = decimal.New(5, -3)

type PriceScenarioRequest struct {
	TenantInfo pagination.TenantInfo
	SchemaID   string
	Content    *formulatemplate.ScenarioContent
	Inputs     map[string]any
}

type ScenarioPrice struct {
	Variables map[string]any
	Amount    string
	Valid     bool
	Error     string
}

func (s *Service) PriceScenario(ctx context.Context, req *PriceScenarioRequest) *ScenarioPrice {
	evaluation := s.TestExpression(
		contextvariablecache.With(ratetablecache.With(ctx)),
		candidateRequest(req.SchemaID, req.TenantInfo, req.Content, req.Inputs),
	)

	price := &ScenarioPrice{Variables: scenarioVariables(req.Content, req.Inputs)}
	if !evaluation.Valid {
		price.Error = evaluationError(evaluation)
		return price
	}

	amount, ok := evaluation.Result.(decimal.Decimal)
	if !ok {
		price.Error = "The expression evaluated to something other than an amount"
		return price
	}

	price.Valid = true
	price.Amount = amount.String()
	return price
}

type ScenarioTraceRequest struct {
	TenantInfo pagination.TenantInfo
	TemplateID pulid.ID
	TestCaseID pulid.ID
	Candidate  *formulatemplate.ScenarioContent
}

type scenarioRun struct {
	content    *formulatemplate.ScenarioContent
	evaluation *TestExpressionResponse
}

func (r *scenarioRun) valid() bool {
	return r.evaluation != nil && r.evaluation.Valid
}

func (r *scenarioRun) raw() *string {
	if !r.valid() {
		return nil
	}

	return decimalText(r.evaluation.RawAmount())
}

func (r *scenarioRun) billed() *string {
	if !r.valid() {
		return nil
	}

	amount, ok := r.evaluation.Result.(decimal.Decimal)
	if !ok {
		return nil
	}

	return decimalText(amount)
}

func (r *scenarioRun) clamp() (string, *decimal.Decimal) {
	if !r.valid() || r.evaluation.Guardrail == nil || !r.evaluation.Guardrail.Applied {
		return "", nil
	}

	guardrail := r.evaluation.Guardrail
	if guardrail.Bound == "max" {
		return boundMaximum, guardrail.MaxCharge
	}

	return boundMinimum, guardrail.MinCharge
}

func (r *scenarioRun) terms() ([]string, map[string]*string) {
	order := make([]string, 0, 4)
	values := make(map[string]*string, 4)
	if r.evaluation == nil {
		return order, values
	}

	for _, statement := range r.evaluation.Statements {
		if statement.Kind != formulatemplatetypes.StatementLet || statement.Name == "" {
			continue
		}
		if _, seen := values[statement.Name]; !seen {
			order = append(order, statement.Name)
		}
		if statement.Error != "" {
			values[statement.Name] = nil
			continue
		}
		values[statement.Name] = traceText(statement.Value)
	}

	return order, values
}

func (s *Service) ScenarioTrace(
	ctx context.Context,
	req *ScenarioTraceRequest,
) (*formulatemplateports.ScenarioTrace, error) {
	log := s.l.With(
		zap.String("operation", "ScenarioTrace"),
		zap.String("testCaseID", req.TestCaseID.String()),
	)

	if req.Candidate == nil {
		return nil, errortypes.NewValidationError(
			"candidate",
			errortypes.ErrRequired,
			"The editor's content is required to explain a scenario",
		)
	}
	multiErr := errortypes.NewMultiError()
	ValidateScenarioContent(multiErr, "candidate", req.Candidate)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	template, err := s.getTemplateByIDWithTenant(ctx, req.TemplateID, req.TenantInfo)
	if err != nil {
		log.Error("failed to get template", zap.Error(err))
		return nil, err
	}

	testCase, err := s.testCaseRepo.GetByID(ctx, formulatemplateports.GetTestCaseByIDRequest{
		TenantInfo: req.TenantInfo,
		TestCaseID: req.TestCaseID,
		TemplateID: req.TemplateID,
	})
	if err != nil {
		log.Error("failed to get test case", zap.Error(err))
		return nil, err
	}

	recorded, err := s.recordedBaseline(ctx, template, testCase, req.TenantInfo)
	if err != nil {
		log.Error("failed to resolve the recorded content", zap.Error(err))
		return nil, err
	}

	ctx = contextvariablecache.With(ratetablecache.With(ctx))
	before := s.runTracedScenario(ctx, template.SchemaID, req.TenantInfo, recorded, testCase)
	after := s.runTracedScenario(ctx, template.SchemaID, req.TenantInfo, req.Candidate, testCase)

	trace := buildScenarioTrace(testCase, before, after)
	trace.BaselineLabel = baselineLabel(testCase.RecordedOnVersion)
	trace.RecordedOnVersion = testCase.RecordedOnVersion
	trace.RecordedBy = recorderOf(testCase)
	trace.FallbackVerdict, trace.FallbackWhy = fallbackScenarioCause(trace)

	return trace, nil
}

func (s *Service) recordedBaseline(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	testCase *formulatemplate.TestCase,
	tenantInfo pagination.TenantInfo,
) (*formulatemplate.ScenarioContent, error) {
	if testCase.RecordedContent != nil {
		return testCase.RecordedContent, nil
	}

	if testCase.RecordedOnVersion != nil && *testCase.RecordedOnVersion > 0 {
		version, err := s.lookupVersion(ctx, template.ID, *testCase.RecordedOnVersion, tenantInfo)
		if err != nil {
			return nil, err
		}
		if version != nil {
			return version.ScenarioContent(), nil
		}
	}

	return template.ScenarioContent(), nil
}

func (s *Service) runTracedScenario(
	ctx context.Context,
	schemaID string,
	tenantInfo pagination.TenantInfo,
	content *formulatemplate.ScenarioContent,
	testCase *formulatemplate.TestCase,
) *scenarioRun {
	req := candidateRequest(schemaID, tenantInfo, content, testCase.Variables)
	req.TraceStatements = true

	return &scenarioRun{content: content, evaluation: s.TestExpression(ctx, req)}
}

func buildScenarioTrace(
	testCase *formulatemplate.TestCase,
	before, after *scenarioRun,
) *formulatemplateports.ScenarioTrace {
	trace := &formulatemplateports.ScenarioTrace{
		Expected:       testCase.ExpectedAmount.String(),
		Actual:         after.billed(),
		ScenarioName:   testCase.Name,
		ScenarioInputs: testCase.Variables,
		ExpressionChanged: strings.TrimSpace(before.content.Expression) !=
			strings.TrimSpace(after.content.Expression),
		PolicyChanged:       !samePolicy(before.content, after.content),
		RecordedRaw:         before.raw(),
		RecordedBilled:      before.billed(),
		CandidateRaw:        after.raw(),
		CandidateExpression: after.content.Expression,
	}
	trace.RecordedClamp, _ = before.clamp()
	trace.CandidateClamp, _ = after.clamp()

	if trace.Actual != nil {
		actual := decimal.RequireFromString(*trace.Actual)
		trace.Delta = decimalText(actual.Sub(testCase.ExpectedAmount))
	}
	if !after.valid() {
		trace.Error = evaluationError(after.evaluation)
	}

	rows := inputRows(testCase.Variables, before.content, after.content)
	rows = append(rows, termRows(before, after)...)
	rows = appendIfDiffers(rows, formulatemplateports.ScenarioRowFormula, "formula",
		before.raw(), after.raw())
	rows = append(rows, policyRows(before, after)...)
	rows = appendIfDiffers(rows, formulatemplateports.ScenarioRowBilled, "billed",
		before.billed(), after.billed())
	trace.Rows = rows

	return trace
}

func inputRows(
	inputs map[string]any,
	before, after *formulatemplate.ScenarioContent,
) []formulatemplateports.ScenarioTraceRow {
	beforeDefaults := definedDefaults(before)
	rows := make([]formulatemplateports.ScenarioTraceRow, 0, 2)
	afterNames := make(map[string]struct{}, len(after.VariableDefinitions))

	for _, definition := range after.VariableDefinitions {
		if definition == nil || definition.Name == "" {
			continue
		}
		afterNames[definition.Name] = struct{}{}
		if _, set := inputs[definition.Name]; set {
			continue
		}
		var was *string
		if value, declared := beforeDefaults[definition.Name]; declared {
			was = value
		}
		rows = appendIfDiffers(rows, formulatemplateports.ScenarioRowInput, definition.Name,
			was, traceText(definition.DefaultValue))
	}

	for _, definition := range before.VariableDefinitions {
		if definition == nil || definition.Name == "" {
			continue
		}
		if _, set := inputs[definition.Name]; set {
			continue
		}
		if _, kept := afterNames[definition.Name]; kept {
			continue
		}
		rows = appendIfDiffers(rows, formulatemplateports.ScenarioRowInput, definition.Name,
			traceText(definition.DefaultValue), nil)
	}

	return rows
}

func definedDefaults(content *formulatemplate.ScenarioContent) map[string]*string {
	defaults := make(map[string]*string, len(content.VariableDefinitions))
	for _, definition := range content.VariableDefinitions {
		if definition == nil || definition.Name == "" {
			continue
		}
		defaults[definition.Name] = traceText(definition.DefaultValue)
	}

	return defaults
}

func termRows(before, after *scenarioRun) []formulatemplateports.ScenarioTraceRow {
	beforeOrder, beforeValues := before.terms()
	afterOrder, afterValues := after.terms()

	names := make([]string, 0, len(beforeOrder)+len(afterOrder))
	seen := make(map[string]struct{}, cap(names))
	for _, name := range append(afterOrder, beforeOrder...) {
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}

	rows := make([]formulatemplateports.ScenarioTraceRow, 0, len(names))
	for _, name := range names {
		rows = appendIfDiffers(rows, formulatemplateports.ScenarioRowTerm, name,
			beforeValues[name], afterValues[name])
	}

	return rows
}

func policyRows(before, after *scenarioRun) []formulatemplateports.ScenarioTraceRow {
	beforeBound, beforeLimit := before.clamp()
	afterBound, afterLimit := after.clamp()

	rows := make([]formulatemplateports.ScenarioTraceRow, 0, 2)
	for _, bound := range []string{boundMinimum, boundMaximum} {
		var was, now *string
		if beforeBound == bound && beforeLimit != nil {
			was = decimalText(*beforeLimit)
		}
		if afterBound == bound && afterLimit != nil {
			now = decimalText(*afterLimit)
		}
		rows = appendIfDiffers(rows, formulatemplateports.ScenarioRowPolicy, bound, was, now)
	}

	return rows
}

func appendIfDiffers(
	rows []formulatemplateports.ScenarioTraceRow,
	kind formulatemplateports.ScenarioTraceRowKind,
	name string,
	before, after *string,
) []formulatemplateports.ScenarioTraceRow {
	if sameTraceValue(before, after) {
		return rows
	}

	return append(rows, formulatemplateports.ScenarioTraceRow{
		Kind:   kind,
		Name:   name,
		Before: before,
		After:  after,
	})
}

func sameTraceValue(before, after *string) bool {
	if before == nil || after == nil {
		return before == nil && after == nil
	}
	if *before == *after {
		return true
	}

	left, leftErr := decimal.NewFromString(*before)
	right, rightErr := decimal.NewFromString(*after)
	if leftErr != nil || rightErr != nil {
		return false
	}

	return left.Sub(right).Abs().LessThan(traceNearTolerance)
}

func samePolicy(before, after *formulatemplate.ScenarioContent) bool {
	left := before.ChargePolicy().Normalized()
	right := after.ChargePolicy().Normalized()

	return decimalutils.NullEqual(left.MinCharge, right.MinCharge) &&
		decimalutils.NullEqual(left.MaxCharge, right.MaxCharge) &&
		left.RoundingMode == right.RoundingMode &&
		left.RoundingPrecision == right.RoundingPrecision
}

func traceText(value any) *string {
	switch typed := value.(type) {
	case nil:
		return nil
	case string:
		return &typed
	case bool:
		text := fmt.Sprintf("%t", typed)
		return &text
	case decimal.Decimal:
		return decimalText(typed)
	case float64:
		return decimalText(decimal.NewFromFloat(typed))
	case float32:
		return decimalText(decimal.NewFromFloat32(typed))
	case int:
		return decimalText(decimal.NewFromInt(int64(typed)))
	case int32:
		return decimalText(decimal.NewFromInt32(typed))
	case int64:
		return decimalText(decimal.NewFromInt(typed))
	default:
		text := fmt.Sprint(typed)
		return &text
	}
}

func decimalText(value decimal.Decimal) *string {
	text := value.String()
	return &text
}

func evaluationError(evaluation *TestExpressionResponse) string {
	if evaluation == nil {
		return "The expression could not be evaluated"
	}
	if message := strings.TrimSpace(evaluation.Error); message != "" {
		return message
	}

	return strings.TrimSpace(evaluation.Message)
}

func baselineLabel(recordedOnVersion *int64) string {
	switch {
	case recordedOnVersion == nil:
		return savedVersionLabel
	case *recordedOnVersion == 0:
		return recordedDraftLabel
	default:
		return fmt.Sprintf("v%d", *recordedOnVersion)
	}
}

func recorderOf(testCase *formulatemplate.TestCase) *formulatemplateports.ScenarioRecorder {
	if testCase.RecordedBy == nil || testCase.RecordedBy.ID.IsNil() {
		return nil
	}

	return &formulatemplateports.ScenarioRecorder{
		ID:   testCase.RecordedBy.ID,
		Name: testCase.RecordedBy.Name,
	}
}

func fallbackScenarioCause(
	trace *formulatemplateports.ScenarioTrace,
) (formulatemplateports.ScenarioVerdict, string) {
	if trace.Error != "" {
		return formulatemplateports.ScenarioVerdictBug,
			"The formula fails on this load now: " + trace.Error
	}

	input := firstRow(trace.Rows, formulatemplateports.ScenarioRowInput)
	term := largestTermMove(trace.Rows)

	if input != nil && !trace.ExpressionChanged {
		why := fmt.Sprintf(
			"`%s`'s default changed from %s to %s after this scenario was recorded on %s",
			input.Name,
			plainValue(input.Before),
			plainValue(input.After),
			trace.BaselineLabel,
		)
		if term != nil {
			why += fmt.Sprintf(", which moves `%s` from %s to %s",
				term.Name, plainValue(term.Before), plainValue(term.After))
		}
		why += ". The formula itself didn't change, so this looks like a rate update rather than a bug."
		return formulatemplateports.ScenarioVerdictIntended, why
	}

	if trace.RecordedClamp != "" && trace.CandidateClamp == "" {
		return formulatemplateports.ScenarioVerdictUnclear, fmt.Sprintf(
			"This load used to land on the %s; it no longer does, so it bills the formula's %s.",
			trace.RecordedClamp,
			moneyValue(trace.CandidateRaw),
		)
	}

	if trace.CandidateClamp != "" && trace.RecordedClamp == "" {
		limit := policyLimit(trace.Rows, trace.CandidateClamp)
		return formulatemplateports.ScenarioVerdictUnclear, fmt.Sprintf(
			"This load now hits the %s of %s, so the formula's %s is replaced.",
			trace.CandidateClamp,
			moneyValue(limit),
			moneyValue(trace.CandidateRaw),
		)
	}

	if term != nil {
		why := fmt.Sprintf("`%s` went from %s to %s on this load",
			term.Name, plainValue(term.Before), plainValue(term.After))
		if trace.ExpressionChanged {
			return formulatemplateports.ScenarioVerdictUnclear,
				why + ", from an edit to the expression."
		}
		return formulatemplateports.ScenarioVerdictIntended, why + "."
	}

	return formulatemplateports.ScenarioVerdictIntended, fmt.Sprintf(
		"Nothing in the template changed for this load since %s; the expected %s doesn't "+
			"match what that version billed either, so the expectation itself is likely off.",
		trace.BaselineLabel,
		moneyValue(&trace.Expected),
	)
}

func firstRow(
	rows []formulatemplateports.ScenarioTraceRow,
	kind formulatemplateports.ScenarioTraceRowKind,
) *formulatemplateports.ScenarioTraceRow {
	for index := range rows {
		if rows[index].Kind == kind {
			return &rows[index]
		}
	}

	return nil
}

func largestTermMove(
	rows []formulatemplateports.ScenarioTraceRow,
) *formulatemplateports.ScenarioTraceRow {
	var best *formulatemplateports.ScenarioTraceRow
	bestMove := decimal.Zero
	for index := range rows {
		row := &rows[index]
		if row.Kind != formulatemplateports.ScenarioRowTerm ||
			row.Before == nil || row.After == nil {
			continue
		}
		before, beforeErr := decimal.NewFromString(*row.Before)
		after, afterErr := decimal.NewFromString(*row.After)
		if beforeErr != nil || afterErr != nil {
			continue
		}
		move := after.Sub(before).Abs()
		if best == nil || move.GreaterThan(bestMove) {
			best = row
			bestMove = move
		}
	}

	return best
}

func policyLimit(rows []formulatemplateports.ScenarioTraceRow, bound string) *string {
	for _, row := range rows {
		if row.Kind == formulatemplateports.ScenarioRowPolicy && row.Name == bound {
			return row.After
		}
	}

	return nil
}

func plainValue(value *string) string {
	if value == nil {
		return "empty"
	}

	return *value
}

func moneyValue(value *string) string {
	if value == nil {
		return "nothing"
	}

	amount, err := decimal.NewFromString(*value)
	if err != nil {
		return *value
	}

	return money.FormatDollars(amount)
}
