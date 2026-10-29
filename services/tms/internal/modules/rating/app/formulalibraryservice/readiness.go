package formulalibraryservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariffverdict"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulalibraryports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	ReadinessPass = "pass"
	ReadinessWarn = "warn"
	ReadinessFail = "fail"

	ReadinessCheckDetails  = "details"
	ReadinessCheckSource   = "source"
	ReadinessCheckTests    = "tests"
	ReadinessCheckCoverage = "coverage"
	ReadinessCheckStatus   = "status"
	ReadinessCheckReviewer = "reviewer"
	ReadinessCheckAge      = "submissionAge"
)

type ReadinessCheck struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`

	refusal string
}

type ReadinessResponse struct {
	CanSubmit   bool                           `json:"canSubmit"`
	CanApprove  bool                           `json:"canApprove"`
	Checks      []ReadinessCheck               `json:"checks"`
	TestTotal   int                            `json:"testTotal"`
	TestPassed  int                            `json:"testPassed"`
	TestFailing []string                       `json:"testFailing,omitempty"`
	Coverage    *tariffverdict.CoverageSummary `json:"coverage"`
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
		if check.Status != ReadinessFail {
			continue
		}
		message := check.refusal
		if message == "" {
			message = check.Detail
		}
		multiErr.Add(check.Key, errortypes.ErrInvalid, "{0}", message)
	}
}

func (s *Service) gate(
	ctx context.Context,
	library *formulalibrary.FormulaLibrary,
) (*submissionGate, error) {
	content, err := s.checkContent(ctx, library)
	if err != nil {
		return nil, err
	}
	locale := i18n.FromContext(ctx)
	return &submissionGate{
		details: detailsCheck(locale, library),
		source:  sourceCheck(locale, content),
		tests:   testsCheck(locale, content),
		content: content,
	}, nil
}

type ReadinessRequest struct {
	TenantInfo pagination.TenantInfo
	LibraryID  pulid.ID
}

func (s *Service) Readiness(
	ctx context.Context,
	req *ReadinessRequest,
) (*ReadinessResponse, error) {
	library, err := s.repo.GetByID(ctx, &formulalibraryports.GetLibraryByIDRequest{
		TenantInfo: req.TenantInfo,
		LibraryID:  req.LibraryID,
	})
	if err != nil {
		return nil, err
	}

	gate, err := s.gate(ctx, library)
	if err != nil {
		return nil, err
	}

	locale := i18n.FromContext(ctx)
	status, reviewer := reviewChecks(locale, library, req.TenantInfo.UserID)
	age := ageCheck(locale, library, timeutils.NowUnix())
	coverage := tariffverdict.SummarizeCoverage(library.Source, gate.content.tests.Coverage)
	blocking := gate.blocking()

	return &ReadinessResponse{
		CanSubmit: !blocking &&
			(library.Status == formulatemplate.StatusDraft ||
				library.Status == formulatemplate.StatusInactive),
		CanApprove: !blocking &&
			library.Status == formulatemplate.StatusInReview &&
			reviewer.Status != ReadinessFail &&
			age.Status != ReadinessFail,
		Checks: append(
			gate.checks(),
			coverageCheck(locale, gate.content, coverage),
			status,
			reviewer,
			age,
		),
		TestTotal:   len(gate.content.tests.Tests),
		TestPassed:  gate.content.passed(),
		TestFailing: gate.content.failing(),
		Coverage:    coverage,
	}, nil
}

func detailsCheck(locale i18n.Locale, library *formulalibrary.FormulaLibrary) ReadinessCheck {
	check := ReadinessCheck{
		Key:   ReadinessCheckDetails,
		Label: i18n.Translate(locale, "Description"),
	}
	if strings.TrimSpace(library.Description) == "" {
		check.Status = ReadinessFail
		check.Detail = i18n.Translate(locale, "Description is missing")
		return check
	}
	check.Status = ReadinessPass
	check.Detail = i18n.Translate(locale, "Described")
	return check
}

func sourceCheck(locale i18n.Locale, content *contentCheck) ReadinessCheck {
	check := ReadinessCheck{
		Key:   ReadinessCheckSource,
		Label: i18n.Translate(locale, "Source checks clean"),
	}
	if content.checks() {
		check.Status = ReadinessPass
		check.Detail = i18n.Translate(locale, "No errors")
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

func testsCheck(locale i18n.Locale, content *contentCheck) ReadinessCheck {
	check := ReadinessCheck{Key: ReadinessCheckTests, Label: i18n.Translate(locale, "Tests pass")}
	total := len(content.tests.Tests)
	passed := content.passed()
	switch {
	case !content.checks():
		check.Status = ReadinessFail
		check.Detail = i18n.Translate(locale, "The tests run once the source checks")
	case total == 0:
		check.Status = ReadinessFail
		check.Detail = i18n.Translate(locale, "Add at least one test")
		check.refusal = i18n.Translate(locale, "Add at least one test block before submitting")
	case content.tests.Error != nil || passed < total:
		check.Status = ReadinessFail
		check.Detail = i18n.Translate(locale, "{0} of {1} pass", passed, total)
		check.refusal = i18n.Translate(
			locale,
			"Only {0} of {1} tests pass ({2}); fix the library or its tests before submitting",
			passed,
			total,
			strings.Join(tariffverdict.FailingTests(locale, content.tests), "; "),
		)
	default:
		check.Status = ReadinessPass
		check.Detail = i18n.Translate(locale, "{0} of {1} pass", passed, total)
	}
	return check
}

func coverageCheck(
	locale i18n.Locale,
	content *contentCheck,
	coverage *tariffverdict.CoverageSummary,
) ReadinessCheck {
	verdict := tariffverdict.CoverageCheck(locale, content.checks(), coverage)
	return ReadinessCheck{
		Key:    ReadinessCheckCoverage,
		Label:  verdict.Label,
		Status: verdict.Status,
		Detail: verdict.Detail,
	}
}

func reviewChecks(
	locale i18n.Locale,
	library *formulalibrary.FormulaLibrary,
	userID pulid.ID,
) (status, reviewer ReadinessCheck) {
	status = ReadinessCheck{
		Key:    ReadinessCheckStatus,
		Label:  i18n.Translate(locale, "Review state"),
		Status: ReadinessPass,
	}
	switch library.Status {
	case formulatemplate.StatusInReview:
		status.Detail = i18n.Translate(locale, "In review; awaiting a decision")
	case formulatemplate.StatusActive:
		status.Detail = i18n.Translate(
			locale,
			"Active; editing the source returns it to draft, and approved templates keep the version they pinned",
		)
	case formulatemplate.StatusDraft, formulatemplate.StatusInactive:
		status.Detail = i18n.Translate(locale, "Ready to submit for review")
	}

	reviewer = ReadinessCheck{
		Key:    ReadinessCheckReviewer,
		Label:  i18n.Translate(locale, "Independent reviewer"),
		Status: ReadinessPass,
	}
	switch {
	case library.Status != formulatemplate.StatusInReview:
		reviewer.Detail = i18n.Translate(locale, "Applies once the library is in review")
	case library.SubmittedByID != nil && *library.SubmittedByID == userID:
		reviewer.Status = ReadinessFail
		reviewer.Detail = i18n.Translate(
			locale,
			"You submitted this library; someone else must approve it",
		)
	default:
		reviewer.Detail = i18n.Translate(locale, "You did not submit this library")
	}
	return status, reviewer
}

func ageCheck(
	locale i18n.Locale,
	library *formulalibrary.FormulaLibrary,
	now int64,
) ReadinessCheck {
	check := ReadinessCheck{
		Key:    ReadinessCheckAge,
		Label:  i18n.Translate(locale, "Submission age"),
		Status: ReadinessPass,
	}
	if library.Status != formulatemplate.StatusInReview || library.SubmittedAt == nil {
		check.Detail = i18n.Translate(locale, "Applies once the library is in review")
		return check
	}
	if formulatemplate.SubmissionIsStale(library.SubmittedAt, now) {
		check.Status = ReadinessFail
		check.Detail = i18n.Translate(
			locale,
			"Older than the 14-day limit, so it must be resubmitted",
		)
		return check
	}
	check.Detail = i18n.Translate(locale, "Submitted recently")
	return check
}

type Reviewer struct {
	ID            pulid.ID `json:"id"`
	Name          string   `json:"name"`
	Username      string   `json:"username"`
	ProfilePicURL string   `json:"profilePicUrl,omitempty"`
}

func (s *Service) ListReviewers(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]Reviewer, error) {
	if s.roles == nil {
		return nil, errortypes.NewBusinessError(
			"Reviewers for formula libraries cannot be listed on this server",
		)
	}

	permitted, err := s.roles.ListUsersWithPermission(
		ctx,
		permissionports.ListUsersWithPermissionRequest{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			Resource:       permission.ResourceFormulaLibrary,
			Operation:      permission.OpApprove,
			Now:            timeutils.NowUnix(),
		},
	)
	if err != nil {
		return nil, err
	}

	reviewers := make([]Reviewer, 0, len(permitted))
	for _, user := range permitted {
		if user.UserID.IsNil() || user.UserID == tenantInfo.UserID {
			continue
		}
		reviewers = append(reviewers, Reviewer{
			ID:            user.UserID,
			Name:          user.Name,
			Username:      user.Username,
			ProfilePicURL: user.ProfilePicURL,
		})
	}
	return reviewers, nil
}

func (s *Service) isEligibleReviewer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	reviewerID pulid.ID,
) (bool, error) {
	reviewers, err := s.ListReviewers(ctx, tenantInfo)
	if err != nil {
		return false, err
	}
	for _, reviewer := range reviewers {
		if reviewer.ID == reviewerID {
			return true, nil
		}
	}
	return false, nil
}
