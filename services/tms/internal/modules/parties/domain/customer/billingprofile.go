package customer

import (
	"context"
	"errors"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/internal/modules/rating/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

const defaultBillingCycleTimezone = "UTC"

const (
	maxInvoicePrefixLength   = 20
	maxTaxExemptNumberLength = 50
	currencyCodeLength       = 3
)

type CustomerBillingProfile struct {
	keel.Table `keel:"customer_billing_profiles"`

	ID                         pulid.ID            `json:"id"                         keel:"id,pk,autoid:cbp_"`
	BusinessUnitID             pulid.ID            `json:"businessUnitId"             keel:"business_unit_id,pk,tenant"`
	OrganizationID             pulid.ID            `json:"organizationId"             keel:"organization_id,pk,tenant"`
	CustomerID                 pulid.ID            `json:"customerId"                 keel:"customer_id,pk"`
	InvoiceDelivery            InvoiceDelivery     `json:"invoiceDelivery"            keel:"invoice_delivery"`
	BillingCycle               BillingCycle        `json:"billingCycle"               keel:"billing_cycle"`
	BillingCycleAnchorDay      int16               `json:"billingCycleAnchorDay"      keel:"billing_cycle_anchor_day"`
	BillingCycleTimezone       string              `json:"billingCycleTimezone"       keel:"billing_cycle_timezone"`
	LastBilledPeriodEnd        *int64              `json:"lastBilledPeriodEnd"        keel:"last_billed_period_end"`
	PaymentTerm                PaymentTerm         `json:"paymentTerm"                keel:"payment_term,nullzero"`
	HasBillingControlOverrides bool                `json:"hasBillingControlOverrides" keel:"has_billing_control_overrides"`
	CreditLimit                decimal.NullDecimal `json:"creditLimit"                keel:"credit_limit"`
	CreditBalance              decimal.Decimal     `json:"creditBalance"              keel:"credit_balance"`
	CreditStatus               CreditStatus        `json:"creditStatus"               keel:"credit_status"`
	EnforceCreditLimit         bool                `json:"enforceCreditLimit"         keel:"enforce_credit_limit"`
	AutoCreditHold             bool                `json:"autoCreditHold"             keel:"auto_credit_hold"`
	CreditHoldReason           string              `json:"creditHoldReason"           keel:"credit_hold_reason,nullzero"`
	// CreditHoldAutomatic marks a Hold or Warning the credit review placed
	// against open receivables; only the review lifts it.
	CreditHoldAutomatic bool `json:"creditHoldAutomatic" keel:"credit_hold_automatic"`
	// CreditReviewedAt is when CreditBalance, the open receivables, was last
	// computed by the credit review.
	CreditReviewedAt            *int64 `json:"creditReviewedAt"            keel:"credit_reviewed_at"`
	AutoSendInvoiceOnGeneration bool   `json:"autoSendInvoiceOnGeneration" keel:"auto_send_invoice_on_generation"`
	// SendPaymentReminders lets the organization's payment reminders reach
	// this customer; off for a customer who must never be dunned by email.
	SendPaymentReminders bool `json:"sendPaymentReminders" keel:"send_payment_reminders"`
	// SendStatements lets the organization's monthly statements reach this
	// customer.
	SendStatements bool `json:"sendStatements" keel:"send_statements"`
	// EmailInvoiceEnabled and EDIInvoiceEnabled are the delivery channels an
	// invoice may leave through. Auto-send after posting honours both.
	EmailInvoiceEnabled        bool                `json:"emailInvoiceEnabled"        keel:"email_invoice_enabled"`
	EDIInvoiceEnabled          bool                `json:"ediInvoiceEnabled"          keel:"edi_invoice_enabled"`
	SplitBy                    InvoiceSplitKey     `json:"splitBy"                    keel:"split_by"`
	SectionBy                  InvoiceSectionKey   `json:"sectionBy"                  keel:"section_by"`
	InvoiceDetail              InvoiceDetail       `json:"invoiceDetail"              keel:"invoice_detail"`
	MinConsolidatedAmount      decimal.NullDecimal `json:"minConsolidatedAmount"      keel:"min_consolidated_amount"`
	MinConsolidatedAmountMinor *int64              `json:"minConsolidatedAmountMinor" keel:"min_consolidated_amount_minor"`
	MaxShipmentsPerInvoice     int16               `json:"maxShipmentsPerInvoice"     keel:"max_shipments_per_invoice"`
	InvoiceNumberFormat        InvoiceNumberFormat `json:"invoiceNumberFormat"        keel:"invoice_number_format"`
	CustomerInvoicePrefix      string              `json:"customerInvoicePrefix"      keel:"customer_invoice_prefix,nullzero"`
	InvoiceCopies              int16               `json:"invoiceCopies"              keel:"invoice_copies"`
	RevenueAccountID           *pulid.ID           `json:"revenueAccountId"           keel:"revenue_account_id"`
	ARAccountID                *pulid.ID           `json:"arAccountId"                keel:"ar_account_id"`
	ApplyLateCharges           bool                `json:"applyLateCharges"           keel:"apply_late_charges"`
	LateChargeRate             decimal.NullDecimal `json:"lateChargeRate"             keel:"late_charge_rate"`
	GracePeriodDays            int16               `json:"gracePeriodDays"            keel:"grace_period_days"`
	TaxExempt                  bool                `json:"taxExempt"                  keel:"tax_exempt"`
	TaxExemptNumber            string              `json:"taxExemptNumber"            keel:"tax_exempt_number,nullzero"`
	EnforceCustomerBillingReq  bool                `json:"enforceCustomerBillingReq"  keel:"enforce_customer_billing_req"`
	ValidateCustomerRates      bool                `json:"validateCustomerRates"      keel:"validate_customer_rates"`
	AutoTransfer               bool                `json:"autoTransfer"               keel:"auto_transfer"`
	AutoMarkReadyToBill        bool                `json:"autoMarkReadyToBill"        keel:"auto_mark_ready_to_bill"`
	AutoBill                   bool                `json:"autoBill"                   keel:"auto_bill"`
	// AutoApprove lets a shipment that passes every billing requirement clear the
	// billing queue without a biller clicking Approve, leaving the queue holding
	// only the freight that actually needs a human.
	//
	// It is reachable only on the automatic transfer path, so an organization that
	// has not enabled automatic queue transfer cannot be auto-approving anything.
	AutoApprove                     bool              `json:"autoApprove"                     keel:"auto_approve"`
	CountLateOnlyOnAppointmentStops bool              `json:"countLateOnlyOnAppointmentStops" keel:"count_late_only_on_appointment_stops"`
	AutoApplyAccessorials           bool              `json:"autoApplyAccessorials"           keel:"auto_apply_accessorials"`
	BillingCurrency                 string            `json:"billingCurrency"                 keel:"billing_currency"`
	RequirePONumber                 bool              `json:"requirePONumber"                 keel:"require_po_number"`
	RequireBOLNumber                bool              `json:"requireBOLNumber"                keel:"require_bol_number"`
	RequireDeliveryNumber           bool              `json:"requireDeliveryNumber"           keel:"require_delivery_number"`
	DefaultBillerID                 *pulid.ID         `json:"defaultBillerId"                 keel:"default_biller_id"`
	BillingNotes                    string            `json:"billingNotes"                    keel:"billing_notes,nullzero"`
	FuelSurchargeMode               FuelSurchargeMode `json:"fuelSurchargeMode"               keel:"fuel_surcharge_mode"`
	FuelSurchargeProgramID          *pulid.ID         `json:"fuelSurchargeProgramId"          keel:"fuel_surcharge_program_id"`
	// FactoringCompanyID names the factor this customer's invoices are
	// assigned to: they post remitted to it, with its notice of assignment.
	FactoringCompanyID *pulid.ID `json:"factoringCompanyId" keel:"factoring_company_id"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	// Relationships
	BusinessUnit         *tenant.BusinessUnit                `json:"-"`
	Organization         *tenant.Organization                `json:"-"`
	DefaultBiller        *tenant.User                        `json:"defaultBiller"                  keel:"belongs:default_biller_id"`
	RevenueAccount       *glaccount.GLAccount                `json:"revenueAccount"                 keel:"belongs:revenue_account_id"`
	ARAccount            *glaccount.GLAccount                `json:"arAccount"                      keel:"belongs:ar_account_id"`
	DocumentTypes        []*documenttype.DocumentType        `json:"documentTypes"                  keel:"m2m:customer_billing_profile_document_types.billing_profile_id.document_type_id"`
	FuelSurchargeProgram *fuelsurcharge.FuelSurchargeProgram `json:"fuelSurchargeProgram,omitempty" keel:"belongs:fuel_surcharge_program_id"`
	FactoringCompany     *factoringcompany.FactoringCompany  `json:"factoringCompany,omitempty"     keel:"belongs:factoring_company_id"`
}

func (b *CustomerBillingProfile) BeforeUpdate(_ context.Context) error {
	b.applyDefaults()

	return nil
}

func (b *CustomerBillingProfile) BeforeCreate(_ context.Context) error {
	b.applyDefaults()
	if b.InvoiceDelivery == "" {
		b.InvoiceDelivery = InvoiceDeliveryPerShipment
	}
	if b.BillingCycle == "" {
		b.BillingCycle = BillingCycleImmediate
	}
	if b.BillingCycleAnchorDay == 0 {
		b.BillingCycleAnchorDay = 1
	}
	if b.BillingCycleTimezone == "" {
		b.BillingCycleTimezone = defaultBillingCycleTimezone
	}
	if b.PaymentTerm == "" {
		b.PaymentTerm = PaymentTermNet30
	}
	if b.CreditStatus == "" {
		b.CreditStatus = CreditStatusActive
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
	if b.InvoiceNumberFormat == "" {
		b.InvoiceNumberFormat = InvoiceNumberFormatDefault
	}
	if b.InvoiceCopies == 0 {
		b.InvoiceCopies = 1
	}
	if b.BillingCurrency == "" {
		b.BillingCurrency = "USD"
	}
	if b.FuelSurchargeMode == "" {
		b.FuelSurchargeMode = FuelSurchargeModeNone
	}

	return nil
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

func NewDefaultBillingProfile(orgID, buID, customerID pulid.ID) *CustomerBillingProfile {
	return &CustomerBillingProfile{
		OrganizationID:              orgID,
		BusinessUnitID:              buID,
		CustomerID:                  customerID,
		AutoSendInvoiceOnGeneration: true,
		SendPaymentReminders:        true,
		SendStatements:              true,
		EmailInvoiceEnabled:         true,
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
		BillingCycleTimezone:        defaultBillingCycleTimezone,
		SplitBy:                     InvoiceSplitKeyCustomer,
		SectionBy:                   InvoiceSectionKeyShipment,
		InvoiceDetail:               InvoiceDetailDetailed,
		PaymentTerm:                 PaymentTermNet30,
		CreditStatus:                CreditStatusActive,
		InvoiceNumberFormat:         InvoiceNumberFormatDefault,
		InvoiceCopies:               1,
		BillingCurrency:             "USD",
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

func (b *CustomerBillingProfile) BeforeSave(context.Context) error {
	b.applyDefaults()

	return nil
}

func (b *CustomerBillingProfile) applyDefaults() {
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
		b.BillingCycleTimezone = defaultBillingCycleTimezone
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
}
