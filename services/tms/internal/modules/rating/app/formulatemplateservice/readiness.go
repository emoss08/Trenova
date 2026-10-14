package formulatemplateservice

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/rating/app/formula/contextvariablecache"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetablecache"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const (
	ReadinessPass = "pass"
	ReadinessWarn = "warn"
	ReadinessFail = "fail"
)

const (
	ReadinessCheckDetails    = "details"
	ReadinessCheckExpression = "expression"
	ReadinessCheckNullables  = "nullableFields"
	ReadinessCheckScenarios  = "scenarios"
	ReadinessCheckBreakdown  = "breakdown"
	ReadinessCheckStatus     = "status"
	ReadinessCheckReviewer   = "reviewer"
	ReadinessCheckAge        = "submissionAge"
)

var reconcileTolerance = decimal.New(5, -3)

type ReadinessRequest struct {
	TenantInfo pagination.TenantInfo
	TemplateID pulid.ID
}

type ReadinessCheck struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`

	refusal string
}

func (c *ReadinessCheck) refusalMessage() string {
	if c.refusal != "" {
		return c.refusal
	}

	return c.Detail
}

type ReadinessResponse struct {
	CanSubmit       bool             `json:"canSubmit"`
	CanApprove      bool             `json:"canApprove"`
	Checks          []ReadinessCheck `json:"checks"`
	ScenarioTotal   int              `json:"scenarioTotal"`
	ScenarioPassed  int              `json:"scenarioPassed"`
	ScenarioFailing []string         `json:"scenarioFailing,omitempty"`
}

type submissionGate struct {
	details    ReadinessCheck
	expression ReadinessCheck
	scenarios  ReadinessCheck
}

func (g *submissionGate) checks() []ReadinessCheck {
	return []ReadinessCheck{g.details, g.expression, g.scenarios}
}

func (g *submissionGate) blocking() bool {
	for _, check := range g.checks() {
		if check.Status == ReadinessFail {
			return true
		}
	}

	return false
}

func (g *submissionGate) addRefusals(multiErr *errortypes.MultiError) {
	for _, check := range g.checks() {
		if check.Status == ReadinessFail {
			multiErr.Add(check.Key, errortypes.ErrInvalid, "{0}", check.refusalMessage())
		}
	}
}

func (s *Service) Readiness(
	ctx context.Context,
	req *ReadinessRequest,
) (*ReadinessResponse, error) {
	log := s.l.With(
		zap.String("operation", "Readiness"),
		zap.String("templateID", req.TemplateID.String()),
	)

	template, err := s.getTemplateByIDWithTenant(ctx, req.TemplateID, req.TenantInfo)
	if err != nil {
		log.Error("failed to get formula template", zap.Error(err))
		return nil, err
	}

	ctx = contextvariablecache.With(ratetablecache.With(ctx))

	resp := &ReadinessResponse{Checks: make([]ReadinessCheck, 0, 8)}

	gate, err := s.evaluateSubmissionGate(ctx, template, req.TenantInfo, resp)
	if err != nil {
		log.Error("failed to evaluate scenarios for readiness", zap.Error(err))
		return nil, err
	}

	statusCheck, reviewerCheck := reviewStateChecks(template, req.TenantInfo.UserID)
	ageCheck := submissionAgeCheck(template, timeutils.NowUnix())

	resp.Checks = append(resp.Checks,
		gate.details,
		gate.expression,
		s.nullableReadiness(ctx, template),
		gate.scenarios,
	)
	if breakdownCheck, ok := s.breakdownReadiness(ctx, template, req.TenantInfo); ok {
		resp.Checks = append(resp.Checks, breakdownCheck)
	}
	resp.Checks = append(resp.Checks, statusCheck, reviewerCheck, ageCheck)

	blocking := gate.blocking()
	resp.CanSubmit = !blocking && submittable(template.Status)
	resp.CanApprove = !blocking &&
		template.Status == formulatemplate.StatusInReview &&
		reviewerCheck.Status != ReadinessFail &&
		ageCheck.Status != ReadinessFail

	return resp, nil
}

func (s *Service) evaluateSubmissionGate(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	tenantInfo pagination.TenantInfo,
	resp *ReadinessResponse,
) (*submissionGate, error) {
	scenarios, err := s.scenarioReadiness(ctx, template, tenantInfo, resp)
	if err != nil {
		return nil, err
	}

	return &submissionGate{
		details:    detailsReadiness(template),
		expression: s.expressionReadiness(ctx, template),
		scenarios:  scenarios,
	}, nil
}

func submittable(status formulatemplate.Status) bool {
	return status == formulatemplate.StatusDraft || status == formulatemplate.StatusInactive
}

func reviewStateChecks(
	template *formulatemplate.FormulaTemplate,
	userID pulid.ID,
) (ReadinessCheck, ReadinessCheck) {
	status := ReadinessCheck{Key: ReadinessCheckStatus, Label: "Review state"}
	switch template.Status {
	case formulatemplate.StatusDraft, formulatemplate.StatusInactive:
		status.Status = ReadinessPass
		status.Detail = "Ready to submit for review"
	case formulatemplate.StatusInReview:
		status.Status = ReadinessPass
		status.Detail = "In review; awaiting a decision"
	case formulatemplate.StatusActive:
		status.Status = ReadinessPass
		status.Detail = "Active; editing the content will return it to draft"
	default:
		status.Status = ReadinessWarn
		status.Detail = "Unknown status " + template.Status.String()
	}

	reviewer := ReadinessCheck{Key: ReadinessCheckReviewer, Label: "Independent reviewer"}
	switch {
	case template.Status != formulatemplate.StatusInReview:
		reviewer.Status = ReadinessPass
		reviewer.Detail = "Applies once the template is in review"
	case template.SubmittedByID != nil && *template.SubmittedByID == userID:
		reviewer.Status = ReadinessFail
		reviewer.Detail = "You submitted this template; someone else must approve it"
	default:
		reviewer.Status = ReadinessPass
		reviewer.Detail = "You did not submit this template"
	}

	return status, reviewer
}

func detailsReadiness(template *formulatemplate.FormulaTemplate) ReadinessCheck {
	check := ReadinessCheck{Key: ReadinessCheckDetails, Label: "Name and description"}

	nameMissing := strings.TrimSpace(template.Name) == ""
	descriptionMissing := strings.TrimSpace(template.Description) == ""
	switch {
	case nameMissing && descriptionMissing:
		check.Status = ReadinessFail
		check.Detail = "Name and description are missing"
	case nameMissing:
		check.Status = ReadinessFail
		check.Detail = "Name is missing"
	case descriptionMissing:
		check.Status = ReadinessFail
		check.Detail = "Description is missing"
	default:
		check.Status = ReadinessPass
		check.Detail = "Named and described"
	}

	return check
}

func (s *Service) expressionReadiness(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
) ReadinessCheck {
	check := ReadinessCheck{Key: ReadinessCheckExpression, Label: "Expression is valid"}

	candidate := *template
	candidate.NormalizeRounding()

	domainErrs := errortypes.NewMultiError()
	candidate.Validate(domainErrs)
	for _, fieldErr := range domainErrs.Errors {
		if fieldErr.Field == "name" || fieldErr.Field == "description" {
			continue
		}
		check.Status = ReadinessFail
		check.Detail = fieldErr.Error()
		return check
	}

	if err := s.validateExpression(ctx, &candidate); err != nil {
		check.Status = ReadinessFail
		check.Detail = expressionErrorMessage(err)
		return check
	}

	check.Status = ReadinessPass
	check.Detail = "Compiles, and every rate table it names exists"
	return check
}

func (s *Service) nullableReadiness(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
) ReadinessCheck {
	check := ReadinessCheck{Key: ReadinessCheckNullables, Label: "Safe on every shipment"}

	variables := template.ScenarioContent().VariableDefaults()

	fields := make([]string, 0, 2)
	expressions := []string{template.Expression}
	for _, def := range template.BreakdownDefinitions {
		if def != nil {
			expressions = append(expressions, def.Expression)
		}
	}
	for _, expression := range expressions {
		found, err := s.formulaService.UnguardedNullableFields(
			ctx,
			expression,
			template.SchemaID,
			variables,
		)
		if err != nil {
			continue
		}
		for _, item := range found {
			fields = appendUnique(fields, item.Field)
		}
	}

	details := make([]string, 0, 2)
	if len(fields) > 0 {
		details = append(details, fmt.Sprintf(
			"%s can be empty: %s",
			stringutils.CountNoun(len(fields), "value", "values"),
			strings.Join(fields, ", "),
		))
	}
	if template.Type == formulatemplate.TemplateTypeFreightCharge &&
		slices.ContainsFunc(expressions, func(expression string) bool {
			return s.formulaService.ReferencesField(expression, storedTotalField)
		}) {
		details = append(details, storedTotalField+
			" is the total stored before this rating, so the freight charge trails by one save")
	}
	if len(details) > 0 {
		check.Status = ReadinessWarn
		check.Detail = strings.Join(details, "; ")
		return check
	}

	check.Status = ReadinessPass
	check.Detail = "Every optional field is guarded"
	return check
}

func (s *Service) scenarioReadiness(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	tenantInfo pagination.TenantInfo,
	resp *ReadinessResponse,
) (ReadinessCheck, error) {
	check := ReadinessCheck{Key: ReadinessCheckScenarios, Label: "Scenarios pass"}

	cases, err := s.testCaseRepo.ListByTemplate(ctx, formulatemplateports.ListTestCasesRequest{
		TenantInfo: tenantInfo,
		TemplateID: template.ID,
	})
	if err != nil {
		return check, err
	}

	if len(cases) == 0 {
		check.Status = ReadinessFail
		check.Detail = "Add at least one"
		check.refusal = "Add at least one test scenario before submitting"
		return check, nil
	}

	run := s.runCasesAgainstCandidate(
		ctx,
		template.SchemaID,
		tenantInfo,
		cases,
		template.ScenarioContent(),
	)

	resp.ScenarioTotal = run.Total
	resp.ScenarioPassed = run.Passed
	check.Detail = fmt.Sprintf("%d of %d pass", run.Passed, run.Total)

	if run.Failed == 0 {
		check.Status = ReadinessPass
		return check, nil
	}

	for _, result := range run.Results {
		if !result.Passed {
			resp.ScenarioFailing = append(resp.ScenarioFailing, result.Name)
		}
	}
	check.Status = ReadinessFail
	check.refusal = fmt.Sprintf(
		"Only %d of %d test scenarios pass; fix the formula or the scenarios before submitting",
		run.Passed,
		run.Total,
	)
	return check, nil
}

func (s *Service) breakdownReadiness(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	tenantInfo pagination.TenantInfo,
) (ReadinessCheck, bool) {
	lines := make([]*formulatypes.BreakdownDefinition, 0, len(template.BreakdownDefinitions))
	for _, def := range template.BreakdownDefinitions {
		if def != nil {
			lines = append(lines, def)
		}
	}
	if len(lines) == 0 {
		return ReadinessCheck{}, false
	}

	check := ReadinessCheck{Key: ReadinessCheckBreakdown, Label: "Breakdown reconciles"}

	req := candidateRequest(template.SchemaID, tenantInfo, template.ScenarioContent(), nil)
	req.Breakdowns = lines
	result := s.TestExpression(ctx, req)
	if !result.Valid {
		check.Status = ReadinessWarn
		check.Detail = "The formula does not run on the sample load, so the lines cannot be checked"
		return check, true
	}

	failed := 0
	sum := decimal.Zero
	for _, item := range result.Breakdown {
		if item.Error != "" {
			failed++
			continue
		}
		sum = sum.Add(item.Amount)
	}
	if failed > 0 {
		check.Status = ReadinessWarn
		check.Detail = fmt.Sprintf(
			"%s fail on the sample load",
			stringutils.CountNoun(failed, "line", "lines"),
		)
		return check, true
	}

	raw := result.RawAmount()
	if sum.Sub(raw).Abs().GreaterThan(reconcileTolerance) {
		check.Status = ReadinessWarn
		check.Detail = fmt.Sprintf(
			"Lines don't add up to the formula: they come to %s and the formula to %s",
			money.FormatDollars(sum),
			money.FormatDollars(raw),
		)
		return check, true
	}

	if result.Guardrail != nil && result.Guardrail.Applied {
		check.Status = ReadinessWarn
		check.Detail = fmt.Sprintf(
			"The %s moved the total from %s, so the invoice lines won't match it",
			boundNoun(result.Guardrail),
			money.FormatDollars(raw),
		)
		return check, true
	}

	check.Status = ReadinessPass
	check.Detail = "Lines explain the total"
	return check, true
}

func (r *TestExpressionResponse) RawAmount() decimal.Decimal {
	switch {
	case r.Guardrail != nil:
		return r.Guardrail.RawAmount
	case r.Rounding != nil:
		return r.Rounding.UnroundedAmount
	}

	amount, _ := r.Result.(decimal.Decimal)
	return amount
}

func boundNoun(guardrail *formulatemplatetypes.GuardrailResult) string {
	if guardrail != nil && guardrail.Bound == "max" {
		return "maximum"
	}

	return "minimum"
}

func appendUnique(values []string, value string) []string {
	if slices.Contains(values, value) {
		return values
	}
	return append(values, value)
}

const agingSubmissionAfter int64 = 7 * 24 * 60 * 60

func submissionAgeCheck(template *formulatemplate.FormulaTemplate, now int64) ReadinessCheck {
	check := ReadinessCheck{Key: ReadinessCheckAge, Label: "Submission age"}
	if template.Status != formulatemplate.StatusInReview || template.SubmittedAt == nil {
		check.Status = ReadinessPass
		check.Detail = "Applies once the template is in review"
		return check
	}

	waited := now - *template.SubmittedAt
	days := waited / (24 * 60 * 60)
	switch {
	case formulatemplate.SubmissionIsStale(template.SubmittedAt, now):
		check.Status = ReadinessFail
		check.Detail = fmt.Sprintf(
			"Submitted %d days ago; older than the 14-day limit, so it must be resubmitted",
			days,
		)
	case waited > agingSubmissionAfter:
		check.Status = ReadinessWarn
		check.Detail = fmt.Sprintf("Submitted %d days ago; decide soon or it expires at 14", days)
	default:
		check.Status = ReadinessPass
		check.Detail = "Submitted recently"
	}
	return check
}
