package permission

import (
	"maps"
	"strings"
)

// recordIDPrefixes is the prefix every id of a resource's records carries, for
// the resources whose records are one kind with one prefix. A resource that
// covers several kinds of record (a qualification is a verification, a
// clearinghouse query or a medical card, each with its own prefix) is left
// out, since its prefix would be wrong for all but one of them; its kinds are
// declared one by one in recordKinds.
//
// Tool schemas name the resource an id parameter takes, and the runtime reads
// the prefix here to tell a model it sent a carrier's id where a customer's
// belonged. Before, an id of the wrong kind passed every check, the service
// answered "not found", and the model told the person the record did not
// exist.
var recordIDPrefixes = map[Resource]string{
	ResourceShipment:          "shp_",
	ResourceShipmentMove:      "sm_",
	ResourceShipmentComment:   "shc_",
	ResourceOrder:             "ord_",
	ResourceRecurringShipment: "rsh_",
	ResourceServiceFailure:    "sf_",
	ResourceCustomer:          "cus_",
	ResourceCarrier:           "car_",
	ResourceLocation:          "loc_",
	ResourceWorker:            "wrk_",
	ResourceTractor:           "trac_",
	ResourceTrailer:           "tr_",
	ResourceCommodity:         "com_",
	ResourceHazardousMaterial: "hm_",
	ResourceDocument:          "doc_",
	ResourceInvoice:           "inv_",
	ResourceInvoiceDispute:    "idsp_",
	ResourceInvoiceRun:        "invrun_",
	ResourceRateAgreement:     "rag_",
	ResourceCustomerPayment:   "cpay_",
	ResourceReport:            "rd_",
	ResourceDashboard:         "rdb_",
	ResourceWorkerCredential:  "wcred_",
	ResourceWorkerTraining:    "wtrn_",
	ResourceWorkerSafetyEvent: "wsev_",
	ResourcePerformanceReview: "prev_",
	ResourceWorkerInjury:      "winj_",
	ResourceWorkerLeave:       "wlc_",
	ResourceDriverSettlement:  "dstl_",
	ResourceSettlementDispute: "dsd_",
	ResourceCaptureBatch:      "cbat_",

	ResourceOrganization:              "org_",
	ResourceUser:                      "usr_",
	ResourceAgentRun:                  "ar_",
	ResourceAgentMemory:               "amem_",
	ResourceInsight:                   "inst_",
	ResourceTableChangeAlert:          "tcas_",
	ResourceEmailProfile:              "emlprof_",
	ResourceInboundMessage:            "imsg_",
	ResourceInboundMailbox:            "imbx_",
	ResourceShipmentStop:              "stp_",
	ResourceShipmentHold:              "shh_",
	ResourceHoldReason:                "hr_",
	ResourceServiceFailureReasonCode:  "sfrc_",
	ResourcePermit:                    "pmt_",
	ResourceServiceType:               "st_",
	ResourceShipmentType:              "sht_",
	ResourceDocumentType:              "dt_",
	ResourceEquipmentType:             "et_",
	ResourceEquipmentManufacturer:     "em_",
	ResourceFleetCode:                 "fc_",
	ResourceLocationCategory:          "lc_",
	ResourceCarrierCapacityPosting:    "ccp_",
	ResourceRateConfirmation:          "ratecon_",
	ResourceCarrierSettlement:         "carstl_",
	ResourceCarrierInvoiceMatch:       "cim_",
	ResourceRoutingGuide:              "rg_",
	ResourceTender:                    "tnd_",
	ResourceAccessorialCharge:         "acc_",
	ResourceFormulaTemplate:           "ft_",
	ResourceRateMatrix:                "rmx_",
	ResourceGeneralLedgerAccount:      "gla_",
	ResourceFiscalPeriod:              "fp_",
	ResourceManualJournal:             "mjr_",
	ResourceJournalEntry:              "je_",
	ResourceJournalReversal:           "jrev_",
	ResourceBankReceipt:               "brcpt_",
	ResourceBankReceiptWorkItem:       "brwi_",
	ResourcePayCode:                   "payc_",
	ResourcePayAdvance:                "padv_",
	ResourceEscrowAccount:             "escr_",
	ResourceRecurringDeduction:        "rded_",
	ResourceRecurringEarning:          "rern_",
	ResourceDriverExpense:             "dexp_",
	ResourceFuelCard:                  "fcard_",
	ResourceFuelPurchase:              "fpur_",
	ResourceFuelPurchaseImport:        "fpib_",
	ResourceWorkerPTO:                 "wrkpto_",
	ResourceWorkerChecklist:           "wcl_",
	ResourceWorkerChecklistTemplate:   "wclt_",
	ResourceWorkerCredentialType:      "wct_",
	ResourceWorkerRecognition:         "wrec_",
	ResourceWorkerDOTTest:             "wdot_",
	ResourceTrainingCourse:            "trnc_",
	ResourcePerformanceReviewTemplate: "prt_",
	ResourceShiftTemplate:             "shft_",
	ResourceShiftSwap:                 "sswp_",
}

var resourcesByIDPrefix = func() map[string]Resource {
	out := make(map[string]Resource, len(recordIDPrefixes))
	for resource, prefix := range recordIDPrefixes {
		out[prefix] = resource
	}

	return out
}()

// IDPrefix is the prefix every id of this resource's records carries, when
// the resource is one kind of record with one prefix.
func (r Resource) IDPrefix() (string, bool) {
	prefix, ok := recordIDPrefixes[r]

	return prefix, ok
}

// ResourceOfIDPrefix is the resource whose records carry the prefix.
func ResourceOfIDPrefix(prefix string) (Resource, bool) {
	resource, ok := resourcesByIDPrefix[prefix]

	return resource, ok
}

// RecordIDPrefixes is the whole table, for the test that holds each prefix
// to the one the domain mints.
func RecordIDPrefixes() map[Resource]string {
	return maps.Clone(recordIDPrefixes)
}

// Noun is how one of the resource's records is named in a sentence: "worker
// credential", "shipment".
func (r Resource) Noun() string {
	return strings.ReplaceAll(string(r), "_", " ")
}
