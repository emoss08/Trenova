package billingqueuereviewservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingapprovalrun"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueuereview"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) ApplyBulkAction(
	ctx context.Context,
	req *billingqueueports.ApplyBulkActionRequest,
	actor *permissionports.RequestActor,
) error {
	switch req.Action {
	case billingapprovalrun.RunActionPost:
		_, err := s.Post(ctx, &billingqueueports.BillingQueueItemRequest{
			ItemID:     req.ItemID,
			TenantInfo: req.TenantInfo,
		}, actor)
		return err
	case billingapprovalrun.RunActionHold:
		return s.bulkHold(ctx, req, actor)
	case billingapprovalrun.RunActionAssign:
		return s.bulkAssign(ctx, req, actor)
	case billingapprovalrun.RunActionCancel:
		return s.bulkCancel(ctx, req, actor)
	case billingapprovalrun.RunActionRequestPaperwork:
		return s.bulkResolve(ctx, req, actor, paperworkIssue)
	case billingapprovalrun.RunActionResolveIssue:
		code := billingqueue.IssueCode(req.Params.IssueCode)
		return s.bulkResolve(ctx, req, actor, func(issue *billingqueue.Issue) (string, bool) {
			return req.Params.OptionKey, issue.Code == code &&
				hasOption(issue, req.Params.OptionKey)
		})
	case billingapprovalrun.RunActionApprove:
		return errortypes.NewValidationError(
			"action",
			errortypes.ErrInvalidOperation,
			"Approve runs through the approval check",
		)
	default:
		return errortypes.NewValidationError("action", errortypes.ErrInvalid, "Unknown bulk action")
	}
}

func (s *Service) bulkHold(
	ctx context.Context,
	req *billingqueueports.ApplyBulkActionRequest,
	actor *permissionports.RequestActor,
) error {
	item, err := s.queue.GetByID(ctx, &billingqueueports.GetBillingQueueItemByIDRequest{
		ItemID:     req.ItemID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return err
	}
	if !billingqueuereview.IsReviewable(item.Status) {
		return notApplicable("only an item still in review can be held")
	}

	code := billingqueue.HoldReasonCode(req.Params.HoldReasonCode)
	_, err = s.queue.UpdateStatus(ctx, &billingqueueports.UpdateBillingQueueStatusRequest{
		ItemID:         req.ItemID,
		NewStatus:      billingqueue.StatusOnHold,
		HoldReasonCode: &code,
		ReviewNotes:    req.Params.Reason,
		TenantInfo:     req.TenantInfo,
	}, actor)

	return err
}

func (s *Service) bulkAssign(
	ctx context.Context,
	req *billingqueueports.ApplyBulkActionRequest,
	actor *permissionports.RequestActor,
) error {
	item, err := s.queue.GetByID(ctx, &billingqueueports.GetBillingQueueItemByIDRequest{
		ItemID:     req.ItemID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return err
	}
	if billingqueue.IsTerminalStatus(item.Status) || item.Status == billingqueue.StatusApproved {
		return notApplicable("a decided item keeps its biller")
	}
	if item.AssignedBillerID != nil && *item.AssignedBillerID == req.Params.BillerID {
		return notApplicable("already assigned to this biller")
	}

	_, err = s.queue.AssignBiller(ctx, &billingqueueports.AssignBillerRequest{
		ItemID:     req.ItemID,
		BillerID:   req.Params.BillerID,
		TenantInfo: req.TenantInfo,
	}, actor)

	return err
}

func (s *Service) bulkCancel(
	ctx context.Context,
	req *billingqueueports.ApplyBulkActionRequest,
	actor *permissionports.RequestActor,
) error {
	item, err := s.queue.GetByID(ctx, &billingqueueports.GetBillingQueueItemByIDRequest{
		ItemID:     req.ItemID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return err
	}
	if billingqueue.IsTerminalStatus(item.Status) || item.Status == billingqueue.StatusApproved {
		return notApplicable("only an undecided item can be canceled")
	}

	_, err = s.queue.UpdateStatus(ctx, &billingqueueports.UpdateBillingQueueStatusRequest{
		ItemID:       req.ItemID,
		NewStatus:    billingqueue.StatusCanceled,
		CancelReason: req.Params.Reason,
		TenantInfo:   req.TenantInfo,
	}, actor)

	return err
}

type issuePicker func(issue *billingqueue.Issue) (optionKey string, ok bool)

func (s *Service) bulkResolve(
	ctx context.Context,
	req *billingqueueports.ApplyBulkActionRequest,
	actor *permissionports.RequestActor,
	pick issuePicker,
) error {
	if s.reviewRepo == nil {
		return notApplicable("issues are not available")
	}
	issues, err := s.reviewRepo.ListIssues(ctx, req.TenantInfo, req.ItemID)
	if err != nil {
		return err
	}

	issueID, optionKey := pulid.Nil, ""
	for _, issue := range issues {
		if !issue.IsOpen() {
			continue
		}
		if key, ok := pick(issue); ok {
			issueID, optionKey = issue.ID, key
			break
		}
	}
	if issueID.IsNil() {
		return notApplicable("no open issue on the item takes this action")
	}

	_, err = s.ResolveIssue(ctx, &billingqueueports.ResolveBillingQueueIssueRequest{
		ItemID:     req.ItemID,
		IssueID:    issueID,
		OptionKey:  optionKey,
		TenantInfo: req.TenantInfo,
	}, actor)

	return err
}

func paperworkIssue(issue *billingqueue.Issue) (string, bool) {
	if issue.RequestedAt != nil {
		return "", false
	}
	if issue.Code != billingqueue.IssuePODMissing && issue.Code != billingqueue.IssuePODUnsigned {
		return "", false
	}

	return billingqueue.OptionKeyRequest, hasOption(issue, billingqueue.OptionKeyRequest)
}

func hasOption(issue *billingqueue.Issue, key string) bool {
	for _, option := range issue.Options {
		if option.Key == key {
			return true
		}
	}

	return false
}

func notApplicable(reason string) error {
	return fmt.Errorf("%w: %s", billingqueueports.ErrBulkActionNotApplicable, reason)
}
