package invoiceadjustmentengine

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/typeutils"
	"go.uber.org/zap"
)

type adjustmentRun struct {
	adjustment        *invoiceadjustment.InvoiceAdjustment
	source            *invoice.Invoice
	computation       *PreviewComputation
	waivers           []detentionports.DetentionWaiver
	actor             *permissionports.RequestActor
	now               int64
	group             *invoiceadjustment.InvoiceAdjustmentCorrectionGroup
	creditMemo        *invoice.Invoice
	rebillQueueItem   *billingqueue.BillingQueueItem
	replacement       *invoice.Invoice
	writeOffJournalID pulid.ID
}

func (s *Engine) execute(
	ctx context.Context,
	run *adjustmentRun,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	if err := s.startExecution(ctx, run); err != nil {
		return nil, err
	}
	if err := s.issueCreditMemo(ctx, run); err != nil {
		return nil, err
	}
	if err := s.issueReplacement(ctx, run); err != nil {
		return nil, err
	}
	if err := s.recordWriteOff(ctx, run); err != nil {
		return nil, err
	}
	if err := s.recordReconciliationException(ctx, run); err != nil {
		return nil, err
	}
	if err := s.recordExecutionSnapshot(ctx, run); err != nil {
		return nil, err
	}
	if err := s.settleSource(ctx, run); err != nil {
		return nil, err
	}

	return s.finishExecution(ctx, run)
}

func (s *Engine) startExecution(ctx context.Context, run *adjustmentRun) error {
	group, err := s.EnsureCorrectionGroup(ctx, run.source)
	if err != nil {
		return err
	}
	run.group = group
	if run.source.CorrectionGroupID.IsNil() {
		run.source.CorrectionGroupID = group.ID
		if _, err = s.invoiceRepo.Update(ctx, run.source); err != nil {
			return err
		}
	}

	run.now = timeutils.NowUnix()
	adjustment := run.adjustment
	if adjustment.ApprovalRequired {
		adjustment.ApprovedByID = run.actor.UserID
		adjustment.ApprovedAt = &run.now
		adjustment.ApprovalStatus = invoiceadjustment.ApprovalStatusApproved
	}
	adjustment.Status = invoiceadjustment.StatusExecuting
	adjustment.CorrectionGroupID = group.ID
	run.adjustment, err = s.repo.UpdateAdjustment(ctx, adjustment)

	return err
}

func (s *Engine) issueCreditMemo(ctx context.Context, run *adjustmentRun) error {
	preview := run.computation.Preview
	creditMemoItem, err := s.createCreditMemoQueueItem(ctx, run.adjustment, run.source, preview)
	if err != nil {
		return err
	}
	creditMemo, err := s.createCreditMemoInvoice(
		ctx,
		creditMemoItem,
		run.adjustment,
		run.source,
		run.computation.creditLineItems,
		preview,
		run.now,
	)
	if err != nil {
		return err
	}
	run.creditMemo = creditMemo
	run.adjustment.CreditMemoInvoiceID = creditMemo.ID
	if err = s.postCreditMemoLedger(
		ctx,
		run.adjustment,
		creditMemo,
		run.source,
		run.actor,
	); err != nil {
		return err
	}
	if err = accountingsyncports.EnqueueAccountingSync(
		ctx,
		s.accountingSync,
		accountingsyncports.InvoiceSyncRequest(
			creditMemo,
			accountingsync.SyncSourceAdjustmentCreditMemo,
		),
	); err != nil {
		return err
	}
	source, err := s.applyCreditToSource(ctx, run.adjustment, creditMemo, run.source, run.actor)
	if err != nil {
		return err
	}
	run.source = source

	return nil
}

func (s *Engine) issueReplacement(ctx context.Context, run *adjustmentRun) error {
	if run.adjustment.Kind != invoiceadjustment.KindCreditRebill {
		return nil
	}
	rebillQueueItem, err := s.createReplacementQueueItem(
		ctx,
		run.adjustment,
		run.source,
		run.group,
		run.computation.replacementLines,
		run.computation.Preview,
	)
	if err != nil {
		return err
	}
	run.rebillQueueItem = rebillQueueItem
	replacement, err := s.createReplacementDraftInvoice(
		ctx,
		rebillQueueItem,
		run.adjustment,
		run.source,
		run.computation.replacementLines,
		run.computation.Preview,
	)
	if err != nil {
		return err
	}
	run.replacement = replacement
	run.adjustment.ReplacementInvoiceID = replacement.ID
	run.adjustment.RebillQueueItemID = rebillQueueItem.ID

	return nil
}

func (s *Engine) recordWriteOff(ctx context.Context, run *adjustmentRun) error {
	adjustment := run.adjustment
	if adjustment.Kind != invoiceadjustment.KindWriteOff {
		return nil
	}
	journalID, err := s.createWriteOffJournalEntry(
		ctx,
		adjustment,
		run.source,
		run.computation.Preview,
		run.actor,
	)
	if err != nil {
		return err
	}
	run.writeOffJournalID = journalID
	if adjustment.Metadata == nil {
		adjustment.Metadata = make(map[string]any, 1)
	}
	adjustment.Metadata["writeOffJournalEntryId"] = journalID.String()

	return nil
}

func (s *Engine) recordReconciliationException(ctx context.Context, run *adjustmentRun) error {
	if !run.computation.Preview.RequiresReconciliationException {
		return nil
	}
	adjustment := run.adjustment
	if err := s.repo.CreateAdjustmentArtifacts(
		ctx,
		invoiceadjustmentports.CreateAdjustmentArtifactsParams{
			ReconciliationExceptions: []*invoiceadjustment.InvoiceAdjustmentReconciliationException{
				{
					OrganizationID:      adjustment.OrganizationID,
					BusinessUnitID:      adjustment.BusinessUnitID,
					AdjustmentID:        adjustment.ID,
					InvoiceID:           run.source.ID,
					CreditMemoInvoiceID: run.creditMemo.ID,
					Status:              invoiceadjustment.ExceptionStatusOpen,
					Reason:              "Adjustment executed against settled or finance-sensitive invoice state",
					Amount:              run.computation.Preview.CreditTotalAmount,
					Metadata: map[string]any{
						"settlementStatus": run.source.SettlementStatus,
						"disputeStatus":    run.source.DisputeStatus,
					},
				},
			},
		},
	); err != nil {
		return fmt.Errorf("create reconciliation exception: %w", err)
	}

	return nil
}

func (s *Engine) recordExecutionSnapshot(ctx context.Context, run *adjustmentRun) error {
	var rebillQueueItemID, replacementInvoiceID *string
	if run.rebillQueueItem != nil {
		rebillQueueItemID = typeutils.IDString(run.rebillQueueItem.ID)
	}
	if run.replacement != nil {
		replacementInvoiceID = typeutils.IDString(run.replacement.ID)
	}

	adjustment := run.adjustment
	if err := s.repo.CreateAdjustmentArtifacts(
		ctx,
		invoiceadjustmentports.CreateAdjustmentArtifactsParams{
			Snapshots: []*invoiceadjustment.InvoiceAdjustmentSnapshot{{
				OrganizationID: adjustment.OrganizationID,
				BusinessUnitID: adjustment.BusinessUnitID,
				AdjustmentID:   adjustment.ID,
				InvoiceID:      run.source.ID,
				Kind:           invoiceadjustment.SnapshotKindExecution,
				CreatedByID:    run.actor.UserID,
				Payload: map[string]any{
					"sourceInvoice":          s.SnapshotPayload(run.source),
					"creditMemoId":           run.creditMemo.ID.String(),
					"rebillQueueItemId":      rebillQueueItemID,
					"replacementInvoiceId":   replacementInvoiceID,
					"writeOffJournalEntryId": typeutils.IDString(run.writeOffJournalID),
				},
			}},
		},
	); err != nil {
		return fmt.Errorf("create execution snapshot: %w", err)
	}

	return nil
}

func (s *Engine) settleSource(ctx context.Context, run *adjustmentRun) error {
	if run.adjustment.Kind == invoiceadjustment.KindFullReversal {
		return s.voidReversedInvoice(ctx, run.adjustment, run.source, run.actor, run.now)
	}

	return s.syncDetentionBilling(ctx, &detentionSync{
		source:    run.source,
		actor:     run.actor,
		waivers:   run.waivers,
		documents: []*invoice.Invoice{run.creditMemo, run.replacement},
	})
}

func (s *Engine) finishExecution(
	ctx context.Context,
	run *adjustmentRun,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	adjustment := run.adjustment
	adjustment.Status = invoiceadjustment.StatusExecuted
	adjustment.ExecutionError = ""
	adjustment.ReplacementReviewStatus = ReplacementReviewStatus(
		run.computation.Preview.RequiresReplacementInvoiceReview,
	)
	updated, err := s.repo.UpdateAdjustment(ctx, adjustment)
	if err != nil {
		return nil, err
	}

	run.group.CurrentInvoiceID = run.source.ID
	if run.replacement != nil {
		run.group.CurrentInvoiceID = run.replacement.ID
	}
	if _, err = s.repo.UpdateCorrectionGroup(ctx, run.group); err != nil {
		return nil, err
	}

	s.LogAudit(updated, run.actor, permission.OpUpdate, "Invoice adjustment executed")
	s.PublishAdjustedInvoices(ctx, run.actor, run.source, run.creditMemo, run.replacement)
	s.LogAdjustmentEvent("invoice adjustment executed", updated, zap.InfoLevel)
	return updated, nil
}
