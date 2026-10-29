package permission_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/emoss08/keel/schema"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection/accountingconnectiondb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingdrift/accountingdriftdb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountinginbound/accountinginbounddb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping/accountingmappingdb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync/accountingsyncdb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod/fiscalperioddb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount/glaccountdb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry/journalentrydb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalreversal"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/manualjournal/manualjournaldb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory/agentmemorydb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentwait/agentwaitdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiaudit/aiauditdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/analysis/analysisdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation/conversationdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/extensionportal/extensionportaldb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/extensionwebwatch/extensionwebwatchdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/insight/insightdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower/watchtowerdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt/bankreceiptdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceiptworkitem/bankreceiptworkitemdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingtransfer/billingtransferdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment/customerpaymentdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring/factoringdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringledger/factoringledgerdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringremittance/factoringremittancedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission/factoringsubmissiondb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment/invoiceadjustmentdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicedispute/invoicedisputedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicerun/invoicerundb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture/capturedb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document/documentdb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage/inboundmessagedb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edicarrierinvoice/edicarrierinvoicedb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile/ediinboundfiledb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage/edimessagedb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edisync/edisyncdb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editenderchange/editenderchangedb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer/editransferdb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentmanufacturer/equipmentmanufacturerdb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype/equipmenttypedb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode/fleetcodedb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor/tractordb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer/trailerdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate/usstatedb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention/detentiondb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order/orderdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/recurringshipment/recurringshipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure/servicefailuredb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmenthold/shipmentholddb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender/tenderdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carriercapacity/carriercapacitydb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelevent/carrierinteleventdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany/factoringcompanydb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/places/domain/locationcategory/locationcategorydb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary/formulalibrarydb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate/formulatemplatedb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/fuelsurcharge/fuelsurchargedb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement/rateagreementdb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateconfirmation/rateconfirmationdb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateimport/rateimportdb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix/ratematrixdb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge/accessorialchargedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity/commoditydb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype/documenttypedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial/hazardousmaterialdb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason/holdreasondb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype/servicetypedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype/shipmenttypedb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement/carriersettlementdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay/driverpaydb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement/driversettlementdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase/fuelpurchasedb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta/iftadb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payadvance/payadvancedb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payassignment/payassignmentdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/paycode/paycodedb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payescrow/payescrowdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payexpense/payexpensedb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/recurringpay/recurringpaydb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/dotclearinghouse/dotclearinghousedb"
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
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email/emaildb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report/reportdb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/tablechangealert/tablechangealertdb"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type minter func(context.Context) (pulid.ID, error)

var errNoIDPrefix = errors.New("the model's keel table mints no id: tag its id autoid:<prefix>")

func prefixed(table *schema.Table) minter {
	return func(context.Context) (pulid.ID, error) {
		if table.IDPrefix == "" {
			return pulid.Nil, fmt.Errorf("%s: %w", table.Name, errNoIDPrefix)
		}

		return pulid.MustNew(table.IDPrefix), nil
	}
}

var minters = map[permission.Resource]minter{
	permission.ResourceShipment:        prefixed(shipmentdb.ShipmentsSchema()),
	permission.ResourceShipmentMove:    prefixed(shipmentdb.ShipmentMovesSchema()),
	permission.ResourceShipmentComment: prefixed(shipmentdb.ShipmentCommentsSchema()),
	permission.ResourceOrder:           prefixed(orderdb.OrdersSchema()),
	permission.ResourceRecurringShipment: prefixed(
		recurringshipmentdb.RecurringShipmentsSchema(),
	),
	permission.ResourceServiceFailure: prefixed(
		servicefailuredb.ServiceFailuresSchema(),
	),
	permission.ResourceCustomer: prefixed(customerdb.CustomersSchema()),
	permission.ResourceFactoringCompany: prefixed(
		factoringcompanydb.FactoringCompaniesSchema(),
	),
	permission.ResourceCarrier:   prefixed(carrierdb.CarriersSchema()),
	permission.ResourceLocation:  prefixed(locationdb.LocationsSchema()),
	permission.ResourceWorker:    prefixed(workerdb.WorkersSchema()),
	permission.ResourceTractor:   prefixed(tractordb.TractorsSchema()),
	permission.ResourceTrailer:   prefixed(trailerdb.TrailersSchema()),
	permission.ResourceCommodity: prefixed(commoditydb.CommoditiesSchema()),
	permission.ResourceHazardousMaterial: prefixed(
		hazardousmaterialdb.HazardousMaterialsSchema(),
	),
	permission.ResourceDocument:            prefixed(documentdb.DocumentsSchema()),
	permission.ResourceInvoice:             prefixed(invoicedb.InvoicesSchema()),
	permission.ResourceInvoiceDispute:      prefixed(invoicedisputedb.InvoiceDisputesSchema()),
	permission.ResourceFactoringAgreement:  prefixed(factoringdb.AgreementsSchema()),
	permission.ResourceFactoringNoa:        prefixed(factoringdb.NoasSchema()),
	permission.ResourceFactoringSubmission: prefixed(factoringsubmissiondb.SubmissionsSchema()),
	permission.ResourceFactoringReserve:    prefixed(factoringledgerdb.ReserveEntriesSchema()),
	permission.ResourceFactoringRemittance: prefixed(factoringremittancedb.RemittancesSchema()),
	permission.ResourceInvoiceRun:          prefixed(invoicerundb.InvoiceRunsSchema()),
	permission.ResourceRateAgreement:       prefixed(rateagreementdb.RateAgreementsSchema()),
	permission.ResourceCustomerPayment:     prefixed(customerpaymentdb.PaymentsSchema()),
	permission.ResourceReport:              prefixed(reportdb.ReportDefinitionsSchema()),
	permission.ResourceDashboard:           prefixed(reportdb.DashboardsSchema()),
	permission.ResourceWorkerCredential: prefixed(
		workercredentialdb.WorkerCredentialsSchema(),
	),
	permission.ResourceWorkerTraining: prefixed(
		workertrainingdb.WorkerTrainingRecordsSchema(),
	),
	permission.ResourceWorkerSafetyEvent: prefixed(
		workersafetydb.WorkerSafetyEventsSchema(),
	),
	permission.ResourcePerformanceReview: prefixed(
		workerreviewdb.PerformanceReviewsSchema(),
	),
	permission.ResourceWorkerInjury:      prefixed(workerinjurydb.WorkerInjuriesSchema()),
	permission.ResourceWorkerLeave:       prefixed(workerleavedb.WorkerLeaveCasesSchema()),
	permission.ResourceDriverSettlement:  prefixed(driversettlementdb.SettlementsSchema()),
	permission.ResourceSettlementDispute: prefixed(driversettlementdb.DisputesSchema()),
	permission.ResourceCaptureBatch:      prefixed(capturedb.CaptureBatchesSchema()),
	permission.ResourceOrganization:      prefixed(tenantdb.OrganizationsSchema()),
	permission.ResourceUser:              prefixed(tenantdb.UsersSchema()),
	permission.ResourceAgentRun:          prefixed(agentdb.AgentRunsSchema()),
	permission.ResourceAgentMemory:       prefixed(agentmemorydb.MemoriesSchema()),
	permission.ResourceInsight:           prefixed(insightdb.InsightsSchema()),
	permission.ResourceTableChangeAlert: prefixed(
		tablechangealertdb.TCASubscriptionsSchema(),
	),
	permission.ResourceEmailProfile: prefixed(emaildb.ProfilesSchema()),
	permission.ResourceInboundMessage: prefixed(
		inboundmessagedb.InboundMessagesSchema(),
	),
	permission.ResourceInboundMailbox:           prefixed(inboundmessagedb.MailboxesSchema()),
	permission.ResourceShipmentStop:             prefixed(shipmentdb.StopsSchema()),
	permission.ResourceShipmentHold:             prefixed(shipmentholddb.ShipmentHoldsSchema()),
	permission.ResourceHoldReason:               prefixed(holdreasondb.HoldReasonsSchema()),
	permission.ResourceServiceFailureReasonCode: prefixed(servicefailuredb.ReasonCodesSchema()),
	permission.ResourcePermit:                   prefixed(permitdb.PermitsSchema()),
	permission.ResourceServiceType:              prefixed(servicetypedb.ServiceTypesSchema()),
	permission.ResourceShipmentType:             prefixed(shipmenttypedb.ShipmentTypesSchema()),
	permission.ResourceDocumentType:             prefixed(documenttypedb.DocumentTypesSchema()),
	permission.ResourceEquipmentType:            prefixed(equipmenttypedb.EquipmentTypesSchema()),
	permission.ResourceEquipmentManufacturer: prefixed(
		equipmentmanufacturerdb.EquipmentManufacturersSchema(),
	),
	permission.ResourceFleetCode: prefixed(fleetcodedb.FleetCodesSchema()),
	permission.ResourceLocationCategory: prefixed(
		locationcategorydb.LocationCategoriesSchema(),
	),
	permission.ResourceCarrierCapacityPosting: prefixed(carriercapacitydb.PostingsSchema()),
	permission.ResourceRateConfirmation: prefixed(
		rateconfirmationdb.RateConfirmationsSchema(),
	),
	permission.ResourceCarrierSettlement: prefixed(
		carriersettlementdb.CarrierSettlementsSchema(),
	),
	permission.ResourceCarrierInvoiceMatch: prefixed(
		carriersettlementdb.InvoiceMatchesSchema(),
	),
	permission.ResourceRoutingGuide: prefixed(tenderdb.RoutingGuidesSchema()),
	permission.ResourceTender:       prefixed(tenderdb.TendersSchema()),
	permission.ResourceAccessorialCharge: prefixed(
		accessorialchargedb.AccessorialChargesSchema(),
	),
	permission.ResourceFormulaTemplate: prefixed(
		formulatemplatedb.FormulaTemplatesSchema(),
	),
	permission.ResourceFormulaLibrary: prefixed(
		formulalibrarydb.FormulaLibrariesSchema(),
	),
	permission.ResourceRateMatrix:           prefixed(ratematrixdb.RateMatrixesSchema()),
	permission.ResourceGeneralLedgerAccount: prefixed(glaccountdb.GLAccountsSchema()),
	permission.ResourceFiscalPeriod:         prefixed(fiscalperioddb.FiscalPeriodsSchema()),
	permission.ResourceManualJournal:        prefixed(manualjournaldb.RequestsSchema()),
	permission.ResourceJournalEntry:         prefixed(journalentrydb.JournalEntriesSchema()),
	permission.ResourceBankReceipt:          prefixed(bankreceiptdb.BankReceiptsSchema()),
	permission.ResourceBankReceiptWorkItem:  prefixed(bankreceiptworkitemdb.WorkItemsSchema()),
	permission.ResourcePayCode:              prefixed(paycodedb.PayCodesSchema()),
	permission.ResourcePayAdvance:           prefixed(payadvancedb.PayAdvancesSchema()),
	permission.ResourceEscrowAccount:        prefixed(payescrowdb.EscrowAccountsSchema()),
	permission.ResourceRecurringDeduction:   prefixed(recurringpaydb.RecurringDeductionsSchema()),
	permission.ResourceRecurringEarning:     prefixed(recurringpaydb.RecurringEarningsSchema()),
	permission.ResourceDriverExpense:        prefixed(payexpensedb.ExpensesSchema()),
	permission.ResourceFuelCard:             prefixed(fuelpurchasedb.FuelCardsSchema()),
	permission.ResourceFuelPurchase:         prefixed(fuelpurchasedb.FuelPurchasesSchema()),
	permission.ResourceFuelPurchaseImport:   prefixed(fuelpurchasedb.ImportBatchesSchema()),
	permission.ResourceWorkerPTO:            prefixed(workerdb.WorkerPTOsSchema()),
	permission.ResourceWorkerChecklist: prefixed(
		workerchecklistdb.WorkerChecklistsSchema(),
	),
	permission.ResourceWorkerChecklistTemplate: prefixed(
		workerchecklistdb.WorkerChecklistTemplatesSchema(),
	),
	permission.ResourceWorkerCredentialType: prefixed(
		workercredentialdb.WorkerCredentialTypesSchema(),
	),
	permission.ResourceWorkerRecognition: prefixed(
		workerdisciplinedb.WorkerRecognitionsSchema(),
	),
	permission.ResourceWorkerDOTTest: prefixed(workerdotdb.WorkerDOTTestsSchema()),
	permission.ResourceTrainingCourse: prefixed(
		workertrainingdb.TrainingCoursesSchema(),
	),
	permission.ResourcePerformanceReviewTemplate: prefixed(
		workerreviewdb.PerformanceReviewTemplatesSchema(),
	),
	permission.ResourceShiftTemplate: prefixed(workerscheduledb.ShiftTemplatesSchema()),
	permission.ResourceShiftSwap: prefixed(
		workerscheduledb.ShiftSwapRequestsSchema(),
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
	permission.KindEmploymentVerification: prefixed(
		employmentdb.WorkerEmploymentVerificationsSchema(),
	),
	permission.KindClearinghouseQuery: prefixed(
		dotclearinghousedb.WorkerClearinghouseQueriesSchema(),
	),
	permission.KindDOTRandomPool:      prefixed(dotrandomdb.DOTRandomPoolsSchema()),
	permission.KindDOTRandomDraw:      prefixed(dotrandomdb.DOTRandomDrawsSchema()),
	permission.KindDOTRandomSelection: prefixed(dotrandomdb.DOTRandomDrawEntriesSchema()),
	permission.KindAIAuditEvent:       prefixed(aiauditdb.AIAuditEventsSchema()),
	permission.KindAIAuditExport:      prefixed(aiauditdb.AIAuditExportsSchema()),
	permission.KindAssistantThread:    prefixed(conversationdb.ThreadsSchema()),
	permission.KindWait:               prefixed(agentwaitdb.WaitsSchema()),
	permission.KindPortalSite:         prefixed(extensionportaldb.PortalSitesSchema()),
	permission.KindPortalRun:          prefixed(extensionportaldb.PortalRunsSchema()),
	permission.KindAnalysisRun:        prefixed(analysisdb.RunsSchema()),
	permission.KindWebWatch:           prefixed(extensionwebwatchdb.WebWatchesSchema()),
	permission.KindWatchtowerItem:     prefixed(watchtowerdb.ItemsSchema()),
	permission.KindAccountingConnection: prefixed(
		accountingconnectiondb.AccountingConnectionsSchema(),
	),
	permission.KindAccountingMapping: prefixed(accountingmappingdb.AccountingMappingsSchema()),
	permission.KindAccountingSyncRecord: prefixed(
		accountingsyncdb.AccountingSyncRecordsSchema(),
	),
	permission.KindAccountingInboundChange: prefixed(
		accountinginbounddb.AccountingInboundChangesSchema(),
	),
	permission.KindAccountingDriftFinding: prefixed(
		accountingdriftdb.AccountingDriftFindingsSchema(),
	),
	permission.KindCarrierIntelligenceEvent: prefixed(
		carrierinteleventdb.CarrierIntelEventsSchema(),
	),
	permission.KindInsurancePolicy:   prefixed(carrierdb.CarrierInsurancePoliciesSchema()),
	permission.KindCarrierAssignment: prefixed(shipmentdb.CarrierAssignmentsSchema()),
	permission.KindAdditionalCharge:  prefixed(shipmentdb.AdditionalChargesSchema()),
	permission.KindBillingTransferRun: prefixed(
		billingtransferdb.BillingTransferRunsSchema(),
	),
	permission.KindDetentionOccurrence: prefixed(detentiondb.DetentionOccurrencesSchema()),
	permission.KindTenderOffer:         prefixed(tenderdb.TenderOffersSchema()),
	permission.KindOrderCharge:         prefixed(orderdb.OrderChargesSchema()),
	permission.KindBillingQueueItem:    prefixed(billingqueuedb.BillingQueueItemsSchema()),
	permission.KindInvoiceLine:         prefixed(invoicedb.InvoiceLinesSchema()),
	permission.KindInvoiceAdjustment: prefixed(
		invoiceadjustmentdb.InvoiceAdjustmentsSchema(),
	),
	permission.KindInvoiceRunGroup: prefixed(invoicerundb.InvoiceRunGroupsSchema()),
	permission.KindInvoiceRunItem:  prefixed(invoicerundb.InvoiceRunGroupItemsSchema()),
	permission.KindCreditMemoApplication: prefixed(
		customerpaymentdb.CreditMemoApplicationsSchema(),
	),
	permission.KindRateAgreementLane: prefixed(rateagreementdb.RateAgreementRulesSchema()),
	permission.KindRateImport:        prefixed(rateimportdb.RateImportBatchesSchema()),
	permission.KindFuelIndex:         prefixed(fuelsurchargedb.FuelIndexesSchema()),
	permission.KindFuelIndexPrice:    prefixed(fuelsurchargedb.FuelIndexPricesSchema()),
	permission.KindReportRun:         prefixed(reportdb.ReportRunsSchema()),
	permission.KindReportSchedule:    prefixed(reportdb.ReportSchedulesSchema()),
	permission.KindEDIPartner:        prefixed(edidb.EDIPartnersSchema()),
	permission.KindEDIInboundFile:    prefixed(ediinboundfiledb.EDIInboundFilesSchema()),
	permission.KindEDITransfer:       prefixed(editransferdb.EDITransfersSchema()),
	permission.KindEDIMessage:        prefixed(edimessagedb.EDIMessagesSchema()),
	permission.KindEDIShipmentLink:   prefixed(edisyncdb.ShipmentLinksSchema()),
	permission.KindEDITenderChange:   prefixed(editenderchangedb.TenderChangesSchema()),
	permission.KindEDITransferChange: prefixed(edisyncdb.TransferChangesSchema()),
	permission.KindEDICarrierInvoice: prefixed(edicarrierinvoicedb.CarrierInvoicesSchema()),
	permission.KindScannedDocument:   prefixed(capturedb.CaptureItemsSchema()),
	permission.KindIFTAJurisdiction:  prefixed(iftadb.JurisdictionsSchema()),
	permission.KindIFTAReturn:        prefixed(iftadb.ReturnsSchema()),
	permission.KindIFTAMileageEntry:  prefixed(iftadb.JurisdictionMileageEntriesSchema()),
	permission.KindUSState:           prefixed(usstatedb.UsStatesSchema()),
	permission.KindPayEvent:          prefixed(driversettlementdb.PayEventsSchema()),
	permission.KindPayProfile:        prefixed(driverpaydb.PayProfilesSchema()),
	permission.KindPayAssignment:     prefixed(payassignmentdb.WorkerPayAssignmentsSchema()),
	permission.KindPayrollExport:     prefixed(timesheetdb.PayrollExportsSchema()),
	permission.KindShiftAssignment: prefixed(
		workerscheduledb.WorkerShiftAssignmentsSchema(),
	),
	permission.KindChecklistItem: prefixed(
		workerchecklistdb.WorkerChecklistItemsSchema(),
	),
	permission.KindSafetyViolation: prefixed(
		fleetsafetydb.WorkerSafetyViolationsSchema(),
	),
	permission.KindLeaveDay: prefixed(workerleavedb.WorkerLeaveEntriesSchema()),
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
