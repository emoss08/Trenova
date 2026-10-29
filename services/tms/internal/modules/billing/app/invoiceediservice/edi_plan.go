package invoiceediservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

func (s *Service) planEDISend(
	ctx context.Context,
	req *invoiceports.SendInvoiceEDIRequest,
	actor *permissionports.RequestActor,
) (*invoice.Invoice, *invoiceports.InvoiceEDISendPlan, error) {
	if req == nil {
		return nil, nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}
	if actor == nil {
		return nil, nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrRequired,
			"Actor is required",
		)
	}
	entity, err := s.repo.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         req.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, nil, err
	}
	if entity.Status != invoice.StatusPosted {
		return nil, nil, errortypes.NewValidationError(
			"invoiceId",
			errortypes.ErrInvalidOperation,
			"Only a posted invoice can be sent by EDI",
		)
	}
	plans, err := s.ResolveEDISendPlans(ctx, &invoiceports.ResolveInvoiceEDISendPlansRequest{
		TenantInfo: req.TenantInfo,
		Invoices:   []*invoice.Invoice{entity},
	})
	if err != nil {
		return nil, nil, err
	}
	plan := plans[entity.ID]
	if plan == nil || !plan.Enabled || len(plan.Blockers) > 0 {
		blocker := ediBlockerProfileDisabled
		if plan != nil && len(plan.Blockers) > 0 {
			blocker = plan.Blockers[0]
		}
		return nil, nil, errortypes.NewValidationError(
			"invoiceId",
			errortypes.ErrInvalidOperation,
			blocker,
		)
	}
	if !req.Force {
		switch entity.EDISendStatus {
		case invoice.EDISendStatusQueued, invoice.EDISendStatusSending:
			return nil, nil, errortypes.NewValidationError(
				"invoiceId",
				errortypes.ErrInvalidOperation,
				"An EDI send for this invoice is already in progress",
			)
		case invoice.EDISendStatusSent:
			return nil, nil, errortypes.NewValidationError(
				"invoiceId",
				errortypes.ErrInvalidOperation,
				"This invoice has already been sent by EDI; resend it with force",
			)
		case invoice.EDISendStatusNotSent,
			invoice.EDISendStatusNotConfigured,
			invoice.EDISendStatusGenerated,
			invoice.EDISendStatusFailed,
			invoice.EDISendStatusDeadLettered:
		}
	}

	return entity, plan, nil
}

func (s *Service) PreviewSendEDI(
	ctx context.Context,
	req *invoiceports.SendInvoiceEDIRequest,
	actor *permissionports.RequestActor,
) (*invoiceports.InvoiceEDISendPreview, error) {
	entity, plan, err := s.planEDISend(ctx, req, actor)
	if err != nil {
		return nil, err
	}
	if s.workflowStarter == nil || !s.workflowStarter.Enabled() {
		return nil, errortypes.NewBusinessError(
			"Invoice EDI delivery requires workflow processing to be enabled",
		).WithInternal(kernelports.ErrWorkflowStarterDisabled)
	}

	return &invoiceports.InvoiceEDISendPreview{Invoice: entity, Plan: plan}, nil
}

// ediPlanFor is the EDI half of a send plan. A failure to resolve it must not
// stop the email plan, so it degrades to an unconfigured plan with the error.
func (s *Service) PlanFor(
	ctx context.Context,
	entity *invoice.Invoice,
	tenantInfo pagination.TenantInfo,
) *invoiceports.InvoiceEDISendPlan {
	plans, err := s.ResolveEDISendPlans(ctx, &invoiceports.ResolveInvoiceEDISendPlansRequest{
		TenantInfo: tenantInfo,
		Invoices:   []*invoice.Invoice{entity},
	})
	if err != nil {
		s.l.Warn("failed to resolve invoice EDI plan", zap.Error(err))
		return &invoiceports.InvoiceEDISendPlan{
			InvoiceID: entity.ID,
			Status:    entity.EDISendStatus,
			Blockers:  []string{"EDI configuration could not be read: " + err.Error()},
		}
	}
	if plan := plans[entity.ID]; plan != nil {
		return plan
	}

	return &invoiceports.InvoiceEDISendPlan{InvoiceID: entity.ID, Status: entity.EDISendStatus}
}
