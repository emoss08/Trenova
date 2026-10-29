package invoiceadjustmentengine

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/app/billingnumber"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/identity/app/exchangeratestamp"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func refuseUnexecutable(adjustment *invoiceadjustment.InvoiceAdjustment) error {
	switch adjustment.Status { //nolint:exhaustive // every other status may run
	case invoiceadjustment.StatusDraft:
		return errortypes.NewValidationError(
			"adjustmentId",
			errortypes.ErrInvalidOperation,
			"Draft adjustments cannot be executed",
		)
	case invoiceadjustment.StatusRejected:
		return errortypes.NewValidationError(
			"adjustmentId",
			errortypes.ErrInvalidOperation,
			"Rejected adjustments cannot be executed",
		)
	default:
		return nil
	}
}

func (s *Engine) EnsureCorrectionGroup(
	ctx context.Context,
	entity *invoice.Invoice,
) (*invoiceadjustment.InvoiceAdjustmentCorrectionGroup, error) {
	rootID := entity.ID
	if entity.CorrectionGroupID.IsNotNil() {
		group, err := s.repo.GetCorrectionGroup(
			ctx,
			invoiceadjustmentports.GetCorrectionGroupRequest{
				ID: entity.CorrectionGroupID,
				TenantInfo: pagination.TenantInfo{
					OrgID: entity.OrganizationID,
					BuID:  entity.BusinessUnitID,
				},
			},
		)
		if err == nil && group != nil {
			return group, nil
		}
	}

	if group, err := s.repo.GetCorrectionGroupByRootInvoice(
		ctx,
		invoiceadjustmentports.GetCorrectionGroupByRootInvoiceRequest{
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

func (s *Engine) ExecuteApprovedAdjustment(
	ctx context.Context,
	adjustmentID pulid.ID,
	req *invoiceadjustmentports.InvoiceAdjustmentRequest,
	actor *permissionports.RequestActor,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	var tenantInfo pagination.TenantInfo
	if req != nil {
		tenantInfo = req.TenantInfo
	}
	adjustment, err := s.repo.GetByID(ctx, invoiceadjustmentports.GetInvoiceAdjustmentRequest{
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
	if err = refuseUnexecutable(adjustment); err != nil {
		return nil, err
	}

	waivers, err := detentionWaiversOf(adjustment)
	if err != nil {
		return nil, err
	}

	lockedInvoice, err := s.repo.LockInvoiceForUpdate(
		ctx,
		invoiceadjustmentports.LockInvoiceAdjustmentRequest{
			InvoiceID:  adjustment.OriginalInvoiceID,
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	// A rejection holds the same lock, so the status read before it may be
	// stale: read it again now that nobody else can change it.
	adjustment, err = s.repo.GetByID(ctx, invoiceadjustmentports.GetInvoiceAdjustmentRequest{
		ID:         adjustmentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if adjustment.Status == invoiceadjustment.StatusExecuted {
		return adjustment, nil
	}
	if err = refuseUnexecutable(adjustment); err != nil {
		return nil, err
	}

	computation, err := s.ComputePreview(
		ctx,
		s.executionRequest(adjustment, req, tenantInfo),
		adjustment.ID,
	)
	if err != nil {
		return nil, err
	}
	computation.Invoice = lockedInvoice
	if len(computation.Preview.Errors) > 0 {
		adjustment.Status = invoiceadjustment.StatusExecutionFailed
		adjustment.ExecutionError = PreviewErrorsToMultiError(computation.Preview.Errors).Error()
		s.LogAdjustmentEvent(
			"invoice adjustment execution failed revalidation",
			adjustment,
			zap.WarnLevel,
		)
		return s.repo.UpdateAdjustment(ctx, adjustment)
	}

	return s.execute(ctx, &adjustmentRun{
		adjustment:  adjustment,
		source:      lockedInvoice,
		computation: computation,
		waivers:     waivers,
		actor:       actor,
	})
}

func (s *Engine) executionRequest(
	adjustment *invoiceadjustment.InvoiceAdjustment,
	req *invoiceadjustmentports.InvoiceAdjustmentRequest,
	tenantInfo pagination.TenantInfo,
) *invoiceadjustmentports.InvoiceAdjustmentRequest {
	if req == nil || req.InvoiceID.IsNil() {
		req = &invoiceadjustmentports.InvoiceAdjustmentRequest{
			InvoiceID:      adjustment.OriginalInvoiceID,
			Kind:           adjustment.Kind,
			RebillStrategy: adjustment.RebillStrategy,
			Reason:         adjustment.Reason,
			IdempotencyKey: adjustment.IdempotencyKey,
			TenantInfo:     tenantInfo,
			Lines:          s.RequestLinesFromAdjustment(adjustment),
		}
	}
	req.TenantInfo = tenantInfo

	return req
}

func (s *Engine) createCreditMemoQueueItem(
	ctx context.Context,
	adjustment *invoiceadjustment.InvoiceAdjustment,
	sourceInvoice *invoice.Invoice,
	preview *invoiceadjustmentports.InvoiceAdjustmentPreview,
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

func (s *Engine) createCreditMemoInvoice(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	adjustment *invoiceadjustment.InvoiceAdjustment,
	sourceInvoice *invoice.Invoice,
	lines []*invoice.InvoiceLine,
	preview *invoiceadjustmentports.InvoiceAdjustmentPreview,
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

func (s *Engine) createReplacementQueueItem(
	ctx context.Context,
	adjustment *invoiceadjustment.InvoiceAdjustment,
	sourceInvoice *invoice.Invoice,
	group *invoiceadjustment.InvoiceAdjustmentCorrectionGroup,
	lines []*invoice.InvoiceLine,
	preview *invoiceadjustmentports.InvoiceAdjustmentPreview,
) (*billingqueue.BillingQueueItem, error) {
	number, err := s.numberer().Next(ctx, &billingnumber.Request{
		TenantInfo: pagination.TenantInfo{
			OrgID: adjustment.OrganizationID,
			BuID:  adjustment.BusinessUnitID,
		},
		BillType: billingqueue.BillTypeInvoice,
		PayerID:  sourceInvoice.CustomerID,
	})
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

func (s *Engine) SnapshotPayload(entity *invoice.Invoice) map[string]any {
	payload := jsonutils.MustToJSON(entity)
	payload["paymentSummary"] = map[string]any{
		"appliedAmount":     entity.AppliedAmount,
		"openBalanceAmount": entity.OpenBalanceAmount(),
		"settlementStatus":  entity.SettlementStatus,
		"disputeStatus":     entity.DisputeStatus,
	}
	return payload
}
