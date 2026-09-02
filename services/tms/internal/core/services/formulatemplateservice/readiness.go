package formulatemplateservice

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/services/formula/contextvariablecache"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetablecache"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

// Readiness check statuses. A failing check blocks the step it belongs to; a
// warning is advice the author can act on but is not held to.
const (
	ReadinessPass = "pass"
	ReadinessWarn = "warn"
	ReadinessFail = "fail"
)

// Readiness check keys, stable so the Studio can map them to controls.
const (
	ReadinessCheckStatus      = "status"
	ReadinessCheckReviewer    = "reviewer"
	ReadinessCheckAge         = "submissionAge"
	ReadinessCheckExpression  = "expression"
	ReadinessCheckDescription = "description"
	ReadinessCheckNullables   = "nullableFields"
	ReadinessCheckScenarios   = "scenarios"
)

type ReadinessRequest struct {
	TenantInfo pagination.TenantInfo
	TemplateID pulid.ID
}

type ReadinessCheck struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// ReadinessResponse is the review gate, computed ahead of time. It answers
// what Submit and Approve would refuse and why, so the Studio can show the
// author the list before they press the button instead of after.
type ReadinessResponse struct {
	CanSubmit       bool             `json:"canSubmit"`
	CanApprove      bool             `json:"canApprove"`
	Checks          []ReadinessCheck `json:"checks"`
	ScenarioTotal   int              `json:"scenarioTotal"`
	ScenarioPassed  int              `json:"scenarioPassed"`
	ScenarioFailing []string         `json:"scenarioFailing,omitempty"`
}

// Readiness runs every check Submit and Approve enforce, without changing
// anything, and reports each one. The checks are the same code the
// transitions call, so the list can never disagree with the gate.
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

	ctx = ratetablecache.With(ctx)
	ctx = contextvariablecache.With(ctx)

	resp := &ReadinessResponse{Checks: make([]ReadinessCheck, 0, 6)}
	blocking := false

	statusCheck, reviewerCheck := reviewStateChecks(template, req.TenantInfo.UserID)
	ageCheck := submissionAgeCheck(template, timeutils.NowUnix())
	resp.Checks = append(resp.Checks, statusCheck, reviewerCheck, ageCheck)

	expressionCheck := s.expressionReadiness(ctx, template)
	resp.Checks = append(resp.Checks, expressionCheck)
	blocking = blocking || expressionCheck.Status == ReadinessFail

	resp.Checks = append(resp.Checks, descriptionReadiness(template))
	resp.Checks = append(resp.Checks, s.nullableReadiness(ctx, template))

	scenarioCheck, err := s.scenarioReadiness(ctx, template, req.TenantInfo, resp)
	if err != nil {
		log.Error("failed to evaluate scenarios for readiness", zap.Error(err))
		return nil, err
	}
	resp.Checks = append(resp.Checks, scenarioCheck)
	blocking = blocking || scenarioCheck.Status == ReadinessFail

	resp.CanSubmit = !blocking && submittable(template.Status)
	resp.CanApprove = !blocking &&
		template.Status == formulatemplate.StatusInReview &&
		reviewerCheck.Status != ReadinessFail &&
		ageCheck.Status != ReadinessFail

	return resp, nil
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

func (s *Service) expressionReadiness(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
) ReadinessCheck {
	check := ReadinessCheck{Key: ReadinessCheckExpression, Label: "Expression and rate tables"}

	candidate := *template
	if err := s.validateTemplate(ctx, &candidate); err != nil {
		check.Status = ReadinessFail
		check.Detail = expressionErrorMessage(err)
		return check
	}

	check.Status = ReadinessPass
	check.Detail = "Compiles, and every rate table it names exists"
	return check
}

func descriptionReadiness(template *formulatemplate.FormulaTemplate) ReadinessCheck {
	check := ReadinessCheck{Key: ReadinessCheckDescription, Label: "Description"}
	if strings.TrimSpace(template.Description) == "" {
		check.Status = ReadinessWarn
		check.Detail = "Say what this template prices and when it applies; reviewers read this first"
		return check
	}

	check.Status = ReadinessPass
	check.Detail = "Present"
	return check
}

func (s *Service) nullableReadiness(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
) ReadinessCheck {
	check := ReadinessCheck{Key: ReadinessCheckNullables, Label: "Optional shipment fields"}

	variables := make(map[string]any, len(template.VariableDefinitions))
	for _, def := range template.VariableDefinitions {
		if def != nil && def.DefaultValue != nil {
			variables[def.Name] = def.DefaultValue
		}
	}

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

	if len(fields) > 0 {
		check.Status = ReadinessWarn
		check.Detail = fmt.Sprintf(
			"%s can be empty on a shipment; guard with coalesce so those shipments still price",
			strings.Join(fields, ", "),
		)
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
	check := ReadinessCheck{Key: ReadinessCheckScenarios, Label: "Scenarios"}

	cases, err := s.testCaseRepo.ListByTemplate(ctx, repositories.ListTestCasesRequest{
		TenantInfo: tenantInfo,
		TemplateID: template.ID,
	})
	if err != nil {
		return check, err
	}

	if len(cases) == 0 {
		check.Status = ReadinessWarn
		check.Detail = "No scenarios pin this template's behaviour; pin a preview so a future edit cannot silently change the price"
		return check, nil
	}

	run := s.runCasesAgainstCandidate(ctx, template.SchemaID, tenantInfo, cases, &TestCaseCandidate{
		Expression:           template.Expression,
		VariableDefinitions:  template.VariableDefinitions,
		BreakdownDefinitions: template.BreakdownDefinitions,
		MinCharge:            template.MinCharge,
		MaxCharge:            template.MaxCharge,
		RoundingMode:         template.RoundingMode,
		RoundingPrecision:    template.RoundingPrecision,
	})

	resp.ScenarioTotal = run.Total
	resp.ScenarioPassed = run.Passed

	if run.Failed == 0 {
		check.Status = ReadinessPass
		check.Detail = fmt.Sprintf("%d of %d passing", run.Passed, run.Total)
		return check, nil
	}

	for _, result := range run.Results {
		if !result.Passed {
			resp.ScenarioFailing = append(resp.ScenarioFailing, result.Name)
		}
	}
	check.Status = ReadinessFail
	check.Detail = fmt.Sprintf(
		"%d of %d failing: %s",
		run.Failed,
		run.Total,
		strings.Join(resp.ScenarioFailing, ", "),
	)
	return check, nil
}

func appendUnique(values []string, value string) []string {
	if slices.Contains(values, value) {
		return values
	}
	return append(values, value)
}

const agingSubmissionAfter int64 = 7 * 24 * 60 * 60

// submissionAgeCheck warns a reviewer that a submission has been waiting and
// fails once it has waited past the expiry, when approval is refused and the
// sweep will return it to draft.
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
