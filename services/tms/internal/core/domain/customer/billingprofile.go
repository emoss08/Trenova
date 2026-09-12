package customer

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*CustomerBillingProfile)(nil)

const (
	maxInvoicePrefixLength   = 20
	maxTaxExemptNumberLength = 50
	currencyCodeLength       = 3

	defaultConsolidationLookbackDays = 30
)

type CustomerBillingProfile struct {
	bun.BaseModel `bun:"table:customer_billing_profiles,alias:cbp" json:"-"`

	ID                                        pulid.ID                                  `json:"id"                                        bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID                            pulid.ID                                  `json:"businessUnitId"                            bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID                            pulid.ID                                  `json:"organizationId"                            bun:"organization_id,pk,notnull,type:VARCHAR(100)"`
	CustomerID                                pulid.ID                                  `json:"customerId"                                bun:"customer_id,pk,notnull,type:VARCHAR(100)"`
	InvoiceDelivery                           InvoiceDelivery                           `json:"invoiceDelivery"                           bun:"invoice_delivery,type:customer_invoice_delivery_enum,notnull,default:'PerShipment'"`
	BillingCycle                              BillingCycle                              `json:"billingCycle"                              bun:"billing_cycle,type:customer_billing_cycle_enum,notnull,default:'Immediate'"`
	BillingCycleAnchorDay                     int16                                     `json:"billingCycleAnchorDay"                     bun:"billing_cycle_anchor_day,type:SMALLINT,notnull,default:1"`
	BillingCycleTimezone                      string                                    `json:"billingCycleTimezone"                      bun:"billing_cycle_timezone,type:VARCHAR(64),notnull,default:'UTC'"`
	LastBilledPeriodEnd                       *int64                                    `json:"lastBilledPeriodEnd"                       bun:"last_billed_period_end,type:BIGINT,nullzero"`
	PaymentTerm                               PaymentTerm                               `json:"paymentTerm"                               bun:"payment_term,type:payment_term_enum,nullzero,default:'Net30'"`
	HasBillingControlOverrides                bool                                      `json:"hasBillingControlOverrides"                bun:"has_billing_control_overrides,type:BOOLEAN,notnull"`
	CreditLimit                               decimal.NullDecimal                       `json:"creditLimit"                               bun:"credit_limit,type:NUMERIC(12,2),nullzero"`
	CreditBalance                             decimal.Decimal                           `json:"creditBalance"                             bun:"credit_balance,type:NUMERIC(12,2),notnull,default:0"`
	CreditStatus                              CreditStatus                              `json:"creditStatus"                              bun:"credit_status,type:credit_status_enum,notnull,default:'Active'"`
	EnforceCreditLimit                        bool                                      `json:"enforceCreditLimit"                        bun:"enforce_credit_limit,type:BOOLEAN,notnull"`
	AutoCreditHold                            bool                                      `json:"autoCreditHold"                            bun:"auto_credit_hold,type:BOOLEAN,notnull"`
	CreditHoldReason                          string                                    `json:"creditHoldReason"                          bun:"credit_hold_reason,type:TEXT,nullzero"`
	AutoSendInvoiceOnGeneration               bool                                      `json:"autoSendInvoiceOnGeneration"               bun:"auto_send_invoice_on_generation,type:BOOLEAN,notnull"`
	SplitBy                                   InvoiceSplitKey                           `json:"splitBy"                                   bun:"split_by,type:invoice_split_key_enum,notnull,default:'Customer'"`
	SectionBy                                 InvoiceSectionKey                         `json:"sectionBy"                                 bun:"section_by,type:invoice_section_key_enum,notnull,default:'Shipment'"`
	InvoiceDetail                             InvoiceDetail                             `json:"invoiceDetail"                             bun:"invoice_detail,type:invoice_detail_enum,notnull,default:'Detailed'"`
	ConsolidationLookbackDays                 int16                                     `json:"consolidationLookbackDays"                 bun:"consolidation_lookback_days,type:SMALLINT,notnull,default:30"`
	MinConsolidatedAmount                     decimal.NullDecimal                       `json:"minConsolidatedAmount"                     bun:"min_consolidated_amount,type:NUMERIC(19,4),nullzero"`
	MinConsolidatedAmountMinor                *int64                                    `json:"minConsolidatedAmountMinor"                bun:"min_consolidated_amount_minor,type:BIGINT,nullzero"`
	MaxShipmentsPerInvoice                    int16                                     `json:"maxShipmentsPerInvoice"                    bun:"max_shipments_per_invoice,type:SMALLINT,notnull"`
	InvoiceNumberFormat                       InvoiceNumberFormat                       `json:"invoiceNumberFormat"                       bun:"invoice_number_format,type:invoice_number_format_enum,notnull,default:'Default'"`
	CustomerInvoicePrefix                     string                                    `json:"customerInvoicePrefix"                     bun:"customer_invoice_prefix,type:VARCHAR(20),nullzero"`
	InvoiceCopies                             int8                                      `json:"invoiceCopies"                             bun:"invoice_copies,type:SMALLINT,notnull,default:1"`
	RevenueAccountID                          *pulid.ID                                 `json:"revenueAccountId"                          bun:"revenue_account_id,type:VARCHAR(100),nullzero"`
	ARAccountID                               *pulid.ID                                 `json:"arAccountId"                               bun:"ar_account_id,type:VARCHAR(100),nullzero"`
	ApplyLateCharges                          bool                                      `json:"applyLateCharges"                          bun:"apply_late_charges,type:BOOLEAN,notnull"`
	LateChargeRate                            decimal.NullDecimal                       `json:"lateChargeRate"                            bun:"late_charge_rate,type:NUMERIC(5,2),nullzero"`
	GracePeriodDays                           int8                                      `json:"gracePeriodDays"                           bun:"grace_period_days,type:SMALLINT,notnull"`
	TaxExempt                                 bool                                      `json:"taxExempt"                                 bun:"tax_exempt,type:BOOLEAN,notnull"`
	TaxExemptNumber                           string                                    `json:"taxExemptNumber"                           bun:"tax_exempt_number,type:VARCHAR(50),nullzero"`
	EnforceCustomerBillingReq                 bool                                      `json:"enforceCustomerBillingReq"                 bun:"enforce_customer_billing_req,type:BOOLEAN,notnull"`
	ValidateCustomerRates                     bool                                      `json:"validateCustomerRates"                     bun:"validate_customer_rates,type:BOOLEAN,notnull"`
	AutoTransfer                              bool                                      `json:"autoTransfer"                              bun:"auto_transfer,type:BOOLEAN,notnull"`
	AutoMarkReadyToBill                       bool                                      `json:"autoMarkReadyToBill"                       bun:"auto_mark_ready_to_bill,type:BOOLEAN,notnull"`
	AutoBill                                  bool                                      `json:"autoBill"                                  bun:"auto_bill,type:BOOLEAN,notnull"`
	CountLateOnlyOnAppointmentStops           bool                                      `json:"countLateOnlyOnAppointmentStops"           bun:"count_late_only_on_appointment_stops,type:BOOLEAN,notnull"`
	AutoApplyAccessorials                     bool                                      `json:"autoApplyAccessorials"                     bun:"auto_apply_accessorials,type:BOOLEAN,notnull"`
	BillingCurrency                           string                                    `json:"billingCurrency"                           bun:"billing_currency,type:VARCHAR(3),notnull,default:'USD'"`
	RequirePONumber                           bool                                      `json:"requirePONumber"                           bun:"require_po_number,type:BOOLEAN,notnull"`
	RequireBOLNumber                          bool                                      `json:"requireBOLNumber"                          bun:"require_bol_number,type:BOOLEAN,notnull"`
	RequireDeliveryNumber                     bool                                      `json:"requireDeliveryNumber"                     bun:"require_delivery_number,type:BOOLEAN,notnull"`
	InvoiceAdjustmentSupportingDocumentPolicy InvoiceAdjustmentSupportingDocumentPolicy `json:"invoiceAdjustmentSupportingDocumentPolicy" bun:"invoice_adjustment_supporting_document_policy,type:invoice_adjustment_supporting_document_policy_enum,notnull,default:'Inherit'"`
	DefaultBillerID                           *pulid.ID                                 `json:"defaultBillerId"                           bun:"default_biller_id,type:VARCHAR(100),nullzero"`
	BillingNotes                              string                                    `json:"billingNotes"                              bun:"billing_notes,type:TEXT,nullzero"`
	FuelSurchargeMode                         FuelSurchargeMode                         `json:"fuelSurchargeMode"                         bun:"fuel_surcharge_mode,type:customer_fuel_surcharge_mode_enum,notnull,default:'None'"`
	FuelSurchargeProgramID                    *pulid.ID                                 `json:"fuelSurchargeProgramId"                    bun:"fuel_surcharge_program_id,type:VARCHAR(100),nullzero"`
	// UseFactoring                bool                 `json:"useFactoring"                bun:"use_factoring,type:BOOLEAN,notnull,default:false"`
	// FactoringCompanyID          *pulid.ID            `json:"factoringCompanyId"          bun:"factoring_company_id,type:VARCHAR(100),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit         *tenant.BusinessUnit                `json:"-"                              bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization         *tenant.Organization                `json:"-"                              bun:"rel:belongs-to,join:organization_id=id"`
	DefaultBiller        *tenant.User                        `json:"defaultBiller"                  bun:"rel:belongs-to,join:default_biller_id=id"`
	RevenueAccount       *glaccount.GLAccount                `json:"revenueAccount"                 bun:"rel:belongs-to,join:revenue_account_id=id"`
	ARAccount            *glaccount.GLAccount                `json:"arAccount"                      bun:"rel:belongs-to,join:ar_account_id=id"`
	DocumentTypes        []*documenttype.DocumentType        `json:"documentTypes"                  bun:"m2m:customer_billing_profile_document_types,join:BillingProfile=DocumentType"`
	FuelSurchargeProgram *fuelsurcharge.FuelSurchargeProgram `json:"fuelSurchargeProgram,omitempty" bun:"rel:belongs-to,join:fuel_surcharge_program_id=id"`
	// FactoringCompany     *FactoringCompany            `json:"factoringCompany" bun:"rel:belongs-to,join:factoring_company_id=id"`
}

func (b *CustomerBillingProfile) GetID() string {
	return b.ID.String()
}

func (b *CustomerBillingProfile) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(b,
		validation.Field(&b.InvoiceDelivery,
			validation.Required.Error("Invoice delivery is required"),
			domainvalidation.ValidEnum[InvoiceDelivery]("Invoice delivery is invalid"),
		),
		validation.Field(&b.BillingCycle,
			validation.Required.Error("Billing cycle is required"),
			domainvalidation.ValidEnum[BillingCycle]("Billing cycle is invalid"),
		),
		// The anchor means a weekday for the weekly cycles and a day of the month
		// for the rest. Capped at 28 so a monthly anchor never silently shifts in
		// February.
		validation.Field(&b.BillingCycleAnchorDay,
			validation.When(
				b.BillingCycle == BillingCycleWeekly || b.BillingCycle == BillingCycleBiWeekly,
				validation.Min(int16(0)).Error("Day of week must be between 0 and 6"),
				validation.Max(int16(6)).Error("Day of week must be between 0 and 6"),
			),
			validation.When(
				b.BillingCycle == BillingCycleSemiMonthly ||
					b.BillingCycle == BillingCycleMonthly ||
					b.BillingCycle == BillingCycleQuarterly,
				validation.Min(int16(1)).Error("Day of month must be between 1 and 28"),
				validation.Max(int16(28)).Error("Day of month must be between 1 and 28"),
			),
		),
		// A period boundary evaluated in the wrong zone moves loads into the wrong
		// month, so the zone is required rather than defaulted at read time.
		validation.Field(&b.BillingCycleTimezone,
			validation.Required.Error("Billing cycle timezone is required"),
			validation.By(validTimezone),
		),
		validation.Field(&b.SplitBy,
			validation.Required.Error("Invoice split key is required"),
			domainvalidation.ValidEnum[InvoiceSplitKey]("Invoice split key is invalid"),
		),
		validation.Field(&b.SectionBy,
			validation.Required.Error("Invoice section key is required"),
			domainvalidation.ValidEnum[InvoiceSectionKey]("Invoice section key is invalid"),
		),
		validation.Field(&b.InvoiceDetail,
			validation.Required.Error("Invoice detail is required"),
			domainvalidation.ValidEnum[InvoiceDetail]("Invoice detail is invalid"),
		),
		validation.Field(&b.ConsolidationLookbackDays,
			validation.Min(int16(0)).Error("Lookback cannot be negative"),
			validation.Max(int16(365)).Error("Lookback cannot exceed a year"),
		),
		// Zero means unbounded. The ceiling is what one PDF and one EDI 210 can
		// carry without becoming unusable.
		validation.Field(&b.MaxShipmentsPerInvoice,
			validation.Min(int16(0)).Error("Maximum shipments per invoice cannot be negative"),
			validation.Max(int16(500)).Error("Maximum shipments per invoice cannot exceed 500"),
		),
		validation.Field(&b.PaymentTerm,
			domainvalidation.ValidEnum[PaymentTerm]("Payment term is invalid"),
		),
		validation.Field(&b.CreditStatus,
			domainvalidation.ValidEnum[CreditStatus]("Credit status is invalid"),
		),
		validation.Field(&b.InvoiceNumberFormat,
			domainvalidation.ValidEnum[InvoiceNumberFormat]("Invoice number format is invalid"),
		),
		// A custom prefix format with no prefix would silently fall back to the
		// default numbering, which is the opposite of what was asked for.
		validation.Field(&b.CustomerInvoicePrefix,
			validation.When(
				b.InvoiceNumberFormat == InvoiceNumberFormatCustomPrefix,
				validation.Required.Error("Enter the invoice prefix to use for this customer"),
			),
			validation.Length(0, maxInvoicePrefixLength).
				Error("Invoice prefix cannot be longer than 20 characters"),
		),
		validation.Field(&b.InvoiceCopies,
			validation.Min(int8(1)).Error("At least one invoice copy is required"),
		),
		validation.Field(&b.GracePeriodDays,
			validation.Min(int8(0)).Error("Grace period cannot be negative"),
		),
		validation.Field(&b.TaxExemptNumber,
			validation.When(
				b.TaxExempt,
				validation.Required.Error("A tax exempt customer must record its exemption number"),
			),
			validation.Length(0, maxTaxExemptNumberLength).
				Error("Tax exempt number cannot be longer than 50 characters"),
		),
		validation.Field(&b.BillingCurrency,
			validation.Required.Error("Billing currency is required"),
			validation.Length(currencyCodeLength, currencyCodeLength).
				Error("Billing currency must be a three letter code"),
		),
		validation.Field(&b.InvoiceAdjustmentSupportingDocumentPolicy,
			domainvalidation.ValidEnum[InvoiceAdjustmentSupportingDocumentPolicy](
				"Supporting document policy is invalid",
			),
		),
		validation.Field(&b.FuelSurchargeMode,
			validation.Required.Error("Fuel surcharge mode is required"),
			domainvalidation.ValidEnum[FuelSurchargeMode]("Fuel surcharge mode is invalid"),
		),
		validation.Field(&b.FuelSurchargeProgramID,
			validation.When(
				b.FuelSurchargeMode == FuelSurchargeModeProgram,
				validation.Required.Error(
					"Select the fuel surcharge program to apply for this customer",
				),
			),
		),
	))

	// The old model let a customer be weekly and per-shipment at once, which is
	// two different answers to the same question. Neither half is meaningful
	// without the other.
	if b.InvoiceDelivery == InvoiceDeliveryConsolidated && !b.BillingCycle.IsPeriodic() {
		multiErr.Add(
			"billingCycle",
			errortypes.ErrInvalid,
			"A statement-billed customer needs a billing cycle longer than Immediate",
		)
	}
	if b.InvoiceDelivery != InvoiceDeliveryConsolidated && b.BillingCycle.IsPeriodic() {
		multiErr.Add(
			"invoiceDelivery",
			errortypes.ErrInvalid,
			"A billing cycle longer than Immediate only applies to statement billing",
		)
	}
}

func validTimezone(value any) error {
	name, ok := value.(string)
	if !ok || name == "" {
		return nil
	}
	if !timeutils.IsValidLocation(name) {
		return errors.New("Billing cycle timezone must be a valid IANA time zone")
	}

	return nil
}

// IsStatementBilled reports whether this customer's freight accumulates onto a
// periodic statement rather than being invoiced as it is approved.
//
// Both halves are required because either alone is meaningless: a consolidated
// customer on an Immediate cycle has no period to accumulate into, and a
// per-shipment customer on a monthly cycle has nothing to accumulate. The
// profile's cross-field validation makes both combinations unreachable; this
// method states the invariant at every read site rather than repeating it.
func (b *CustomerBillingProfile) IsStatementBilled() bool {
	return b.InvoiceDelivery == InvoiceDeliveryConsolidated && b.BillingCycle.IsPeriodic()
}

func (b *CustomerBillingProfile) AppliesFuelSurcharge() bool {
	return b.FuelSurchargeMode == FuelSurchargeModeProgram &&
		b.FuelSurchargeProgramID != nil && !b.FuelSurchargeProgramID.IsNil()
}

// NewDefaultBillingProfile is the profile a customer gets when one is not
// supplied. Every billing automation it turns on is named here rather than left
// to a column default: the fields below carry no bun default, so what this
// constructor omits is stored false.
func NewDefaultBillingProfile(orgID, buID, customerID pulid.ID) *CustomerBillingProfile {
	return &CustomerBillingProfile{
		OrganizationID:              orgID,
		BusinessUnitID:              buID,
		CustomerID:                  customerID,
		AutoSendInvoiceOnGeneration: true,
		EnforceCustomerBillingReq:   true,
		ValidateCustomerRates:       true,
		AutoTransfer:                true,
		AutoMarkReadyToBill:         true,
		AutoBill:                    true,
		AutoApplyAccessorials:       true,
		FuelSurchargeMode:           FuelSurchargeModeNone,
		InvoiceDelivery:             InvoiceDeliveryPerShipment,
		BillingCycle:                BillingCycleImmediate,
		BillingCycleAnchorDay:       1,
		BillingCycleTimezone:        "UTC",
		SplitBy:                     InvoiceSplitKeyCustomer,
		SectionBy:                   InvoiceSectionKeyShipment,
		InvoiceDetail:               InvoiceDetailDetailed,
		ConsolidationLookbackDays:   defaultConsolidationLookbackDays,
		InvoiceAdjustmentSupportingDocumentPolicy: InvoiceAdjustmentSupportingDocumentPolicyInherit,
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
	if b.InvoiceAdjustmentSupportingDocumentPolicy == "" {
		b.InvoiceAdjustmentSupportingDocumentPolicy = InvoiceAdjustmentSupportingDocumentPolicyInherit
	}
	if b.FuelSurchargeMode == "" {
		b.FuelSurchargeMode = FuelSurchargeModeNone
	}
	if b.InvoiceDelivery == "" {
		b.InvoiceDelivery = InvoiceDeliveryPerShipment
	}
	if b.BillingCycle == "" {
		b.BillingCycle = BillingCycleImmediate
	}
	if b.BillingCycleTimezone == "" {
		b.BillingCycleTimezone = "UTC"
	}
	if b.SplitBy == "" {
		b.SplitBy = InvoiceSplitKeyCustomer
	}
	if b.SectionBy == "" {
		b.SectionBy = InvoiceSectionKeyShipment
	}
	if b.InvoiceDetail == "" {
		b.InvoiceDetail = InvoiceDetailDetailed
	}
	if b.FuelSurchargeMode != FuelSurchargeModeProgram {
		b.FuelSurchargeProgramID = nil
	}

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
