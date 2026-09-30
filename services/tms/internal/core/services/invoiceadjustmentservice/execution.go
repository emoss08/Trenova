package invoiceadjustmentservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/exchangeratestamp"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/typeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func (s *Service) ensureCorrectionGroup(
	ctx context.Context,
	entity *invoice.Invoice,
) (*invoiceadjustment.InvoiceAdjustmentCorrectionGroup, error) {
	rootID := entity.ID
	if entity.CorrectionGroupID.IsNotNil() {
		group, err := s.repo.GetCorrectionGroup(ctx, repositories.GetCorrectionGroupRequest{
			ID: entity.CorrectionGroupID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		})
		if err == nil && group != nil {
			return group, nil
		}
	}

	if group, err := s.repo.GetCorrectionGroupByRootInvoice(
		ctx,
		repositories.GetCorrectionGroupByRootInvoiceRequest{
			RootInvoiceID: rootID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		},
	); err == nil &&
		group != nil {
		return group, nil
	}

	return s.repo.CreateCorrectionGroup(ctx, &invoiceadjustment.InvoiceAdjustmentCorrectionGroup{
		OrganizationID:   entity.OrganizationID,
		BusinessUnitID:   entity.BusinessUnitID,
		RootInvoiceID:    rootID,
		CurrentInvoiceID: rootID,
	})
}

func (s *Service) executeApprovedAdjustment( //nolint:cyclop,funlen // legacy workflow
	ctx context.Context,
	adjustmentID pulid.ID,
	req *servicesports.InvoiceAdjustmentRequest,
	actor *servicesports.RequestActor,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	var tenantInfo pagination.TenantInfo
	if req != nil {
		tenantInfo = req.TenantInfo
	}
	adjustment, err := s.repo.GetByID(ctx, repositories.GetInvoiceAdjustmentRequest{
		ID:         adjustmentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if tenantInfo.OrgID.IsNil() {
		tenantInfo = pagination.TenantInfo{
			OrgID: adjustment.OrganizationID,
			BuID:  adjustment.BusinessUnitID,
		}
	}
	if adjustment.Status == invoiceadjustment.StatusExecuted {
		return adjustment, nil
	}
	if adjustment.Status == invoiceadjustment.StatusDraft {
		return nil, errortypes.NewValidationError(
			"adjustmentId",
			errortypes.ErrInvalidOperation,
			"Draft adjustments cannot be executed",
		)
	}
	if adjustment.Status == invoiceadjustment.StatusRejected {
		return nil, errortypes.NewValidationError(
			"adjustmentId",
			errortypes.ErrInvalidOperation,
			"Rejected adjustments cannot be executed",
		)
	}

	lockedInvoice, err := s.repo.LockInvoiceForUpdate(
		ctx,
		repositories.LockInvoiceAdjustmentRequest{
			InvoiceID:  adjustment.OriginalInvoiceID,
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	if req == nil || req.InvoiceID.IsNil() {
		req = &servicesports.InvoiceAdjustmentRequest{
			InvoiceID:      adjustment.OriginalInvoiceID,
			Kind:           adjustment.Kind,
			RebillStrategy: adjustment.RebillStrategy,
			Reason:         adjustment.Reason,
			IdempotencyKey: adjustment.IdempotencyKey,
			TenantInfo:     tenantInfo,
			Lines:          s.requestLinesFromAdjustment(adjustment),
		}
	}
	req.TenantInfo = tenantInfo

	computation, err := s.computePreview(ctx, req, adjustment.ID)
	if err != nil {
		return nil, err
	}
	computation.invoice = lockedInvoice
	if len(computation.preview.Errors) > 0 {
		adjustment.Status = invoiceadjustment.StatusExecutionFailed
		adjustment.ExecutionError = previewErrorsToMultiError(computation.preview.Errors).Error()
		s.logAdjustmentEvent(
			"invoice adjustment execution failed revalidation",
			adjustment,
			zap.WarnLevel,
		)
		return s.repo.UpdateAdjustment(ctx, adjustment)
	}

	group, err := s.ensureCorrectionGroup(ctx, lockedInvoice)
	if err != nil {
		return nil, err
	}
	if lockedInvoice.CorrectionGroupID.IsNil() {
		lockedInvoice.CorrectionGroupID = group.ID
		if _, err = s.invoiceRepo.Update(ctx, lockedInvoice); err != nil {
			return nil, err
		}
	}

	now := timeutils.NowUnix()
	if adjustment.ApprovalRequired {
		adjustment.ApprovedByID = actor.UserID
		adjustment.ApprovedAt = &now
		adjustment.ApprovalStatus = invoiceadjustment.ApprovalStatusApproved
	}
	adjustment.Status = invoiceadjustment.StatusExecuting
	adjustment.CorrectionGroupID = group.ID
	adjustment, err = s.repo.UpdateAdjustment(ctx, adjustment)
	if err != nil {
		return nil, err
	}

	creditMemoItem, err := s.createCreditMemoQueueItem(
		ctx,
		adjustment,
		lockedInvoice,
		computation.preview,
	)
	if err != nil {
		return nil, err
	}
	creditMemoInvoice, err := s.createCreditMemoInvoice(
		ctx,
		creditMemoItem,
		adjustment,
		lockedInvoice,
		computation.creditLineItems,
		computation.preview,
		now,
	)
	if err != nil {
		return nil, err
	}
	adjustment.CreditMemoInvoiceID = creditMemoInvoice.ID
	if err = s.postCreditMemoLedger(
		ctx,
		adjustment,
		creditMemoInvoice,
		lockedInvoice,
		actor,
	); err != nil {
		return nil, err
	}
	if err = servicesports.EnqueueAccountingSync(
		ctx,
		s.accountingSync,
		servicesports.InvoiceSyncRequest(
			creditMemoInvoice,
			accountingsync.SyncSourceAdjustmentCreditMemo,
		),
	); err != nil {
		return nil, err
	}

	var rebillQueueItem *billingqueue.BillingQueueItem
	var replacementInvoice *invoice.Invoice
	if adjustment.Kind == invoiceadjustment.KindCreditRebill {
		rebillQueueItem, err = s.createReplacementQueueItem(
			ctx,
			adjustment,
			lockedInvoice,
			group,
			computation.replacementLines,
			computation.preview,
		)
		if err != nil {
			return nil, err
		}
		replacementInvoice, err = s.createReplacementDraftInvoice(
			ctx,
			rebillQueueItem,
			adjustment,
			lockedInvoice,
			computation.replacementLines,
			computation.preview,
		)
		if err != nil {
			return nil, err
		}
		adjustment.ReplacementInvoiceID = replacementInvoice.ID
		adjustment.RebillQueueItemID = rebillQueueItem.ID
	}

	var writeOffJournalEntryID pulid.ID
	if adjustment.Kind == invoiceadjustment.KindWriteOff {
		writeOffJournalEntryID, err = s.createWriteOffJournalEntry(
			ctx,
			adjustment,
			lockedInvoice,
			computation.preview,
			actor,
		)
		if err != nil {
			return nil, err
		}
		if adjustment.Metadata == nil {
			adjustment.Metadata = make(map[string]any, 1)
		}
		adjustment.Metadata["writeOffJournalEntryId"] = writeOffJournalEntryID.String()
	}

	if computation.preview.RequiresReconciliationException {
		if _, err = s.db.DBForContext(ctx).
			NewInsert().
			Model(&invoiceadjustment.InvoiceAdjustmentReconciliationException{
				OrganizationID:      adjustment.OrganizationID,
				BusinessUnitID:      adjustment.BusinessUnitID,
				AdjustmentID:        adjustment.ID,
				InvoiceID:           lockedInvoice.ID,
				CreditMemoInvoiceID: creditMemoInvoice.ID,
				Status:              invoiceadjustment.ExceptionStatusOpen,
				Reason:              "Adjustment executed against settled or finance-sensitive invoice state",
				Amount:              computation.preview.CreditTotalAmount,
				Metadata: map[string]any{
					"settlementStatus": lockedInvoice.SettlementStatus,
					"disputeStatus":    lockedInvoice.DisputeStatus,
				},
			}).
			Exec(ctx); err != nil {
			return nil, fmt.Errorf("create reconciliation exception: %w", err)
		}
	}

	var rebillQueueItemID, replacementInvoiceID *string
	if rebillQueueItem != nil {
		rebillQueueItemID = typeutils.IDString(rebillQueueItem.ID)
	}
	if replacementInvoice != nil {
		replacementInvoiceID = typeutils.IDString(replacementInvoice.ID)
	}

	if _, err = s.db.DBForContext(ctx).
		NewInsert().
		Model(&invoiceadjustment.InvoiceAdjustmentSnapshot{
			OrganizationID: adjustment.OrganizationID,
			BusinessUnitID: adjustment.BusinessUnitID,
			AdjustmentID:   adjustment.ID,
			InvoiceID:      lockedInvoice.ID,
			Kind:           invoiceadjustment.SnapshotKindExecution,
			CreatedByID:    actor.UserID,
			Payload: map[string]any{
				"sourceInvoice":          s.snapshotPayload(lockedInvoice),
				"creditMemoId":           creditMemoInvoice.ID.String(),
				"rebillQueueItemId":      rebillQueueItemID,
				"replacementInvoiceId":   replacementInvoiceID,
				"writeOffJournalEntryId": typeutils.IDString(writeOffJournalEntryID),
			},
		}).
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("create execution snapshot: %w", err)
	}

	if adjustment.Kind == invoiceadjustment.KindFullReversal {
		if err = s.voidReversedInvoice(ctx, adjustment, lockedInvoice, actor, now); err != nil {
			return nil, err
		}
	} else if err = s.syncDetentionBilling(
		ctx,
		lockedInvoice,
		actor,
		creditMemoInvoice,
		replacementInvoice,
	); err != nil {
		return nil, err
	}

	adjustment.Status = invoiceadjustment.StatusExecuted
	adjustment.ExecutionError = ""
	adjustment.ReplacementReviewStatus = replacementReviewStatus(
		computation.preview.RequiresReplacementInvoiceReview,
	)
	updated, err := s.repo.UpdateAdjustment(ctx, adjustment)
	if err != nil {
		return nil, err
	}

	group.CurrentInvoiceID = lockedInvoice.ID
	if replacementInvoice != nil {
		group.CurrentInvoiceID = replacementInvoice.ID
	}
	if _, err = s.repo.UpdateCorrectionGroup(ctx, group); err != nil {
		return nil, err
	}

	s.logAudit(updated, actor, permission.OpUpdate, "Invoice adjustment executed")
	s.logAdjustmentEvent("invoice adjustment executed", updated, zap.InfoLevel)
	return updated, nil
}

func (s *Service) createCreditMemoQueueItem(
	ctx context.Context,
	adjustment *invoiceadjustment.InvoiceAdjustment,
	sourceInvoice *invoice.Invoice,
	preview *servicesports.InvoiceAdjustmentPreview,
) (*billingqueue.BillingQueueItem, error) {
	number, err := s.generator.GenerateCreditMemoNumber(
		ctx,
		adjustment.OrganizationID,
		adjustment.BusinessUnitID,
		"",
		"",
	)
	if err != nil {
		return nil, err
	}

	return s.billingQueueRepo.Create(ctx, &billingqueue.BillingQueueItem{
		OrganizationID:            adjustment.OrganizationID,
		BusinessUnitID:            adjustment.BusinessUnitID,
		ShipmentID:                sourceInvoice.ShipmentID,
		OrderID:                   sourceInvoice.OrderID,
		BillToCustomerID:          sourceInvoice.CustomerID,
		AllocatedTotalAmount:      sourceInvoice.TotalAmount.Abs(),
		AllocatedTotalAmountMinor: intutils.Abs(sourceInvoice.TotalAmountMinor),
		Number:                    number,
		Status:                    billingqueue.StatusPosted,
		BillType:                  billingqueue.BillTypeCreditMemo,
		IsAdjustmentOrigin:        true,
		SourceInvoiceID:           &sourceInvoice.ID,
		SourceInvoiceAdjustmentID: &adjustment.ID,
		CorrectionGroupID:         &adjustment.CorrectionGroupID,
		RequiresReplacementReview: preview.RequiresReplacementInvoiceReview,
		RerateVariancePercent:     decimal.Zero,
		AdjustmentContext: map[string]any{
			"kind": adjustment.Kind,
		},
	})
}

func (s *Service) createCreditMemoInvoice(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	adjustment *invoiceadjustment.InvoiceAdjustment,
	sourceInvoice *invoice.Invoice,
	lines []*invoice.InvoiceLine,
	preview *servicesports.InvoiceAdjustmentPreview,
	postedAt int64,
) (*invoice.Invoice, error) {
	subtotal := decimal.Zero
	other := decimal.Zero
	for _, line := range lines {
		if line == nil {
			continue
		}
		if line.Type == invoice.InvoiceLineTypeFreight {
			subtotal = subtotal.Add(line.Amount)
		} else {
			other = other.Add(line.Amount)
		}
	}

	entity := &invoice.Invoice{
		OrganizationID:            sourceInvoice.OrganizationID,
		BusinessUnitID:            sourceInvoice.BusinessUnitID,
		BillingQueueItemID:        item.ID,
		Scope:                     invoice.ScopeAdjustment,
		ShipmentID:                sourceInvoice.ShipmentID,
		OrderID:                   sourceInvoice.OrderID,
		OrderNumber:               sourceInvoice.OrderNumber,
		CustomerID:                sourceInvoice.CustomerID,
		Number:                    item.Number,
		BillType:                  billingqueue.BillTypeCreditMemo,
		Status:                    invoice.StatusPosted,
		PaymentTerm:               sourceInvoice.PaymentTerm,
		CurrencyCode:              sourceInvoice.CurrencyCode,
		InvoiceDate:               preview.AccountingDate,
		DueDate:                   &preview.AccountingDate,
		PostedAt:                  &postedAt,
		ShipmentProNumber:         sourceInvoice.ShipmentProNumber,
		ShipmentBOL:               sourceInvoice.ShipmentBOL,
		ServiceDate:               sourceInvoice.ServiceDate,
		BillToName:                sourceInvoice.BillToName,
		BillToCode:                sourceInvoice.BillToCode,
		BillToAddressLine1:        sourceInvoice.BillToAddressLine1,
		BillToAddressLine2:        sourceInvoice.BillToAddressLine2,
		BillToCity:                sourceInvoice.BillToCity,
		BillToState:               sourceInvoice.BillToState,
		BillToPostalCode:          sourceInvoice.BillToPostalCode,
		BillToCountry:             sourceInvoice.BillToCountry,
		ShipperCustomerID:         sourceInvoice.ShipperCustomerID,
		IsSplitBill:               sourceInvoice.IsSplitBill,
		SubtotalAmount:            subtotal,
		OtherAmount:               other,
		TotalAmount:               preview.CreditTotalAmount.Neg(),
		AppliedAmount:             decimal.Zero,
		SettlementStatus:          invoice.SettlementStatusUnpaid,
		DisputeStatus:             invoice.DisputeStatusNone,
		CorrectionGroupID:         adjustment.CorrectionGroupID,
		SourceInvoiceAdjustmentID: adjustment.ID,
		IsAdjustmentArtifact:      true,
		Lines:                     lines,
	}
	if err := s.stamper.StampInto(ctx, &exchangeratestamp.Request{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		CurrencyCode:   entity.CurrencyCode,
		DocumentDate:   entity.InvoiceDate,
		AccountingDate: postedAt,
	}, &entity.ExchangeRate, &entity.ExchangeRateDate); err != nil {
		return nil, err
	}

	return s.invoiceRepo.Create(ctx, entity)
}

func (s *Service) createReplacementQueueItem(
	ctx context.Context,
	adjustment *invoiceadjustment.InvoiceAdjustment,
	sourceInvoice *invoice.Invoice,
	group *invoiceadjustment.InvoiceAdjustmentCorrectionGroup,
	lines []*invoice.InvoiceLine,
	preview *servicesports.InvoiceAdjustmentPreview,
) (*billingqueue.BillingQueueItem, error) {
	number, err := s.generator.GenerateInvoiceNumber(
		ctx,
		adjustment.OrganizationID,
		adjustment.BusinessUnitID,
		"",
		"",
	)
	if err != nil {
		return nil, err
	}

	return s.billingQueueRepo.Create(ctx, &billingqueue.BillingQueueItem{
		OrganizationID:            adjustment.OrganizationID,
		BusinessUnitID:            adjustment.BusinessUnitID,
		ShipmentID:                sourceInvoice.ShipmentID,
		OrderID:                   sourceInvoice.OrderID,
		BillToCustomerID:          sourceInvoice.CustomerID,
		AllocatedTotalAmount:      preview.RebillTotalAmount.Abs(),
		AllocatedTotalAmountMinor: money.MinorUnits(preview.RebillTotalAmount.Abs()),
		Number:                    number,
		Status:                    replacementQueueStatus(preview.RequiresReplacementInvoiceReview),
		BillType:                  billingqueue.BillTypeInvoice,
		IsAdjustmentOrigin:        true,
		SourceInvoiceID:           &sourceInvoice.ID,
		SourceInvoiceAdjustmentID: &adjustment.ID,
		SourceCreditMemoInvoiceID: &adjustment.CreditMemoInvoiceID,
		CorrectionGroupID:         &group.ID,
		RebillStrategy:            string(adjustment.RebillStrategy),
		RequiresReplacementReview: preview.RequiresReplacementInvoiceReview,
		RerateVariancePercent:     preview.RerateVariancePercent,
		AdjustmentContext: map[string]any{
			"replacementLines":   lines,
			"subtotalAmount":     sumInvoiceLines(lines, invoice.InvoiceLineTypeFreight),
			"otherAmount":        sumInvoiceLines(lines, invoice.InvoiceLineTypeAccessorial),
			"totalAmount":        preview.RebillTotalAmount,
			"accountingDate":     preview.AccountingDate,
			"sourceInvoiceId":    sourceInvoice.ID,
			"correctionGroupId":  group.ID,
			"sourceAdjustmentId": adjustment.ID,
		},
	})
}

func (s *Service) snapshotPayload(entity *invoice.Invoice) map[string]any {
	payload := jsonutils.MustToJSON(entity)
	payload["paymentSummary"] = map[string]any{
		"appliedAmount":     entity.AppliedAmount,
		"openBalanceAmount": entity.OpenBalanceAmount(),
		"settlementStatus":  entity.SettlementStatus,
		"disputeStatus":     entity.DisputeStatus,
	}
	return payload
}
