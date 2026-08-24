package formulatemplateservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/approvalworkflow"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// ApprovalActionRequest is kept as the package's own name for the shared
// request so existing callers and handlers read unchanged.
type ApprovalActionRequest = approvalworkflow.Request

type templateTransition = approvalworkflow.Transition[*formulatemplate.FormulaTemplate, formulatemplate.Status]

func (s *Service) Submit(
	ctx context.Context,
	req *ApprovalActionRequest,
) (*formulatemplate.FormulaTemplate, error) {
	return s.approvals().Apply(ctx, req, templateTransition{
		Operation:    "Submit",
		From:         formulatemplate.StatusDraft,
		To:           formulatemplate.StatusInReview,
		PermissionOp: permission.OpSubmit,
		AuditComment: "Formula template submitted for review",
		Apply: func(
			template *formulatemplate.FormulaTemplate,
			r *ApprovalActionRequest,
			now int64,
		) {
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
	return s.approvals().Apply(ctx, req, templateTransition{
		Operation:    "Approve",
		From:         formulatemplate.StatusInReview,
		To:           formulatemplate.StatusActive,
		PermissionOp: permission.OpApprove,
		AuditComment: "Formula template approved",
		Apply: func(
			template *formulatemplate.FormulaTemplate,
			r *ApprovalActionRequest,
			now int64,
		) {
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
	if err := approvalworkflow.RequireComment(req, "rejecting a formula template"); err != nil {
		return nil, err
	}

	return s.approvals().Apply(ctx, req, templateTransition{
		Operation:    "Reject",
		From:         formulatemplate.StatusInReview,
		To:           formulatemplate.StatusDraft,
		PermissionOp: permission.OpReject,
		AuditComment: "Formula template rejected",
		Apply: func(
			template *formulatemplate.FormulaTemplate,
			r *ApprovalActionRequest,
			_ int64,
		) {
			template.SubmittedByID = nil
			template.SubmittedAt = nil
			template.ReviewComment = r.Comment
		},
	})
}

// approvals binds the shared review cycle to this service's repository, status
// rules and audit log.
func (s *Service) approvals() approvalworkflow.Engine[*formulatemplate.FormulaTemplate, formulatemplate.Status] {
	return approvalworkflow.Engine[*formulatemplate.FormulaTemplate, formulatemplate.Status]{
		Label: "template",
		Load: func(
			ctx context.Context,
			id pulid.ID,
			tenant pagination.TenantInfo,
		) (*formulatemplate.FormulaTemplate, error) {
			return s.getTemplateByIDWithTenant(ctx, id, tenant)
		},
		Save:     s.repo.Update,
		StatusOf: func(t *formulatemplate.FormulaTemplate) formulatemplate.Status { return t.Status },
		SetStatus: func(t *formulatemplate.FormulaTemplate, status formulatemplate.Status) {
			t.Status = status
		},
		CanTransition: formulatemplate.CanTransition,
		Snapshot: func(t *formulatemplate.FormulaTemplate) *formulatemplate.FormulaTemplate {
			snapshot := *t
			return &snapshot
		},
		Audit: func(
			updated, original *formulatemplate.FormulaTemplate,
			operation permission.Operation,
			req *ApprovalActionRequest,
			comment string,
		) {
			s.logAuditAction(
				s.l.With(zap.String("operation", string(operation))),
				updated,
				operation,
				req.TenantInfo.UserID,
				original,
				comment,
			)
		},
	}
}
