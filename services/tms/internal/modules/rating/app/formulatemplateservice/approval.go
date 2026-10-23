package formulatemplateservice

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/approvalworkflow"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type ApprovalActionRequest = approvalworkflow.Request

type templateTransition = approvalworkflow.Transition[*formulatemplate.FormulaTemplate, formulatemplate.Status]

type SubmitRequest struct {
	ApprovalActionRequest
	ReviewerID   pulid.ID
	ReviewerNote string
}

func (s *Service) Submit(
	ctx context.Context,
	req *SubmitRequest,
) (*formulatemplate.FormulaTemplate, error) {
	var submitted *formulatemplate.FormulaTemplate
	err := s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		result, txErr := s.approvals(ctx).
			Apply(txCtx, &req.ApprovalActionRequest, templateTransition{
				Operation:    "Submit",
				From:         formulatemplate.StatusDraft,
				AlsoFrom:     []formulatemplate.Status{formulatemplate.StatusInactive},
				To:           formulatemplate.StatusInReview,
				PermissionOp: permission.OpSubmit,
				AuditComment: "Formula template submitted for review",
				Validate: func(
					vCtx context.Context,
					template *formulatemplate.FormulaTemplate,
					_ *ApprovalActionRequest,
				) error {
					return s.validateSubmission(vCtx, template, req)
				},
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
				AfterSave: func(
					aCtx context.Context,
					updated *formulatemplate.FormulaTemplate,
					r *ApprovalActionRequest,
				) error {
					return s.recordReview(aCtx, updated, r.TenantInfo, &reviewRecord{
						Comment:             r.Comment,
						Decision:            formulatemplate.ReviewDecisionSubmitted,
						RequestedReviewerID: req.ReviewerID,
						ReviewerNote:        req.ReviewerNote,
					})
				},
			})
		if txErr != nil {
			return txErr
		}

		submitted = result
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.notifySubmitted(ctx, req.TenantInfo, submitted)
	if req.ReviewerID.IsNotNil() {
		s.notifyRequestedReviewer(ctx, req.TenantInfo, submitted, req.ReviewerID)
	}

	return submitted, nil
}

type ApproveRequest struct {
	ApprovalActionRequest
	EffectiveFrom *int64
}

func (s *Service) Approve(
	ctx context.Context,
	req *ApprovalActionRequest,
) (*formulatemplate.FormulaTemplate, error) {
	return s.ApproveFrom(ctx, &ApproveRequest{ApprovalActionRequest: *req})
}

func (s *Service) ApproveFrom(
	ctx context.Context,
	approveReq *ApproveRequest,
) (*formulatemplate.FormulaTemplate, error) {
	req := &approveReq.ApprovalActionRequest
	effectiveFrom := timeutils.NowUnix()
	if approveReq.EffectiveFrom != nil {
		if err := validateEffectiveFrom(*approveReq.EffectiveFrom); err != nil {
			return nil, err
		}
		effectiveFrom = max(*approveReq.EffectiveFrom, effectiveFrom)
	}

	var approved *formulatemplate.FormulaTemplate
	var pinned *contentCheck
	err := s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		result, txErr := s.approvals(ctx).Apply(txCtx, req, templateTransition{
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

				if formulatemplate.SubmissionIsStale(
					template.SubmittedAt,
					s.approvals(vCtx).Clock(),
				) {
					return staleSubmissionError()
				}

				if err := s.refuseRevisionAuthor(vCtx, template, r.TenantInfo); err != nil {
					return err
				}

				content, err := s.approvalContent(vCtx, template)
				if err != nil {
					return err
				}
				pinned = content
				return nil
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
				applyContent(template, pinned)
				template.CurrentVersionNumber++
			},
			AfterSave: func(
				aCtx context.Context,
				updated *formulatemplate.FormulaTemplate,
				r *ApprovalActionRequest,
			) error {
				if err := s.createApprovedSnapshot(
					aCtx,
					updated,
					r.TenantInfo.UserID,
					effectiveFrom,
				); err != nil {
					return err
				}
				return s.recordReview(aCtx, updated, r.TenantInfo, &reviewRecord{
					Comment:  r.Comment,
					Decision: formulatemplate.ReviewDecisionApproved,
				})
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

	s.notifyReviewOutcome(ctx, req.TenantInfo, &reviewOutcome{
		Template:    approved,
		SubmitterID: approved.SubmittedByID,
		Decision:    formulatemplate.ReviewDecisionApproved,
		Comment:     req.Comment,
	})

	return approved, nil
}

func (s *Service) approvalContent(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
) (*contentCheck, error) {
	content, err := s.checkContent(ctx, &contentRequest{
		tenant: templateTenant(template),
		source: template.Source,
		mode:   tariff.LibrariesLatest,
	})
	if err != nil {
		return nil, err
	}
	if refusal := content.refusal(); refusal != nil {
		return nil, refusal
	}

	gate := newSubmissionGate(ctx, template, content)
	multiErr := errortypes.NewMultiError()
	gate.addRefusals(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	dependents, err := s.checkDependents(ctx, template, content)
	if err != nil {
		return nil, err
	}
	if refusal := dependentsRefusal(i18n.FromContext(ctx), dependents); refusal != nil {
		return nil, refusal
	}

	return content, nil
}

func (s *Service) refuseRevisionAuthor(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	tenantInfo pagination.TenantInfo,
) error {
	var baseVersion int64
	latest, err := s.reviewRepo.Latest(ctx, template.ID, tenantInfo)
	if err != nil {
		return err
	}
	if latest != nil {
		baseVersion = latest.BaseVersionNumber
	}

	authors, err := s.versionRepo.ListAuthorIDsSince(
		ctx,
		&formulatemplateports.ListVersionAuthorsRequest{
			TenantInfo:   tenantInfo,
			TemplateID:   template.ID,
			AfterVersion: baseVersion,
		},
	)
	if err != nil {
		return err
	}

	if slices.Contains(authors, tenantInfo.UserID) {
		return errortypes.NewValidationError(
			"approvedById",
			errortypes.ErrInvalid,
			"You authored part of this revision, so another reviewer must approve it",
		)
	}

	return nil
}

func (s *Service) Reject(
	ctx context.Context,
	req *ApprovalActionRequest,
) (*formulatemplate.FormulaTemplate, error) {
	if err := approvalworkflow.RequireComment(req, "rejecting a formula template"); err != nil {
		return nil, err
	}

	billingVersion := s.approvedVersionNumber(ctx, req.EntityID, req.TenantInfo)
	destination := formulatemplate.StatusInactive
	if billingVersion > 0 {
		destination = formulatemplate.StatusDraft
	}

	var submitterID *pulid.ID
	var rejected *formulatemplate.FormulaTemplate
	err := s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		result, txErr := s.approvals(ctx).Apply(txCtx, req, templateTransition{
			Operation:    "Reject",
			From:         formulatemplate.StatusInReview,
			To:           destination,
			PermissionOp: permission.OpReject,
			AuditComment: "Formula template rejected",
			Apply: func(
				template *formulatemplate.FormulaTemplate,
				r *ApprovalActionRequest,
				_ int64,
			) {
				if template.SubmittedByID != nil {
					captured := *template.SubmittedByID
					submitterID = &captured
				}
				template.SubmittedByID = nil
				template.SubmittedAt = nil
				template.ReviewComment = r.Comment
			},
			AfterSave: func(
				aCtx context.Context,
				updated *formulatemplate.FormulaTemplate,
				r *ApprovalActionRequest,
			) error {
				cleared, clearErr := s.clearScheduledVersions(aCtx, updated)
				if clearErr != nil {
					return clearErr
				}
				if cleared > 0 {
					s.l.Info("rejection cleared scheduled versions",
						zap.String("templateID", updated.ID.String()),
						zap.Int64("cleared", cleared),
					)
				}
				return s.recordReview(aCtx, updated, r.TenantInfo, &reviewRecord{
					Comment:  r.Comment,
					Decision: formulatemplate.ReviewDecisionRejected,
				})
			},
		})
		if txErr != nil {
			return txErr
		}

		rejected = result
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.notifyReviewOutcome(ctx, req.TenantInfo, &reviewOutcome{
		Template:       rejected,
		SubmitterID:    submitterID,
		Decision:       formulatemplate.ReviewDecisionRejected,
		Comment:        req.Comment,
		BillingVersion: billingVersion,
	})

	return rejected, nil
}

func (s *Service) approvals(
	ctx context.Context,
) approvalworkflow.Engine[*formulatemplate.FormulaTemplate, formulatemplate.Status] {
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
			opts := make([]auditports.LogOption, 0, 1)
			if operation == permission.OpApprove || operation == permission.OpReject {
				opts = append(opts, auditservice.WithCritical())
			}
			s.logAuditAction(
				ctx,
				s.l.With(zap.String("operation", string(operation))),
				updated,
				operation,
				req.TenantInfo.UserID,
				original,
				comment,
				opts...,
			)
		},
	}
}
