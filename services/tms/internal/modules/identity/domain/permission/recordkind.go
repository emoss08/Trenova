package permission

import "strings"

// RecordKind is one kind of record with one id prefix. A resource whose
// records are all one kind is that kind under its own name (Resource.Kind);
// the kinds declared here are the ones a resource's name cannot stand for: the
// verifications and clearinghouse queries a qualification covers, the pools,
// draws and selections of DOT random testing, and the lines, days and items
// held inside a record of another resource.
//
// A tool parameter that takes an id of one of these, or of any of several
// kinds, names the kinds it takes, and the runtime refuses an id whose prefix
// is none of theirs before the tool reads it.
type RecordKind string

const (
	KindEmploymentVerification   RecordKind = "employment_verification"
	KindClearinghouseQuery       RecordKind = "clearinghouse_query"
	KindDOTRandomPool            RecordKind = "dot_random_pool"
	KindDOTRandomDraw            RecordKind = "dot_random_draw"
	KindDOTRandomSelection       RecordKind = "dot_random_selection"
	KindConversation             RecordKind = "conversation"
	KindWait                     RecordKind = "wait"
	KindWatchtowerItem           RecordKind = "watchtower_item"
	KindAccountingConnection     RecordKind = "accounting_connection"
	KindAccountingMapping        RecordKind = "accounting_mapping"
	KindAccountingSyncRecord     RecordKind = "accounting_sync_record"
	KindAccountingInboundChange  RecordKind = "accounting_inbound_change"
	KindAccountingDriftFinding   RecordKind = "accounting_drift_finding"
	KindCarrierIntelligenceEvent RecordKind = "carrier_intelligence_event"
	KindInsurancePolicy          RecordKind = "insurance_policy"
	KindCarrierAssignment        RecordKind = "carrier_assignment"
	KindAdditionalCharge         RecordKind = "additional_charge"
	KindBillingTransferRun       RecordKind = "billing_transfer_run"
	KindDetentionOccurrence      RecordKind = "detention_occurrence"
	KindTenderOffer              RecordKind = "tender_offer"
	KindOrderCharge              RecordKind = "order_charge"
	KindBillingQueueItem         RecordKind = "billing_queue_item"
	KindInvoiceLine              RecordKind = "invoice_line"
	KindInvoiceAdjustment        RecordKind = "invoice_adjustment"
	KindInvoiceRunGroup          RecordKind = "invoice_run_group"
	KindInvoiceRunItem           RecordKind = "invoice_run_item"
	KindCreditMemoApplication    RecordKind = "credit_memo_application"
	KindRateAgreementLane        RecordKind = "rate_agreement_lane"
	KindRateImport               RecordKind = "rate_import"
	KindFuelIndex                RecordKind = "fuel_index"
	KindFuelIndexPrice           RecordKind = "fuel_index_price"
	KindReportRun                RecordKind = "report_run"
	KindReportSchedule           RecordKind = "report_schedule"
	KindEDIPartner               RecordKind = "edi_partner"
	KindEDIInboundFile           RecordKind = "edi_inbound_file"
	KindEDITransfer              RecordKind = "edi_transfer"
	KindEDIMessage               RecordKind = "edi_message"
	KindEDIShipmentLink          RecordKind = "edi_shipment_link"
	KindEDITenderChange          RecordKind = "edi_tender_change"
	KindEDITransferChange        RecordKind = "edi_transfer_change"
	KindEDICarrierInvoice        RecordKind = "edi_carrier_invoice"
	KindScannedDocument          RecordKind = "scanned_document"
	KindIFTAJurisdiction         RecordKind = "ifta_jurisdiction"
	KindIFTAReturn               RecordKind = "ifta_return"
	KindIFTAMileageEntry         RecordKind = "ifta_mileage_entry"
	KindUSState                  RecordKind = "us_state"
	KindPayEvent                 RecordKind = "pay_event"
	KindPayProfile               RecordKind = "pay_profile"
	KindPayAssignment            RecordKind = "pay_assignment"
	KindPayrollExport            RecordKind = "payroll_export"
	KindShiftAssignment          RecordKind = "shift_assignment"
	KindChecklistItem            RecordKind = "checklist_item"
	KindSafetyViolation          RecordKind = "safety_violation"
	KindLeaveDay                 RecordKind = "leave_day"
)

type recordKindSpec struct {
	prefix   string
	resource Resource
	noun     string
}

// recordKinds holds each declared kind to the prefix the domain mints for it,
// the resource whose permission governs it, and what one is called.
var recordKinds = map[RecordKind]recordKindSpec{
	KindEmploymentVerification:  {"wemv_", ResourceQualification, "employment verification"},
	KindClearinghouseQuery:      {"wchq_", ResourceQualification, "clearinghouse query"},
	KindDOTRandomPool:           {"drpool_", ResourceDOTRandomPool, "DOT random pool"},
	KindDOTRandomDraw:           {"drdraw_", ResourceDOTRandomPool, "DOT random draw"},
	KindDOTRandomSelection:      {"drde_", ResourceDOTRandomPool, "DOT random selection"},
	KindConversation:            {"athr_", ResourceAssistant, "conversation"},
	KindWait:                    {"awt_", ResourceAgentRun, "wait"},
	KindWatchtowerItem:          {"wt_", ResourceWatchtower, "watchtower item"},
	KindAccountingConnection:    {"acctc_", ResourceAccountingIntegration, "accounting connection"},
	KindAccountingMapping:       {"acctm_", ResourceAccountingIntegration, "accounting mapping"},
	KindAccountingSyncRecord:    {"acctsr_", ResourceAccountingSync, "accounting sync record"},
	KindAccountingInboundChange: {"acctic_", ResourceAccountingSync, "accounting inbound change"},
	KindAccountingDriftFinding:  {"acctdf_", ResourceAccountingSync, "accounting drift finding"},
	KindCarrierIntelligenceEvent: {
		"cievt_", ResourceCarrierIntelligence, "carrier intelligence event",
	},
	KindInsurancePolicy:       {"carins_", ResourceCarrier, "insurance policy"},
	KindCarrierAssignment:     {"casn_", ResourceShipmentMove, "carrier assignment"},
	KindAdditionalCharge:      {"ac_", ResourceShipment, "additional charge"},
	KindBillingTransferRun:    {"btr_", ResourceShipment, "billing transfer run"},
	KindDetentionOccurrence:   {"dto_", ResourceDetentionPolicy, "detention occurrence"},
	KindTenderOffer:           {"tof_", ResourceTender, "tender offer"},
	KindOrderCharge:           {"ordchg_", ResourceOrder, "order charge"},
	KindBillingQueueItem:      {"bqi_", ResourceBillingQueue, "billing queue item"},
	KindInvoiceLine:           {"invl_", ResourceInvoice, "invoice line"},
	KindInvoiceAdjustment:     {"iadj_", ResourceInvoice, "invoice adjustment"},
	KindInvoiceRunGroup:       {"invrg_", ResourceInvoiceRun, "invoice run group"},
	KindInvoiceRunItem:        {"invrgi_", ResourceInvoiceRun, "invoice run item"},
	KindCreditMemoApplication: {"cma_", ResourceCustomerPayment, "credit memo application"},
	KindRateAgreementLane:     {"ragr_", ResourceRateAgreement, "rate agreement lane"},
	KindRateImport:            {"rib_", ResourceRateAgreement, "rate import"},
	KindFuelIndex:             {"fidx_", ResourceFuelSurchargeProgram, "fuel index"},
	KindFuelIndexPrice:        {"fip_", ResourceFuelSurchargeProgram, "fuel index price"},
	KindReportRun:             {"rrun_", ResourceReport, "report run"},
	KindReportSchedule:        {"rsch_", ResourceReport, "report schedule"},
	KindEDIPartner:            {"edip_", ResourceEDI, "EDI partner"},
	KindEDIInboundFile:        {"ediinf_", ResourceEDI, "EDI inbound file"},
	KindEDITransfer:           {"edilt_", ResourceEDI, "EDI transfer"},
	KindEDIMessage:            {"edimsg_", ResourceEDI, "EDI message"},
	KindEDIShipmentLink:       {"edislink_", ResourceEDI, "EDI shipment link"},
	KindEDITenderChange:       {"editcg_", ResourceEDI, "EDI tender change"},
	KindEDITransferChange:     {"editc_", ResourceEDI, "EDI transfer change"},
	KindEDICarrierInvoice:     {"edici_", ResourceEDI, "EDI carrier invoice"},
	KindScannedDocument:       {"citm_", ResourceCaptureBatch, "scanned document"},
	KindIFTAJurisdiction:      {"ifj_", ResourceFuelPurchase, "IFTA jurisdiction"},
	KindIFTAReturn:            {"ifr_", ResourceIFTAReturn, "IFTA return"},
	KindIFTAMileageEntry:      {"ifme_", ResourceIFTAJurisdictionMileage, "IFTA mileage entry"},
	KindUSState:               {"us_", ResourcePermit, "US state"},
	KindPayEvent:              {"dpe_", ResourceDriverSettlement, "pay event"},
	KindPayProfile:            {"dpp_", ResourceDriverPayProfile, "pay profile"},
	KindPayAssignment:         {"wpa_", ResourceDriverPayProfile, "pay assignment"},
	KindPayrollExport:         {"pxb_", ResourceTimesheet, "payroll export"},
	KindShiftAssignment:       {"wsa_", ResourceWorkerSchedule, "shift assignment"},
	KindChecklistItem:         {"wcli_", ResourceWorkerChecklist, "checklist item"},
	KindSafetyViolation:       {"wsvi_", ResourceWorkerSafetyEvent, "safety violation"},
	KindLeaveDay:              {"wle_", ResourceWorkerLeave, "leave day"},
}

var kindsByIDPrefix = func() map[string]RecordKind {
	out := make(map[string]RecordKind, len(recordKinds)+len(recordIDPrefixes))
	for resource, prefix := range recordIDPrefixes {
		out[prefix] = RecordKind(resource)
	}
	for kind, spec := range recordKinds {
		out[spec.prefix] = kind
	}

	return out
}()

// Kind is the one kind of record the resource covers, when it covers one.
func (r Resource) Kind() (RecordKind, bool) {
	if _, ok := recordIDPrefixes[r]; !ok {
		return "", false
	}

	return RecordKind(r), true
}

// IDPrefix is the prefix every id of this kind carries.
func (k RecordKind) IDPrefix() (string, bool) {
	if spec, ok := recordKinds[k]; ok {
		return spec.prefix, true
	}

	return Resource(k).IDPrefix()
}

// Resource is the resource whose permission governs records of this kind.
func (k RecordKind) Resource() Resource {
	if spec, ok := recordKinds[k]; ok {
		return spec.resource
	}

	return Resource(k)
}

// Noun is how one record of this kind is named in a sentence: "clearinghouse
// query", "shipment".
func (k RecordKind) Noun() string {
	if spec, ok := recordKinds[k]; ok {
		return spec.noun
	}

	return strings.ReplaceAll(string(k), "_", " ")
}

// Known reports whether ids of this kind have a prefix the runtime can check.
func (k RecordKind) Known() bool {
	_, ok := k.IDPrefix()

	return ok
}

// RecordKindOfIDPrefix is the kind of record that carries the prefix, among
// both the single-kind resources and the kinds declared here.
func RecordKindOfIDPrefix(prefix string) (RecordKind, bool) {
	kind, ok := kindsByIDPrefix[prefix]

	return kind, ok
}

// RecordKindPrefixes is the declared kinds' table, for the test that holds
// each prefix to the one the domain mints.
func RecordKindPrefixes() map[RecordKind]string {
	out := make(map[RecordKind]string, len(recordKinds))
	for kind, spec := range recordKinds {
		out[kind] = spec.prefix
	}

	return out
}
