package formulatemplateservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/pkg/approvalworkflow"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type ListReviewsRequest struct {
	TenantInfo pagination.TenantInfo
	TemplateID pulid.ID
}

// ListReviews returns the template's review history, newest first.
func (s *Service) ListReviews(
	ctx context.Context,
	req *ListReviewsRequest,
) ([]*formulatemplate.Review, error) {
	if _, err := s.getTemplateByIDWithTenant(ctx, req.TemplateID, req.TenantInfo); err != nil {
		return nil, err
	}

	reviews, err := s.reviewRepo.ListByTemplate(
		ctx,
		&formulatemplateports.ListTemplateReviewsRequest{
			TemplateID: req.TemplateID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	if reviews == nil {
		reviews = []*formulatemplate.Review{}
	}

	return reviews, nil
}

// RequestChanges sends a template back to its author without closing the
// review round. Unlike Reject, the submission stamps stay, so the history
// shows one conversation: submitted, changes requested, resubmitted, decided.
func (s *Service) RequestChanges(
	ctx context.Context,
	req *ApprovalActionRequest,
) (*formulatemplate.FormulaTemplate, error) {
	if err := approvalworkflow.RequireComment(req, "requesting changes"); err != nil {
		return nil, err
	}

	var submitterID *pulid.ID
	var returned *formulatemplate.FormulaTemplate
	err := s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		result, txErr := s.approvals(ctx).Apply(txCtx, req, templateTransition{
			Operation:    "RequestChanges",
			From:         formulatemplate.StatusInReview,
			To:           formulatemplate.StatusDraft,
			PermissionOp: permission.OpReject,
			AuditComment: "Changes requested on formula template",
			Validate: func(
				_ context.Context,
				template *formulatemplate.FormulaTemplate,
				r *ApprovalActionRequest,
			) error {
				if template.SubmittedByID != nil && *template.SubmittedByID == r.TenantInfo.UserID {
					return errortypes.NewValidationError(
						"approvedById",
						errortypes.ErrInvalid,
						"A template's submitter cannot review it; ask another reviewer or archive it",
					)
				}
				return nil
			},
			Apply: func(
				template *formulatemplate.FormulaTemplate,
				r *ApprovalActionRequest,
				_ int64,
			) {
				if template.SubmittedByID != nil {
					captured := *template.SubmittedByID
					submitterID = &captured
				}
				template.ReviewComment = r.Comment
			},
			AfterSave: func(
				aCtx context.Context,
				updated *formulatemplate.FormulaTemplate,
				r *ApprovalActionRequest,
			) error {
				return s.recordReview(aCtx, updated, r.TenantInfo, &reviewRecord{
					Comment:  r.Comment,
					Decision: formulatemplate.ReviewDecisionChangesRequested,
				})
			},
		})
		if txErr != nil {
			return txErr
		}

		returned = result
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.notifyReviewOutcome(ctx, req.TenantInfo, &reviewOutcome{
		Template:    returned,
		SubmitterID: submitterID,
		Decision:    formulatemplate.ReviewDecisionChangesRequested,
		Comment:     req.Comment,
	})

	return returned, nil
}

type ExpireStaleSubmissionsRequest struct {
	// Now is the moment the sweep runs; zero means the wall clock.
	Now   int64
	Limit int
}

type ExpireStaleSubmissionsResult struct {
	Expired []pulid.ID `json:"expired"`
}

// ExpireStaleSubmissions returns every template that has waited in review
// longer than the expiry to draft, records the expiry in its history, and
// tells the author. It runs across tenants: each template carries its own.
func (s *Service) ExpireStaleSubmissions(
	ctx context.Context,
	req *ExpireStaleSubmissionsRequest,
) (*ExpireStaleSubmissionsResult, error) {
	now := req.Now
	if now == 0 {
		now = timeutils.NowUnix()
	}
	log := s.l.With(zap.String("operation", "ExpireStaleSubmissions"))

	stale, err := s.reviewRepo.ListStaleSubmissions(
		ctx,
		&formulatemplateports.ListStaleSubmissionsRequest{
			SubmittedBefore: now - formulatemplate.SubmissionExpiry,
			Limit:           req.Limit,
		},
	)
	if err != nil {
		return nil, err
	}

	result := &ExpireStaleSubmissionsResult{Expired: make([]pulid.ID, 0, len(stale))}
	for _, template := range stale {
		if template == nil || !formulatemplate.SubmissionIsStale(template.SubmittedAt, now) {
			continue
		}
		tenantInfo := pagination.TenantInfo{
			OrgID: template.OrganizationID,
			BuID:  template.BusinessUnitID,
		}
		tenantCtx := dbscope.WithTenant(ctx, tenantInfo.DBTenant())
		if expireErr := s.expireSubmission(tenantCtx, template, tenantInfo); expireErr != nil {
			log.Error("failed to expire stale submission",
				zap.String("templateID", template.ID.String()),
				zap.Error(expireErr),
			)
			continue
		}
		result.Expired = append(result.Expired, template.ID)
	}

	return result, nil
}

func (s *Service) expireSubmission(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	tenantInfo pagination.TenantInfo,
) error {
	var submitterID *pulid.ID
	if template.SubmittedByID != nil {
		captured := *template.SubmittedByID
		submitterID = &captured
	}

	original := *template
	template.Status = formulatemplate.StatusDraft
	template.SubmittedByID = nil
	template.SubmittedAt = nil
	template.ReviewComment = "Submission expired after waiting for review"

	var updated *formulatemplate.FormulaTemplate
	err := s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		saved, txErr := s.repo.Update(txCtx, template)
		if txErr != nil {
			return txErr
		}

		if txErr = s.recordReview(txCtx, saved, tenantInfo, &reviewRecord{
			Comment:  template.ReviewComment,
			Decision: formulatemplate.ReviewDecisionExpired,
		}); txErr != nil {
			return txErr
		}

		updated = saved
		return nil
	})
	if err != nil {
		return err
	}

	s.logAuditAction(
		ctx,
		s.l.With(zap.String("operation", "ExpireSubmission")),
		updated,
		permission.OpUpdate,
		pulid.Nil,
		&original,
		"Formula template submission expired",
	)

	s.notifyReviewOutcome(ctx, tenantInfo, &reviewOutcome{
		Template:    updated,
		SubmitterID: submitterID,
		Decision:    formulatemplate.ReviewDecisionExpired,
		Comment:     template.ReviewComment,
	})

	return nil
}

// recordReview appends one entry to the template's history. Submissions open
// or continue a round; every other decision lands in the round that is open.
// The approved base version is captured at submission so the diff a reviewer
// saw is reconstructible later.
type reviewRecord struct {
	Comment             string
	Decision            formulatemplate.ReviewDecision
	RequestedReviewerID pulid.ID
	ReviewerNote        string
}

func (s *Service) recordReview(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	tenantInfo pagination.TenantInfo,
	record *reviewRecord,
) error {
	decision := record.Decision
	latest, err := s.reviewRepo.Latest(ctx, template.ID, tenantInfo)
	if err != nil {
		return err
	}

	round := formulatemplate.NextRound(latest)
	if decision != formulatemplate.ReviewDecisionSubmitted {
		round = 1
		if latest != nil {
			round = latest.Round
		}
	}

	review := &formulatemplate.Review{
		TemplateID:     template.ID,
		OrganizationID: template.OrganizationID,
		BusinessUnitID: template.BusinessUnitID,
		Round:          round,
		Decision:       decision,
		Comment:        record.Comment,
		ReviewerNote:   strings.TrimSpace(record.ReviewerNote),
	}
	if record.RequestedReviewerID.IsNotNil() {
		reviewerID := record.RequestedReviewerID
		review.RequestedReviewerID = &reviewerID
	}
	if decision != formulatemplate.ReviewDecisionExpired && !tenantInfo.UserID.IsNil() {
		actor := tenantInfo.UserID
		review.ActorID = &actor
	}
	if decision == formulatemplate.ReviewDecisionSubmitted {
		review.BaseVersionNumber = s.latestApprovedVersionNumber(ctx, template, tenantInfo)
	} else if latest != nil {
		review.BaseVersionNumber = latest.BaseVersionNumber
	}

	_, err = s.reviewRepo.Create(ctx, review)
	return err
}

func (s *Service) latestApprovedVersionNumber(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	tenantInfo pagination.TenantInfo,
) int64 {
	return s.approvedVersionNumber(ctx, template.ID, tenantInfo)
}

func (s *Service) approvedVersionNumber(
	ctx context.Context,
	templateID pulid.ID,
	tenantInfo pagination.TenantInfo,
) int64 {
	version, err := s.versionRepo.GetLatestByStatus(
		ctx,
		&formulatemplateports.GetLatestVersionByStatusRequest{
			TemplateID: templateID,
			TenantInfo: tenantInfo,
			Status:     formulatemplate.StatusActive,
		},
	)
	if err != nil || version == nil {
		return 0
	}
	return version.VersionNumber
}

// staleSubmissionError is the refusal a reviewer sees when the submission has
// waited past the expiry; the author must resubmit against current rates.
func staleSubmissionError() error {
	return errortypes.NewValidationError(
		"submittedAt",
		errortypes.ErrInvalid,
		"This submission is older than 14 days; ask the author to resubmit so the review reflects current rates",
	)
}
