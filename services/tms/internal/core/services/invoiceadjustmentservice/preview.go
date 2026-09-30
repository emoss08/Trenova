package invoiceadjustmentservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/formula/contextvariablecache"
	"github.com/emoss08/trenova/internal/core/services/formula/effectiveversioncache"
	"github.com/emoss08/trenova/internal/core/services/invoicelines"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetablecache"
	"github.com/emoss08/trenova/shared/decimalutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func (s *Service) computePreview( //nolint:cyclop,funlen // legacy workflow
	ctx context.Context,
	req *servicesports.InvoiceAdjustmentRequest,
	excludeAdjustmentID pulid.ID,
) (*previewComputation, error) {
	entity, err := s.invoiceRepo.GetByID(ctx, repositories.GetInvoiceByIDRequest{
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

	preview := &servicesports.InvoiceAdjustmentPreview{
		InvoiceID:      entity.ID,
		InvoiceNumber:  entity.Number,
		CurrencyCode:   entity.CurrencyCode,
		Kind:           req.Kind,
		RebillStrategy: req.RebillStrategy,
		Warnings:       make([]string, 0),
		Errors:         make(map[string][]string),
		Lines: make(
			[]*servicesports.InvoiceAdjustmentPreviewLine,
			0,
			len(entity.Lines),
		),
	}

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

	preview.AccountingDate = s.resolveAccountingDate(ctx, entity, control, preview)
	s.validateSettlementPolicy(entity, control, preview)
	if strings.TrimSpace(req.Reason) == "" &&
		control.AdjustmentReasonRequirement == tenant.RequirementPolicyRequired {
		appendPreviewError(preview, "reason", "Adjustment reason is required by policy")
	}
	s.validateAttachments(ctx, req, preview)

	switch {
	case entity.CorrectionGroupID.IsNotNil():
		preview.CorrectionGroupID = entity.CorrectionGroupID
	case true:
		if group, groupErr := s.repo.GetCorrectionGroupByRootInvoice(
			ctx,
			repositories.GetCorrectionGroupByRootInvoiceRequest{
				RootInvoiceID: entity.ID,
				TenantInfo:    req.TenantInfo,
			},
		); groupErr == nil &&
			group != nil {
			preview.CorrectionGroupID = group.ID
		} else {
			preview.CorrectionGroupID = pulid.MustNew("icg_")
		}
	}

	usageByLine, err := s.repo.GetInvoiceLineCreditUsage(
		ctx,
		repositories.GetInvoiceLineCreditUsageRequest{
			InvoiceID:           entity.ID,
			TenantInfo:          req.TenantInfo,
			ExcludeAdjustmentID: excludeAdjustmentID,
		},
	)
	if err != nil {
		return nil, err
	}

	requestedByLine := make(map[string]*servicesports.InvoiceAdjustmentLineInput, len(req.Lines))
	for _, line := range req.Lines {
		if line == nil {
			continue
		}
		requestedByLine[line.OriginalLineID.String()] = line
	}

	lines := make([]*invoiceadjustment.InvoiceAdjustmentLine, 0, len(entity.Lines))
	creditLines := make([]*invoice.InvoiceLine, 0, len(entity.Lines))
	replacementLines := make([]*invoice.InvoiceLine, 0, len(entity.Lines))
	fullScope := len(req.Lines) == 0 || req.Kind == invoiceadjustment.KindFullReversal

	for _, sourceLine := range entity.Lines {
		if sourceLine == nil {
			continue
		}
		input, hasInput := requestedByLine[sourceLine.ID.String()]
		if !fullScope && !hasInput {
			continue
		}

		used := usageByLine[sourceLine.ID.String()]
		eligible := sourceLine.Amount.Abs().Sub(used)
		if eligible.IsNegative() {
			eligible = decimal.Zero
		}

		lineValues := resolvePreviewLineValues(previewLineValuesRequest{
			sourceLine: sourceLine,
			input:      input,
			kind:       req.Kind,
			eligible:   eligible,
		})
		creditAmount := lineValues.creditAmount
		creditQuantity := lineValues.creditQuantity
		rebillAmount := lineValues.rebillAmount
		rebillQuantity := lineValues.rebillQuantity
		description := lineValues.description
		payload := lineValues.payload

		if req.Kind == invoiceadjustment.KindCreditRebill && fullScope &&
			req.RebillStrategy != invoiceadjustment.RebillStrategyRerate {
			rebillAmount = sourceLine.Amount.Abs()
			rebillQuantity = sourceLine.Quantity
		}

		remainingEligibleAmount := eligible.Sub(creditAmount)
		hasEligibilityError := creditAmount.GreaterThan(eligible)
		eligibilityOverageAmount := decimal.Zero
		eligibilityMessage := ""
		if hasEligibilityError {
			eligibilityOverageAmount = creditAmount.Sub(eligible)
			eligibilityMessage = fmt.Sprintf(
				"Line %d exceeds the remaining eligible amount by %s",
				sourceLine.LineNumber,
				eligibilityOverageAmount.StringFixed(4),
			)
			appendPreviewError(preview, "lines", eligibilityMessage)
		}

		linePreview := &servicesports.InvoiceAdjustmentPreviewLine{
			LineNumber:               sourceLine.LineNumber,
			OriginalLineID:           sourceLine.ID,
			Description:              sourceLine.Description,
			EligibleAmount:           eligible,
			AlreadyCreditedAmount:    used,
			RequestedCreditAmount:    creditAmount,
			RequestedRebillAmount:    rebillAmount,
			RemainingEligibleAmount:  remainingEligibleAmount,
			HasEligibilityError:      hasEligibilityError,
			EligibilityOverageAmount: eligibilityOverageAmount,
			EligibilityMessage:       eligibilityMessage,
		}
		preview.Lines = append(preview.Lines, linePreview)

		preview.CreditTotalAmount = preview.CreditTotalAmount.Add(creditAmount)
		preview.RebillTotalAmount = preview.RebillTotalAmount.Add(rebillAmount)

		lines = append(lines, &invoiceadjustment.InvoiceAdjustmentLine{
			OriginalInvoiceID:       entity.ID,
			OriginalLineID:          sourceLine.ID,
			LineNumber:              sourceLine.LineNumber,
			Description:             description,
			CreditQuantity:          creditQuantity,
			CreditAmount:            creditAmount,
			RemainingEligibleAmount: linePreview.RemainingEligibleAmount,
			RebillQuantity:          rebillQuantity,
			RebillAmount:            rebillAmount,
			ReplacementPayload:      payload,
		})

		if creditAmount.GreaterThan(decimal.Zero) {
			unitPrice := creditAmount
			if creditQuantity.GreaterThan(decimal.Zero) {
				unitPrice = creditAmount.Div(creditQuantity)
			}
			creditLine := &invoice.InvoiceLine{
				LineNumber:  sourceLine.LineNumber,
				Type:        sourceLine.Type,
				Description: description,
				Quantity:    creditQuantity,
				UnitPrice:   unitPrice.Neg(),
				Amount:      creditAmount.Neg(),
			}
			creditLine.CopyChargeDetail(sourceLine)
			creditLines = append(creditLines, creditLine)
		}

		if rebillAmount.GreaterThan(decimal.Zero) {
			unitPrice := rebillAmount
			if rebillQuantity.GreaterThan(decimal.Zero) {
				unitPrice = rebillAmount.Div(rebillQuantity)
			}
			replacementLine := &invoice.InvoiceLine{
				LineNumber:  len(replacementLines) + 1,
				Type:        sourceLine.Type,
				Description: description,
				Quantity:    decimalutils.Max(rebillQuantity, decimal.NewFromInt(1)),
				UnitPrice:   unitPrice,
				Amount:      rebillAmount,
			}
			replacementLine.CopyChargeDetail(sourceLine)
			replacementLines = append(replacementLines, replacementLine)
		}
	}

	if req.Kind == invoiceadjustment.KindCreditRebill &&
		req.RebillStrategy == invoiceadjustment.RebillStrategyRerate {
		rerateLines, rerateTotal, rerateVariance, rerateErr := s.computeRerate(
			ctx,
			entity,
			req.TenantInfo,
		)
		if rerateErr != nil {
			appendPreviewError(preview, "rebillStrategy", rerateErr.Error())
		} else {
			replacementLines = rerateLines
			preview.RebillTotalAmount = rerateTotal
			preview.RerateVariancePercent = rerateVariance
		}
	}
	if req.Kind == invoiceadjustment.KindCreditRebill {
		if err = s.guardDetentionRebill(ctx, preview, req.TenantInfo, entity); err != nil {
			return nil, err
		}
	}

	preview.NetDeltaAmount = preview.RebillTotalAmount.Sub(preview.CreditTotalAmount)
	preview.WouldCreateUnappliedCredit = preview.CreditTotalAmount.GreaterThan(
		entity.OpenBalanceAmount(),
	)
	if req.Kind == invoiceadjustment.KindWriteOff &&
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
	if req.Kind == invoiceadjustment.KindWriteOff && entity.OpenBalanceAmount().IsZero() {
		appendPreviewError(preview, "invoiceId", "Write-offs require a remaining invoice balance")
	}
	if preview.RequiresReconciliationException {
		preview.Warnings = append(
			preview.Warnings,
			"Execution will create a reconciliation exception for finance follow-up",
		)
	}

	return &previewComputation{
		invoice:           entity,
		correctionGroupID: preview.CorrectionGroupID,
		control:           control,
		accountingControl: accountingControl,
		preview:           preview,
		lines:             lines,
		creditLineItems:   creditLines,
		replacementLines:  replacementLines,
	}, nil
}

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

func (s *Service) applyCreditBalancePolicy(
	preview *servicesports.InvoiceAdjustmentPreview,
	entity *invoice.Invoice,
	control *tenant.InvoiceAdjustmentControl,
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
	case tenant.CustomerCreditBalancePolicyDisallow:
		appendPreviewError(
			preview,
			"creditTotalAmount",
			"Policy disallows customer credit balance outcomes",
		)
	case tenant.CustomerCreditBalancePolicyAllowUnappliedCredit:
		switch control.OverCreditPolicy {
		case tenant.OverCreditPolicyBlock:
			appendPreviewError(
				preview,
				"creditTotalAmount",
				"Adjustment would create unapplied customer credit because of payment state",
			)
		case tenant.OverCreditPolicyAllowWithApproval:
			preview.RequiresApproval = true
		}
	}

	if entity.SettlementStatus != invoice.SettlementStatusUnpaid {
		preview.RequiresReconciliationException = true
	}
}

func (s *Service) applyApprovalPolicy(
	preview *servicesports.InvoiceAdjustmentPreview,
	entity *invoice.Invoice,
	control *tenant.InvoiceAdjustmentControl,
) {
	amount := preview.CreditTotalAmount.Abs()
	if preview.Kind == invoiceadjustment.KindWriteOff {
		switch control.WriteOffApprovalPolicy {
		case tenant.WriteOffApprovalPolicyDisallow:
			appendPreviewError(preview, "kind", "Write-offs are disallowed by policy")
		case tenant.WriteOffApprovalPolicyAlwaysRequireApproval:
			preview.RequiresApproval = true
		case tenant.WriteOffApprovalPolicyRequireApprovalAboveThreshold:
			if amount.GreaterThanOrEqual(control.WriteOffApprovalThreshold) {
				preview.RequiresApproval = true
			}
		}
	} else {
		//nolint:exhaustive // only actionable enum states require explicit handling here
		switch control.StandardAdjustmentApprovalPolicy {
		case tenant.ApprovalPolicyAlways:
			preview.RequiresApproval = true
		case tenant.ApprovalPolicyAmountThreshold:
			if amount.GreaterThanOrEqual(control.StandardAdjustmentApprovalThreshold) {
				preview.RequiresApproval = true
			}
		}
	}

	if entity.SettlementStatus != invoice.SettlementStatusUnpaid {
		preview.RequiresReconciliationException = true
	}
}

func (s *Service) applyReplacementReviewPolicy(
	preview *servicesports.InvoiceAdjustmentPreview,
	control *tenant.InvoiceAdjustmentControl,
) {
	if preview.Kind != invoiceadjustment.KindCreditRebill {
		return
	}
	//nolint:exhaustive // only actionable enum states require explicit handling here
	switch control.ReplacementInvoiceReviewPolicy {
	case tenant.ReplacementInvoiceReviewPolicyAlwaysRequireReview:
		preview.RequiresReplacementInvoiceReview = true
	case tenant.ReplacementInvoiceReviewPolicyRequireReviewWhenEconomicTermsChange:
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
func (s *Service) computeRerate( //nolint:gocritic // stable API shape
	ctx context.Context,
	entity *invoice.Invoice,
	tenantInfo pagination.TenantInfo,
) ([]*invoice.InvoiceLine, decimal.Decimal, decimal.Decimal, error) {
	legIDs := entity.LegShipmentIDs()
	if len(legIDs) == 0 {
		return nil, decimal.Zero, decimal.Zero, errortypes.NewValidationError(
			"invoiceId",
			errortypes.ErrInvalidOperation,
			"Invoice has no shipment legs to rerate",
		)
	}

	control, err := s.shipmentCtrlRepo.Get(ctx, repositories.GetShipmentControlRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, decimal.Zero, decimal.Zero, err
	}

	// Every leg of a multi-leg order re-rates against the same tenant's rate
	// tables, so the invoice reads them once instead of once per leg.
	ctx = ratetablecache.With(ctx)
	ctx = contextvariablecache.With(ctx)
	ctx = effectiveversioncache.With(ctx)

	lines := make([]*invoice.InvoiceLine, 0, len(entity.Lines))
	total := decimal.Zero
	nextLineNumber := 1
	for _, legID := range legIDs {
		shp, legErr := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
			ID:         legID,
			TenantInfo: tenantInfo,
			ShipmentOptions: repositories.ShipmentOptions{
				ExpandShipmentDetails: true,
			},
		})
		if legErr != nil {
			return nil, decimal.Zero, decimal.Zero, legErr
		}
		if legErr = s.commercial.Recalculate(ctx, shp, control, tenantInfo.UserID); legErr != nil {
			return nil, decimal.Zero, decimal.Zero, legErr
		}
		if legErr = invoicelines.HydrateAccessorials(
			ctx,
			s.accessorialRepo,
			tenantInfo,
			shp,
		); legErr != nil {
			return nil, decimal.Zero, decimal.Zero, legErr
		}

		resolution, legErr := shipment.ResolveShares(shp, shp.ChargeAllocations)
		if legErr != nil {
			return nil, decimal.Zero, decimal.Zero, legErr
		}
		share := resolution.ShareFor(entity.CustomerID)
		if share == nil {
			return nil, decimal.Zero, decimal.Zero, errortypes.NewValidationError(
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
			nextLineNumber,
		)
		lines = append(lines, legLines...)
		nextLineNumber += len(legLines)
		total = total.Add(share.TotalAmount)
	}

	// Anything wider than one shipment can carry unattributed lines, and dropping
	// them here would under-bill the rebill. Keying this off a nil order id lost
	// them for every consolidated invoice.
	if entity.Scope != invoice.ScopeShipment {
		for _, line := range entity.Lines {
			if line == nil || !line.ShipmentID.IsNil() {
				continue
			}
			carried := &invoice.InvoiceLine{
				LineNumber:  nextLineNumber,
				Type:        line.Type,
				Description: line.Description,
				Quantity:    line.Quantity,
				UnitPrice:   line.UnitPrice,
				Amount:      line.Amount,
			}
			carried.CopyChargeDetail(line)
			lines = append(lines, carried)
			nextLineNumber++
			total = total.Add(line.Amount)
		}
	}

	variance := decimal.Zero
	if entity.TotalAmount.GreaterThan(decimal.Zero) && !total.Equal(entity.TotalAmount) {
		variance = total.Sub(entity.TotalAmount).
			Abs().
			Div(entity.TotalAmount).
			Mul(decimal.NewFromInt(100))
	}
	return lines, total, variance, nil
}

func appendPreviewError(preview *servicesports.InvoiceAdjustmentPreview, field, message string) {
	preview.Errors[field] = append(preview.Errors[field], message)
}

func previewErrorsToMultiError(errors map[string][]string) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	for field, messages := range errors {
		for _, message := range messages {
			multiErr.Add(field, errortypes.ErrInvalidOperation, message)
		}
	}
	return multiErr
}
