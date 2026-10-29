package invoiceadjustmentengine

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/app/invoicelines"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/invoiceadjustmentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/chargeallocation"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff/effectiveversioncache"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetablecache"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func resolvePreviewLineValues(req previewLineValuesRequest) previewLineValues {
	values := previewLineValues{
		creditAmount:   req.eligible,
		creditQuantity: req.sourceLine.Quantity,
		rebillAmount:   decimal.Zero,
		rebillQuantity: decimal.Zero,
		description:    req.sourceLine.Description,
		payload:        map[string]any{},
	}

	if req.input == nil || req.kind == invoiceadjustment.KindFullReversal {
		return values
	}

	if !req.input.CreditAmount.IsZero() {
		values.creditAmount = req.input.CreditAmount
	}
	if !req.input.CreditQuantity.IsZero() {
		values.creditQuantity = req.input.CreditQuantity
	}
	if !req.input.RebillAmount.IsZero() {
		values.rebillAmount = req.input.RebillAmount
	}
	if !req.input.RebillQuantity.IsZero() {
		values.rebillQuantity = req.input.RebillQuantity
	}
	if strings.TrimSpace(req.input.Description) != "" {
		values.description = req.input.Description
	}
	values.payload = req.input.ReplacementPayload

	return values
}

func appendPreviewError(
	preview *invoiceadjustmentports.InvoiceAdjustmentPreview,
	field, message string,
) {
	preview.Errors[field] = append(preview.Errors[field], message)
}

func PreviewErrorsToMultiError(errors map[string][]string) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	for field, messages := range errors {
		for _, message := range messages {
			multiErr.Add(field, errortypes.ErrInvalidOperation, message)
		}
	}
	return multiErr
}

func (s *Engine) ComputePreview(
	ctx context.Context,
	req *invoiceadjustmentports.InvoiceAdjustmentRequest,
	excludeAdjustmentID pulid.ID,
) (*PreviewComputation, error) {
	entity, err := s.invoiceRepo.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         req.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	control, err := s.adjustmentCtrlRepo.GetByOrgID(ctx, req.TenantInfo.OrgID)
	if err != nil {
		return nil, err
	}

	accountingControl, err := s.accountingRepo.GetByOrgID(ctx, req.TenantInfo.OrgID)
	if err != nil {
		return nil, err
	}

	preview := newPreview(entity, req)
	validateAdjustableInvoice(preview, entity, req)
	preview.AccountingDate = s.resolveAccountingDate(ctx, entity, control, preview)
	s.validateSettlementPolicy(entity, control, preview)
	if strings.TrimSpace(req.Reason) == "" &&
		control.AdjustmentReasonRequirement == invoiceadjustmentcontrol.RequirementPolicyRequired {
		appendPreviewError(preview, "reason", "Adjustment reason is required by policy")
	}
	s.validateAttachments(ctx, req, preview)
	preview.CorrectionGroupID = s.previewCorrectionGroupID(ctx, entity, req.TenantInfo)

	usageByLine, err := s.repo.GetInvoiceLineCreditUsage(
		ctx,
		invoiceadjustmentports.GetInvoiceLineCreditUsageRequest{
			InvoiceID:           entity.ID,
			TenantInfo:          req.TenantInfo,
			ExcludeAdjustmentID: excludeAdjustmentID,
		},
	)
	if err != nil {
		return nil, err
	}

	acc := collectPreviewLines(preview, entity, req, usageByLine)
	s.applyRerate(ctx, preview, entity, req, acc)
	if req.Kind == invoiceadjustment.KindCreditRebill {
		if err = s.guardDetentionRebill(ctx, preview, req.TenantInfo, entity); err != nil {
			return nil, err
		}
	}

	s.finishPreview(preview, entity, control)

	return &PreviewComputation{
		Invoice:           entity,
		correctionGroupID: preview.CorrectionGroupID,
		control:           control,
		accountingControl: accountingControl,
		Preview:           preview,
		Lines:             acc.lines,
		creditLineItems:   acc.creditLines,
		replacementLines:  acc.replacementLines,
	}, nil
}

func newPreview(
	entity *invoice.Invoice,
	req *invoiceadjustmentports.InvoiceAdjustmentRequest,
) *invoiceadjustmentports.InvoiceAdjustmentPreview {
	return &invoiceadjustmentports.InvoiceAdjustmentPreview{
		InvoiceID:      entity.ID,
		InvoiceNumber:  entity.Number,
		CurrencyCode:   entity.CurrencyCode,
		Kind:           req.Kind,
		RebillStrategy: req.RebillStrategy,
		Warnings:       make([]string, 0),
		Errors:         make(map[string][]string),
		Lines: make(
			[]*invoiceadjustmentports.InvoiceAdjustmentPreviewLine,
			0,
			len(entity.Lines),
		),
	}
}

func validateAdjustableInvoice(
	preview *invoiceadjustmentports.InvoiceAdjustmentPreview,
	entity *invoice.Invoice,
	req *invoiceadjustmentports.InvoiceAdjustmentRequest,
) {
	if entity.Status != invoice.StatusPosted {
		appendPreviewError(preview, "invoiceId", "Only posted invoices may be adjusted")
	}
	if req.Kind == invoiceadjustment.KindFullReversal && entity.AppliedAmountMinor > 0 {
		appendPreviewError(
			preview,
			"invoiceId",
			"Unapply the customer payments and credit memos on this invoice before reversing it in full",
		)
	}
}

func (s *Engine) previewCorrectionGroupID(
	ctx context.Context,
	entity *invoice.Invoice,
	tenantInfo pagination.TenantInfo,
) pulid.ID {
	if entity.CorrectionGroupID.IsNotNil() {
		return entity.CorrectionGroupID
	}
	group, err := s.repo.GetCorrectionGroupByRootInvoice(
		ctx,
		invoiceadjustmentports.GetCorrectionGroupByRootInvoiceRequest{
			RootInvoiceID: entity.ID,
			TenantInfo:    tenantInfo,
		},
	)
	if err == nil && group != nil {
		return group.ID
	}

	return pulid.MustNew("icg_")
}

func requestedLinesByID(
	req *invoiceadjustmentports.InvoiceAdjustmentRequest,
) map[string]*invoiceadjustmentports.InvoiceAdjustmentLineInput {
	requestedByLine := make(
		map[string]*invoiceadjustmentports.InvoiceAdjustmentLineInput,
		len(req.Lines),
	)
	for _, line := range req.Lines {
		if line == nil {
			continue
		}
		requestedByLine[line.OriginalLineID.String()] = line
	}

	return requestedByLine
}

func (s *Engine) applyRerate(
	ctx context.Context,
	preview *invoiceadjustmentports.InvoiceAdjustmentPreview,
	entity *invoice.Invoice,
	req *invoiceadjustmentports.InvoiceAdjustmentRequest,
	acc *previewAccumulator,
) {
	if req.Kind != invoiceadjustment.KindCreditRebill ||
		req.RebillStrategy != invoiceadjustment.RebillStrategyRerate {
		return
	}
	rerate, err := s.computeRerate(ctx, entity, req.TenantInfo)
	if err != nil {
		appendPreviewError(preview, "rebillStrategy", err.Error())
		return
	}
	acc.replacementLines = rerate.lines
	preview.RebillTotalAmount = rerate.total
	preview.RerateVariancePercent = rerate.variance
}

func (s *Engine) finishPreview(
	preview *invoiceadjustmentports.InvoiceAdjustmentPreview,
	entity *invoice.Invoice,
	control *invoiceadjustmentcontrol.InvoiceAdjustmentControl,
) {
	preview.NetDeltaAmount = preview.RebillTotalAmount.Sub(preview.CreditTotalAmount)
	preview.WouldCreateUnappliedCredit = preview.CreditTotalAmount.GreaterThan(
		entity.OpenBalanceAmount(),
	)
	if preview.Kind == invoiceadjustment.KindWriteOff &&
		preview.CreditTotalAmount.GreaterThan(entity.OpenBalanceAmount()) {
		appendPreviewError(
			preview,
			"creditTotalAmount",
			"Write-off amount cannot exceed the invoice open balance",
		)
	}
	s.applyCreditBalancePolicy(preview, entity, control)
	s.applyApprovalPolicy(preview, entity, control)
	s.applyReplacementReviewPolicy(preview, control)
	if preview.CreditTotalAmount.IsZero() {
		appendPreviewError(preview, "lines", "Adjustment must produce a non-zero credit amount")
	}
	if preview.Kind == invoiceadjustment.KindWriteOff && entity.OpenBalanceAmount().IsZero() {
		appendPreviewError(preview, "invoiceId", "Write-offs require a remaining invoice balance")
	}
	if preview.RequiresReconciliationException {
		preview.Warnings = append(
			preview.Warnings,
			"Execution will create a reconciliation exception for finance follow-up",
		)
	}
}

func (s *Engine) applyCreditBalancePolicy(
	preview *invoiceadjustmentports.InvoiceAdjustmentPreview,
	entity *invoice.Invoice,
	control *invoiceadjustmentcontrol.InvoiceAdjustmentControl,
) {
	if !preview.WouldCreateUnappliedCredit {
		return
	}
	if preview.Kind == invoiceadjustment.KindWriteOff {
		appendPreviewError(
			preview,
			"creditTotalAmount",
			"Write-offs cannot create unapplied customer credit",
		)
		return
	}

	switch control.CustomerCreditBalancePolicy {
	case invoiceadjustmentcontrol.CustomerCreditBalancePolicyDisallow:
		appendPreviewError(
			preview,
			"creditTotalAmount",
			"Policy disallows customer credit balance outcomes",
		)
	case invoiceadjustmentcontrol.CustomerCreditBalancePolicyAllowUnappliedCredit:
		switch control.OverCreditPolicy {
		case invoiceadjustmentcontrol.OverCreditPolicyBlock:
			appendPreviewError(
				preview,
				"creditTotalAmount",
				"Adjustment would create unapplied customer credit because of payment state",
			)
		case invoiceadjustmentcontrol.OverCreditPolicyAllowWithApproval:
			preview.RequiresApproval = true
		}
	}

	if entity.SettlementStatus != invoice.SettlementStatusUnpaid {
		preview.RequiresReconciliationException = true
	}
}

func (s *Engine) applyApprovalPolicy(
	preview *invoiceadjustmentports.InvoiceAdjustmentPreview,
	entity *invoice.Invoice,
	control *invoiceadjustmentcontrol.InvoiceAdjustmentControl,
) {
	amount := preview.CreditTotalAmount.Abs()
	if preview.Kind == invoiceadjustment.KindWriteOff {
		switch control.WriteOffApprovalPolicy {
		case invoiceadjustmentcontrol.WriteOffApprovalPolicyDisallow:
			appendPreviewError(preview, "kind", "Write-offs are disallowed by policy")
		case invoiceadjustmentcontrol.WriteOffApprovalPolicyAlwaysRequireApproval:
			preview.RequiresApproval = true
		case invoiceadjustmentcontrol.WriteOffApprovalPolicyRequireApprovalAboveThreshold:
			if amount.GreaterThanOrEqual(control.WriteOffApprovalThreshold) {
				preview.RequiresApproval = true
			}
		}
	} else {
		//nolint:exhaustive // only actionable enum states require explicit handling here
		switch control.StandardAdjustmentApprovalPolicy {
		case invoiceadjustmentcontrol.ApprovalPolicyAlways:
			preview.RequiresApproval = true
		case invoiceadjustmentcontrol.ApprovalPolicyAmountThreshold:
			if amount.GreaterThanOrEqual(control.StandardAdjustmentApprovalThreshold) {
				preview.RequiresApproval = true
			}
		}
	}

	if entity.SettlementStatus != invoice.SettlementStatusUnpaid {
		preview.RequiresReconciliationException = true
	}
}

func (s *Engine) applyReplacementReviewPolicy(
	preview *invoiceadjustmentports.InvoiceAdjustmentPreview,
	control *invoiceadjustmentcontrol.InvoiceAdjustmentControl,
) {
	if preview.Kind != invoiceadjustment.KindCreditRebill {
		return
	}
	//nolint:exhaustive // only actionable enum states require explicit handling here
	switch control.ReplacementInvoiceReviewPolicy {
	case invoiceadjustmentcontrol.ReplacementInvoiceReviewPolicyAlwaysRequireReview:
		preview.RequiresReplacementInvoiceReview = true
	case invoiceadjustmentcontrol.ReplacementInvoiceReviewPolicyRequireReviewWhenEconomicTermsChange:
		if preview.RebillStrategy == invoiceadjustment.RebillStrategyRerate {
			preview.RequiresReplacementInvoiceReview = preview.RerateVariancePercent.GreaterThan(
				control.RerateVarianceTolerancePercent,
			)
		} else {
			preview.RequiresReplacementInvoiceReview = !preview.RebillTotalAmount.Equal(
				preview.CreditTotalAmount,
			)
		}
	}
}

// computeRerate rebuilds replacement lines from the current rating of every leg billed
// by the invoice — one leg for a single-shipment invoice, the line-attributed set for a
// grouped (order) invoice. Order-level lines (no leg attribution) are carried forward
// at their billed amount; rerating only re-prices legs.
type rerateResult struct {
	lines    []*invoice.InvoiceLine
	total    decimal.Decimal
	variance decimal.Decimal
}

func (s *Engine) computeRerate(
	ctx context.Context,
	entity *invoice.Invoice,
	tenantInfo pagination.TenantInfo,
) (*rerateResult, error) {
	legIDs := entity.LegShipmentIDs()
	if len(legIDs) == 0 {
		return nil, errortypes.NewValidationError(
			"invoiceId",
			errortypes.ErrInvalidOperation,
			"Invoice has no shipment legs to rerate",
		)
	}

	control, err := s.shipmentCtrlRepo.Get(ctx, tenantports.GetShipmentControlRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	// Every leg of a multi-leg order re-rates against the same tenant's rate
	// tables, so the invoice reads them once instead of once per leg.
	ctx = ratetablecache.With(ctx)
	ctx = effectiveversioncache.With(ctx)

	result := &rerateResult{
		lines:    make([]*invoice.InvoiceLine, 0, len(entity.Lines)),
		total:    decimal.Zero,
		variance: decimal.Zero,
	}
	for _, legID := range legIDs {
		if err = s.rerateLeg(ctx, &rerateLegRequest{
			entity:     entity,
			legID:      legID,
			control:    control,
			tenantInfo: tenantInfo,
		}, result); err != nil {
			return nil, err
		}
	}

	// Anything wider than one shipment can carry unattributed lines, and dropping
	// them here would under-bill the rebill. Keying this off a nil order id lost
	// them for every consolidated invoice.
	if entity.Scope != invoice.ScopeShipment {
		carryUnattributedLines(entity, result)
	}

	if entity.TotalAmount.GreaterThan(decimal.Zero) && !result.total.Equal(entity.TotalAmount) {
		result.variance = result.total.Sub(entity.TotalAmount).
			Abs().
			Div(entity.TotalAmount).
			Mul(decimal.NewFromInt(100))
	}
	return result, nil
}

type rerateLegRequest struct {
	entity     *invoice.Invoice
	legID      pulid.ID
	control    *shipmentcontrol.ShipmentControl
	tenantInfo pagination.TenantInfo
}

func (s *Engine) rerateLeg(
	ctx context.Context,
	req *rerateLegRequest,
	result *rerateResult,
) error {
	shp, err := s.shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         req.legID,
		TenantInfo: req.tenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return err
	}
	if err = s.commercial.Recalculate(ctx, shp, req.control, req.tenantInfo.UserID); err != nil {
		return err
	}
	if err = invoicelines.HydrateAccessorials(
		ctx,
		s.accessorialRepo,
		req.tenantInfo,
		shp,
	); err != nil {
		return err
	}

	resolution, err := chargeallocation.ResolveShares(ctx, shp, shp.ChargeAllocations)
	if err != nil {
		return err
	}
	share := resolution.ShareFor(req.entity.CustomerID)
	if share == nil {
		return errortypes.NewValidationError(
			"invoiceId",
			errortypes.ErrInvalidOperation,
			"This payer no longer has a share of shipment {0}",
			shp.ProNumber,
		)
	}

	legLines := invoicelines.ForShipmentShare(
		billingqueue.BillTypeInvoice,
		shp,
		share,
		len(result.lines)+1,
	)
	result.lines = append(result.lines, legLines...)
	result.total = result.total.Add(share.TotalAmount)

	return nil
}

func carryUnattributedLines(entity *invoice.Invoice, result *rerateResult) {
	for _, line := range entity.Lines {
		if line == nil || !line.ShipmentID.IsNil() {
			continue
		}
		carried := &invoice.InvoiceLine{
			LineNumber:  len(result.lines) + 1,
			Type:        line.Type,
			Description: line.Description,
			Quantity:    line.Quantity,
			UnitPrice:   line.UnitPrice,
			Amount:      line.Amount,
		}
		carried.CopyChargeDetail(line)
		result.lines = append(result.lines, carried)
		result.total = result.total.Add(line.Amount)
	}
}
