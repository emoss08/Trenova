package recordkind

import (
	"maps"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
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
var recordIDPrefixes = map[permission.Resource]string{ //nolint:exhaustive // only single-kind resources have a prefix
	permission.ResourceShipment:            "shp_",
	permission.ResourceShipmentMove:        "sm_",
	permission.ResourceShipmentComment:     "shc_",
	permission.ResourceOrder:               "ord_",
	permission.ResourceRecurringShipment:   "rsh_",
	permission.ResourceServiceFailure:      "sf_",
	permission.ResourceCustomer:            "cus_",
	permission.ResourceFactoringCompany:    "fac_",
	permission.ResourceCarrier:             "car_",
	permission.ResourceLocation:            "loc_",
	permission.ResourceWorker:              "wrk_",
	permission.ResourceTractor:             "trac_",
	permission.ResourceTrailer:             "tr_",
	permission.ResourceCommodity:           "com_",
	permission.ResourceHazardousMaterial:   "hm_",
	permission.ResourceDocument:            "doc_",
	permission.ResourceInvoice:             "inv_",
	permission.ResourceInvoiceDispute:      "idsp_",
	permission.ResourceFactoringAgreement:  "facagr_",
	permission.ResourceFactoringNoa:        "facnoa_",
	permission.ResourceFactoringSubmission: "facsub_",
	permission.ResourceFactoringReserve:    "facres_",
	permission.ResourceFactoringRemittance: "facrem_",
	permission.ResourceInvoiceRun:          "invrun_",
	permission.ResourceRateAgreement:       "rag_",
	permission.ResourceCustomerPayment:     "cpay_",
	permission.ResourceReport:              "rd_",
	permission.ResourceDashboard:           "rdb_",
	permission.ResourceWorkerCredential:    "wcred_",
	permission.ResourceWorkerTraining:      "wtrn_",
	permission.ResourceWorkerSafetyEvent:   "wsev_",
	permission.ResourcePerformanceReview:   "prev_",
	permission.ResourceWorkerInjury:        "winj_",
	permission.ResourceWorkerLeave:         "wlc_",
	permission.ResourceDriverSettlement:    "dstl_",
	permission.ResourceSettlementDispute:   "dsd_",
	permission.ResourceCaptureBatch:        "cbat_",

	permission.ResourceOrganization:              "org_",
	permission.ResourceUser:                      "usr_",
	permission.ResourceAgentRun:                  "ar_",
	permission.ResourceAgentMemory:               "amem_",
	permission.ResourceInsight:                   "inst_",
	permission.ResourceTableChangeAlert:          "tcas_",
	permission.ResourceEmailProfile:              "emlprof_",
	permission.ResourceInboundMessage:            "imsg_",
	permission.ResourceInboundMailbox:            "imbx_",
	permission.ResourceShipmentStop:              "stp_",
	permission.ResourceShipmentHold:              "shh_",
	permission.ResourceHoldReason:                "hr_",
	permission.ResourceServiceFailureReasonCode:  "sfrc_",
	permission.ResourcePermit:                    "pmt_",
	permission.ResourceServiceType:               "st_",
	permission.ResourceShipmentType:              "sht_",
	permission.ResourceDocumentType:              "dt_",
	permission.ResourceEquipmentType:             "et_",
	permission.ResourceEquipmentManufacturer:     "em_",
	permission.ResourceFleetCode:                 "fc_",
	permission.ResourceLocationCategory:          "lc_",
	permission.ResourceCarrierCapacityPosting:    "ccp_",
	permission.ResourceRateConfirmation:          "ratecon_",
	permission.ResourceCarrierSettlement:         "carstl_",
	permission.ResourceCarrierInvoiceMatch:       "cim_",
	permission.ResourceRoutingGuide:              "rg_",
	permission.ResourceTender:                    "tnd_",
	permission.ResourceAccessorialCharge:         "acc_",
	permission.ResourceFormulaTemplate:           "ft_",
	permission.ResourceFormulaLibrary:            "fl_",
	permission.ResourceRateMatrix:                "rmx_",
	permission.ResourceGeneralLedgerAccount:      "gla_",
	permission.ResourceFiscalPeriod:              "fp_",
	permission.ResourceManualJournal:             "mjr_",
	permission.ResourceJournalEntry:              "je_",
	permission.ResourceJournalReversal:           "jrev_",
	permission.ResourceBankReceipt:               "brcpt_",
	permission.ResourceBankReceiptWorkItem:       "brwi_",
	permission.ResourcePayCode:                   "payc_",
	permission.ResourcePayAdvance:                "padv_",
	permission.ResourceEscrowAccount:             "escr_",
	permission.ResourceRecurringDeduction:        "rded_",
	permission.ResourceRecurringEarning:          "rern_",
	permission.ResourceDriverExpense:             "dexp_",
	permission.ResourceFuelCard:                  "fcard_",
	permission.ResourceFuelPurchase:              "fpur_",
	permission.ResourceFuelPurchaseImport:        "fpib_",
	permission.ResourceWorkerPTO:                 "wrkpto_",
	permission.ResourceWorkerChecklist:           "wcl_",
	permission.ResourceWorkerChecklistTemplate:   "wclt_",
	permission.ResourceWorkerCredentialType:      "wct_",
	permission.ResourceWorkerRecognition:         "wrec_",
	permission.ResourceWorkerDOTTest:             "wdot_",
	permission.ResourceTrainingCourse:            "trnc_",
	permission.ResourcePerformanceReviewTemplate: "prt_",
	permission.ResourceShiftTemplate:             "shft_",
	permission.ResourceShiftSwap:                 "sswp_",
}

var resourcesByIDPrefix = func() map[string]permission.Resource {
	out := make(map[string]permission.Resource, len(recordIDPrefixes))
	for resource, prefix := range recordIDPrefixes {
		out[prefix] = resource
	}

	return out
}()

// ResourcePrefix is the prefix every id of the resource's records carries,
// when the resource is one kind of record with one prefix.
func ResourcePrefix(r permission.Resource) (string, bool) {
	prefix, ok := recordIDPrefixes[r]

	return prefix, ok
}

// ResourceOfIDPrefix is the resource whose records carry the prefix.
func ResourceOfIDPrefix(prefix string) (permission.Resource, bool) {
	resource, ok := resourcesByIDPrefix[prefix]

	return resource, ok
}

// ResourcePrefixes is the whole table, for the test that holds each prefix
// to the one the domain mints.
func ResourcePrefixes() map[permission.Resource]string {
	return maps.Clone(recordIDPrefixes)
}
