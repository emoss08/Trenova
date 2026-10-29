package invoicepostingservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingcontrolpolicyservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceaction"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicevalidation"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/app/accountingcontrolpolicyservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/jobs/billingjobs/billingcontract"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

//nolint:nestif // existing validation flow mirrors business rule nesting
func (s *Service) Post( //nolint:funlen // legacy workflow
	ctx context.Context,
	req *invoiceports.PostInvoiceRequest,
	actor *permissionports.RequestActor,
) (*invoice.Invoice, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}

	var posted *invoice.Invoice
	newlyPosted := false
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		entity, getErr := s.repo.GetByID(txCtx, invoiceports.GetInvoiceByIDRequest{
			ID:         req.InvoiceID,
			TenantInfo: req.TenantInfo,
		})
		if getErr != nil {
			return getErr
		}

		auditActor := actor.AuditActor()
		previous := *entity
		now := timeutils.NowUnix()

		alreadyPosted, planErr := s.planPost(txCtx, entity, req, now)
		if planErr != nil {
			return planErr
		}
		if !alreadyPosted {
			if guardErr := s.guardPoster(txCtx, req, actor); guardErr != nil {
				return guardErr
			}
			if guardErr := s.guardMemoPoster(
				txCtx,
				entity,
				actor,
				req.TenantInfo,
			); guardErr != nil {
				return guardErr
			}
		}

		if alreadyPosted {
			queueUpdate, err := s.markBillingQueueItemPosted(txCtx, entity, req.TenantInfo)
			if err != nil {
				return err
			}

			s.logBillingQueuePosted(queueUpdate, auditActor)
			s.rec.PublishBillingQueueInvalidation(txCtx, queueUpdate, auditActor)

			posted = entity
			return nil
		}

		entity.PDFDocumentID = pulid.Nil
		entity.PDFDocument = nil
		updated, updateErr := s.repo.Update(txCtx, entity)
		if updateErr != nil {
			return updateErr
		}
		newlyPosted = true

		legs, shipErr := s.markInvoicedLegs(txCtx, entity, now, req.TenantInfo)
		if shipErr != nil {
			return shipErr
		}

		if derivErr := s.recomputeOrdersForLegs(
			txCtx,
			req.TenantInfo,
			entity.OrderID,
			legs,
		); derivErr != nil {
			return derivErr
		}

		queueUpdate, updateErr := s.markBillingQueueItemPosted(txCtx, updated, req.TenantInfo)
		if updateErr != nil {
			return updateErr
		}

		if postErr := s.createInvoiceJournalPosting(txCtx, updated, actor); postErr != nil {
			return postErr
		}

		if syncErr := accountingsyncports.EnqueueAccountingSync(
			txCtx,
			s.accountingSync,
			accountingsyncports.InvoiceSyncRequest(
				updated,
				accountingsync.PostedSourceEvent(updated.BillType),
			),
		); syncErr != nil {
			return syncErr
		}

		posted = updated

		s.rec.LogAction(
			updated,
			auditActor,
			permission.OpUpdate,
			&previous,
			updated,
			"Invoice posted",
		)
		s.rec.PublishInvalidation(txCtx, updated, auditActor, "updated", updated)
		s.logBillingQueuePosted(queueUpdate, auditActor)
		s.rec.PublishBillingQueueInvalidation(txCtx, queueUpdate, auditActor)
		s.notifyReconciliationWarning(txCtx, updated, legs)

		return nil
	})
	if err != nil {
		return nil, err
	}
	s.edi.EnqueueAfterPost(ctx, posted, req.TenantInfo, actor)
	if newlyPosted {
		s.generatePDFAfterPost(ctx, posted, req.TenantInfo, actor)
	}

	return posted, nil
}

func (s *Service) generatePDFAfterPost(
	ctx context.Context,
	entity *invoice.Invoice,
	tenantInfo pagination.TenantInfo,
	actor *permissionports.RequestActor,
) {
	if entity == nil || s.workflowStarter == nil || !s.workflowStarter.Enabled() {
		return
	}
	if _, err := s.pdf.GeneratePDF(ctx, &invoiceports.InvoicePreviewRequest{
		InvoiceID:  entity.ID,
		TenantInfo: tenantInfo,
	}, actor); err != nil {
		s.l.Warn(
			"failed to start invoice PDF generation after post",
			zap.String("invoiceId", entity.ID.String()),
			zap.Error(err),
		)
	}
}

func (s *Service) notifyReconciliationWarning(
	ctx context.Context,
	entity *invoice.Invoice,
	legs []*shipment.Shipment,
) {
	if s.accountingRepo == nil || s.notificationService == nil || entity == nil || len(legs) == 0 {
		return
	}

	control, err := s.accountingRepo.GetByOrgID(ctx, entity.OrganizationID)
	if err != nil || control == nil {
		return
	}

	if control.ReconciliationMode != accountingcontrol.ReconciliationModeWarnOnly ||
		!control.NotifyOnReconciliationException {
		return
	}

	expectedTotal, err := invoicevalidation.ReconciliationExpectedTotal(ctx, entity, legs)
	if err != nil {
		s.l.Warn("failed to compute the reconciliation total", zap.Error(err))
		return
	}
	discrepancy := entity.TotalAmount.Sub(expectedTotal).Abs()
	if !discrepancy.GreaterThan(control.ReconciliationToleranceAmount) {
		return
	}

	if _, err = s.notificationService.Create(ctx, &notification.Notification{
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: &entity.BusinessUnitID,
		EventType:      "invoice_reconciliation_warning",
		Priority:       notification.PriorityMedium,
		Channel:        notification.ChannelGlobal,
		Title:          "Invoice posted with reconciliation warning",
		Message:        "A posted invoice exceeded the organization reconciliation tolerance and requires follow-up.",
		Data: map[string]any{
			"invoiceTotal":      entity.TotalAmount.String(),
			"expectedTotal":     expectedTotal.String(),
			"toleranceAmount":   control.ReconciliationToleranceAmount.String(),
			"discrepancyAmount": discrepancy.String(),
		},
		RelatedEntities: invoicevalidation.ReconciliationRelatedEntities(entity),
		Source:          "invoiceservice.Post",
	}); err != nil {
		s.l.Warn("failed to create reconciliation warning notification", zap.Error(err))
	}
}

// recomputeOrdersForLegs re-derives every order touched by a posting: the invoice's
// own order (grouped path) plus each leg's parent order (single-shipment path). The
// dedupe matters because all grouped-invoice legs share one order.
func (s *Service) recomputeOrdersForLegs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	invoiceOrderID pulid.ID,
	legs []*shipment.Shipment,
) error {
	if s.orderDerivation == nil {
		return nil
	}

	orderIDs := make(map[pulid.ID]struct{}, 1+len(legs))
	if !invoiceOrderID.IsNil() {
		orderIDs[invoiceOrderID] = struct{}{}
	}
	for _, leg := range legs {
		if leg == nil || leg.OrderID.IsNil() {
			continue
		}
		orderIDs[leg.OrderID] = struct{}{}
	}

	for orderID := range orderIDs {
		if err := s.orderDerivation.RecomputeOrder(ctx, tenantInfo, orderID); err != nil {
			return err
		}
	}

	return nil
}

// markInvoicedLegs flips every shipment billed by this invoice to Invoiced. For a
// single-shipment invoice that is one leg; for a grouped (order) invoice it is the set
// of legs carried on the invoice lines. It returns the legs for downstream
// reconciliation notification.
//
// A split-billed leg is invoiced only once every payer's share has posted: while
// another payer's item is still open the leg keeps its status and only records
// that it has been billed.
func (s *Service) markInvoicedLegs(
	ctx context.Context,
	entity *invoice.Invoice,
	now int64,
	tenantInfo pagination.TenantInfo,
) ([]*shipment.Shipment, error) {
	changes, err := s.planInvoicedLegs(ctx, entity, now, tenantInfo)
	if err != nil {
		return nil, err
	}

	legs := make([]*shipment.Shipment, 0, len(changes))
	for _, change := range changes {
		if _, err = s.shipmentRepo.UpdateDerivedState(ctx, change.After); err != nil {
			return nil, err
		}
		legs = append(legs, change.After)
	}

	return legs, nil
}

// planInvoicedLegs loads the legs the invoice bills and marks each as posting
// leaves it, without saving: billed now, and Invoiced unless another payer's
// item on it is still open.
func (s *Service) planInvoicedLegs(
	ctx context.Context,
	entity *invoice.Invoice,
	now int64,
	tenantInfo pagination.TenantInfo,
) ([]LegChange, error) {
	legs, err := invoicevalidation.LoadInvoiceLegs(ctx, s.shipmentRepo, entity, tenantInfo, true)
	if err != nil {
		return nil, err
	}

	stillOpen, err := s.legsWithOpenSiblingItems(ctx, entity, legs, tenantInfo)
	if err != nil {
		return nil, err
	}

	changes := make([]LegChange, 0, len(legs))
	for _, shp := range legs {
		before := *shp
		shp.BilledAt = &now
		shp.RateLocked = true
		if _, open := stillOpen[shp.ID]; !open {
			shp.Status = shipmentvocab.StatusInvoiced
		}
		changes = append(changes, LegChange{Before: &before, After: shp})
	}

	return changes, nil
}

// legsWithOpenSiblingItems reports which legs still have an invoice item that
// belongs to another payer and has not posted. Items this invoice bills are not
// siblings: they post in the same transaction, a moment after this runs.
func (s *Service) legsWithOpenSiblingItems(
	ctx context.Context,
	entity *invoice.Invoice,
	legs []*shipment.Shipment,
	tenantInfo pagination.TenantInfo,
) (map[pulid.ID]struct{}, error) {
	open := make(map[pulid.ID]struct{})
	if len(legs) == 0 {
		return open, nil
	}

	legIDs := make([]pulid.ID, 0, len(legs))
	for _, shp := range legs {
		legIDs = append(legIDs, shp.ID)
	}
	active, err := s.billingQueueRepo.ListActiveInvoiceItemsByShipmentIDs(
		ctx,
		&billingqueueports.ListActiveInvoiceItemsRequest{
			TenantInfo:  tenantInfo,
			ShipmentIDs: legIDs,
		},
	)
	if err != nil {
		return nil, err
	}

	for shipmentID, items := range active {
		for _, item := range items {
			if item == nil || item.ID == entity.BillingQueueItemID ||
				item.InvoiceID == entity.ID {
				continue
			}
			open[shipmentID] = struct{}{}
			break
		}
	}

	return open, nil
}

func (s *Service) markBillingQueueItemPosted(
	ctx context.Context,
	entity *invoice.Invoice,
	tenantInfo pagination.TenantInfo,
) (*invoiceaction.PostedBillingQueueResult, error) {
	item, err := s.billingQueueRepo.GetByID(ctx, &billingqueueports.GetBillingQueueItemByIDRequest{
		ItemID:     entity.BillingQueueItemID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if item.Status == billingqueue.StatusPosted {
		return &invoiceaction.PostedBillingQueueResult{
			Updated: item,
		}, nil
	}

	previous := *item
	item.Status = billingqueue.StatusPosted

	updated, err := s.billingQueueRepo.Update(ctx, item)
	if err != nil {
		return nil, err
	}

	// Settle every sibling billing-queue item this invoice bills so none linger in
	// Approved, without touching items that belong to other still-draft invoices.
	// The order and shipment ids are only a fallback for rows written before the
	// invoice back-link existed.
	if _, sweepErr := s.billingQueueRepo.MarkPostedForInvoice(
		ctx,
		&billingqueueports.MarkPostedForInvoiceRequest{
			TenantInfo:  tenantInfo,
			InvoiceID:   entity.ID,
			OrderID:     entity.OrderID,
			ShipmentIDs: entity.LegShipmentIDs(),
			PayerID:     entity.CustomerID,
		},
	); sweepErr != nil {
		return nil, sweepErr
	}

	return &invoiceaction.PostedBillingQueueResult{
		Previous: &previous,
		Updated:  updated,
	}, nil
}

func (s *Service) EnqueueAutoPost(
	ctx context.Context,
	entity *invoice.Invoice,
	actor *permissionports.RequestActor,
) error {
	if entity == nil {
		return errortypes.NewValidationError(
			"invoice",
			errortypes.ErrRequired,
			"Invoice is required",
		)
	}

	if !s.workflowStarter.Enabled() {
		return kernelports.ErrWorkflowStarterDisabled
	}

	auditActor := actor.AuditActor()
	payload := &billingcontract.AutoPostInvoicePayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: entity.OrganizationID,
			BusinessUnitID: entity.BusinessUnitID,
			UserID:         auditActor.UserID,
			Timestamp:      timeutils.NowUnix(),
		},
		InvoiceID:     entity.ID,
		PrincipalType: auditActor.PrincipalType,
		PrincipalID:   auditActor.PrincipalID,
		APIKeyID:      auditActor.APIKeyID,
	}

	workflowID := fmt.Sprintf(
		"invoice-auto-post-%s-%s-%s",
		entity.OrganizationID.String(),
		entity.BusinessUnitID.String(),
		entity.ID.String(),
	)

	_, err := s.workflowStarter.StartWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:            workflowID,
			TaskQueue:     temporaltype.TaskQueueBilling.String(),
			StaticSummary: "Auto-post invoice " + entity.Number,
		},
		billingcontract.AutoPostInvoiceWorkflowName,
		payload,
	)
	return err
}

func (s *Service) accountingPolicyService() *accountingcontrolpolicyservice.Service {
	if s.accountingPolicy != nil {
		return s.accountingPolicy
	}
	return accountingcontrolpolicyservice.New(
		accountingcontrolpolicyservice.Params{Logger: zap.NewNop()},
	)
}

func (s *Service) billingPolicyService() *billingcontrolpolicyservice.Service {
	if s.billingPolicy != nil {
		return s.billingPolicy
	}
	return billingcontrolpolicyservice.New(billingcontrolpolicyservice.Params{Logger: zap.NewNop()})
}

func (s *Service) logBillingQueuePosted(
	result *invoiceaction.PostedBillingQueueResult,
	actor kernelports.AuditActor,
) {
	if result == nil || result.Previous == nil || result.Updated == nil {
		return
	}

	if err := s.auditService.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceBillingQueue,
		ResourceID:     result.Updated.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		OrganizationID: result.Updated.OrganizationID,
		BusinessUnitID: result.Updated.BusinessUnitID,
		CurrentState:   jsonutils.MustToJSON(result.Updated),
		PreviousState:  jsonutils.MustToJSON(result.Previous),
	},
		auditservice.WithComment("Billing queue item marked posted from invoice"),
		auditservice.WithDiff(result.Previous, result.Updated),
	); err != nil {
		s.l.Error("failed to log billing queue invoice posting", zap.Error(err))
	}
}
