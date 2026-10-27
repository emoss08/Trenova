package formulatemplateservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	ReadinessPass = "pass"
	ReadinessWarn = "warn"
	ReadinessFail = "fail"
)

const (
	ReadinessCheckDetails    = "details"
	ReadinessCheckSource     = "source"
	ReadinessCheckTests      = "tests"
	ReadinessCheckLibraries  = "libraries"
	ReadinessCheckTables     = "tables"
	ReadinessCheckDependents = "dependents"
	ReadinessCheckCoverage   = "coverage"
	ReadinessCheckStatus     = "status"
	ReadinessCheckReviewer   = "reviewer"
	ReadinessCheckAge        = "submissionAge"
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

	refusal string
}

func (c *ReadinessCheck) refusalMessage() string {
	if c.refusal != "" {
		return c.refusal
	}
	return c.Detail
}

type ReadinessResponse struct {
	CanSubmit        bool                    `json:"canSubmit"`
	CanApprove       bool                    `json:"canApprove"`
	Checks           []ReadinessCheck        `json:"checks"`
	TestTotal        int                     `json:"testTotal"`
	TestPassed       int                     `json:"testPassed"`
	TestFailing      []string                `json:"testFailing,omitempty"`
	Coverage         *tariff.CoverageSummary `json:"coverage"`
	Dependents       []DependentFailure      `json:"dependents"`
	PendingSchedules []ScheduledWindow       `json:"pendingSchedules"`
}

type submissionGate struct {
	details ReadinessCheck
	source  ReadinessCheck
	tests   ReadinessCheck
	content *contentCheck
}

func (g *submissionGate) checks() []ReadinessCheck {
	return []ReadinessCheck{g.details, g.source, g.tests}
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

	resp := &ReadinessResponse{
		Checks:     make([]ReadinessCheck, 0, 8),
		Dependents: []DependentFailure{},
	}

	gate, err := s.evaluateSubmissionGate(ctx, template)
	if err != nil {
		log.Error("failed to check the template's source", zap.Error(err))
		return nil, err
	}
	resp.TestTotal = len(gate.content.tests.Tests)
	resp.TestPassed = gate.content.passedTests()
	resp.TestFailing = gate.content.failingTests()
	resp.Coverage = tariff.SummarizeCoverage(template.Source, gate.content.tests.Coverage)
	locale := i18n.FromContext(ctx)

	libraries, err := s.libraryReadiness(ctx, template)
	if err != nil {
		log.Error("failed to compare imported library versions", zap.Error(err))
		return nil, err
	}

	dependents := ReadinessCheck{
		Key:   ReadinessCheckDependents,
		Label: i18n.Translate(locale, "Contracts still fit"),
	}
	if gate.content.checks() {
		result, gateErr := s.checkDependents(ctx, template, gate.content)
		if gateErr != nil {
			log.Error("failed to check the template's dependents", zap.Error(gateErr))
			return nil, gateErr
		}
		resp.Dependents = result.Failures
		dependents = dependentsReadiness(ctx, result)
	} else {
		dependents.Status = ReadinessWarn
		dependents.Detail = i18n.Translate(
			locale,
			"Fix the source first; contracts are checked against its inputs",
		)
	}

	statusCheck, reviewerCheck := reviewStateChecks(locale, template, req.TenantInfo.UserID)
	ageCheck := submissionAgeCheck(locale, template, timeutils.NowUnix())

	resp.Checks = append(resp.Checks,
		gate.details,
		gate.source,
		gate.tests,
		coverageReadiness(locale, gate.content, resp.Coverage),
		libraries,
		tableReadiness(locale, template, gate.content),
		dependents,
		statusCheck,
		reviewerCheck,
		ageCheck,
	)

	resp.PendingSchedules, err = s.pendingSchedules(ctx, template.ID, req.TenantInfo)
	if err != nil {
		log.Error("failed to list pending schedules", zap.Error(err))
		return nil, err
	}

	blocking := gate.blocking()
	resp.CanSubmit = !blocking && submittable(template.Status)
	resp.CanApprove = !blocking &&
		template.Status == formulatemplate.StatusInReview &&
		reviewerCheck.Status != ReadinessFail &&
		ageCheck.Status != ReadinessFail &&
		dependents.Status != ReadinessFail

	return resp, nil
}

func (s *Service) evaluateSubmissionGate(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
) (*submissionGate, error) {
	content, err := s.checkContent(ctx, &contentRequest{
		tenant: templateTenant(template),
		source: template.Source,
		mode:   tariff.LibrariesLatest,
	})
	if err != nil {
		return nil, err
	}

	return newSubmissionGate(ctx, template, content), nil
}

func newSubmissionGate(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	content *contentCheck,
) *submissionGate {
	locale := i18n.FromContext(ctx)
	return &submissionGate{
		details: detailsReadiness(locale, template),
		source:  sourceReadiness(locale, content),
		tests:   testsReadiness(locale, content),
		content: content,
	}
}

func submittable(status formulatemplate.Status) bool {
	return status == formulatemplate.StatusDraft || status == formulatemplate.StatusInactive
}

func reviewStateChecks(
	locale i18n.Locale,
	template *formulatemplate.FormulaTemplate,
	userID pulid.ID,
) (ReadinessCheck, ReadinessCheck) {
	status := ReadinessCheck{
		Key:   ReadinessCheckStatus,
		Label: i18n.Translate(locale, "Review state"),
	}
	switch template.Status {
	case formulatemplate.StatusDraft, formulatemplate.StatusInactive:
		status.Status = ReadinessPass
		status.Detail = i18n.Translate(locale, "Ready to submit for review")
	case formulatemplate.StatusInReview:
		status.Status = ReadinessPass
		status.Detail = i18n.Translate(locale, "In review; awaiting a decision")
	case formulatemplate.StatusActive:
		status.Status = ReadinessPass
		status.Detail = i18n.Translate(
			locale,
			"Active; editing the source will return it to draft",
		)
	default:
		status.Status = ReadinessWarn
		status.Detail = i18n.Translate(locale, "Unknown status {0}", template.Status.String())
	}

	reviewer := ReadinessCheck{
		Key:   ReadinessCheckReviewer,
		Label: i18n.Translate(locale, "Independent reviewer"),
	}
	switch {
	case template.Status != formulatemplate.StatusInReview:
		reviewer.Status = ReadinessPass
		reviewer.Detail = i18n.Translate(locale, "Applies once the template is in review")
	case template.SubmittedByID != nil && *template.SubmittedByID == userID:
		reviewer.Status = ReadinessFail
		reviewer.Detail = i18n.Translate(
			locale,
			"You submitted this template; someone else must approve it",
		)
	default:
		reviewer.Status = ReadinessPass
		reviewer.Detail = i18n.Translate(locale, "You did not submit this template")
	}

	return status, reviewer
}

func detailsReadiness(
	locale i18n.Locale,
	template *formulatemplate.FormulaTemplate,
) ReadinessCheck {
	check := ReadinessCheck{
		Key:   ReadinessCheckDetails,
		Label: i18n.Translate(locale, "Name and description"),
	}

	nameMissing := strings.TrimSpace(template.Name) == ""
	descriptionMissing := strings.TrimSpace(template.Description) == ""
	switch {
	case nameMissing && descriptionMissing:
		check.Status = ReadinessFail
		check.Detail = i18n.Translate(locale, "Name and description are missing")
	case nameMissing:
		check.Status = ReadinessFail
		check.Detail = i18n.Translate(locale, "Name is missing")
	case descriptionMissing:
		check.Status = ReadinessFail
		check.Detail = i18n.Translate(locale, "Description is missing")
	default:
		check.Status = ReadinessPass
		check.Detail = i18n.Translate(locale, "Named and described")
	}

	return check
}

func sourceReadiness(locale i18n.Locale, content *contentCheck) ReadinessCheck {
	check := ReadinessCheck{
		Key:   ReadinessCheckSource,
		Label: i18n.Translate(locale, "Source checks clean"),
	}
	if content.checks() {
		check.Status = ReadinessPass
		check.Detail = i18n.Translate(
			locale,
			"No errors against the schema, the rate tables and the charge codes",
		)
		return check
	}

	check.Status = ReadinessFail
	check.Detail = i18n.Translate(locale, "The source does not check")
	if first := content.checked.Check.FirstError(); first != nil {
		check.Detail = first.Message.Localize(locale)
	}
	check.refusal = i18n.Translate(
		locale,
		"Fix the source's errors before submitting: {0}",
		check.Detail,
	)
	return check
}

func testsReadiness(locale i18n.Locale, content *contentCheck) ReadinessCheck {
	check := ReadinessCheck{
		Key:   ReadinessCheckTests,
		Label: i18n.Translate(locale, "Tests pass"),
	}
	total := 0
	if content.tests != nil {
		total = len(content.tests.Tests)
	}
	passed := content.passedTests()

	switch {
	case !content.checks():
		check.Status = ReadinessFail
		check.Detail = i18n.Translate(locale, "The tests run once the source checks")
	case total == 0:
		check.Status = ReadinessFail
		check.Detail = i18n.Translate(locale, "Add at least one test")
		check.refusal = i18n.Translate(locale, "Add at least one test block before submitting")
	case content.tests.Error != nil:
		check.Status = ReadinessFail
		check.Detail = i18n.Translate(locale, "The tests could not run")
	case passed < total:
		check.Status = ReadinessFail
		check.Detail = i18n.Translate(locale, "{0} of {1} pass", passed, total)
		check.refusal = i18n.Translate(
			locale,
			"Only {0} of {1} tests pass ({2}); fix the formula or the tests before submitting",
			passed,
			total,
			strings.Join(tariff.FailingTests(locale, content.tests), "; "),
		)
	default:
		check.Status = ReadinessPass
		check.Detail = i18n.Translate(locale, "{0} of {1} pass", passed, total)
	}

	return check
}

func coverageReadiness(
	locale i18n.Locale,
	content *contentCheck,
	coverage *tariff.CoverageSummary,
) ReadinessCheck {
	verdict := tariff.CoverageCheck(locale, content.checks(), coverage)
	return ReadinessCheck{
		Key:    ReadinessCheckCoverage,
		Label:  verdict.Label,
		Status: verdict.Status,
		Detail: verdict.Detail,
	}
}

func (s *Service) libraryReadiness(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
) (ReadinessCheck, error) {
	locale := i18n.FromContext(ctx)
	check := ReadinessCheck{
		Key:   ReadinessCheckLibraries,
		Label: i18n.Translate(locale, "Library versions"),
	}
	if len(template.Imports) == 0 {
		check.Status = ReadinessPass
		check.Detail = i18n.Translate(locale, "Imports no organization library")
		return check, nil
	}

	latest, err := s.resolver.LatestLibraries(ctx, templateTenant(template), template.Source)
	if err != nil {
		return check, err
	}
	newest := make(map[string]tarifftypes.Import, len(latest))
	for _, version := range latest {
		newest[version.Path()] = version.Pin()
	}

	stale := make([]string, 0, len(template.Imports))
	for _, imp := range template.Imports {
		current, ok := newest[imp.Path]
		if ok && current.VersionNumber != imp.VersionNumber {
			stale = append(stale, i18n.Translate(
				locale,
				"{0} v{1} (v{2} is approved)",
				imp.Path,
				imp.VersionNumber,
				current.VersionNumber,
			))
		}
	}
	if len(stale) > 0 {
		check.Status = ReadinessWarn
		check.Detail = i18n.Translate(
			locale,
			"Approval pins the newest approved versions: {0}",
			strings.Join(stale, ", "),
		)
		return check, nil
	}

	check.Status = ReadinessPass
	check.Detail = i18n.Translate(
		locale,
		"Pins {0, plural, one {# library} other {# libraries}} at the newest approved version",
		len(template.Imports),
	)
	return check, nil
}

func tableReadiness(
	locale i18n.Locale,
	template *formulatemplate.FormulaTemplate,
	content *contentCheck,
) ReadinessCheck {
	check := ReadinessCheck{
		Key:    ReadinessCheckTables,
		Label:  i18n.Translate(locale, "Rate table versions"),
		Status: ReadinessPass,
	}
	latest := []tarifftypes.TablePin{}
	if content != nil && content.checked != nil {
		latest = content.checked.Tables
	}
	if len(latest) == 0 {
		check.Detail = i18n.Translate(locale, "Reads no rate table")
		return check
	}

	pinned := make(map[pulid.ID]int64, len(template.Tables))
	for _, pin := range template.Tables {
		pinned[pin.RateMatrixID] = pin.VersionNumber
	}
	newer := make([]string, 0, len(latest))
	for _, pin := range latest {
		current, ok := pinned[pin.RateMatrixID]
		if ok && current != pin.VersionNumber {
			newer = append(newer, i18n.Translate(
				locale,
				"{0} v{1} (v{2} is published)",
				pin.Code,
				current,
				pin.VersionNumber,
			))
		}
	}
	if len(newer) > 0 {
		check.Status = ReadinessWarn
		check.Detail = i18n.Translate(
			locale,
			"Approval pins the newest published versions: {0}",
			strings.Join(newer, ", "),
		)
		return check
	}

	check.Detail = i18n.Translate(
		locale,
		"Pins {0, plural, one {# rate table} other {# rate tables}} at the newest published version",
		len(latest),
	)
	return check
}

func dependentsReadiness(ctx context.Context, gate *dependentsGate) ReadinessCheck {
	locale := i18n.FromContext(ctx)
	check := ReadinessCheck{
		Key:   ReadinessCheckDependents,
		Label: i18n.Translate(locale, "Contracts still fit"),
	}
	switch {
	case gate.blocking():
		check.Status = ReadinessFail
		check.Detail = errortypes.LocalizedText(ctx, dependentsRefusal(locale, gate))
	case !gate.Checked:
		check.Status = ReadinessPass
		check.Detail = i18n.Translate(
			locale,
			"The inputs are declared as in the approved version, so every contract still fits",
		)
	case gate.Sets == 0:
		check.Status = ReadinessPass
		check.Detail = i18n.Translate(
			locale,
			"No contract or shipment sets this template's inputs yet",
		)
	default:
		check.Status = ReadinessPass
		check.Detail = i18n.Translate(
			locale,
			"Every contract and shipment fits ({0, plural, one {# distinct set of values} other {# distinct sets of values}} checked)",
			gate.Sets,
		)
	}
	return check
}

const agingSubmissionAfter int64 = 7 * 24 * 60 * 60

func submissionAgeCheck(
	locale i18n.Locale,
	template *formulatemplate.FormulaTemplate,
	now int64,
) ReadinessCheck {
	check := ReadinessCheck{
		Key:   ReadinessCheckAge,
		Label: i18n.Translate(locale, "Submission age"),
	}
	if template.Status != formulatemplate.StatusInReview || template.SubmittedAt == nil {
		check.Status = ReadinessPass
		check.Detail = i18n.Translate(locale, "Applies once the template is in review")
		return check
	}

	waited := now - *template.SubmittedAt
	days := waited / (24 * 60 * 60)
	switch {
	case formulatemplate.SubmissionIsStale(template.SubmittedAt, now):
		check.Status = ReadinessFail
		check.Detail = i18n.Translate(
			locale,
			"Submitted {0, plural, one {# day} other {# days}} ago; older than the 14-day limit, so it must be resubmitted",
			days,
		)
	case waited > agingSubmissionAfter:
		check.Status = ReadinessWarn
		check.Detail = i18n.Translate(
			locale,
			"Submitted {0, plural, one {# day} other {# days}} ago; decide soon or it expires at 14",
			days,
		)
	default:
		check.Status = ReadinessPass
		check.Detail = i18n.Translate(locale, "Submitted recently")
	}
	return check
}
