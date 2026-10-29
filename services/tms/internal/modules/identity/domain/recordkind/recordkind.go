package recordkind

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/textcase"
)

// Kind is one kind of record with one id prefix. A resource whose
// records are all one kind is that kind under its own name (OfResource);
// the kinds declared here are the ones a resource's name cannot stand for: the
// verifications and clearinghouse queries a qualification covers, the pools,
// draws and selections of DOT random testing, and the lines, days and items
// held inside a record of another resource.
//
// A tool parameter that takes an id of one of these, or of any of several
// kinds, names the kinds it takes, and the runtime refuses an id whose prefix
// is none of theirs before the tool reads it.
type Kind string

const (
	EmploymentVerification   Kind = "employment_verification"
	ClearinghouseQuery       Kind = "clearinghouse_query"
	DOTRandomPool            Kind = "dot_random_pool"
	DOTRandomDraw            Kind = "dot_random_draw"
	DOTRandomSelection       Kind = "dot_random_selection"
	AssistantThread          Kind = "assistant_thread"
	Wait                     Kind = "wait"
	PortalSite               Kind = "portal_site"
	PortalRun                Kind = "portal_run"
	AnalysisRun              Kind = "analysis_run"
	WebWatch                 Kind = "web_watch"
	WatchtowerItem           Kind = "watchtower_item"
	AccountingConnection     Kind = "accounting_connection"
	AccountingMapping        Kind = "accounting_mapping"
	AccountingSyncRecord     Kind = "accounting_sync_record"
	AccountingInboundChange  Kind = "accounting_inbound_change"
	AccountingDriftFinding   Kind = "accounting_drift_finding"
	CarrierIntelligenceEvent Kind = "carrier_intelligence_event"
	InsurancePolicy          Kind = "insurance_policy"
	CarrierAssignment        Kind = "carrier_assignment"
	AdditionalCharge         Kind = "additional_charge"
	BillingTransferRun       Kind = "billing_transfer_run"
	DetentionOccurrence      Kind = "detention_occurrence"
	TenderOffer              Kind = "tender_offer"
	OrderCharge              Kind = "order_charge"
	BillingQueueItem         Kind = "billing_queue_item"
	InvoiceLine              Kind = "invoice_line"
	InvoiceAdjustment        Kind = "invoice_adjustment"
	InvoiceRunGroup          Kind = "invoice_run_group"
	InvoiceRunItem           Kind = "invoice_run_item"
	CreditMemoApplication    Kind = "credit_memo_application" //nolint:gosec // G101: a record kind, not a credential
	RateAgreementLane        Kind = "rate_agreement_lane"
	RateImport               Kind = "rate_import"
	FuelIndex                Kind = "fuel_index"
	FuelIndexPrice           Kind = "fuel_index_price"
	ReportRun                Kind = "report_run"
	ReportSchedule           Kind = "report_schedule"
	EDIPartner               Kind = "edi_partner"
	EDIInboundFile           Kind = "edi_inbound_file"
	EDITransfer              Kind = "edi_transfer"
	EDIMessage               Kind = "edi_message"
	EDIShipmentLink          Kind = "edi_shipment_link"
	EDITenderChange          Kind = "edi_tender_change"
	EDITransferChange        Kind = "edi_transfer_change"
	EDICarrierInvoice        Kind = "edi_carrier_invoice"
	ScannedDocument          Kind = "scanned_document"
	IFTAJurisdiction         Kind = "ifta_jurisdiction"
	IFTAReturn               Kind = "ifta_return"
	IFTAMileageEntry         Kind = "ifta_mileage_entry"
	USState                  Kind = "us_state"
	PayEvent                 Kind = "pay_event"
	PayProfile               Kind = "pay_profile"
	PayAssignment            Kind = "pay_assignment"
	PayrollExport            Kind = "payroll_export"
	ShiftAssignment          Kind = "shift_assignment"
	ChecklistItem            Kind = "checklist_item"
	SafetyViolation          Kind = "safety_violation"
	LeaveDay                 Kind = "leave_day"
	AIAuditEvent             Kind = "ai_audit_event"
	AIAuditExport            Kind = "ai_audit_export"
)

type recordKindSpec struct {
	prefix   string
	resource permission.Resource
	noun     string
}

// recordKinds holds each declared kind to the prefix the domain mints for it,
// the resource whose permission governs it, and what one is called.
var recordKinds = map[Kind]recordKindSpec{
	EmploymentVerification: {"wemv_", permission.ResourceQualification, "employment verification"},
	ClearinghouseQuery:     {"wchq_", permission.ResourceQualification, "clearinghouse query"},
	DOTRandomPool:          {"drpool_", permission.ResourceDOTRandomPool, "DOT random pool"},
	DOTRandomDraw:          {"drdraw_", permission.ResourceDOTRandomPool, "DOT random draw"},
	DOTRandomSelection:     {"drde_", permission.ResourceDOTRandomPool, "DOT random selection"},
	AssistantThread:        {"athr_", permission.ResourceAssistant, "conversation"},
	Wait:                   {"awt_", permission.ResourceAgentRun, "wait"},
	PortalSite:             {"apst_", permission.ResourceAgentExtension, "portal site"},
	PortalRun:              {"aprn_", permission.ResourceAgentExtension, "portal task"},
	AnalysisRun:            {"aanr_", permission.ResourceAgentRun, "analysis"},
	WebWatch:               {"awwt_", permission.ResourceAgentExtension, "web watch"},
	WatchtowerItem:         {"wt_", permission.ResourceWatchtower, "watchtower item"},
	AccountingConnection: {
		"acctc_",
		permission.ResourceAccountingIntegration,
		"accounting connection",
	},
	AccountingMapping: {
		"acctm_",
		permission.ResourceAccountingIntegration,
		"accounting mapping",
	},
	AccountingSyncRecord: {
		"acctsr_",
		permission.ResourceAccountingSync,
		"accounting sync record",
	},
	AccountingInboundChange: {
		"acctic_",
		permission.ResourceAccountingSync,
		"accounting inbound change",
	},
	AccountingDriftFinding: {
		"acctdf_",
		permission.ResourceAccountingSync,
		"accounting drift finding",
	},
	CarrierIntelligenceEvent: {
		"cievt_", permission.ResourceCarrierIntelligence, "carrier intelligence event",
	},
	InsurancePolicy:       {"carins_", permission.ResourceCarrier, "insurance policy"},
	CarrierAssignment:     {"casn_", permission.ResourceShipmentMove, "carrier assignment"},
	AdditionalCharge:      {"ac_", permission.ResourceShipment, "additional charge"},
	BillingTransferRun:    {"btr_", permission.ResourceShipment, "billing transfer run"},
	DetentionOccurrence:   {"dto_", permission.ResourceDetention, "detention occurrence"},
	TenderOffer:           {"tof_", permission.ResourceTender, "tender offer"},
	OrderCharge:           {"ordchg_", permission.ResourceOrder, "order charge"},
	BillingQueueItem:      {"bqi_", permission.ResourceBillingQueue, "billing queue item"},
	InvoiceLine:           {"invl_", permission.ResourceInvoice, "invoice line"},
	InvoiceAdjustment:     {"iadj_", permission.ResourceInvoice, "invoice adjustment"},
	InvoiceRunGroup:       {"invrg_", permission.ResourceInvoiceRun, "invoice run group"},
	InvoiceRunItem:        {"invrgi_", permission.ResourceInvoiceRun, "invoice run item"},
	CreditMemoApplication: {"cma_", permission.ResourceCustomerPayment, "credit memo application"},
	RateAgreementLane:     {"ragr_", permission.ResourceRateAgreement, "rate agreement lane"},
	RateImport:            {"rib_", permission.ResourceRateAgreement, "rate import"},
	FuelIndex:             {"fidx_", permission.ResourceFuelSurchargeProgram, "fuel index"},
	FuelIndexPrice:        {"fip_", permission.ResourceFuelSurchargeProgram, "fuel index price"},
	ReportRun:             {"rrun_", permission.ResourceReport, "report run"},
	ReportSchedule:        {"rsch_", permission.ResourceReport, "report schedule"},
	EDIPartner:            {"edip_", permission.ResourceEDI, "EDI partner"},
	EDIInboundFile:        {"ediinf_", permission.ResourceEDI, "EDI inbound file"},
	EDITransfer:           {"edilt_", permission.ResourceEDI, "EDI transfer"},
	EDIMessage:            {"edimsg_", permission.ResourceEDI, "EDI message"},
	EDIShipmentLink:       {"edislink_", permission.ResourceEDI, "EDI shipment link"},
	EDITenderChange:       {"editcg_", permission.ResourceEDI, "EDI tender change"},
	EDITransferChange:     {"editc_", permission.ResourceEDI, "EDI transfer change"},
	EDICarrierInvoice:     {"edici_", permission.ResourceEDI, "EDI carrier invoice"},
	ScannedDocument:       {"citm_", permission.ResourceCaptureBatch, "scanned document"},
	IFTAJurisdiction:      {"ifj_", permission.ResourceFuelPurchase, "IFTA jurisdiction"},
	IFTAReturn:            {"ifr_", permission.ResourceIFTAReturn, "IFTA return"},
	IFTAMileageEntry: {
		"ifme_",
		permission.ResourceIFTAJurisdictionMileage,
		"IFTA mileage entry",
	},
	USState:         {"us_", permission.ResourcePermit, "US state"},
	PayEvent:        {"dpe_", permission.ResourceDriverSettlement, "pay event"},
	PayProfile:      {"dpp_", permission.ResourceDriverPayProfile, "pay profile"},
	PayAssignment:   {"wpa_", permission.ResourceDriverPayProfile, "pay assignment"},
	PayrollExport:   {"pxb_", permission.ResourceTimesheet, "payroll export"},
	ShiftAssignment: {"wsa_", permission.ResourceWorkerSchedule, "shift assignment"},
	ChecklistItem:   {"wcli_", permission.ResourceWorkerChecklist, "checklist item"},
	SafetyViolation: {"wsvi_", permission.ResourceWorkerSafetyEvent, "safety violation"},
	LeaveDay:        {"wle_", permission.ResourceWorkerLeave, "leave day"},
	AIAuditEvent:    {"aiae_", permission.ResourceAIAuditTrail, "AI audit event"},
	AIAuditExport:   {"aiax_", permission.ResourceAIAuditTrail, "AI audit export"},
}

var kindsByIDPrefix = func() map[string]Kind {
	out := make(map[string]Kind, len(recordKinds)+len(recordIDPrefixes))
	for resource, prefix := range recordIDPrefixes {
		out[prefix] = Kind(resource)
	}
	for kind, spec := range recordKinds {
		out[spec.prefix] = kind
	}

	return out
}()

// OfResource is the one kind of record the resource covers, when it covers
// one.
func OfResource(r permission.Resource) (Kind, bool) {
	if _, ok := recordIDPrefixes[r]; !ok {
		return "", false
	}

	return Kind(r), true
}

// IDPrefix is the prefix every id of this kind carries.
func (k Kind) IDPrefix() (string, bool) {
	if spec, ok := recordKinds[k]; ok {
		return spec.prefix, true
	}

	return ResourcePrefix(permission.Resource(k))
}

// Resource is the resource whose permission governs records of this kind.
func (k Kind) Resource() permission.Resource {
	if spec, ok := recordKinds[k]; ok {
		return spec.resource
	}

	return permission.Resource(k)
}

// Noun is how one record of this kind is named in a sentence: "clearinghouse
// query", "shipment".
func (k Kind) Noun() string {
	if spec, ok := recordKinds[k]; ok {
		return spec.noun
	}

	return textcase.HumanizeSnakeCasePhrase(string(k))
}

// Known reports whether ids of this kind have a prefix the runtime can check.
func (k Kind) Known() bool {
	_, ok := k.IDPrefix()

	return ok
}

// OfIDPrefix is the kind of record that carries the prefix, among
// both the single-kind resources and the kinds declared here.
func OfIDPrefix(prefix string) (Kind, bool) {
	kind, ok := kindsByIDPrefix[prefix]

	return kind, ok
}

// Prefixes is the declared kinds' table, for the test that holds
// each prefix to the one the domain mints.
func Prefixes() map[Kind]string {
	out := make(map[Kind]string, len(recordKinds))
	for kind, spec := range recordKinds {
		out[kind] = spec.prefix
	}

	return out
}

// All is every kind of record whose ids the runtime can check: each
// single-kind resource under its own name and each declared kind, sorted. A
// table of kinds kept anywhere else is held to it.
func All() []Kind {
	out := make([]Kind, 0, len(recordIDPrefixes)+len(recordKinds))
	for resource := range recordIDPrefixes {
		out = append(out, Kind(resource))
	}
	for kind := range recordKinds {
		out = append(out, kind)
	}
	slices.Sort(out)

	return out
}
