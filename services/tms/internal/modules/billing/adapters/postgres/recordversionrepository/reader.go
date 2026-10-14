package recordversionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/pkg/buncolgen/factoringcol"
	"github.com/emoss08/trenova/pkg/buncolgen/factoringcompanycol"

	"github.com/emoss08/trenova/pkg/buncolgen/accountingsynccol"
	"github.com/emoss08/trenova/pkg/buncolgen/bankreceiptcol"
	"github.com/emoss08/trenova/pkg/buncolgen/bankreceiptworkitemcol"
	"github.com/emoss08/trenova/pkg/buncolgen/billingqueuecol"
	"github.com/emoss08/trenova/pkg/buncolgen/carriercapacitycol"
	"github.com/emoss08/trenova/pkg/buncolgen/carriercol"
	"github.com/emoss08/trenova/pkg/buncolgen/carrierintelcol"
	"github.com/emoss08/trenova/pkg/buncolgen/carriersettlementcol"
	"github.com/emoss08/trenova/pkg/buncolgen/commoditycol"
	"github.com/emoss08/trenova/pkg/buncolgen/customercol"
	"github.com/emoss08/trenova/pkg/buncolgen/customerpaymentcol"
	"github.com/emoss08/trenova/pkg/buncolgen/detentioncol"
	"github.com/emoss08/trenova/pkg/buncolgen/documentcol"
	"github.com/emoss08/trenova/pkg/buncolgen/driverpaycol"
	"github.com/emoss08/trenova/pkg/buncolgen/driversettlementcol"
	"github.com/emoss08/trenova/pkg/buncolgen/edicol"
	"github.com/emoss08/trenova/pkg/buncolgen/fiscalperiodcol"
	"github.com/emoss08/trenova/pkg/buncolgen/fuelpurchasecol"
	"github.com/emoss08/trenova/pkg/buncolgen/hazardousmaterialcol"
	"github.com/emoss08/trenova/pkg/buncolgen/iftacol"
	"github.com/emoss08/trenova/pkg/buncolgen/inboundmessagecol"
	"github.com/emoss08/trenova/pkg/buncolgen/insightcol"
	"github.com/emoss08/trenova/pkg/buncolgen/invoiceadjustmentcol"
	"github.com/emoss08/trenova/pkg/buncolgen/invoicecol"
	"github.com/emoss08/trenova/pkg/buncolgen/invoiceruncol"
	"github.com/emoss08/trenova/pkg/buncolgen/journalentrycol"
	"github.com/emoss08/trenova/pkg/buncolgen/journalreversalcol"
	"github.com/emoss08/trenova/pkg/buncolgen/locationcol"
	"github.com/emoss08/trenova/pkg/buncolgen/manualjournalcol"
	"github.com/emoss08/trenova/pkg/buncolgen/ordercol"
	"github.com/emoss08/trenova/pkg/buncolgen/permitcol"
	"github.com/emoss08/trenova/pkg/buncolgen/rateagreementcol"
	"github.com/emoss08/trenova/pkg/buncolgen/rateconfirmationcol"
	"github.com/emoss08/trenova/pkg/buncolgen/rateimportcol"
	"github.com/emoss08/trenova/pkg/buncolgen/recurringshipmentcol"
	"github.com/emoss08/trenova/pkg/buncolgen/reportcol"
	"github.com/emoss08/trenova/pkg/buncolgen/servicefailurecol"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"
	"github.com/emoss08/trenova/pkg/buncolgen/tablechangealertcol"
	"github.com/emoss08/trenova/pkg/buncolgen/tendercol"
	"github.com/emoss08/trenova/pkg/buncolgen/tractorcol"
	"github.com/emoss08/trenova/pkg/buncolgen/trailercol"
	"github.com/emoss08/trenova/pkg/buncolgen/watchtowercol"
	"github.com/emoss08/trenova/pkg/buncolgen/workercol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalreversal"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/manualjournal"
	"github.com/emoss08/trenova/internal/modules/agent/domain/insight"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicerun"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/editenderchangerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/editransferchangerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/editransferrepository"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order"
	"github.com/emoss08/trenova/internal/modules/operations/domain/recurringshipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carriercapacity"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateimport"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/permit"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/tablechangealert"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

// Params wires the reader.
type Params struct {
	fx.In

	DB *postgres.Connection
}

// Reader answers "what version is this record now" for the records agent
// tools act on.
//
// One reader rather than a method per tool: each tool holds only the narrow
// service it acts through — holds, comments, assignments — and none of those
// hands back a version. The tool names its target; this resolves it. Adding a
// resource is one line in the table below, and a resource not in it is
// reported rather than guessed.
type Reader struct {
	db *postgres.Connection
}

func New(p Params) agentports.RecordVersionReader {
	return &Reader{db: p.DB}
}

// lookup is one resource's query: which model, how to scope it to the tenant,
// which column is the id, and how to read the version off the row.
type lookup struct {
	model   func() versioned
	scope   func(*bun.SelectQuery, pagination.TenantInfo) *bun.SelectQuery
	idEq    string
	version func(versioned) int64
	kinds   map[string]lookup
}

type versioned any

var fuelImportLookup = lookup{
	model: func() versioned { return new(fuelpurchase.ImportBatch) },
	scope: fuelpurchasecol.ImportBatchScopeTenant,
	idEq:  fuelpurchasecol.ImportBatchColumns.ID.Eq(),
	version: versionOf(
		func(entity *fuelpurchase.ImportBatch) int64 { return entity.Version },
	),
}

var lookups = map[permission.Resource]lookup{
	permission.ResourceOrder: {
		model:   func() versioned { return new(order.Order) },
		scope:   ordercol.OrderScopeTenant,
		idEq:    ordercol.OrderColumns.ID.Eq(),
		version: versionOf(func(entity *order.Order) int64 { return entity.Version }),
	},
	permission.ResourceShipmentComment: {
		model:   func() versioned { return new(shipment.ShipmentComment) },
		scope:   shipmentcol.ShipmentCommentScopeTenant,
		idEq:    shipmentcol.ShipmentCommentColumns.ID.Eq(),
		version: versionOf(func(entity *shipment.ShipmentComment) int64 { return entity.Version }),
	},
	permission.ResourceRecurringShipment: {
		model: func() versioned { return new(recurringshipment.RecurringShipment) },
		scope: recurringshipmentcol.RecurringShipmentScopeTenant,
		idEq:  recurringshipmentcol.RecurringShipmentColumns.ID.Eq(),
		version: versionOf(
			func(entity *recurringshipment.RecurringShipment) int64 { return entity.Version },
		),
	},
	permission.ResourceWorkerSchedule: {
		model: func() versioned { return new(worker.WorkerShiftAssignment) },
		scope: workercol.WorkerShiftAssignmentScopeTenant,
		idEq:  workercol.WorkerShiftAssignmentColumns.ID.Eq(),
		version: versionOf(
			func(entity *worker.WorkerShiftAssignment) int64 { return entity.Version },
		),
	},
	permission.ResourceShiftSwap: {
		model:   func() versioned { return new(worker.ShiftSwapRequest) },
		scope:   workercol.ShiftSwapRequestScopeTenant,
		idEq:    workercol.ShiftSwapRequestColumns.ID.Eq(),
		version: versionOf(func(entity *worker.ShiftSwapRequest) int64 { return entity.Version }),
	},
	permission.ResourceCarrier: {
		model:   func() versioned { return new(carrier.Carrier) },
		scope:   carriercol.CarrierScopeTenant,
		idEq:    carriercol.CarrierColumns.ID.Eq(),
		version: versionOf(func(entity *carrier.Carrier) int64 { return entity.Version }),
	},
	permission.ResourceCustomer: {
		model:   func() versioned { return new(customer.Customer) },
		scope:   customercol.CustomerScopeTenant,
		idEq:    customercol.CustomerColumns.ID.Eq(),
		version: versionOf(func(entity *customer.Customer) int64 { return entity.Version }),
	},
	permission.ResourceShipment: {
		model:   func() versioned { return new(shipment.Shipment) },
		scope:   shipmentcol.ShipmentScopeTenant,
		idEq:    shipmentcol.ShipmentColumns.ID.Eq(),
		version: versionOf(func(entity *shipment.Shipment) int64 { return entity.Version }),
	},
	permission.ResourceShipmentMove: {
		model:   func() versioned { return new(shipment.ShipmentMove) },
		scope:   shipmentcol.ShipmentMoveScopeTenant,
		idEq:    shipmentcol.ShipmentMoveColumns.ID.Eq(),
		version: versionOf(func(entity *shipment.ShipmentMove) int64 { return entity.Version }),
	},
	permission.ResourceDocument: {
		model:   func() versioned { return new(document.Document) },
		scope:   documentcol.DocumentScopeTenant,
		idEq:    documentcol.DocumentColumns.ID.Eq(),
		version: versionOf(func(entity *document.Document) int64 { return entity.Version }),
	},
	permission.ResourceDashboard: {
		model:   func() versioned { return new(report.Dashboard) },
		scope:   reportcol.DashboardScopeTenant,
		idEq:    reportcol.DashboardColumns.ID.Eq(),
		version: versionOf(func(entity *report.Dashboard) int64 { return entity.Version }),
	},
	permission.ResourceWorkerPTO: {
		model:   func() versioned { return new(worker.WorkerPTO) },
		scope:   workercol.WorkerPTOScopeTenant,
		idEq:    workercol.WorkerPTOColumns.ID.Eq(),
		version: versionOf(func(entity *worker.WorkerPTO) int64 { return entity.Version }),
	},
	permission.ResourceBillingQueue: {
		model: func() versioned { return new(billingqueue.BillingQueueItem) },
		scope: billingqueuecol.BillingQueueItemScopeTenant,
		idEq:  billingqueuecol.BillingQueueItemColumns.ID.Eq(),
		version: versionOf(
			func(entity *billingqueue.BillingQueueItem) int64 { return entity.Version },
		),
	},
	permission.ResourceBankReceipt: {
		model:   func() versioned { return new(bankreceipt.BankReceipt) },
		scope:   bankreceiptcol.BankReceiptScopeTenant,
		idEq:    bankreceiptcol.BankReceiptColumns.ID.Eq(),
		version: versionOf(func(entity *bankreceipt.BankReceipt) int64 { return entity.Version }),
	},
	permission.ResourceBankReceiptWorkItem: {
		model: func() versioned { return new(bankreceiptworkitem.WorkItem) },
		scope: bankreceiptworkitemcol.WorkItemScopeTenant,
		idEq:  bankreceiptworkitemcol.WorkItemColumns.ID.Eq(),
		version: versionOf(
			func(entity *bankreceiptworkitem.WorkItem) int64 { return entity.Version },
		),
	},
	permission.ResourceInsight: {
		model:   func() versioned { return new(insight.Insight) },
		scope:   insightcol.InsightScopeTenant,
		idEq:    insightcol.InsightColumns.ID.Eq(),
		version: versionOf(func(entity *insight.Insight) int64 { return entity.Version }),
	},
	permission.ResourceInboundMessage: {
		model: func() versioned { return new(inboundmessage.InboundMessage) },
		scope: inboundmessagecol.InboundMessageScopeTenant,
		idEq:  inboundmessagecol.InboundMessageColumns.ID.Eq(),
		version: versionOf(
			func(entity *inboundmessage.InboundMessage) int64 { return entity.Version },
		),
	},
	permission.ResourceWorker: {
		model:   func() versioned { return new(worker.Worker) },
		scope:   workercol.WorkerScopeTenant,
		idEq:    workercol.WorkerColumns.ID.Eq(),
		version: versionOf(func(entity *worker.Worker) int64 { return entity.Version }),
	},
	permission.ResourceServiceFailure: {
		model: func() versioned { return new(servicefailure.ServiceFailure) },
		scope: servicefailurecol.ServiceFailureScopeTenant,
		idEq:  servicefailurecol.ServiceFailureColumns.ID.Eq(),
		version: versionOf(
			func(entity *servicefailure.ServiceFailure) int64 { return entity.Version },
		),
	},
	permission.ResourceReport: {kinds: map[string]lookup{
		"rd_": {
			model: func() versioned { return new(report.ReportDefinition) },
			scope: reportcol.ReportDefinitionScopeTenant,
			idEq:  reportcol.ReportDefinitionColumns.ID.Eq(),
			version: versionOf(
				func(entity *report.ReportDefinition) int64 { return entity.Version },
			),
		},
		"rsch_": {
			model: func() versioned { return new(report.ReportSchedule) },
			scope: reportcol.ReportScheduleScopeTenant,
			idEq:  reportcol.ReportScheduleColumns.ID.Eq(),
			version: versionOf(
				func(entity *report.ReportSchedule) int64 { return entity.Version },
			),
		},
		"rrun_": {
			model:   func() versioned { return new(report.ReportRun) },
			scope:   reportcol.ReportRunScopeTenant,
			idEq:    reportcol.ReportRunColumns.ID.Eq(),
			version: versionOf(func(entity *report.ReportRun) int64 { return entity.Version }),
		},
	}},
	// The detention tools act on an occurrence under the detention policy
	// resource, which is the permission a person needs to act on one.
	permission.ResourceDetentionPolicy: {
		model: func() versioned { return new(detention.DetentionOccurrence) },
		scope: detentioncol.DetentionOccurrenceScopeTenant,
		idEq:  detentioncol.DetentionOccurrenceColumns.ID.Eq(),
		version: versionOf(
			func(entity *detention.DetentionOccurrence) int64 { return entity.Version },
		),
	},
	// The sync tools act on one ledger record under the accounting sync
	// resource; its version moves each time it is claimed, sent or failed.
	permission.ResourceAccountingSync: {
		model: func() versioned { return new(accountingsync.AccountingSyncRecord) },
		scope: accountingsynccol.AccountingSyncRecordScopeTenant,
		idEq:  accountingsynccol.AccountingSyncRecordColumns.ID.Eq(),
		version: func(v versioned) int64 {
			record, ok := v.(*accountingsync.AccountingSyncRecord)
			if !ok {
				return 0
			}
			return record.Version
		},
	},
	// A tender is withdrawn as a whole; its version moves as the tender
	// advances or ends.
	permission.ResourceTender: {
		model: func() versioned { return new(tender.Tender) },
		scope: tendercol.TenderScopeTenant,
		idEq:  tendercol.TenderColumns.ID.Eq(),
		version: func(v versioned) int64 {
			entity, ok := v.(*tender.Tender)
			if !ok {
				return 0
			}
			return entity.Version
		},
	},
	permission.ResourceRateConfirmation: {
		model: func() versioned { return new(rateconfirmation.RateConfirmation) },
		scope: rateconfirmationcol.RateConfirmationScopeTenant,
		idEq:  rateconfirmationcol.RateConfirmationColumns.ID.Eq(),
		version: func(v versioned) int64 {
			entity, ok := v.(*rateconfirmation.RateConfirmation)
			if !ok {
				return 0
			}
			return entity.Version
		},
	},
	// An invoice is posted or sent as a whole; its version moves with each
	// change to it, so an approval refuses one edited since it was proposed.
	permission.ResourceInvoice: {
		model: func() versioned { return new(invoice.Invoice) },
		scope: invoicecol.InvoiceScopeTenant,
		idEq:  invoicecol.InvoiceColumns.ID.Eq(),
		version: func(v versioned) int64 {
			entity, ok := v.(*invoice.Invoice)
			if !ok {
				return 0
			}
			return entity.Version
		},
	},
	permission.ResourceEDI: {kinds: map[string]lookup{
		"edilt_": {
			model:   func() versioned { return new(edi.EDITransfer) },
			scope:   editransferrepository.ScopeTenant,
			idEq:    edicol.EDITransferColumns.ID.Eq(),
			version: versionOf(func(entity *edi.EDITransfer) int64 { return entity.Version }),
		},
		"editcg_": {
			model:   func() versioned { return new(edi.TenderChange) },
			scope:   editenderchangerepository.ScopeTenant,
			idEq:    edicol.TenderChangeColumns.ID.Eq(),
			version: versionOf(func(entity *edi.TenderChange) int64 { return entity.Version }),
		},
		"editc_": {
			model:   func() versioned { return new(edi.TransferChange) },
			scope:   editransferchangerepository.ScopeTenant,
			idEq:    edicol.TransferChangeColumns.ID.Eq(),
			version: versionOf(func(entity *edi.TransferChange) int64 { return entity.Version }),
		},
		"edimsg_": {
			model:   func() versioned { return new(edi.EDIMessage) },
			scope:   edicol.EDIMessageScopeTenant,
			idEq:    edicol.EDIMessageColumns.ID.Eq(),
			version: versionOf(func(entity *edi.EDIMessage) int64 { return entity.Version }),
		},
		"ediinf_": {
			model:   func() versioned { return new(edi.EDIInboundFile) },
			scope:   edicol.EDIInboundFileScopeTenant,
			idEq:    edicol.EDIInboundFileColumns.ID.Eq(),
			version: versionOf(func(entity *edi.EDIInboundFile) int64 { return entity.Version }),
		},
	}},
	permission.ResourceFuelPurchase: {kinds: map[string]lookup{
		"fpur_": {
			model: func() versioned { return new(fuelpurchase.FuelPurchase) },
			scope: fuelpurchasecol.FuelPurchaseScopeTenant,
			idEq:  fuelpurchasecol.FuelPurchaseColumns.ID.Eq(),
			version: versionOf(
				func(entity *fuelpurchase.FuelPurchase) int64 { return entity.Version },
			),
		},
		"fpib_": fuelImportLookup,
	}},
	permission.ResourceFuelPurchaseImport: fuelImportLookup,
	permission.ResourceFuelCard: {
		model:   func() versioned { return new(fuelpurchase.FuelCard) },
		scope:   fuelpurchasecol.FuelCardScopeTenant,
		idEq:    fuelpurchasecol.FuelCardColumns.ID.Eq(),
		version: versionOf(func(entity *fuelpurchase.FuelCard) int64 { return entity.Version }),
	},
	permission.ResourceIFTAReturn: {
		model:   func() versioned { return new(ifta.Return) },
		scope:   iftacol.ReturnScopeTenant,
		idEq:    iftacol.ReturnColumns.ID.Eq(),
		version: versionOf(func(entity *ifta.Return) int64 { return entity.Version }),
	},
	permission.ResourceIFTAJurisdictionMileage: {
		model: func() versioned { return new(ifta.JurisdictionMileageEntry) },
		scope: iftacol.JurisdictionMileageEntryScopeTenant,
		idEq:  iftacol.JurisdictionMileageEntryColumns.ID.Eq(),
		version: versionOf(
			func(entity *ifta.JurisdictionMileageEntry) int64 { return entity.Version },
		),
	},
	permission.ResourceRateAgreement: {kinds: map[string]lookup{
		"rag_": {
			model: func() versioned { return new(rateagreement.RateAgreement) },
			scope: rateagreementcol.RateAgreementScopeTenant,
			idEq:  rateagreementcol.RateAgreementColumns.ID.Eq(),
			version: versionOf(
				func(entity *rateagreement.RateAgreement) int64 { return entity.Version },
			),
		},
		"rib_": {
			model: func() versioned { return new(rateimport.RateImportBatch) },
			scope: rateimportcol.RateImportBatchScopeTenant,
			idEq:  rateimportcol.RateImportBatchColumns.ID.Eq(),
			version: versionOf(
				func(entity *rateimport.RateImportBatch) int64 { return entity.Version },
			),
		},
	}},
	permission.ResourceInvoiceDispute: {
		model:   func() versioned { return new(invoice.InvoiceDispute) },
		scope:   invoicecol.InvoiceDisputeScopeTenant,
		idEq:    invoicecol.InvoiceDisputeColumns.ID.Eq(),
		version: versionOf(func(entity *invoice.InvoiceDispute) int64 { return entity.Version }),
	},
	permission.ResourceFactoringNoa: {
		model:   func() versioned { return new(factoring.Noa) },
		scope:   factoringcol.NoaScopeTenant,
		idEq:    factoringcol.NoaColumns.ID.Eq(),
		version: versionOf(func(entity *factoring.Noa) int64 { return entity.Version }),
	},
	permission.ResourceFactoringSubmission: {
		model:   func() versioned { return new(factoring.Submission) },
		scope:   factoringcol.SubmissionScopeTenant,
		idEq:    factoringcol.SubmissionColumns.ID.Eq(),
		version: versionOf(func(entity *factoring.Submission) int64 { return entity.Version }),
	},
	permission.ResourceFactoringCompany: {
		model: func() versioned { return new(factoringcompany.FactoringCompany) },
		scope: factoringcompanycol.FactoringCompanyScopeTenant,
		idEq:  factoringcompanycol.FactoringCompanyColumns.ID.Eq(),
		version: versionOf(
			func(entity *factoringcompany.FactoringCompany) int64 { return entity.Version },
		),
	},
	permission.ResourceInvoiceRun: {
		model:   func() versioned { return new(invoicerun.InvoiceRun) },
		scope:   invoiceruncol.InvoiceRunScopeTenant,
		idEq:    invoiceruncol.InvoiceRunColumns.ID.Eq(),
		version: versionOf(func(entity *invoicerun.InvoiceRun) int64 { return entity.Version }),
	},
	permission.ResourceCustomerPayment: {
		model:   func() versioned { return new(customerpayment.Payment) },
		scope:   customerpaymentcol.PaymentScopeTenant,
		idEq:    customerpaymentcol.PaymentColumns.ID.Eq(),
		version: versionOf(func(entity *customerpayment.Payment) int64 { return entity.Version }),
	},
	agentports.RecordInvoiceAdjustment: {
		model: func() versioned { return new(invoiceadjustment.InvoiceAdjustment) },
		scope: invoiceadjustmentcol.InvoiceAdjustmentScopeTenant,
		idEq:  invoiceadjustmentcol.InvoiceAdjustmentColumns.ID.Eq(),
		version: versionOf(
			func(entity *invoiceadjustment.InvoiceAdjustment) int64 { return entity.Version },
		),
	},
	agentports.RecordCreditMemoApplication: {
		model: func() versioned { return new(customerpayment.CreditMemoApplication) },
		scope: customerpaymentcol.CreditMemoApplicationScopeTenant,
		idEq:  customerpaymentcol.CreditMemoApplicationColumns.ID.Eq(),
		version: versionOf(
			func(entity *customerpayment.CreditMemoApplication) int64 { return entity.UpdatedAt },
		),
	},
	// Likewise the carrier intelligence tools act on one event.
	permission.ResourceCarrierIntelligence: {
		model: func() versioned { return new(carrierintel.CarrierIntelEvent) },
		scope: carrierintelcol.CarrierIntelEventScopeTenant,
		idEq:  carrierintelcol.CarrierIntelEventColumns.ID.Eq(),
		version: versionOf(
			func(entity *carrierintel.CarrierIntelEvent) int64 { return entity.Version },
		),
	},
	permission.ResourceDriverSettlement: {
		model: func() versioned { return new(driversettlement.Settlement) },
		scope: driversettlementcol.SettlementScopeTenant,
		idEq:  driversettlementcol.SettlementColumns.ID.Eq(),
		version: func(v versioned) int64 {
			entity, ok := v.(*driversettlement.Settlement)
			if !ok {
				return 0
			}
			return entity.Version
		},
	},
	permission.ResourceCarrierSettlement: {
		model: func() versioned { return new(carriersettlement.CarrierSettlement) },
		scope: carriersettlementcol.CarrierSettlementScopeTenant,
		idEq:  carriersettlementcol.CarrierSettlementColumns.ID.Eq(),
		version: func(v versioned) int64 {
			entity, ok := v.(*carriersettlement.CarrierSettlement)
			if !ok {
				return 0
			}
			return entity.Version
		},
	},
	permission.ResourceCarrierInvoiceMatch: {
		model: func() versioned { return new(carriersettlement.InvoiceMatch) },
		scope: carriersettlementcol.InvoiceMatchScopeTenant,
		idEq:  carriersettlementcol.InvoiceMatchColumns.ID.Eq(),
		version: func(v versioned) int64 {
			entity, ok := v.(*carriersettlement.InvoiceMatch)
			if !ok {
				return 0
			}
			return entity.Version
		},
	},
	permission.ResourceEscrowAccount: {
		model: func() versioned { return new(driverpay.EscrowAccount) },
		scope: driverpaycol.EscrowAccountScopeTenant,
		idEq:  driverpaycol.EscrowAccountColumns.ID.Eq(),
		version: func(v versioned) int64 {
			entity, ok := v.(*driverpay.EscrowAccount)
			if !ok {
				return 0
			}
			return entity.Version
		},
	},
	permission.ResourcePayAdvance: {
		model: func() versioned { return new(driverpay.PayAdvance) },
		scope: driverpaycol.PayAdvanceScopeTenant,
		idEq:  driverpaycol.PayAdvanceColumns.ID.Eq(),
		version: func(v versioned) int64 {
			entity, ok := v.(*driverpay.PayAdvance)
			if !ok {
				return 0
			}
			return entity.Version
		},
	},
	permission.ResourceRecurringDeduction: {
		model: func() versioned { return new(driverpay.RecurringDeduction) },
		scope: driverpaycol.RecurringDeductionScopeTenant,
		idEq:  driverpaycol.RecurringDeductionColumns.ID.Eq(),
		version: func(v versioned) int64 {
			entity, ok := v.(*driverpay.RecurringDeduction)
			if !ok {
				return 0
			}
			return entity.Version
		},
	},
	permission.ResourceManualJournal: {
		model:   func() versioned { return new(manualjournal.Request) },
		scope:   manualjournalcol.RequestScopeTenant,
		idEq:    manualjournalcol.RequestColumns.ID.Eq(),
		version: versionOf(func(entity *manualjournal.Request) int64 { return entity.Version }),
	},
	permission.ResourceJournalReversal: {
		model:   func() versioned { return new(journalreversal.Reversal) },
		scope:   journalreversalcol.ReversalScopeTenant,
		idEq:    journalreversalcol.ReversalColumns.ID.Eq(),
		version: versionOf(func(entity *journalreversal.Reversal) int64 { return entity.Version }),
	},
	permission.ResourceJournalEntry: {
		model:   func() versioned { return new(journalentry.JournalEntry) },
		scope:   journalentrycol.JournalEntryScopeTenant,
		idEq:    journalentrycol.JournalEntryColumns.ID.Eq(),
		version: versionOf(func(entity *journalentry.JournalEntry) int64 { return entity.Version }),
	},
	permission.ResourceFiscalPeriod: {
		model:   func() versioned { return new(fiscalperiod.FiscalPeriod) },
		scope:   fiscalperiodcol.FiscalPeriodScopeTenant,
		idEq:    fiscalperiodcol.FiscalPeriodColumns.ID.Eq(),
		version: versionOf(func(entity *fiscalperiod.FiscalPeriod) int64 { return entity.Version }),
	},
	permission.ResourceRecurringEarning: {
		model: func() versioned { return new(driverpay.RecurringEarning) },
		scope: driverpaycol.RecurringEarningScopeTenant,
		idEq:  driverpaycol.RecurringEarningColumns.ID.Eq(),
		version: func(v versioned) int64 {
			entity, ok := v.(*driverpay.RecurringEarning)
			if !ok {
				return 0
			}
			return entity.Version
		},
	},
	permission.ResourceWorkerDOTTest: {
		model:   func() versioned { return new(worker.WorkerDOTTest) },
		scope:   workercol.WorkerDOTTestScopeTenant,
		idEq:    workercol.WorkerDOTTestColumns.ID.Eq(),
		version: versionOf(func(entity *worker.WorkerDOTTest) int64 { return entity.Version }),
	},
	permission.ResourceDOTRandomPool: {kinds: map[string]lookup{
		"drdraw_": {
			model:   func() versioned { return new(worker.DOTRandomDraw) },
			scope:   workercol.DOTRandomDrawScopeTenant,
			idEq:    workercol.DOTRandomDrawColumns.ID.Eq(),
			version: versionOf(func(entity *worker.DOTRandomDraw) int64 { return entity.Version }),
		},
		"drde_": {
			model: func() versioned { return new(worker.DOTRandomDrawEntry) },
			scope: workercol.DOTRandomDrawEntryScopeTenant,
			idEq:  workercol.DOTRandomDrawEntryColumns.ID.Eq(),
			version: versionOf(
				func(entity *worker.DOTRandomDrawEntry) int64 { return entity.Version },
			),
		},
	}},
	permission.ResourceWorkerSafetyEvent: {kinds: map[string]lookup{
		"wsev_": {
			model: func() versioned { return new(worker.WorkerSafetyEvent) },
			scope: workercol.WorkerSafetyEventScopeTenant,
			idEq:  workercol.WorkerSafetyEventColumns.ID.Eq(),
			version: versionOf(
				func(entity *worker.WorkerSafetyEvent) int64 { return entity.Version },
			),
		},
		"wsvi_": {
			model: func() versioned { return new(worker.WorkerSafetyViolation) },
			scope: workercol.WorkerSafetyViolationScopeTenant,
			idEq:  workercol.WorkerSafetyViolationColumns.ID.Eq(),
			version: versionOf(
				func(entity *worker.WorkerSafetyViolation) int64 { return entity.Version },
			),
		},
	}},
	permission.ResourceWorkerRecognition: {
		model:   func() versioned { return new(worker.WorkerRecognition) },
		scope:   workercol.WorkerRecognitionScopeTenant,
		idEq:    workercol.WorkerRecognitionColumns.ID.Eq(),
		version: versionOf(func(entity *worker.WorkerRecognition) int64 { return entity.Version }),
	},
	permission.ResourceWorkerLeave: {kinds: map[string]lookup{
		"wlc_": {
			model: func() versioned { return new(worker.WorkerLeaveCase) },
			scope: workercol.WorkerLeaveCaseScopeTenant,
			idEq:  workercol.WorkerLeaveCaseColumns.ID.Eq(),
			version: versionOf(
				func(entity *worker.WorkerLeaveCase) int64 { return entity.Version },
			),
		},
		"wle_": {
			model: func() versioned { return new(worker.WorkerLeaveEntry) },
			scope: workercol.WorkerLeaveEntryScopeTenant,
			idEq:  workercol.WorkerLeaveEntryColumns.ID.Eq(),
			version: versionOf(
				func(entity *worker.WorkerLeaveEntry) int64 { return entity.Version },
			),
		},
	}},
	permission.ResourceWorkerTraining: {
		model: func() versioned { return new(worker.WorkerTrainingRecord) },
		scope: workercol.WorkerTrainingRecordScopeTenant,
		idEq:  workercol.WorkerTrainingRecordColumns.ID.Eq(),
		version: versionOf(
			func(entity *worker.WorkerTrainingRecord) int64 { return entity.Version },
		),
	},
	permission.ResourceWorkerChecklist: {kinds: map[string]lookup{
		"wcl_": {
			model: func() versioned { return new(worker.WorkerChecklist) },
			scope: workercol.WorkerChecklistScopeTenant,
			idEq:  workercol.WorkerChecklistColumns.ID.Eq(),
			version: versionOf(
				func(entity *worker.WorkerChecklist) int64 { return entity.Version },
			),
		},
		"wcli_": {
			model: func() versioned { return new(worker.WorkerChecklistItem) },
			scope: workercol.WorkerChecklistItemScopeTenant,
			idEq:  workercol.WorkerChecklistItemColumns.ID.Eq(),
			version: versionOf(
				func(entity *worker.WorkerChecklistItem) int64 { return entity.Version },
			),
		},
	}},
	permission.ResourcePerformanceReview: {
		model:   func() versioned { return new(worker.PerformanceReview) },
		scope:   workercol.PerformanceReviewScopeTenant,
		idEq:    workercol.PerformanceReviewColumns.ID.Eq(),
		version: versionOf(func(entity *worker.PerformanceReview) int64 { return entity.Version }),
	},
	permission.ResourceQualification: {
		model: func() versioned { return new(worker.WorkerEmploymentVerification) },
		scope: workercol.WorkerEmploymentVerificationScopeTenant,
		idEq:  workercol.WorkerEmploymentVerificationColumns.ID.Eq(),
		version: versionOf(
			func(entity *worker.WorkerEmploymentVerification) int64 { return entity.Version },
		),
	},
	permission.ResourceWorkerCredential: {
		model:   func() versioned { return new(worker.WorkerCredential) },
		scope:   workercol.WorkerCredentialScopeTenant,
		idEq:    workercol.WorkerCredentialColumns.ID.Eq(),
		version: versionOf(func(entity *worker.WorkerCredential) int64 { return entity.Version }),
	},
	permission.ResourceWorkerInjury: {
		model:   func() versioned { return new(worker.WorkerInjury) },
		scope:   workercol.WorkerInjuryScopeTenant,
		idEq:    workercol.WorkerInjuryColumns.ID.Eq(),
		version: versionOf(func(entity *worker.WorkerInjury) int64 { return entity.Version }),
	},
	permission.ResourceTimesheet: {kinds: map[string]lookup{
		"pxb_": {
			model:   func() versioned { return new(worker.PayrollExport) },
			scope:   workercol.PayrollExportScopeTenant,
			idEq:    workercol.PayrollExportColumns.ID.Eq(),
			version: versionOf(func(entity *worker.PayrollExport) int64 { return entity.Version }),
		},
	}},
	permission.ResourcePermit: {
		model:   func() versioned { return new(permit.Permit) },
		scope:   permitcol.PermitScopeTenant,
		idEq:    permitcol.PermitColumns.ID.Eq(),
		version: versionOf(func(entity *permit.Permit) int64 { return entity.Version }),
	},
	permission.ResourceSettlementDispute: {
		model:   func() versioned { return new(driversettlement.Dispute) },
		scope:   driversettlementcol.DisputeScopeTenant,
		idEq:    driversettlementcol.DisputeColumns.ID.Eq(),
		version: versionOf(func(entity *driversettlement.Dispute) int64 { return entity.Version }),
	},
	permission.ResourceDriverExpense: {
		model:   func() versioned { return new(driverpay.Expense) },
		scope:   driverpaycol.ExpenseScopeTenant,
		idEq:    driverpaycol.ExpenseColumns.ID.Eq(),
		version: versionOf(func(entity *driverpay.Expense) int64 { return entity.Version }),
	}, permission.ResourceCommodity: {
		model:   func() versioned { return new(commodity.Commodity) },
		scope:   commoditycol.CommodityScopeTenant,
		idEq:    commoditycol.CommodityColumns.ID.Eq(),
		version: versionOf(func(entity *commodity.Commodity) int64 { return entity.Version }),
	},
	permission.ResourceCarrierCapacityPosting: {
		model:   func() versioned { return new(carriercapacity.Posting) },
		scope:   carriercapacitycol.PostingScopeTenant,
		idEq:    carriercapacitycol.PostingColumns.ID.Eq(),
		version: versionOf(func(entity *carriercapacity.Posting) int64 { return entity.Version }),
	},
	permission.ResourceHazardousMaterial: {
		model: func() versioned { return new(hazardousmaterial.HazardousMaterial) },
		scope: hazardousmaterialcol.HazardousMaterialScopeTenant,
		idEq:  hazardousmaterialcol.HazardousMaterialColumns.ID.Eq(),
		version: versionOf(
			func(entity *hazardousmaterial.HazardousMaterial) int64 { return entity.Version },
		),
	},
	permission.ResourceLocation: {
		model:   func() versioned { return new(location.Location) },
		scope:   locationcol.LocationScopeTenant,
		idEq:    locationcol.LocationColumns.ID.Eq(),
		version: versionOf(func(entity *location.Location) int64 { return entity.Version }),
	},
	permission.ResourceTractor: {
		model:   func() versioned { return new(tractor.Tractor) },
		scope:   tractorcol.TractorScopeTenant,
		idEq:    tractorcol.TractorColumns.ID.Eq(),
		version: versionOf(func(entity *tractor.Tractor) int64 { return entity.Version }),
	},
	permission.ResourceTrailer: {
		model:   func() versioned { return new(trailer.Trailer) },
		scope:   trailercol.TrailerScopeTenant,
		idEq:    trailercol.TrailerColumns.ID.Eq(),
		version: versionOf(func(entity *trailer.Trailer) int64 { return entity.Version }),
	},
	permission.ResourceTableChangeAlert: {
		model: func() versioned { return new(tablechangealert.TCASubscription) },
		scope: tablechangealertcol.TCASubscriptionScopeTenant,
		idEq:  tablechangealertcol.TCASubscriptionColumns.ID.Eq(),
		version: versionOf(
			func(entity *tablechangealert.TCASubscription) int64 { return entity.Version },
		),
	},
	permission.ResourceWatchtower: {
		model:   func() versioned { return new(watchtower.Item) },
		scope:   watchtowercol.ItemScopeTenant,
		idEq:    watchtowercol.ItemColumns.ID.Eq(),
		version: versionOf(func(entity *watchtower.Item) int64 { return entity.Version }),
	},
}

func versionOf[T any](read func(*T) int64) func(versioned) int64 {
	return func(v versioned) int64 {
		entity, ok := v.(*T)
		if !ok {
			return 0
		}

		return read(entity)
	}
}

// Version reads the record's current version inside the tenant. A record that
// is not there is an error, not a zero: a proposal against a deleted record
// must not compare equal to anything.
func (r *Reader) Version(
	ctx context.Context,
	tenant pagination.TenantInfo,
	target agentports.ToolTarget,
) (int64, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int64, error) {
		entry, ok := lookups[target.Resource]
		if ok && entry.kinds != nil {
			entry, ok = entry.kinds[target.ID.Prefix()]
		}
		if !ok {
			return 0, fmt.Errorf("%w: %s", agentports.ErrRecordVersionUnsupported, target.Resource)
		}

		record := entry.model()
		query := r.db.DBForContext(ctx).NewSelect().Model(record)
		query = entry.scope(query, tenant).Where(entry.idEq, target.ID).Limit(1)
		if err := query.Scan(ctx); err != nil {
			return 0, fmt.Errorf("read %s %s version: %w", target.Resource, target.ID, err)
		}

		return entry.version(record), nil
	})
}
