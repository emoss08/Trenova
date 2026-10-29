package recordversionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync/accountingsyncdb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod/fiscalperioddb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry/journalentrydb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalreversal/journalreversaldb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/manualjournal/manualjournaldb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/insight/insightdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower/watchtowerdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt/bankreceiptdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceiptworkitem/bankreceiptworkitemdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment/customerpaymentdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring/factoringdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringremittance/factoringremittancedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission/factoringsubmissiondb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment/invoiceadjustmentdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicedispute/invoicedisputedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicerun/invoicerundb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document/documentdb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage/inboundmessagedb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile/ediinboundfiledb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage/edimessagedb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edisync/edisyncdb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editenderchange/editenderchangedb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer/editransferdb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor/tractordb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer/trailerdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention/detentiondb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order/orderdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/recurringshipment/recurringshipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure/servicefailuredb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender/tenderdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carriercapacity/carriercapacitydb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel/carrierinteldb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany/factoringcompanydb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement/rateagreementdb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateconfirmation/rateconfirmationdb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateimport/rateimportdb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity/commoditydb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial/hazardousmaterialdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement/carriersettlementdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement/driversettlementdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase/fuelpurchasedb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta/iftadb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payadvance/payadvancedb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payescrow/payescrowdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payexpense/payexpensedb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/recurringpay/recurringpaydb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/dotrandom/dotrandomdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/employment/employmentdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/fleetsafety/fleetsafetydb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/permit/permitdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/timesheet/timesheetdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerchecklist/workerchecklistdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential/workercredentialdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerdiscipline/workerdisciplinedb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerdot/workerdotdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerinjury/workerinjurydb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerleave/workerleavedb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerreview/workerreviewdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workersafety/workersafetydb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerschedule/workerscheduledb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workertraining/workertrainingdb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report/reportdb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/tablechangealert/tablechangealertdb"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type Reader struct {
	db *postgres.Connection
}

func New(p Params) agenttoolports.RecordVersionReader {
	return &Reader{db: p.DB}
}

type lookup struct {
	read  func(id pulid.ID) keel.Projection
	kinds map[string]lookup
}

func by(read func(id pulid.ID) keel.Projection) lookup {
	return lookup{read: read}
}

var fuelImportLookup = by(func(id pulid.ID) keel.Projection {
	t := fuelpurchasedb.ImportBatches
	return t.Pick(t.Version).Where(t.ID.Eq(id))
})

var lookups = map[permission.Resource]lookup{
	permission.ResourceOrder: by(func(id pulid.ID) keel.Projection {
		t := orderdb.Orders
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceShipmentComment: by(func(id pulid.ID) keel.Projection {
		t := shipmentdb.ShipmentComments
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceRecurringShipment: by(func(id pulid.ID) keel.Projection {
		t := recurringshipmentdb.RecurringShipments
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceWorkerSchedule: by(func(id pulid.ID) keel.Projection {
		t := workerscheduledb.WorkerShiftAssignments
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceShiftSwap: by(func(id pulid.ID) keel.Projection {
		t := workerscheduledb.ShiftSwapRequests
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceCarrier: by(func(id pulid.ID) keel.Projection {
		t := carrierdb.Carriers
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceCustomer: by(func(id pulid.ID) keel.Projection {
		t := customerdb.Customers
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceShipment: by(func(id pulid.ID) keel.Projection {
		t := shipmentdb.Shipments
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceShipmentMove: by(func(id pulid.ID) keel.Projection {
		t := shipmentdb.ShipmentMoves
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceDocument: by(func(id pulid.ID) keel.Projection {
		t := documentdb.Documents
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceDashboard: by(func(id pulid.ID) keel.Projection {
		t := reportdb.Dashboards
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceWorkerPTO: by(func(id pulid.ID) keel.Projection {
		t := workerdb.WorkerPTOs
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceBillingQueue: by(func(id pulid.ID) keel.Projection {
		t := billingqueuedb.BillingQueueItems
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceBankReceipt: by(func(id pulid.ID) keel.Projection {
		t := bankreceiptdb.BankReceipts
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceBankReceiptWorkItem: by(func(id pulid.ID) keel.Projection {
		t := bankreceiptworkitemdb.WorkItems
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceInsight: by(func(id pulid.ID) keel.Projection {
		t := insightdb.Insights
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceInboundMessage: by(func(id pulid.ID) keel.Projection {
		t := inboundmessagedb.InboundMessages
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceWorker: by(func(id pulid.ID) keel.Projection {
		t := workerdb.Workers
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceServiceFailure: by(func(id pulid.ID) keel.Projection {
		t := servicefailuredb.ServiceFailures
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceReport: {kinds: map[string]lookup{
		"rd_": by(func(id pulid.ID) keel.Projection {
			t := reportdb.ReportDefinitions
			return t.Pick(t.Version).Where(t.ID.Eq(id))
		}),
		"rsch_": by(func(id pulid.ID) keel.Projection {
			t := reportdb.ReportSchedules
			return t.Pick(t.Version).Where(t.ID.Eq(id))
		}),
		"rrun_": by(func(id pulid.ID) keel.Projection {
			t := reportdb.ReportRuns
			return t.Pick(t.Version).Where(t.ID.Eq(id))
		}),
	}},
	permission.ResourceDetention: by(func(id pulid.ID) keel.Projection {
		t := detentiondb.DetentionOccurrences
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceAccountingSync: by(func(id pulid.ID) keel.Projection {
		t := accountingsyncdb.AccountingSyncRecords
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceTender: by(func(id pulid.ID) keel.Projection {
		t := tenderdb.Tenders
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceRateConfirmation: by(func(id pulid.ID) keel.Projection {
		t := rateconfirmationdb.RateConfirmations
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceInvoice: by(func(id pulid.ID) keel.Projection {
		t := invoicedb.Invoices
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceEDI: {kinds: map[string]lookup{
		"edilt_": by(func(id pulid.ID) keel.Projection {
			t := editransferdb.EDITransfers
			return t.Pick(t.Version).Where(t.ID.Eq(id))
		}),
		"editcg_": by(func(id pulid.ID) keel.Projection {
			t := editenderchangedb.TenderChanges
			return t.Pick(t.Version).Where(t.ID.Eq(id))
		}),
		"editc_": by(func(id pulid.ID) keel.Projection {
			t := edisyncdb.TransferChanges
			return t.Pick(t.Version).Where(t.ID.Eq(id))
		}),
		"edimsg_": by(func(id pulid.ID) keel.Projection {
			t := edimessagedb.EDIMessages
			return t.Pick(t.Version).Where(t.ID.Eq(id))
		}),
		"ediinf_": by(func(id pulid.ID) keel.Projection {
			t := ediinboundfiledb.EDIInboundFiles
			return t.Pick(t.Version).Where(t.ID.Eq(id))
		}),
	}},
	permission.ResourceFuelPurchase: {kinds: map[string]lookup{
		"fpur_": by(func(id pulid.ID) keel.Projection {
			t := fuelpurchasedb.FuelPurchases
			return t.Pick(t.Version).Where(t.ID.Eq(id))
		}),
		"fpib_": fuelImportLookup,
	}},
	permission.ResourceFuelCard: by(func(id pulid.ID) keel.Projection {
		t := fuelpurchasedb.FuelCards
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceIFTAReturn: by(func(id pulid.ID) keel.Projection {
		t := iftadb.Returns
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceIFTAJurisdictionMileage: by(func(id pulid.ID) keel.Projection {
		t := iftadb.JurisdictionMileageEntries
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceRateAgreement: {kinds: map[string]lookup{
		"rag_": by(func(id pulid.ID) keel.Projection {
			t := rateagreementdb.RateAgreements
			return t.Pick(t.Version).Where(t.ID.Eq(id))
		}),
		"rib_": by(func(id pulid.ID) keel.Projection {
			t := rateimportdb.RateImportBatches
			return t.Pick(t.Version).Where(t.ID.Eq(id))
		}),
	}},
	permission.ResourceInvoiceDispute: by(func(id pulid.ID) keel.Projection {
		t := invoicedisputedb.InvoiceDisputes
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceFactoringNoa: by(func(id pulid.ID) keel.Projection {
		t := factoringdb.Noas
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceFactoringSubmission: by(func(id pulid.ID) keel.Projection {
		t := factoringsubmissiondb.Submissions
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceFactoringRemittance: by(func(id pulid.ID) keel.Projection {
		t := factoringremittancedb.Remittances
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceFactoringCompany: by(func(id pulid.ID) keel.Projection {
		t := factoringcompanydb.FactoringCompanies
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceInvoiceRun: by(func(id pulid.ID) keel.Projection {
		t := invoicerundb.InvoiceRuns
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceCustomerPayment: by(func(id pulid.ID) keel.Projection {
		t := customerpaymentdb.Payments
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	agenttoolports.RecordInvoiceAdjustment: by(func(id pulid.ID) keel.Projection {
		t := invoiceadjustmentdb.InvoiceAdjustments
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	agenttoolports.RecordCreditMemoApplication: by(func(id pulid.ID) keel.Projection {
		t := customerpaymentdb.CreditMemoApplications
		return t.Pick(t.UpdatedAt).Where(t.ID.Eq(id))
	}),
	permission.ResourceCarrierIntelligence: by(func(id pulid.ID) keel.Projection {
		t := carrierinteldb.CarrierIntelEvents
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceDriverSettlement: by(func(id pulid.ID) keel.Projection {
		t := driversettlementdb.Settlements
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceCarrierSettlement: by(func(id pulid.ID) keel.Projection {
		t := carriersettlementdb.CarrierSettlements
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceCarrierInvoiceMatch: by(func(id pulid.ID) keel.Projection {
		t := carriersettlementdb.InvoiceMatches
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceEscrowAccount: by(func(id pulid.ID) keel.Projection {
		t := payescrowdb.EscrowAccounts
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourcePayAdvance: by(func(id pulid.ID) keel.Projection {
		t := payadvancedb.PayAdvances
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceRecurringDeduction: by(func(id pulid.ID) keel.Projection {
		t := recurringpaydb.RecurringDeductions
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceManualJournal: by(func(id pulid.ID) keel.Projection {
		t := manualjournaldb.Requests
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceJournalReversal: by(func(id pulid.ID) keel.Projection {
		t := journalreversaldb.Reversals
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceJournalEntry: by(func(id pulid.ID) keel.Projection {
		t := journalentrydb.JournalEntries
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceFiscalPeriod: by(func(id pulid.ID) keel.Projection {
		t := fiscalperioddb.FiscalPeriods
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceRecurringEarning: by(func(id pulid.ID) keel.Projection {
		t := recurringpaydb.RecurringEarnings
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceWorkerDOTTest: by(func(id pulid.ID) keel.Projection {
		t := workerdotdb.WorkerDOTTests
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceDOTRandomPool: {kinds: map[string]lookup{
		"drdraw_": by(func(id pulid.ID) keel.Projection {
			t := dotrandomdb.DOTRandomDraws
			return t.Pick(t.Version).Where(t.ID.Eq(id))
		}),
		"drde_": by(func(id pulid.ID) keel.Projection {
			t := dotrandomdb.DOTRandomDrawEntries
			return t.Pick(t.Version).Where(t.ID.Eq(id))
		}),
	}},
	permission.ResourceWorkerSafetyEvent: {kinds: map[string]lookup{
		"wsev_": by(func(id pulid.ID) keel.Projection {
			t := workersafetydb.WorkerSafetyEvents
			return t.Pick(t.Version).Where(t.ID.Eq(id))
		}),
		"wsvi_": by(func(id pulid.ID) keel.Projection {
			t := fleetsafetydb.WorkerSafetyViolations
			return t.Pick(t.Version).Where(t.ID.Eq(id))
		}),
	}},
	permission.ResourceWorkerRecognition: by(func(id pulid.ID) keel.Projection {
		t := workerdisciplinedb.WorkerRecognitions
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceWorkerLeave: {kinds: map[string]lookup{
		"wlc_": by(func(id pulid.ID) keel.Projection {
			t := workerleavedb.WorkerLeaveCases
			return t.Pick(t.Version).Where(t.ID.Eq(id))
		}),
		"wle_": by(func(id pulid.ID) keel.Projection {
			t := workerleavedb.WorkerLeaveEntries
			return t.Pick(t.Version).Where(t.ID.Eq(id))
		}),
	}},
	permission.ResourceWorkerTraining: by(func(id pulid.ID) keel.Projection {
		t := workertrainingdb.WorkerTrainingRecords
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceWorkerChecklist: {kinds: map[string]lookup{
		"wcl_": by(func(id pulid.ID) keel.Projection {
			t := workerchecklistdb.WorkerChecklists
			return t.Pick(t.Version).Where(t.ID.Eq(id))
		}),
		"wcli_": by(func(id pulid.ID) keel.Projection {
			t := workerchecklistdb.WorkerChecklistItems
			return t.Pick(t.Version).Where(t.ID.Eq(id))
		}),
	}},
	permission.ResourcePerformanceReview: by(func(id pulid.ID) keel.Projection {
		t := workerreviewdb.PerformanceReviews
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceQualification: by(func(id pulid.ID) keel.Projection {
		t := employmentdb.WorkerEmploymentVerifications
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceWorkerCredential: by(func(id pulid.ID) keel.Projection {
		t := workercredentialdb.WorkerCredentials
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceWorkerInjury: by(func(id pulid.ID) keel.Projection {
		t := workerinjurydb.WorkerInjuries
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceTimesheet: {kinds: map[string]lookup{
		"pxb_": by(func(id pulid.ID) keel.Projection {
			t := timesheetdb.PayrollExports
			return t.Pick(t.Version).Where(t.ID.Eq(id))
		}),
	}},
	permission.ResourcePermit: by(func(id pulid.ID) keel.Projection {
		t := permitdb.Permits
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceSettlementDispute: by(func(id pulid.ID) keel.Projection {
		t := driversettlementdb.Disputes
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceDriverExpense: by(func(id pulid.ID) keel.Projection {
		t := payexpensedb.Expenses
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceCommodity: by(func(id pulid.ID) keel.Projection {
		t := commoditydb.Commodities
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceCarrierCapacityPosting: by(func(id pulid.ID) keel.Projection {
		t := carriercapacitydb.Postings
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceHazardousMaterial: by(func(id pulid.ID) keel.Projection {
		t := hazardousmaterialdb.HazardousMaterials
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceLocation: by(func(id pulid.ID) keel.Projection {
		t := locationdb.Locations
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceTractor: by(func(id pulid.ID) keel.Projection {
		t := tractordb.Tractors
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceTrailer: by(func(id pulid.ID) keel.Projection {
		t := trailerdb.Trailers
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceTableChangeAlert: by(func(id pulid.ID) keel.Projection {
		t := tablechangealertdb.TCASubscriptions
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceWatchtower: by(func(id pulid.ID) keel.Projection {
		t := watchtowerdb.Items
		return t.Pick(t.Version).Where(t.ID.Eq(id))
	}),
	permission.ResourceFuelPurchaseImport: fuelImportLookup,
}

func (r *Reader) Version(
	ctx context.Context,
	_ pagination.TenantInfo,
	target agenttoolports.ToolTarget,
) (int64, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int64, error) {
		entry, ok := lookups[target.Resource]
		if ok && entry.kinds != nil {
			entry, ok = entry.kinds[target.ID.Prefix()]
		}
		if !ok {
			return 0, fmt.Errorf(
				"%w: %s",
				agenttoolports.ErrRecordVersionUnsupported,
				target.Resource,
			)
		}

		version, err := keel.IntoOne[int64](ctx, r.db.Keel(ctx), entry.read(target.ID))
		if err != nil {
			return 0, fmt.Errorf("read %s %s version: %w", target.Resource, target.ID, err)
		}

		return version, nil
	})
}
