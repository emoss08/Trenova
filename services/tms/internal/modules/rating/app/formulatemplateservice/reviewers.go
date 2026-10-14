package formulatemplateservice

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/contextvariablecache"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetablecache"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type Reviewer struct {
	ID            pulid.ID `json:"id"`
	Name          string   `json:"name"`
	Username      string   `json:"username"`
	ProfilePicURL string   `json:"profilePicUrl,omitempty"`
}

type ListReviewersRequest struct {
	TenantInfo pagination.TenantInfo
	TemplateID pulid.ID
}

func (s *Service) ListReviewers(
	ctx context.Context,
	req *ListReviewersRequest,
) ([]Reviewer, error) {
	if _, err := s.getTemplateByIDWithTenant(ctx, req.TemplateID, req.TenantInfo); err != nil {
		return nil, err
	}

	return s.eligibleReviewers(ctx, req.TenantInfo)
}

func (s *Service) eligibleReviewers(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]Reviewer, error) {
	if s.roles == nil {
		return nil, errortypes.NewBusinessError(
			"Reviewers for formula templates cannot be listed on this server",
		)
	}

	permitted, err := s.roles.ListUsersWithPermission(
		ctx,
		permissionports.ListUsersWithPermissionRequest{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			Resource:       permission.ResourceFormulaTemplate,
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

func (s *Service) validateSubmission(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	req *SubmitRequest,
) error {
	multiErr := errortypes.NewMultiError()

	if utf8.RuneCountInString(strings.TrimSpace(req.ReviewerNote)) >
		formulatemplate.MaxReviewerNoteLength {
		multiErr.Add(
			"reviewerNote",
			errortypes.ErrInvalid,
			"A note for the reviewer cannot be longer than 4,000 characters",
		)
	}

	if req.ReviewerID.IsNotNil() {
		eligible, err := s.isEligibleReviewer(ctx, req.TenantInfo, req.ReviewerID)
		if err != nil {
			return err
		}
		if !eligible {
			multiErr.Add(
				"reviewerId",
				errortypes.ErrInvalid,
				"Choose an active reviewer, other than you, who can approve formula templates",
			)
		}
	}

	gate, err := s.evaluateSubmissionGate(
		contextvariablecache.With(ratetablecache.With(ctx)),
		template,
		req.TenantInfo,
		&ReadinessResponse{},
	)
	if err != nil {
		return err
	}
	gate.addRefusals(multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *Service) isEligibleReviewer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	reviewerID pulid.ID,
) (bool, error) {
	reviewers, err := s.eligibleReviewers(ctx, tenantInfo)
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
