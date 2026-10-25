package tenant

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
	_ validationframework.TenantedEntity = (*InvoiceAdjustmentControl)(nil)
)

type InvoiceAdjustmentControl struct {
	keel.Table `keel:"invoice_adjustment_controls"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:iac_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	PartiallyPaidInvoiceAdjustmentPolicy AdjustmentEligibilityPolicy `json:"partiallyPaidInvoiceAdjustmentPolicy" keel:"partially_paid_invoice_adjustment_policy"`
	PaidInvoiceAdjustmentPolicy          AdjustmentEligibilityPolicy `json:"paidInvoiceAdjustmentPolicy"          keel:"paid_invoice_adjustment_policy"`
	DisputedInvoiceAdjustmentPolicy      AdjustmentEligibilityPolicy `json:"disputedInvoiceAdjustmentPolicy"      keel:"disputed_invoice_adjustment_policy"`

	AdjustmentAccountingDatePolicy AdjustmentAccountingDatePolicy `json:"adjustmentAccountingDatePolicy" keel:"adjustment_accounting_date_policy"`
	ClosedPeriodAdjustmentPolicy   ClosedPeriodAdjustmentPolicy   `json:"closedPeriodAdjustmentPolicy"   keel:"closed_period_adjustment_policy"`

	AdjustmentReasonRequirement RequirementPolicy `json:"adjustmentReasonRequirement" keel:"adjustment_reason_requirement"`

	StandardAdjustmentApprovalPolicy    ApprovalPolicy         `json:"standardAdjustmentApprovalPolicy"    keel:"standard_adjustment_approval_policy"`
	StandardAdjustmentApprovalThreshold decimal.Decimal        `json:"standardAdjustmentApprovalThreshold" keel:"standard_adjustment_approval_threshold,nullzero"`
	WriteOffApprovalPolicy              WriteOffApprovalPolicy `json:"writeOffApprovalPolicy"              keel:"write_off_approval_policy"`
	WriteOffApprovalThreshold           decimal.Decimal        `json:"writeOffApprovalThreshold"           keel:"write_off_approval_threshold,nullzero"`
	RequireSeparateApprover             bool                   `json:"requireSeparateApprover"             keel:"require_separate_approver"`

	RerateVarianceTolerancePercent decimal.Decimal                `json:"rerateVarianceTolerancePercent" keel:"rerate_variance_tolerance_percent"`
	ReplacementInvoiceReviewPolicy ReplacementInvoiceReviewPolicy `json:"replacementInvoiceReviewPolicy" keel:"replacement_invoice_review_policy"`

	CustomerCreditBalancePolicy       CustomerCreditBalancePolicy       `json:"customerCreditBalancePolicy"       keel:"customer_credit_balance_policy"`
	OverCreditPolicy                  OverCreditPolicy                  `json:"overCreditPolicy"                  keel:"over_credit_policy"`
	SupersededInvoiceVisibilityPolicy SupersededInvoiceVisibilityPolicy `json:"supersededInvoiceVisibilityPolicy" keel:"superseded_invoice_visibility_policy"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *BusinessUnit `json:"businessUnit,omitempty"`
	Organization *Organization `json:"organization,omitempty"`
}

func (iac *InvoiceAdjustmentControl) BeforeCreate(_ context.Context) error {
	if iac.PartiallyPaidInvoiceAdjustmentPolicy == "" {
		iac.PartiallyPaidInvoiceAdjustmentPolicy = AdjustmentEligibilityAllowWithApproval
	}
	if iac.PaidInvoiceAdjustmentPolicy == "" {
		iac.PaidInvoiceAdjustmentPolicy = AdjustmentEligibilityDisallow
	}
	if iac.DisputedInvoiceAdjustmentPolicy == "" {
		iac.DisputedInvoiceAdjustmentPolicy = AdjustmentEligibilityAllowWithApproval
	}
	if iac.AdjustmentAccountingDatePolicy == "" {
		iac.AdjustmentAccountingDatePolicy = AdjustmentAccountingDateUseOriginalIfOpenElseNextOpen
	}
	if iac.ClosedPeriodAdjustmentPolicy == "" {
		iac.ClosedPeriodAdjustmentPolicy = ClosedPeriodAdjustmentPolicyPostInNextOpenPeriodWithApproval
	}
	if iac.AdjustmentReasonRequirement == "" {
		iac.AdjustmentReasonRequirement = RequirementPolicyRequired
	}
	if iac.StandardAdjustmentApprovalPolicy == "" {
		iac.StandardAdjustmentApprovalPolicy = ApprovalPolicyAmountThreshold
	}
	if iac.WriteOffApprovalPolicy == "" {
		iac.WriteOffApprovalPolicy = WriteOffApprovalPolicyRequireApprovalAboveThreshold
	}
	if iac.ReplacementInvoiceReviewPolicy == "" {
		iac.ReplacementInvoiceReviewPolicy = ReplacementInvoiceReviewPolicyRequireReviewWhenEconomicTermsChange
	}
	if iac.CustomerCreditBalancePolicy == "" {
		iac.CustomerCreditBalancePolicy = CustomerCreditBalancePolicyAllowUnappliedCredit
	}
	if iac.OverCreditPolicy == "" {
		iac.OverCreditPolicy = OverCreditPolicyBlock
	}
	if iac.SupersededInvoiceVisibilityPolicy == "" {
		iac.SupersededInvoiceVisibilityPolicy = SupersededInvoiceVisibilityPolicyShowCurrentOnlyExternally
	}

	return nil
}

func (iac *InvoiceAdjustmentControl) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		iac,
		validation.Field(&iac.PartiallyPaidInvoiceAdjustmentPolicy, validation.Required),
		validation.Field(&iac.PaidInvoiceAdjustmentPolicy, validation.Required),
		validation.Field(&iac.DisputedInvoiceAdjustmentPolicy, validation.Required),
		validation.Field(&iac.AdjustmentAccountingDatePolicy, validation.Required),
		validation.Field(&iac.ClosedPeriodAdjustmentPolicy, validation.Required),
		validation.Field(&iac.AdjustmentReasonRequirement, validation.Required),
		validation.Field(&iac.StandardAdjustmentApprovalPolicy, validation.Required),
		validation.Field(&iac.WriteOffApprovalPolicy, validation.Required),
		validation.Field(&iac.ReplacementInvoiceReviewPolicy, validation.Required),
		validation.Field(&iac.CustomerCreditBalancePolicy, validation.Required),
		validation.Field(&iac.OverCreditPolicy, validation.Required),
		validation.Field(&iac.SupersededInvoiceVisibilityPolicy, validation.Required),
	))
}

func (iac *InvoiceAdjustmentControl) GetID() pulid.ID {
	return iac.ID
}

func (iac *InvoiceAdjustmentControl) GetTableName() string {
	return "invoice_adjustment_controls"
}

func (iac *InvoiceAdjustmentControl) GetOrganizationID() pulid.ID {
	return iac.OrganizationID
}

func (iac *InvoiceAdjustmentControl) GetBusinessUnitID() pulid.ID {
	return iac.BusinessUnitID
}
