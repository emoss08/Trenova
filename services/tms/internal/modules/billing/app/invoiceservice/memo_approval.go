package invoiceservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const memoRaisedByKey = "raisedById"

// A standalone credit memo takes money off a customer's balance just as an
// invoice adjustment does, so it answers to the same approval policy. One the
// policy would send for approval is never posted in the same step that raised
// it, and where the organization requires a separate approver, whoever raised
// it cannot post it.

func memoNeedsApproval(control *tenant.InvoiceAdjustmentControl, entity *invoice.Invoice) bool {
	if control == nil || entity == nil ||
		entity.Scope != invoice.ScopeMemo ||
		entity.BillType != billingqueue.BillTypeCreditMemo {
		return false
	}

	switch control.StandardAdjustmentApprovalPolicy { //nolint:exhaustive // None needs nothing
	case tenant.ApprovalPolicyAlways:
		return true
	case tenant.ApprovalPolicyAmountThreshold:
		return entity.TotalAmount.Abs().GreaterThanOrEqual(control.StandardAdjustmentApprovalThreshold)
	default:
		return false
	}
}

func (s *Service) adjustmentApprovalControl(
	ctx context.Context,
	orgID pulid.ID,
) (*tenant.InvoiceAdjustmentControl, error) {
	if s.adjustmentCtrlRepo == nil {
		return nil, nil
	}

	control, err := s.adjustmentCtrlRepo.GetByOrgID(ctx, orgID)
	if errortypes.IsNotFoundError(err) {
		return nil, nil
	}

	return control, err
}

func (s *Service) refuseAutoPostingMemo(
	ctx context.Context,
	req *invoiceports.CreateMemoRequest,
	entity *invoice.Invoice,
) error {
	if !req.AutoPost {
		return nil
	}

	control, err := s.adjustmentApprovalControl(ctx, entity.OrganizationID)
	if err != nil {
		return err
	}
	if !memoNeedsApproval(control, entity) {
		return nil
	}

	return errortypes.NewValidationError(
		"autoPost",
		errortypes.ErrInvalidOperation,
		"This credit memo needs approval before it is posted. Save it as a draft for an approver to post.",
	)
}

func (s *Service) guardMemoPoster(
	ctx context.Context,
	entity *invoice.Invoice,
	actor *permissionports.RequestActor,
	tenantInfo pagination.TenantInfo,
) error {
	if actor == nil || actor.UserID.IsNil() {
		return nil
	}

	control, err := s.adjustmentApprovalControl(ctx, entity.OrganizationID)
	if err != nil {
		return err
	}
	if control == nil || !control.RequireSeparateApprover || !memoNeedsApproval(control, entity) {
		return nil
	}

	item, err := s.billingQueueRepo.GetByID(ctx, &billingqueueports.GetBillingQueueItemByIDRequest{
		ItemID:     entity.BillingQueueItemID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}
	if memoRaisedBy(item) != actor.UserID {
		return nil
	}

	return errortypes.NewValidationError(
		"invoiceId",
		errortypes.ErrInvalidOperation,
		"You raised this credit memo, so someone else must approve and post it",
	)
}

func memoRaisedBy(item *billingqueue.BillingQueueItem) pulid.ID {
	if item == nil {
		return pulid.Nil
	}
	raw, ok := item.AdjustmentContext[memoRaisedByKey].(string)
	if !ok {
		return pulid.Nil
	}

	return pulid.ID(raw)
}
