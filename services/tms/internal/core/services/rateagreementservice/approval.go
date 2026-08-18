package rateagreementservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/approvalworkflow"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// ApprovalActionRequest is the shared review request, named locally so handlers
// read the same way they do for formula templates.
type ApprovalActionRequest = approvalworkflow.Request

type agreementTransition = approvalworkflow.Transition[*rateagreement.RateAgreement, rateagreement.Status]

// Submit sends a draft agreement for review.
func (s *Service) Submit(
	ctx context.Context,
	req *ApprovalActionRequest,
) (*rateagreement.RateAgreement, error) {
	return s.approvals().Apply(ctx, req, agreementTransition{
		Operation:    "Submit",
		From:         rateagreement.StatusDraft,
		To:           rateagreement.StatusInReview,
		PermissionOp: permission.OpSubmit,
		AuditComment: "Rate agreement submitted for review",
		Apply: func(
			agreement *rateagreement.RateAgreement,
			r *ApprovalActionRequest,
			now int64,
		) {
			userID := r.TenantInfo.UserID
			agreement.SubmittedByID = &userID
			agreement.SubmittedAt = &now
			agreement.ReviewComment = r.Comment
		},
	})
}

// Approve activates an agreement, which is the moment it starts pricing
// shipments. Nothing else in the system turns rating on for a contract.
func (s *Service) Approve(
	ctx context.Context,
	req *ApprovalActionRequest,
) (*rateagreement.RateAgreement, error) {
	return s.approvals().Apply(ctx, req, agreementTransition{
		Operation:    "Approve",
		From:         rateagreement.StatusInReview,
		To:           rateagreement.StatusActive,
		PermissionOp: permission.OpApprove,
		AuditComment: "Rate agreement approved",
		Apply: func(
			agreement *rateagreement.RateAgreement,
			r *ApprovalActionRequest,
			now int64,
		) {
			userID := r.TenantInfo.UserID
			agreement.ApprovedByID = &userID
			agreement.ApprovedAt = &now
			agreement.ReviewComment = r.Comment
		},
	})
}

// Reject sends an agreement back to draft, and insists on a reason.
func (s *Service) Reject(
	ctx context.Context,
	req *ApprovalActionRequest,
) (*rateagreement.RateAgreement, error) {
	if err := approvalworkflow.RequireComment(req, "rejecting a rate agreement"); err != nil {
		return nil, err
	}

	return s.approvals().Apply(ctx, req, agreementTransition{
		Operation:    "Reject",
		From:         rateagreement.StatusInReview,
		To:           rateagreement.StatusDraft,
		PermissionOp: permission.OpReject,
		AuditComment: "Rate agreement rejected",
		Apply: func(
			agreement *rateagreement.RateAgreement,
			r *ApprovalActionRequest,
			_ int64,
		) {
			agreement.SubmittedByID = nil
			agreement.SubmittedAt = nil
			agreement.ReviewComment = r.Comment
		},
	})
}

// Suspend takes an active agreement out of service without ending it.
//
// This is the lever for a customer on credit hold or a carrier who has let
// their insurance lapse: shipments stop rating against the contract
// immediately, and resuming does not require another approval round.
func (s *Service) Suspend(
	ctx context.Context,
	req *ApprovalActionRequest,
) (*rateagreement.RateAgreement, error) {
	if err := approvalworkflow.RequireComment(req, "suspending a rate agreement"); err != nil {
		return nil, err
	}

	return s.approvals().Apply(ctx, req, agreementTransition{
		Operation:    "Suspend",
		From:         rateagreement.StatusActive,
		To:           rateagreement.StatusSuspended,
		PermissionOp: permission.OpUpdate,
		AuditComment: "Rate agreement suspended",
		Apply: func(
			agreement *rateagreement.RateAgreement,
			r *ApprovalActionRequest,
			_ int64,
		) {
			agreement.ReviewComment = r.Comment
		},
	})
}

// Resume puts a suspended agreement back in service.
func (s *Service) Resume(
	ctx context.Context,
	req *ApprovalActionRequest,
) (*rateagreement.RateAgreement, error) {
	return s.approvals().Apply(ctx, req, agreementTransition{
		Operation:    "Resume",
		From:         rateagreement.StatusSuspended,
		To:           rateagreement.StatusActive,
		PermissionOp: permission.OpUpdate,
		AuditComment: "Rate agreement resumed",
		Apply: func(
			agreement *rateagreement.RateAgreement,
			r *ApprovalActionRequest,
			_ int64,
		) {
			agreement.ReviewComment = r.Comment
		},
	})
}

// Archive retires an agreement permanently. It is kept rather than deleted
// because every quote it ever produced points at it.
func (s *Service) Archive(
	ctx context.Context,
	req *ApprovalActionRequest,
) (*rateagreement.RateAgreement, error) {
	agreement, err := s.repo.GetByID(ctx, &repositories.GetRateAgreementByIDRequest{
		RateAgreementID: req.EntityID,
		TenantInfo:      req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	return s.approvals().Apply(ctx, req, agreementTransition{
		Operation:    "Archive",
		From:         agreement.Status,
		To:           rateagreement.StatusArchived,
		PermissionOp: permission.OpArchive,
		AuditComment: "Rate agreement archived",
		Apply: func(
			a *rateagreement.RateAgreement,
			r *ApprovalActionRequest,
			_ int64,
		) {
			a.ReviewComment = r.Comment
		},
	})
}

// approvals binds the shared review cycle to this service.
func (s *Service) approvals() approvalworkflow.Engine[*rateagreement.RateAgreement, rateagreement.Status] {
	return approvalworkflow.Engine[*rateagreement.RateAgreement, rateagreement.Status]{
		Label: "rate agreement",
		Load: func(
			ctx context.Context,
			id pulid.ID,
			tenant pagination.TenantInfo,
		) (*rateagreement.RateAgreement, error) {
			return s.repo.GetByID(ctx, &repositories.GetRateAgreementByIDRequest{
				RateAgreementID: id,
				TenantInfo:      tenant,
			})
		},
		Save:     s.repo.Update,
		StatusOf: func(a *rateagreement.RateAgreement) rateagreement.Status { return a.Status },
		SetStatus: func(a *rateagreement.RateAgreement, status rateagreement.Status) {
			a.Status = status
		},
		CanTransition: rateagreement.CanTransition,
		Snapshot: func(a *rateagreement.RateAgreement) *rateagreement.RateAgreement {
			snapshot := *a
			return &snapshot
		},
		Audit: func(
			updated, original *rateagreement.RateAgreement,
			operation permission.Operation,
			req *ApprovalActionRequest,
			comment string,
		) {
			s.audit(
				s.l.With(zap.String("operation", string(operation))),
				updated,
				original,
				operation,
				req.TenantInfo.UserID,
				comment,
			)
		},
	}
}

// ReviewAction is the shape every review step shares, so a route table can hold
// them side by side instead of repeating the same handler six times.
type ReviewAction func(
	ctx context.Context,
	req *ApprovalActionRequest,
) (*rateagreement.RateAgreement, error)
