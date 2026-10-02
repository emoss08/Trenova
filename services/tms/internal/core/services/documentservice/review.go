package documentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	metadataResourceID   = "resourceId"
	metadataResourceType = "resourceType"
)

type ReviewRequest struct {
	DocumentID pulid.ID
	TenantInfo pagination.TenantInfo
	Decision   document.ReviewDecision
	Reason     string
	Actor      services.RequestActor
}

type ReviewPlan struct {
	Before *document.Document
	After  *document.Document
}

func (s *Service) PlanReview(ctx context.Context, req *ReviewRequest) (*ReviewPlan, error) {
	if req == nil || req.DocumentID.IsNil() {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("documentId", errortypes.ErrRequired, "Document is required")
		return nil, multiErr
	}

	if !req.Decision.IsValid() {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("decision", errortypes.ErrInvalid, "Decision must be approve or reject")
		return nil, multiErr
	}

	reviewerID := req.Actor.PersonUserID()
	if reviewerID.IsNil() {
		return nil, errortypes.NewAuthorizationError("Only a person can review a document")
	}

	current, err := s.repo.GetByID(ctx, repositories.GetDocumentByIDRequest{
		ID:         req.DocumentID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	after := *current
	now := timeutils.NowUnix()
	switch req.Decision {
	case document.ReviewDecisionApprove:
		err = after.Approve(reviewerID, now)
	case document.ReviewDecisionReject:
		err = after.Reject(reviewerID, now, req.Reason)
	}
	if err != nil {
		return nil, err
	}

	return &ReviewPlan{Before: current, After: &after}, nil
}

func (s *Service) Review(ctx context.Context, req *ReviewRequest) (*document.Document, error) {
	plan, err := s.PlanReview(ctx, req)
	if err != nil {
		return nil, err
	}

	multiErr := errortypes.NewMultiError()
	plan.After.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	reviewed, err := s.repo.Update(ctx, plan.After)
	if err != nil {
		return nil, err
	}

	operation, comment := reviewAuditOperation(req.Decision)
	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceDocument,
		ResourceID:     reviewed.ID.String(),
		Operation:      operation,
		UserID:         req.Actor.AuditActor().UserID,
		PreviousState:  jsonutils.MustToJSON(plan.Before),
		CurrentState:   jsonutils.MustToJSON(reviewed),
		OrganizationID: reviewed.OrganizationID,
		BusinessUnitID: reviewed.BusinessUnitID,
	},
		auditservice.WithComment(comment),
		auditservice.WithDiff(plan.Before, reviewed),
		auditservice.WithRequest(ctx),
		auditservice.WithMetadata(map[string]any{
			metadataResourceID:   reviewed.ResourceID,
			metadataResourceType: reviewed.ResourceType,
			"reason":             reviewed.RejectionReason,
		}),
	); err != nil {
		s.l.Warn("failed to log document review", zap.Error(err))
	}

	s.publishDocumentInvalidation(ctx, reviewed, &req.Actor, operation)

	return reviewed, nil
}

func reviewAuditOperation(
	decision document.ReviewDecision,
) (operation permission.Operation, comment string) {
	if decision == document.ReviewDecisionReject {
		return permission.OpReject, "Document rejected"
	}

	return permission.OpApprove, "Document approved"
}
