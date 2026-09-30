package invoiceadjustmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

func (s *Service) resolveAccountingDate(
	ctx context.Context,
	entity *invoice.Invoice,
	control *tenant.InvoiceAdjustmentControl,
	preview *servicesports.InvoiceAdjustmentPreview,
) int64 {
	originalDate := entity.InvoiceDate
	if control.AdjustmentAccountingDatePolicy == tenant.AdjustmentAccountingDateAlwaysNextOpen {
		nextOpen := s.resolveNextOpenPeriodDate(
			ctx,
			entity.OrganizationID,
			entity.BusinessUnitID,
			originalDate,
		)
		if nextOpen > 0 {
			return nextOpen
		}

		return timeutils.NowUnix()
	}

	period, err := s.fiscalPeriodRepo.GetPeriodByDate(ctx, repositories.GetPeriodByDateRequest{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
		Date:  originalDate,
	})
	if err == nil && period != nil && period.Status == fiscalperiod.StatusOpen {
		return originalDate
	}

	switch control.ClosedPeriodAdjustmentPolicy {
	case tenant.ClosedPeriodAdjustmentPolicyDisallow:
		appendPreviewError(
			preview,
			"accountingDate",
			"Closed-period adjustments are disallowed by policy",
		)
	case tenant.ClosedPeriodAdjustmentPolicyRequireReopen:
		appendPreviewError(
			preview,
			"accountingDate",
			"The accounting period must be reopened before this adjustment can be posted",
		)
	case tenant.ClosedPeriodAdjustmentPolicyPostInNextOpenPeriodWithApproval:
		preview.RequiresApproval = true
		preview.Warnings = append(
			preview.Warnings,
			"Adjustment will post in the next open period and requires approval",
		)
	}

	nextOpen := s.resolveNextOpenPeriodDate(
		ctx,
		entity.OrganizationID,
		entity.BusinessUnitID,
		originalDate,
	)
	if nextOpen > 0 {
		return nextOpen
	}

	return timeutils.NowUnix()
}

func (s *Service) resolveNextOpenPeriodDate(
	ctx context.Context,
	orgID, buID pulid.ID,
	fallback int64,
) int64 {
	period, err := s.fiscalPeriodRepo.GetPeriodByDate(ctx, repositories.GetPeriodByDateRequest{
		OrgID: orgID,
		BuID:  buID,
		Date:  timeutils.NowUnix(),
	})
	if err == nil && period != nil && period.Status == fiscalperiod.StatusOpen {
		return intutils.Max(period.StartDate, fallback)
	}
	return timeutils.NowUnix()
}

func (s *Service) validateSettlementPolicy(
	entity *invoice.Invoice,
	control *tenant.InvoiceAdjustmentControl,
	preview *servicesports.InvoiceAdjustmentPreview,
) {
	//nolint:exhaustive // only actionable enum states require explicit handling here
	switch entity.SettlementStatus {
	case invoice.SettlementStatusPaid:
		//nolint:exhaustive // only actionable enum states require explicit handling here
		switch control.PaidInvoiceAdjustmentPolicy {
		case tenant.AdjustmentEligibilityDisallow:
			appendPreviewError(preview, "invoiceId", "Paid invoices cannot be adjusted by policy")
		case tenant.AdjustmentEligibilityAllowWithApproval:
			preview.RequiresApproval = true
		}
	case invoice.SettlementStatusPartiallyPaid:
		//nolint:exhaustive // only actionable enum states require explicit handling here
		switch control.PartiallyPaidInvoiceAdjustmentPolicy {
		case tenant.AdjustmentEligibilityDisallow:
			appendPreviewError(
				preview,
				"invoiceId",
				"Partially paid invoices cannot be adjusted by policy",
			)
		case tenant.AdjustmentEligibilityAllowWithApproval:
			preview.RequiresApproval = true
		}
	}

	if entity.DisputeStatus == invoice.DisputeStatusDisputed {
		//nolint:exhaustive // only actionable enum states require explicit handling here
		switch control.DisputedInvoiceAdjustmentPolicy {
		case tenant.AdjustmentEligibilityDisallow:
			appendPreviewError(
				preview,
				"invoiceId",
				"Disputed invoices cannot be adjusted by policy",
			)
		case tenant.AdjustmentEligibilityAllowWithApproval:
			preview.RequiresApproval = true
		}
	}

	if entity.SettlementStatus != invoice.SettlementStatusUnpaid {
		preview.RequiresReconciliationException = true
	}
}
