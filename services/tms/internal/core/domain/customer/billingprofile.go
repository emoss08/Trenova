package customer

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*CustomerBillingProfile)(nil)

type CustomerBillingProfile struct {
	bun.BaseModel `bun:"table:customer_billing_profiles,alias:cbp" json:"-"`

	ID                                   pulid.ID             `json:"id"                          bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID                       pulid.ID             `json:"businessUnitId"              bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID                       pulid.ID             `json:"organizationId"              bun:"organization_id,pk,notnull,type:VARCHAR(100)"`
	CustomerID                           pulid.ID             `json:"customerId"                  bun:"customer_id,pk,notnull,type:VARCHAR(100)"`
	BillingCycleType                     BillingCycleType     `json:"billingCycleType"            bun:"billing_cycle_type,type:billing_cycle_type_enum,nullzero,default:'Immediate'"`
	BillingCycleDayOfWeek                *int8                `json:"billingCycleDayOfWeek"       bun:"billing_cycle_day_of_week,type:SMALLINT,nullzero"`
	PaymentTerm                          PaymentTerm          `json:"paymentTerm"                 bun:"payment_term,type:payment_term_enum,nullzero,default:'Net30'"`
	HasBillingControlOverrides           bool                 `json:"hasBillingControlOverrides"  bun:"has_billing_control_overrides,type:BOOLEAN,notnull,default:false"`
	CreditLimit                          decimal.NullDecimal  `json:"creditLimit"                 bun:"credit_limit,type:NUMERIC(12,2),nullzero"`
	CreditBalance                        decimal.Decimal      `json:"creditBalance"               bun:"credit_balance,type:NUMERIC(12,2),notnull,default:0"`
	CreditStatus                         CreditStatus         `json:"creditStatus"                bun:"credit_status,type:credit_status_enum,notnull,default:'Active'"`
	EnforceCreditLimit                   bool                 `json:"enforceCreditLimit"          bun:"enforce_credit_limit,type:BOOLEAN,notnull,default:false"`
	AutoCreditHold                       bool                 `json:"autoCreditHold"              bun:"auto_credit_hold,type:BOOLEAN,notnull,default:false"`
	CreditHoldReason                     string               `json:"creditHoldReason"            bun:"credit_hold_reason,type:TEXT,nullzero"`
	InvoiceMethod                        InvoiceMethod        `json:"invoiceMethod"               bun:"invoice_method,type:invoice_method_enum,notnull,default:'Individual'"`
	SummaryTransmitOnGeneration          bool                 `json:"summaryTransmitOnGeneration" bun:"summary_transmit_on_generation,type:BOOLEAN,notnull,default:true"`
	AllowInvoiceConsolidation            bool                 `json:"allowInvoiceConsolidation"   bun:"allow_invoice_consolidation,type:BOOLEAN,notnull,default:false"`
	ConsolidationPeriodDays              int8                 `json:"consolidationPeriodDays"     bun:"consolidation_period_days,type:INTEGER,notnull,default:7"`
	ConsolidationGroupBy                 ConsolidationGroupBy `json:"consolidationGroupBy"        bun:"consolidation_group_by,type:consolidation_group_by_enum,notnull,default:'None'"`
	InvoiceNumberFormat                  InvoiceNumberFormat  `json:"invoiceNumberFormat"         bun:"invoice_number_format,type:invoice_number_format_enum,notnull,default:'Default'"`
	CustomerInvoicePrefix                string               `json:"customerInvoicePrefix"       bun:"customer_invoice_prefix,type:VARCHAR(20),nullzero"`
	InvoiceCopies                        int8                 `json:"invoiceCopies"               bun:"invoice_copies,type:SMALLINT,notnull,default:1"`
	RevenueAccountID                     *pulid.ID            `json:"revenueAccountId"            bun:"revenue_account_id,type:VARCHAR(100),nullzero"`
	ARAccountID                          *pulid.ID            `json:"arAccountId"                 bun:"ar_account_id,type:VARCHAR(100),nullzero"`
	ApplyLateCharges                     bool                 `json:"applyLateCharges"            bun:"apply_late_charges,type:BOOLEAN,notnull,default:false"`
	LateChargeRate                       decimal.NullDecimal  `json:"lateChargeRate"              bun:"late_charge_rate,type:NUMERIC(5,2),nullzero"`
	GracePeriodDays                      int8                 `json:"gracePeriodDays"             bun:"grace_period_days,type:SMALLINT,notnull,default:0"`
	TaxExempt                            bool                 `json:"taxExempt"                   bun:"tax_exempt,type:BOOLEAN,notnull,default:false"`
	TaxExemptNumber                      string               `json:"taxExemptNumber"             bun:"tax_exempt_number,type:VARCHAR(50),nullzero"`
	EnforceCustomerBillingReq            bool                 `json:"enforceCustomerBillingReq"   bun:"enforce_customer_billing_req,type:BOOLEAN,notnull,default:true"`
	ValidateCustomerRates                bool                 `json:"validateCustomerRates"       bun:"validate_customer_rates,type:BOOLEAN,notnull,default:true"`
	AutoTransfer                         bool                 `json:"autoTransfer"                bun:"auto_transfer,type:BOOLEAN,notnull,default:true"`
	AutoMarkReadyToBill                  bool                 `json:"autoMarkReadyToBill"         bun:"auto_mark_ready_to_bill,type:BOOLEAN,notnull,default:true"`
	AutoBill                             bool                 `json:"autoBill"                    bun:"auto_bill,type:BOOLEAN,notnull,default:true"`
	DetentionBillingEnabled              bool                 `json:"detentionBillingEnabled"     bun:"detention_billing_enabled,type:BOOLEAN,notnull,default:false"`
	DetentionFreeMinutes                 int8                 `json:"detentionFreeMinutes"        bun:"detention_free_minutes,type:SMALLINT,notnull,default:120"`
	DetentionRatePerHour                 decimal.NullDecimal  `json:"detentionRatePerHour"        bun:"detention_rate_per_hour,type:NUMERIC(8,2),nullzero"`
	CountLateOnlyOnAppointmentStops      bool                 `json:"countLateOnlyOnAppointmentStops" bun:"count_late_only_on_appointment_stops,type:BOOLEAN,notnull,default:false"`
	CountDetentionOnlyOnAppointmentStops bool                 `json:"countDetentionOnlyOnAppointmentStops" bun:"count_detention_only_on_appointment_stops,type:BOOLEAN,notnull,default:false"`
	AutoApplyAccessorials                bool                 `json:"autoApplyAccessorials"       bun:"auto_apply_accessorials,type:BOOLEAN,notnull,default:true"`
	BillingCurrency                      string               `json:"billingCurrency"             bun:"billing_currency,type:VARCHAR(3),notnull,default:'USD'"`
	RequirePONumber                      bool                 `json:"requirePONumber"             bun:"require_po_number,type:BOOLEAN,notnull,default:false"`
	RequireBOLNumber                     bool                 `json:"requireBOLNumber"            bun:"require_bol_number,type:BOOLEAN,notnull,default:false"`
	RequireDeliveryNumber                bool                 `json:"requireDeliveryNumber"       bun:"require_delivery_number,type:BOOLEAN,notnull,default:false"`
	BillingNotes                         string               `json:"billingNotes"                bun:"billing_notes,type:TEXT,nullzero"`
	// UseFactoring                bool                 `json:"useFactoring"                bun:"use_factoring,type:BOOLEAN,notnull,default:false"`
	// FuelSurchargeMethod         FuelSurchargeMethod  `json:"fuelSurchargeMethod"         bun:"fuel_surcharge_method,type:fuel_surcharge_method_enum,notnull,default:'None'"`

	// FuelSurchargeProfileID      *pulid.ID            `json:"fuelSurchargeProfileId"      bun:"fuel_surcharge_profile_id,type:VARCHAR(100),nullzero"`
	// FactoringCompanyID          *pulid.ID            `json:"factoringCompanyId"          bun:"factoring_company_id,type:VARCHAR(100),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit   *tenant.BusinessUnit         `json:"-"              bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization   *tenant.Organization         `json:"-"              bun:"rel:belongs-to,join:organization_id=id"`
	RevenueAccount *glaccount.GLAccount         `json:"revenueAccount" bun:"rel:belongs-to,join:revenue_account_id=id"`
	ARAccount      *glaccount.GLAccount         `json:"arAccount"      bun:"rel:belongs-to,join:ar_account_id=id"`
	DocumentTypes  []*documenttype.DocumentType `json:"documentTypes"  bun:"m2m:customer_billing_profile_document_types,join:BillingProfile=DocumentType"`
	// FuelSurchargeProfile *FuelSurchargeProfile        `json:"fuelSurchargeProfile" bun:"rel:belongs-to,join:fuel_surcharge_profile_id=id"`
	// FactoringCompany     *FactoringCompany            `json:"factoringCompany" bun:"rel:belongs-to,join:factoring_company_id=id"`
}

func (b *CustomerBillingProfile) GetID() string {
	return b.ID.String()
}

func NewDefaultBillingProfile(orgID, buID, customerID pulid.ID) *CustomerBillingProfile {
	return &CustomerBillingProfile{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		CustomerID:     customerID,
	}
}

func (b *CustomerBillingProfile) GetTableName() string {
	return "customer_billing_profiles"
}

func (b *CustomerBillingProfile) GetOrganizationID() pulid.ID {
	return b.OrganizationID
}

func (b *CustomerBillingProfile) GetBusinessUnitID() pulid.ID {
	return b.BusinessUnitID
}

func (b *CustomerBillingProfile) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if b.ID.IsNil() {
			b.ID = pulid.MustNew("cbp_")
		}
		b.CreatedAt = now
	case *bun.UpdateQuery:
		b.UpdatedAt = now
	}

	return nil
}
