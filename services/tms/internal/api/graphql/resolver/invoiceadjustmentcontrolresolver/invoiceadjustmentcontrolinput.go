package invoiceadjustmentcontrolresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlinput"
	"github.com/emoss08/trenova/internal/modules/identity/domain/invoiceadjustmentcontrol"
)

func invoiceAdjustmentControlFromInput(
	existing *invoiceadjustmentcontrol.InvoiceAdjustmentControl,
	input *gqlmodel.UpdateInvoiceAdjustmentControlInput,
) (*invoiceadjustmentcontrol.InvoiceAdjustmentControl, error) {
	parse := gqlinput.NewParser()
	entity := &invoiceadjustmentcontrol.InvoiceAdjustmentControl{
		ID:                                   existing.ID,
		BusinessUnitID:                       existing.BusinessUnitID,
		OrganizationID:                       existing.OrganizationID,
		Version:                              int64(input.Version),
		CreatedAt:                            existing.CreatedAt,
		UpdatedAt:                            existing.UpdatedAt,
		BusinessUnit:                         existing.BusinessUnit,
		Organization:                         existing.Organization,
		PartiallyPaidInvoiceAdjustmentPolicy: input.PartiallyPaidInvoiceAdjustmentPolicy,
		PaidInvoiceAdjustmentPolicy:          input.PaidInvoiceAdjustmentPolicy,
		DisputedInvoiceAdjustmentPolicy:      input.DisputedInvoiceAdjustmentPolicy,
		AdjustmentAccountingDatePolicy:       input.AdjustmentAccountingDatePolicy,
		ClosedPeriodAdjustmentPolicy:         input.ClosedPeriodAdjustmentPolicy,
		AdjustmentReasonRequirement:          input.AdjustmentReasonRequirement,
		StandardAdjustmentApprovalPolicy:     input.StandardAdjustmentApprovalPolicy,
		StandardAdjustmentApprovalThreshold: parse.Decimal(
			"standardAdjustmentApprovalThreshold",
			input.StandardAdjustmentApprovalThreshold,
		),
		WriteOffApprovalPolicy: input.WriteOffApprovalPolicy,
		WriteOffApprovalThreshold: parse.Decimal(
			"writeOffApprovalThreshold",
			input.WriteOffApprovalThreshold,
		),
		RequireSeparateApprover: input.RequireSeparateApprover,
		RerateVarianceTolerancePercent: parse.Decimal(
			"rerateVarianceTolerancePercent",
			input.RerateVarianceTolerancePercent,
		),
		ReplacementInvoiceReviewPolicy:    input.ReplacementInvoiceReviewPolicy,
		CustomerCreditBalancePolicy:       input.CustomerCreditBalancePolicy,
		OverCreditPolicy:                  input.OverCreditPolicy,
		SupersededInvoiceVisibilityPolicy: input.SupersededInvoiceVisibilityPolicy,
	}
	if err := parse.Err(); err != nil {
		return nil, err
	}

	return entity, nil
}
