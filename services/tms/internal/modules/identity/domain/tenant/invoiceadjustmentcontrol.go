package tenant

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*InvoiceAdjustmentControl)(nil)
	_ validationframework.TenantedEntity = (*InvoiceAdjustmentControl)(nil)
)

type InvoiceAdjustmentControl struct {
	bun.BaseModel `bun:"table:invoice_adjustment_controls,alias:iac" json:"-"`
	keel.Table    `keel:"invoice_adjustment_controls"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:iac_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull" keel:"organization_id,pk,tenant"`

	PartiallyPaidInvoiceAdjustmentPolicy AdjustmentEligibilityPolicy `json:"partiallyPaidInvoiceAdjustmentPolicy" bun:"partially_paid_invoice_adjustment_policy,type:adjustment_eligibility_policy_enum,notnull,default:'AllowWithApproval'" keel:"partially_paid_invoice_adjustment_policy"`
	PaidInvoiceAdjustmentPolicy          AdjustmentEligibilityPolicy `json:"paidInvoiceAdjustmentPolicy"          bun:"paid_invoice_adjustment_policy,type:adjustment_eligibility_policy_enum,notnull,default:'Disallow'" keel:"paid_invoice_adjustment_policy"`
	DisputedInvoiceAdjustmentPolicy      AdjustmentEligibilityPolicy `json:"disputedInvoiceAdjustmentPolicy"      bun:"disputed_invoice_adjustment_policy,type:adjustment_eligibility_policy_enum,notnull,default:'AllowWithApproval'" keel:"disputed_invoice_adjustment_policy"`

	AdjustmentAccountingDatePolicy AdjustmentAccountingDatePolicy `json:"adjustmentAccountingDatePolicy" bun:"adjustment_accounting_date_policy,type:adjustment_accounting_date_policy_enum,notnull,default:'UseOriginalIfOpenElseNextOpen'" keel:"adjustment_accounting_date_policy"`
	ClosedPeriodAdjustmentPolicy   ClosedPeriodAdjustmentPolicy   `json:"closedPeriodAdjustmentPolicy"   bun:"closed_period_adjustment_policy,type:closed_period_adjustment_policy_enum,notnull,default:'PostInNextOpenPeriodWithApproval'" keel:"closed_period_adjustment_policy"`

	AdjustmentReasonRequirement RequirementPolicy `json:"adjustmentReasonRequirement" bun:"adjustment_reason_requirement,type:requirement_policy_enum,notnull,default:'Required'" keel:"adjustment_reason_requirement"`

	StandardAdjustmentApprovalPolicy    ApprovalPolicy         `json:"standardAdjustmentApprovalPolicy"    bun:"standard_adjustment_approval_policy,type:approval_policy_enum,notnull,default:'AmountThreshold'" keel:"standard_adjustment_approval_policy"`
	StandardAdjustmentApprovalThreshold decimal.Decimal        `json:"standardAdjustmentApprovalThreshold" bun:"standard_adjustment_approval_threshold,type:NUMERIC(19,4),nullzero" keel:"standard_adjustment_approval_threshold,nullzero"`
	WriteOffApprovalPolicy              WriteOffApprovalPolicy `json:"writeOffApprovalPolicy"              bun:"write_off_approval_policy,type:write_off_approval_policy_enum,notnull,default:'RequireApprovalAboveThreshold'" keel:"write_off_approval_policy"`
	WriteOffApprovalThreshold           decimal.Decimal        `json:"writeOffApprovalThreshold"           bun:"write_off_approval_threshold,type:NUMERIC(19,4),nullzero" keel:"write_off_approval_threshold,nullzero"`
	RequireSeparateApprover             bool                   `json:"requireSeparateApprover"             bun:"require_separate_approver,type:BOOLEAN,notnull,default:false" keel:"require_separate_approver"`

	RerateVarianceTolerancePercent decimal.Decimal                `json:"rerateVarianceTolerancePercent" bun:"rerate_variance_tolerance_percent,type:NUMERIC(9,6),notnull,default:0.000000" keel:"rerate_variance_tolerance_percent"`
	ReplacementInvoiceReviewPolicy ReplacementInvoiceReviewPolicy `json:"replacementInvoiceReviewPolicy" bun:"replacement_invoice_review_policy,type:replacement_invoice_review_policy_enum,notnull,default:'RequireReviewWhenEconomicTermsChange'" keel:"replacement_invoice_review_policy"`

	CustomerCreditBalancePolicy       CustomerCreditBalancePolicy       `json:"customerCreditBalancePolicy"       bun:"customer_credit_balance_policy,type:customer_credit_balance_policy_enum,notnull,default:'AllowUnappliedCredit'" keel:"customer_credit_balance_policy"`
	OverCreditPolicy                  OverCreditPolicy                  `json:"overCreditPolicy"                  bun:"over_credit_policy,type:over_credit_policy_enum,notnull,default:'Block'" keel:"over_credit_policy"`
	SupersededInvoiceVisibilityPolicy SupersededInvoiceVisibilityPolicy `json:"supersededInvoiceVisibilityPolicy" bun:"superseded_invoice_visibility_policy,type:superseded_invoice_visibility_policy_enum,notnull,default:'ShowCurrentOnlyExternally'" keel:"superseded_invoice_visibility_policy"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	BusinessUnit *BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
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

func (iac *InvoiceAdjustmentControl) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if iac.ID.IsNil() {
			iac.ID = pulid.MustNew("iac_")
		}
		iac.CreatedAt = now
	case *bun.UpdateQuery:
		iac.UpdatedAt = now
	}

	return nil
}
