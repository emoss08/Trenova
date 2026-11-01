package billingcontrolresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlinput"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
)

func billingControlFromInput(
	existing *billingcontrol.BillingControl,
	input *gqlmodel.UpdateBillingControlInput,
) (*billingcontrol.BillingControl, error) {
	parse := gqlinput.NewParser()
	entity := &billingcontrol.BillingControl{
		ID:                                    existing.ID,
		BusinessUnitID:                        existing.BusinessUnitID,
		OrganizationID:                        existing.OrganizationID,
		Version:                               int64(input.Version),
		CreatedAt:                             existing.CreatedAt,
		UpdatedAt:                             existing.UpdatedAt,
		BusinessUnit:                          existing.BusinessUnit,
		Organization:                          existing.Organization,
		DefaultPaymentTerm:                    input.DefaultPaymentTerm,
		DefaultInvoiceTerms:                   input.DefaultInvoiceTerms,
		DefaultInvoiceFooter:                  input.DefaultInvoiceFooter,
		ShowDueDateOnInvoice:                  input.ShowDueDateOnInvoice,
		ShowBalanceDueOnInvoice:               input.ShowBalanceDueOnInvoice,
		ReadyToBillAssignmentMode:             input.ReadyToBillAssignmentMode,
		BillingQueueTransferMode:              input.BillingQueueTransferMode,
		BillingQueueTransferSchedule:          input.BillingQueueTransferSchedule,
		BillingQueueTransferBatchSize:         input.BillingQueueTransferBatchSize,
		InvoiceDraftCreationMode:              input.InvoiceDraftCreationMode,
		InvoicePostingMode:                    input.InvoicePostingMode,
		AutoInvoiceBatchSize:                  input.AutoInvoiceBatchSize,
		NotifyOnAutoInvoiceCreation:           input.NotifyOnAutoInvoiceCreation,
		ShipmentBillingRequirementEnforcement: input.ShipmentBillingRequirementEnforcement,
		RateValidationEnforcement:             input.RateValidationEnforcement,
		BillingExceptionDisposition:           input.BillingExceptionDisposition,
		NotifyOnBillingExceptions:             input.NotifyOnBillingExceptions,
		RateVarianceTolerancePercent: parse.Decimal(
			"rateVarianceTolerancePercent",
			input.RateVarianceTolerancePercent,
		),
		RateVarianceAutoResolutionMode: input.RateVarianceAutoResolutionMode,
		UnratedShipmentDisposition:     input.UnratedShipmentDisposition,
		FallbackFormulaTemplateID: parse.IDPtr(
			"fallbackFormulaTemplateId",
			input.FallbackFormulaTemplateID,
		),
		RequireRateOverrideReason: input.RequireRateOverrideReason,
		EnforceMarginFloor:        input.EnforceMarginFloor,
		QueueAutoApproveMode:      input.QueueAutoApproveMode,
		QueueAutoApproveMaxAmount: parse.NullDecimal(
			"queueAutoApproveMaxAmount",
			input.QueueAutoApproveMaxAmount,
		),
		QueueAgentReviewEnabled:  input.QueueAgentReviewEnabled,
		LateChargeAssessmentMode: input.LateChargeAssessmentMode,
		LateChargeMinimumAmount: parse.Decimal(
			"lateChargeMinimumAmount",
			input.LateChargeMinimumAmount,
		),
		PaymentRemindersEnabled: input.PaymentRemindersEnabled,
		PaymentReminderDays: parse.Int32s(
			"paymentReminderDays",
			input.PaymentReminderDays,
		),
		MonthlyStatementsEnabled: input.MonthlyStatementsEnabled,
		MonthlyStatementDay: parse.Int16(
			"monthlyStatementDay",
			input.MonthlyStatementDay,
		),
	}
	if err := parse.Err(); err != nil {
		return nil, err
	}

	return entity, nil
}
