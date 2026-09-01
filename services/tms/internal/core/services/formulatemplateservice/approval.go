package formulatemplateservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/approvalworkflow"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
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
	var approved *formulatemplate.FormulaTemplate
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		result, txErr := s.approvals().Apply(txCtx, req, templateTransition{
			Operation:    "Approve",
			From:         formulatemplate.StatusInReview,
			To:           formulatemplate.StatusActive,
			PermissionOp: permission.OpApprove,
			AuditComment: "Formula template approved",
			Validate: func(
				vCtx context.Context,
				template *formulatemplate.FormulaTemplate,
				r *ApprovalActionRequest,
			) error {
				if template.SubmittedByID != nil &&
					*template.SubmittedByID == r.TenantInfo.UserID {
					return errortypes.NewValidationError(
						"approvedById",
						errortypes.ErrInvalid,
						"A template cannot be approved by its submitter",
					)
				}

				if err := s.validateTemplate(vCtx, template); err != nil {
					return err
				}

				return s.requirePassingTestCases(vCtx, template, r.TenantInfo)
			},
			Apply: func(
				template *formulatemplate.FormulaTemplate,
				r *ApprovalActionRequest,
				now int64,
			) {
				userID := r.TenantInfo.UserID
				template.ApprovedByID = &userID
				template.ApprovedAt = &now
				template.ReviewComment = r.Comment
				template.CurrentVersionNumber++
			},
			AfterSave: func(
				aCtx context.Context,
				updated *formulatemplate.FormulaTemplate,
				r *ApprovalActionRequest,
			) error {
				return s.createVersionSnapshot(
					aCtx,
					updated,
					updated.CurrentVersionNumber,
					r.TenantInfo.UserID,
					"Approved",
					nil,
				)
			},
		})
		if txErr != nil {
			return txErr
		}

		approved = result
		return nil
	})
	if err != nil {
		return nil, err
	}

	return approved, nil
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
