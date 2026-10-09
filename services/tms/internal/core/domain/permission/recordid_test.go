package permission_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentwait"
	"github.com/emoss08/trenova/internal/core/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/core/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/billingtransfer"
	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/carriercapacity"
	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/fleetcode"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/holdreason"
	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/domain/invoicerun"
	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/internal/core/domain/journalreversal"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/locationcategory"
	"github.com/emoss08/trenova/internal/core/domain/manualjournal"
	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/permit"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/core/domain/rateimport"
	"github.com/emoss08/trenova/internal/core/domain/ratematrix"
	"github.com/emoss08/trenova/internal/core/domain/recurringshipment"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/tablechangealert"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

type minter func(context.Context) (pulid.ID, error)

// mint inserts nothing: it runs the entity's own insert hook, which is where
// the domain gives a new record its id, and reads the id back.
func mint[T any, P interface {
	*T
	bun.BeforeAppendModelHook
}](id func(P) pulid.ID) minter {
	return func(ctx context.Context) (pulid.ID, error) {
		entity := P(new(T))
		err := entity.BeforeAppendModel(ctx, &bun.InsertQuery{})

		return id(entity), err
	}
}

var minters = map[permission.Resource]minter{
	permission.ResourceShipment: mint(func(e *shipment.Shipment) pulid.ID { return e.ID }),
	permission.ResourceShipmentMove: mint(
		func(e *shipment.ShipmentMove) pulid.ID { return e.ID },
	),
	permission.ResourceShipmentComment: mint(
		func(e *shipment.ShipmentComment) pulid.ID { return e.ID },
	),
	permission.ResourceOrder: mint(func(e *order.Order) pulid.ID { return e.ID }),
	permission.ResourceRecurringShipment: mint(
		func(e *recurringshipment.RecurringShipment) pulid.ID { return e.ID },
	),
	permission.ResourceServiceFailure: mint(
		func(e *servicefailure.ServiceFailure) pulid.ID { return e.ID },
	),
	permission.ResourceCustomer: mint(func(e *customer.Customer) pulid.ID { return e.ID }),
	permission.ResourceCarrier:  mint(func(e *carrier.Carrier) pulid.ID { return e.ID }),
	permission.ResourceLocation: mint(func(e *location.Location) pulid.ID { return e.ID }),
	permission.ResourceWorker:   mint(func(e *worker.Worker) pulid.ID { return e.ID }),
	permission.ResourceTractor:  mint(func(e *tractor.Tractor) pulid.ID { return e.ID }),
	permission.ResourceTrailer:  mint(func(e *trailer.Trailer) pulid.ID { return e.ID }),
	permission.ResourceCommodity: mint(
		func(e *commodity.Commodity) pulid.ID { return e.ID },
	),
	permission.ResourceHazardousMaterial: mint(
		func(e *hazardousmaterial.HazardousMaterial) pulid.ID { return e.ID },
	),
	permission.ResourceDocument: mint(func(e *document.Document) pulid.ID { return e.ID }),
	permission.ResourceInvoice:  mint(func(e *invoice.Invoice) pulid.ID { return e.ID }),
	permission.ResourceInvoiceDispute: mint(
		func(e *invoice.InvoiceDispute) pulid.ID { return e.ID },
	),
	permission.ResourceInvoiceRun: mint(
		func(e *invoicerun.InvoiceRun) pulid.ID { return e.ID },
	),
	permission.ResourceRateAgreement: mint(
		func(e *rateagreement.RateAgreement) pulid.ID { return e.ID },
	),
	permission.ResourceCustomerPayment: mint(
		func(e *customerpayment.Payment) pulid.ID { return e.ID },
	),
	permission.ResourceReport: mint(
		func(e *report.ReportDefinition) pulid.ID { return e.ID },
	),
	permission.ResourceDashboard: mint(func(e *report.Dashboard) pulid.ID { return e.ID }),
	permission.ResourceWorkerCredential: mint(
		func(e *worker.WorkerCredential) pulid.ID { return e.ID },
	),
	permission.ResourceWorkerTraining: mint(
		func(e *worker.WorkerTrainingRecord) pulid.ID { return e.ID },
	),
	permission.ResourceWorkerSafetyEvent: mint(
		func(e *worker.WorkerSafetyEvent) pulid.ID { return e.ID },
	),
	permission.ResourcePerformanceReview: mint(
		func(e *worker.PerformanceReview) pulid.ID { return e.ID },
	),
	permission.ResourceWorkerInjury: mint(
		func(e *worker.WorkerInjury) pulid.ID { return e.ID },
	),
	permission.ResourceWorkerLeave: mint(
		func(e *worker.WorkerLeaveCase) pulid.ID { return e.ID },
	),
	permission.ResourceDriverSettlement: mint(
		func(e *driversettlement.Settlement) pulid.ID { return e.ID },
	),
	permission.ResourceSettlementDispute: mint(
		func(e *driversettlement.Dispute) pulid.ID { return e.ID },
	),
	permission.ResourceCaptureBatch: mint(
		func(e *capture.CaptureBatch) pulid.ID { return e.ID },
	),
	permission.ResourceOrganization: mint(
		func(e *tenant.Organization) pulid.ID { return e.ID },
	),
	permission.ResourceUser: mint(
		func(e *tenant.User) pulid.ID { return e.ID },
	),
	permission.ResourceAgentRun: mint(
		func(e *agent.AgentRun) pulid.ID { return e.ID },
	),
	permission.ResourceAgentMemory: mint(
		func(e *agent.Memory) pulid.ID { return e.ID },
	),
	permission.ResourceInsight: mint(
		func(e *insight.Insight) pulid.ID { return e.ID },
	),
	permission.ResourceTableChangeAlert: mint(
		func(e *tablechangealert.TCASubscription) pulid.ID { return e.ID },
	),
	permission.ResourceEmailProfile: mint(
		func(e *email.Profile) pulid.ID { return e.ID },
	),
	permission.ResourceInboundMessage: mint(
		func(e *inboundmessage.InboundMessage) pulid.ID { return e.ID },
	),
	permission.ResourceInboundMailbox: mint(
		func(e *inboundmessage.Mailbox) pulid.ID { return e.ID },
	),
	permission.ResourceShipmentStop: mint(
		func(e *shipment.Stop) pulid.ID { return e.ID },
	),
	permission.ResourceShipmentHold: mint(
		func(e *shipment.ShipmentHold) pulid.ID { return e.ID },
	),
	permission.ResourceHoldReason: mint(
		func(e *holdreason.HoldReason) pulid.ID { return e.ID },
	),
	permission.ResourceServiceFailureReasonCode: mint(
		func(e *servicefailure.ReasonCode) pulid.ID { return e.ID },
	),
	permission.ResourcePermit: mint(
		func(e *permit.Permit) pulid.ID { return e.ID },
	),
	permission.ResourceServiceType: mint(
		func(e *servicetype.ServiceType) pulid.ID { return e.ID },
	),
	permission.ResourceShipmentType: mint(
		func(e *shipmenttype.ShipmentType) pulid.ID { return e.ID },
	),
	permission.ResourceDocumentType: mint(
		func(e *documenttype.DocumentType) pulid.ID { return e.ID },
	),
	permission.ResourceEquipmentType: mint(
		func(e *equipmenttype.EquipmentType) pulid.ID { return e.ID },
	),
	permission.ResourceEquipmentManufacturer: mint(
		func(e *equipmentmanufacturer.EquipmentManufacturer) pulid.ID { return e.ID },
	),
	permission.ResourceFleetCode: mint(
		func(e *fleetcode.FleetCode) pulid.ID { return e.ID },
	),
	permission.ResourceLocationCategory: mint(
		func(e *locationcategory.LocationCategory) pulid.ID { return e.ID },
	),
	permission.ResourceCarrierCapacityPosting: mint(
		func(e *carriercapacity.Posting) pulid.ID { return e.ID },
	),
	permission.ResourceRateConfirmation: mint(
		func(e *rateconfirmation.RateConfirmation) pulid.ID { return e.ID },
	),
	permission.ResourceCarrierSettlement: mint(
		func(e *carriersettlement.CarrierSettlement) pulid.ID { return e.ID },
	),
	permission.ResourceCarrierInvoiceMatch: mint(
		func(e *carriersettlement.InvoiceMatch) pulid.ID { return e.ID },
	),
	permission.ResourceRoutingGuide: mint(
		func(e *tender.RoutingGuide) pulid.ID { return e.ID },
	),
	permission.ResourceTender: mint(
		func(e *tender.Tender) pulid.ID { return e.ID },
	),
	permission.ResourceAccessorialCharge: mint(
		func(e *accessorialcharge.AccessorialCharge) pulid.ID { return e.ID },
	),
	permission.ResourceFormulaTemplate: mint(
		func(e *formulatemplate.FormulaTemplate) pulid.ID { return e.ID },
	),
	permission.ResourceRateMatrix: mint(
		func(e *ratematrix.RateMatrix) pulid.ID { return e.ID },
	),
	permission.ResourceGeneralLedgerAccount: mint(
		func(e *glaccount.GLAccount) pulid.ID { return e.ID },
	),
	permission.ResourceFiscalPeriod: mint(
		func(e *fiscalperiod.FiscalPeriod) pulid.ID { return e.ID },
	),
	permission.ResourceManualJournal: mint(
		func(e *manualjournal.Request) pulid.ID { return e.ID },
	),
	permission.ResourceJournalEntry: mint(
		func(e *journalentry.JournalEntry) pulid.ID { return e.ID },
	),
	permission.ResourceBankReceipt: mint(
		func(e *bankreceipt.BankReceipt) pulid.ID { return e.ID },
	),
	permission.ResourceBankReceiptWorkItem: mint(
		func(e *bankreceiptworkitem.WorkItem) pulid.ID { return e.ID },
	),
	permission.ResourcePayCode: mint(
		func(e *driverpay.PayCode) pulid.ID { return e.ID },
	),
	permission.ResourcePayAdvance: mint(
		func(e *driverpay.PayAdvance) pulid.ID { return e.ID },
	),
	permission.ResourceEscrowAccount: mint(
		func(e *driverpay.EscrowAccount) pulid.ID { return e.ID },
	),
	permission.ResourceRecurringDeduction: mint(
		func(e *driverpay.RecurringDeduction) pulid.ID { return e.ID },
	),
	permission.ResourceRecurringEarning: mint(
		func(e *driverpay.RecurringEarning) pulid.ID { return e.ID },
	),
	permission.ResourceDriverExpense: mint(
		func(e *driverpay.Expense) pulid.ID { return e.ID },
	),
	permission.ResourceFuelCard: mint(
		func(e *fuelpurchase.FuelCard) pulid.ID { return e.ID },
	),
	permission.ResourceFuelPurchase: mint(
		func(e *fuelpurchase.FuelPurchase) pulid.ID { return e.ID },
	),
	permission.ResourceFuelPurchaseImport: mint(
		func(e *fuelpurchase.ImportBatch) pulid.ID { return e.ID },
	),
	permission.ResourceWorkerPTO: mint(
		func(e *worker.WorkerPTO) pulid.ID { return e.ID },
	),
	permission.ResourceWorkerChecklist: mint(
		func(e *worker.WorkerChecklist) pulid.ID { return e.ID },
	),
	permission.ResourceWorkerChecklistTemplate: mint(
		func(e *worker.WorkerChecklistTemplate) pulid.ID { return e.ID },
	),
	permission.ResourceWorkerCredentialType: mint(
		func(e *worker.WorkerCredentialType) pulid.ID { return e.ID },
	),
	permission.ResourceWorkerRecognition: mint(
		func(e *worker.WorkerRecognition) pulid.ID { return e.ID },
	),
	permission.ResourceWorkerDOTTest: mint(
		func(e *worker.WorkerDOTTest) pulid.ID { return e.ID },
	),
	permission.ResourceTrainingCourse: mint(
		func(e *worker.TrainingCourse) pulid.ID { return e.ID },
	),
	permission.ResourcePerformanceReviewTemplate: mint(
		func(e *worker.PerformanceReviewTemplate) pulid.ID { return e.ID },
	),
	permission.ResourceShiftTemplate: mint(
		func(e *worker.ShiftTemplate) pulid.ID { return e.ID },
	),
	permission.ResourceShiftSwap: mint(
		func(e *worker.ShiftSwapRequest) pulid.ID { return e.ID },
	),
	permission.ResourceJournalReversal: func(context.Context) (pulid.ID, error) {
		return pulid.MustNew(journalreversal.IDPrefix), nil
	},
}

// A prefix the table invents is worse than none: the runtime would refuse a
// real id as the wrong kind. Every entry is held to the id the domain itself
// mints for a new record of that resource.
func TestRecordIDPrefixes_MatchWhatTheDomainMints(t *testing.T) {
	t.Parallel()

	table := permission.RecordIDPrefixes()
	require.NotEmpty(t, table)

	for resource, prefix := range table {
		mintID, tested := minters[resource]
		if !assert.Truef(t, tested, "%s has a prefix and no entity minting it here", resource) {
			continue
		}
		id, err := mintID(t.Context())
		require.NoError(t, err, resource)
		assert.Equalf(t, prefix, id.Prefix(), "%s ids", resource)
	}
}

func TestResourceOfIDPrefix_ReadsTheTableBackwards(t *testing.T) {
	t.Parallel()

	for resource, prefix := range permission.RecordIDPrefixes() {
		got, ok := permission.ResourceOfIDPrefix(prefix)
		require.Truef(t, ok, "prefix %s", prefix)
		assert.Equal(t, resource, got, "two resources share prefix %s", prefix)

		declared, has := resource.IDPrefix()
		assert.True(t, has)
		assert.Equal(t, prefix, declared)
	}

	_, ok := permission.ResourceOfIDPrefix("nope_")
	assert.False(t, ok)
}

func TestResource_Noun(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "worker credential", permission.ResourceWorkerCredential.Noun())
	assert.Equal(t, "shipment", permission.ResourceShipment.Noun())
}

var kindMinters = map[permission.RecordKind]minter{
	permission.KindEmploymentVerification: mint(
		func(e *worker.WorkerEmploymentVerification) pulid.ID { return e.ID },
	),
	permission.KindClearinghouseQuery: mint(
		func(e *worker.WorkerClearinghouseQuery) pulid.ID { return e.ID },
	),
	permission.KindDOTRandomPool: mint(func(e *worker.DOTRandomPool) pulid.ID { return e.ID }),
	permission.KindDOTRandomDraw: mint(func(e *worker.DOTRandomDraw) pulid.ID { return e.ID }),
	permission.KindDOTRandomSelection: mint(
		func(e *worker.DOTRandomDrawEntry) pulid.ID { return e.ID },
	),
	permission.KindConversation: mint(
		func(e *conversation.Thread) pulid.ID { return e.ID },
	),
	permission.KindWait: mint(
		func(e *agentwait.Wait) pulid.ID { return e.ID },
	),
	permission.KindWatchtowerItem: mint(
		func(e *watchtower.Item) pulid.ID { return e.ID },
	),
	permission.KindAccountingConnection: mint(
		func(e *accountingsync.AccountingConnection) pulid.ID { return e.ID },
	),
	permission.KindAccountingMapping: mint(
		func(e *accountingsync.AccountingMapping) pulid.ID { return e.ID },
	),
	permission.KindAccountingSyncRecord: mint(
		func(e *accountingsync.AccountingSyncRecord) pulid.ID { return e.ID },
	),
	permission.KindAccountingInboundChange: mint(
		func(e *accountingsync.AccountingInboundChange) pulid.ID { return e.ID },
	),
	permission.KindAccountingDriftFinding: mint(
		func(e *accountingsync.AccountingDriftFinding) pulid.ID { return e.ID },
	),
	permission.KindCarrierIntelligenceEvent: mint(
		func(e *carrierintel.CarrierIntelEvent) pulid.ID { return e.ID },
	),
	permission.KindInsurancePolicy: mint(
		func(e *carrier.CarrierInsurancePolicy) pulid.ID { return e.ID },
	),
	permission.KindCarrierAssignment: mint(
		func(e *shipment.CarrierAssignment) pulid.ID { return e.ID },
	),
	permission.KindAdditionalCharge: mint(
		func(e *shipment.AdditionalCharge) pulid.ID { return e.ID },
	),
	permission.KindBillingTransferRun: mint(
		func(e *billingtransfer.BillingTransferRun) pulid.ID { return e.ID },
	),
	permission.KindDetentionOccurrence: mint(
		func(e *detention.DetentionOccurrence) pulid.ID { return e.ID },
	),
	permission.KindTenderOffer: mint(
		func(e *tender.TenderOffer) pulid.ID { return e.ID },
	),
	permission.KindOrderCharge: mint(
		func(e *order.OrderCharge) pulid.ID { return e.ID },
	),
	permission.KindBillingQueueItem: mint(
		func(e *billingqueue.BillingQueueItem) pulid.ID { return e.ID },
	),
	permission.KindInvoiceLine: mint(
		func(e *invoice.InvoiceLine) pulid.ID { return e.ID },
	),
	permission.KindInvoiceAdjustment: mint(
		func(e *invoiceadjustment.InvoiceAdjustment) pulid.ID { return e.ID },
	),
	permission.KindInvoiceRunGroup: mint(
		func(e *invoicerun.InvoiceRunGroup) pulid.ID { return e.ID },
	),
	permission.KindInvoiceRunItem: mint(
		func(e *invoicerun.InvoiceRunGroupItem) pulid.ID { return e.ID },
	),
	permission.KindCreditMemoApplication: mint(
		func(e *customerpayment.CreditMemoApplication) pulid.ID { return e.ID },
	),
	permission.KindRateAgreementLane: mint(
		func(e *rateagreement.RateAgreementRule) pulid.ID { return e.ID },
	),
	permission.KindRateImport: mint(
		func(e *rateimport.RateImportBatch) pulid.ID { return e.ID },
	),
	permission.KindFuelIndex: mint(
		func(e *fuelsurcharge.FuelIndex) pulid.ID { return e.ID },
	),
	permission.KindFuelIndexPrice: mint(
		func(e *fuelsurcharge.FuelIndexPrice) pulid.ID { return e.ID },
	),
	permission.KindReportRun: mint(
		func(e *report.ReportRun) pulid.ID { return e.ID },
	),
	permission.KindReportSchedule: mint(
		func(e *report.ReportSchedule) pulid.ID { return e.ID },
	),
	permission.KindEDIPartner: mint(
		func(e *edi.EDIPartner) pulid.ID { return e.ID },
	),
	permission.KindEDIInboundFile: mint(
		func(e *edi.EDIInboundFile) pulid.ID { return e.ID },
	),
	permission.KindEDITransfer: mint(
		func(e *edi.EDITransfer) pulid.ID { return e.ID },
	),
	permission.KindEDIMessage: mint(
		func(e *edi.EDIMessage) pulid.ID { return e.ID },
	),
	permission.KindEDIShipmentLink: mint(
		func(e *edi.ShipmentLink) pulid.ID { return e.ID },
	),
	permission.KindEDITenderChange: mint(
		func(e *edi.TenderChange) pulid.ID { return e.ID },
	),
	permission.KindEDITransferChange: mint(
		func(e *edi.TransferChange) pulid.ID { return e.ID },
	),
	permission.KindEDICarrierInvoice: mint(
		func(e *edi.CarrierInvoice) pulid.ID { return e.ID },
	),
	permission.KindScannedDocument: mint(
		func(e *capture.CaptureItem) pulid.ID { return e.ID },
	),
	permission.KindIFTAJurisdiction: mint(
		func(e *ifta.Jurisdiction) pulid.ID { return e.ID },
	),
	permission.KindIFTAReturn: mint(
		func(e *ifta.Return) pulid.ID { return e.ID },
	),
	permission.KindIFTAMileageEntry: mint(
		func(e *ifta.JurisdictionMileageEntry) pulid.ID { return e.ID },
	),
	permission.KindUSState: mint(
		func(e *usstate.UsState) pulid.ID { return e.ID },
	),
	permission.KindPayEvent: mint(
		func(e *driversettlement.PayEvent) pulid.ID { return e.ID },
	),
	permission.KindPayProfile: mint(
		func(e *driverpay.PayProfile) pulid.ID { return e.ID },
	),
	permission.KindPayAssignment: mint(
		func(e *driverpay.WorkerPayAssignment) pulid.ID { return e.ID },
	),
	permission.KindPayrollExport: mint(
		func(e *worker.PayrollExport) pulid.ID { return e.ID },
	),
	permission.KindShiftAssignment: mint(
		func(e *worker.WorkerShiftAssignment) pulid.ID { return e.ID },
	),
	permission.KindChecklistItem: mint(
		func(e *worker.WorkerChecklistItem) pulid.ID { return e.ID },
	),
	permission.KindSafetyViolation: mint(
		func(e *worker.WorkerSafetyViolation) pulid.ID { return e.ID },
	),
	permission.KindLeaveDay: mint(
		func(e *worker.WorkerLeaveEntry) pulid.ID { return e.ID },
	),
}

// A declared kind's prefix is held to the domain's as a resource's is: a
// wrong one would refuse every real id of the kind.
func TestRecordKindPrefixes_MatchWhatTheDomainMints(t *testing.T) {
	t.Parallel()

	table := permission.RecordKindPrefixes()
	require.NotEmpty(t, table)

	for kind, prefix := range table {
		mintID, tested := kindMinters[kind]
		if !assert.Truef(t, tested, "%s has a prefix and no entity minting it here", kind) {
			continue
		}
		id, err := mintID(t.Context())
		require.NoError(t, err, kind)
		assert.Equalf(t, prefix, id.Prefix(), "%s ids", kind)
	}
}

// One prefix names one kind across both tables, so an id is named by what it
// is when it is refused, and no declared kind repeats a resource's own.
func TestRecordKindOfIDPrefix_ReadsBothTables(t *testing.T) {
	t.Parallel()

	owners := make(map[string]permission.RecordKind)
	for resource, prefix := range permission.RecordIDPrefixes() {
		kind, ok := resource.Kind()
		require.True(t, ok, resource)
		owners[prefix] = kind
	}
	for kind, prefix := range permission.RecordKindPrefixes() {
		other, taken := owners[prefix]
		assert.Falsef(t, taken, "%s and %s share prefix %s", kind, other, prefix)
		owners[prefix] = kind

		_, resourceToo := permission.Resource(kind).IDPrefix()
		assert.Falsef(t, resourceToo, "%s is declared as a kind and registered as a resource", kind)
		assert.True(t, kind.Known())
		assert.NotEmpty(t, kind.Noun())
		assert.NotEmpty(t, kind.Resource())
	}

	for prefix, kind := range owners {
		got, ok := permission.RecordKindOfIDPrefix(prefix)
		require.Truef(t, ok, "prefix %s", prefix)
		assert.Equal(t, kind, got)
		declared, has := got.IDPrefix()
		assert.True(t, has)
		assert.Equal(t, prefix, declared)
	}

	_, ok := permission.RecordKindOfIDPrefix("nope_")
	assert.False(t, ok)
	assert.False(t, permission.RecordKind("nope").Known())
}

func TestRecordKind_NounAndResource(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "clearinghouse query", permission.KindClearinghouseQuery.Noun())
	assert.Equal(t, permission.ResourceQualification, permission.KindClearinghouseQuery.Resource())

	shipment, ok := permission.ResourceShipment.Kind()
	require.True(t, ok)
	assert.Equal(t, "shipment", shipment.Noun())
	assert.Equal(t, permission.ResourceShipment, shipment.Resource())

	_, ok = permission.ResourceQualification.Kind()
	assert.False(t, ok, "a resource covering several kinds is none of them")
}
