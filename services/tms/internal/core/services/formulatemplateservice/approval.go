package formulatemplateservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

type ApprovalActionRequest struct {
	TenantInfo pagination.TenantInfo
	TemplateID pulid.ID
	Comment    string
}

type approvalTransition struct {
	operation    string
	from         formulatemplate.Status
	to           formulatemplate.Status
	permissionOp permission.Operation
	auditComment string
	apply        func(template *formulatemplate.FormulaTemplate, req *ApprovalActionRequest, now int64)
}

func (s *Service) Submit(
	ctx context.Context,
	req *ApprovalActionRequest,
) (*formulatemplate.FormulaTemplate, error) {
	return s.applyApprovalTransition(ctx, req, approvalTransition{
		operation:    "Submit",
		from:         formulatemplate.StatusDraft,
		to:           formulatemplate.StatusInReview,
		permissionOp: permission.OpSubmit,
		auditComment: "Formula template submitted for review",
		apply: func(template *formulatemplate.FormulaTemplate, r *ApprovalActionRequest, now int64) {
			userID := r.TenantInfo.UserID
			template.SubmittedByID = &userID
			template.SubmittedAt = &now
			template.ReviewComment = r.Comment
		},
	})
}

func (s *Service) Approve(
	ctx context.Context,
	req *ApprovalActionRequest,
) (*formulatemplate.FormulaTemplate, error) {
	return s.applyApprovalTransition(ctx, req, approvalTransition{
		operation:    "Approve",
		from:         formulatemplate.StatusInReview,
		to:           formulatemplate.StatusActive,
		permissionOp: permission.OpApprove,
		auditComment: "Formula template approved",
		apply: func(template *formulatemplate.FormulaTemplate, r *ApprovalActionRequest, now int64) {
			userID := r.TenantInfo.UserID
			template.ApprovedByID = &userID
			template.ApprovedAt = &now
			template.ReviewComment = r.Comment
		},
	})
}

func (s *Service) Reject(
	ctx context.Context,
	req *ApprovalActionRequest,
) (*formulatemplate.FormulaTemplate, error) {
	if req.Comment == "" {
		return nil, errortypes.NewValidationError(
			"comment",
			errortypes.ErrRequired,
			"A comment is required when rejecting a formula template",
		)
	}

	return s.applyApprovalTransition(ctx, req, approvalTransition{
		operation:    "Reject",
		from:         formulatemplate.StatusInReview,
		to:           formulatemplate.StatusDraft,
		permissionOp: permission.OpReject,
		auditComment: "Formula template rejected",
		apply: func(template *formulatemplate.FormulaTemplate, r *ApprovalActionRequest, _ int64) {
			template.SubmittedByID = nil
			template.SubmittedAt = nil
			template.ReviewComment = r.Comment
		},
	})
}

//nolint:gocritic // transition descriptors are small fixed literals
func (s *Service) applyApprovalTransition(
	ctx context.Context,
	req *ApprovalActionRequest,
	transition approvalTransition,
) (*formulatemplate.FormulaTemplate, error) {
	log := s.l.With(
		zap.String("operation", transition.operation),
		zap.String("templateID", req.TemplateID.String()),
	)

	template, err := s.getTemplateByIDWithTenant(ctx, req.TemplateID, req.TenantInfo)
	if err != nil {
		log.Error("failed to get formula template", zap.Error(err))
		return nil, err
	}

	if template.Status != transition.from ||
		!formulatemplate.CanTransition(template.Status, transition.to) {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			fmt.Sprintf(
				"Cannot transition template status from %s to %s",
				template.Status,
				transition.to,
			),
		)
	}

	original := *template
	template.Status = transition.to
	transition.apply(template, req, timeutils.NowUnix())

	updated, err := s.repo.Update(ctx, template)
	if err != nil {
		log.Error("failed to update formula template", zap.Error(err))
		return nil, err
	}

	s.logAuditAction(
		log,
		updated,
		transition.permissionOp,
		req.TenantInfo.UserID,
		&original,
		transition.auditComment,
	)

	return updated, nil
}
