package formulalibraryservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
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
	CanSubmit   bool             `json:"canSubmit"`
	CanApprove  bool             `json:"canApprove"`
	Checks      []ReadinessCheck `json:"checks"`
	TestTotal   int              `json:"testTotal"`
	TestPassed  int              `json:"testPassed"`
	TestFailing []string         `json:"testFailing,omitempty"`
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
	return &submissionGate{
		details: detailsCheck(library),
		source:  sourceCheck(ctx, content),
		tests:   testsCheck(content),
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

	status, reviewer := reviewChecks(library, req.TenantInfo.UserID)
	age := ageCheck(library, timeutils.NowUnix())
	blocking := gate.blocking()

	return &ReadinessResponse{
		CanSubmit: !blocking &&
			(library.Status == formulatemplate.StatusDraft ||
				library.Status == formulatemplate.StatusInactive),
		CanApprove: !blocking &&
			library.Status == formulatemplate.StatusInReview &&
			reviewer.Status != ReadinessFail &&
			age.Status != ReadinessFail,
		Checks:      append(gate.checks(), status, reviewer, age),
		TestTotal:   len(gate.content.tests.Tests),
		TestPassed:  gate.content.passed(),
		TestFailing: gate.content.failing(),
	}, nil
}

func detailsCheck(library *formulalibrary.FormulaLibrary) ReadinessCheck {
	check := ReadinessCheck{Key: ReadinessCheckDetails, Label: "Description"}
	if strings.TrimSpace(library.Description) == "" {
		check.Status = ReadinessFail
		check.Detail = "Description is missing"
		return check
	}
	check.Status = ReadinessPass
	check.Detail = "Described"
	return check
}

func sourceCheck(ctx context.Context, content *contentCheck) ReadinessCheck {
	check := ReadinessCheck{Key: ReadinessCheckSource, Label: "Source checks clean"}
	if content.checks() {
		check.Status = ReadinessPass
		check.Detail = "No errors"
		return check
	}
	check.Status = ReadinessFail
	check.Detail = "The source does not check"
	if first := content.checked.Check.FirstError(); first != nil {
		check.Detail = first.Message.Localize(i18n.FromContext(ctx))
	}
	check.refusal = "Fix the source's errors before submitting: " + check.Detail
	return check
}

func testsCheck(content *contentCheck) ReadinessCheck {
	check := ReadinessCheck{Key: ReadinessCheckTests, Label: "Tests pass"}
	total := len(content.tests.Tests)
	passed := content.passed()
	switch {
	case !content.checks():
		check.Status = ReadinessFail
		check.Detail = "The tests run once the source checks"
	case total == 0:
		check.Status = ReadinessFail
		check.Detail = "Add at least one test"
		check.refusal = "Add at least one test block before submitting"
	case content.tests.Error != nil || passed < total:
		check.Status = ReadinessFail
		check.Detail = fmt.Sprintf("%d of %d pass", passed, total)
		check.refusal = fmt.Sprintf(
			"Only %d of %d tests pass (%s); fix the library or its tests before submitting",
			passed, total, strings.Join(content.failing(), ", "),
		)
	default:
		check.Status = ReadinessPass
		check.Detail = fmt.Sprintf("%d of %d pass", passed, total)
	}
	return check
}

func reviewChecks(
	library *formulalibrary.FormulaLibrary,
	userID pulid.ID,
) (status, reviewer ReadinessCheck) {
	status = ReadinessCheck{
		Key:    ReadinessCheckStatus,
		Label:  "Review state",
		Status: ReadinessPass,
	}
	switch library.Status {
	case formulatemplate.StatusInReview:
		status.Detail = "In review; awaiting a decision"
	case formulatemplate.StatusActive:
		status.Detail = "Active; editing the source returns it to draft, and approved templates keep the version they pinned"
	case formulatemplate.StatusDraft, formulatemplate.StatusInactive:
		status.Detail = "Ready to submit for review"
	}

	reviewer = ReadinessCheck{
		Key:    ReadinessCheckReviewer,
		Label:  "Independent reviewer",
		Status: ReadinessPass,
	}
	switch {
	case library.Status != formulatemplate.StatusInReview:
		reviewer.Detail = "Applies once the library is in review"
	case library.SubmittedByID != nil && *library.SubmittedByID == userID:
		reviewer.Status = ReadinessFail
		reviewer.Detail = "You submitted this library; someone else must approve it"
	default:
		reviewer.Detail = "You did not submit this library"
	}
	return status, reviewer
}

func ageCheck(library *formulalibrary.FormulaLibrary, now int64) ReadinessCheck {
	check := ReadinessCheck{Key: ReadinessCheckAge, Label: "Submission age", Status: ReadinessPass}
	if library.Status != formulatemplate.StatusInReview || library.SubmittedAt == nil {
		check.Detail = "Applies once the library is in review"
		return check
	}
	if formulatemplate.SubmissionIsStale(library.SubmittedAt, now) {
		check.Status = ReadinessFail
		check.Detail = "Older than the 14-day limit, so it must be resubmitted"
		return check
	}
	check.Detail = "Submitted recently"
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
