import {
  combineLoaders,
  createCapabilityLoader,
  createPermissionLoader,
} from "@/lib/route-permission";
import { createPrefetchLoader, lazyPrefetch } from "@/lib/route-prefetch";
import { AppLayout } from "@/routes/app-layout";
import { RootLayout } from "@/routes/root-layout";
import { RouteErrorBoundary } from "@trenova/shared/components/error-boundary";
import LoadingSkeleton from "@trenova/shared/components/loading-skeleton";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { OrganizationCapability } from "@trenova/shared/types/organization-capability";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { createBrowserRouter, redirect, type LoaderFunction, type RouteObject } from "react-router";
import { AdminLayout } from "./routes/admin-layout";

const protectedLoader: LoaderFunction = async () => {
  const { checkAuth } = useAuthStore.getState();
  const isAuthenticated = await checkAuth();

  if (!isAuthenticated) {
    return redirect("/login");
  }

  return null;
};

const guestLoader: LoaderFunction = async () => {
  const { checkAuth } = useAuthStore.getState();

  const isAuthenticated = await checkAuth();
  if (isAuthenticated) {
    return redirect("/");
  }

  return null;
};

export const routes: RouteObject[] = [
  {
    element: <RootLayout />,
    errorElement: <RouteErrorBoundary />,
    HydrateFallback: LoadingSkeleton,
    children: [
      {
        element: <AppLayout />,
        loader: protectedLoader,
        children: [
          {
            path: "/",
            loader: combineLoaders(
              protectedLoader,
              createPrefetchLoader(lazyPrefetch(() => import("@/routes/home/page"))),
            ),
            async lazy() {
              const { Home } = await import("@/routes/home/page");
              return { Component: Home };
            },
          },
          {
            path: "/organization/data-retention",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.Organization)),
            async lazy() {
              const { DataRetentionPage } =
                await import("@/routes/organization/data-retention/page");
              return { Component: DataRetentionPage };
            },
          },
          {
            path: "/organization/email-profiles",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.EmailProfile)),
            async lazy() {
              const { EmailProfilesPage } =
                await import("@/routes/organization/email-profiles/page");
              return { Component: EmailProfilesPage };
            },
          },
          {
            path: "/organization/email-logs",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.EmailLog)),
            async lazy() {
              const { EmailLogsPage } = await import("@/routes/organization/email-logs/page");
              return { Component: EmailLogsPage };
            },
          },
          {
            path: "/shipment-management/shipments",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.Shipment),
              createPrefetchLoader(lazyPrefetch(() => import("@/routes/shipment/page"))),
            ),
            async lazy() {
              const { ShipmentsPage } = await import("@/routes/shipment/page");
              return { Component: ShipmentsPage };
            },
          },
          {
            path: "/shipment-management/recurring-shipments",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.RecurringShipment),
            ),
            async lazy() {
              const { RecurringShipmentsPage } = await import("@/routes/recurring-shipment/page");
              return { Component: RecurringShipmentsPage };
            },
          },
          {
            path: "/shipment-management/orders",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.Order)),
            async lazy() {
              const { OrdersPage } = await import("@/routes/order/page");
              return { Component: OrdersPage };
            },
          },
          {
            path: "/shipment-management/service-failures",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.ServiceFailure),
            ),
            async lazy() {
              const { ServiceFailuresPage } = await import("@/routes/service-failure/page");
              return { Component: ServiceFailuresPage };
            },
          },
          {
            path: "/shipment-management/shipments/import",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.Shipment, Operation.Create),
            ),
            async lazy() {
              const { ShipmentImportPage } = await import("@/routes/shipment/import-page");
              return { Component: ShipmentImportPage };
            },
          },
          {
            path: "/shipment-management/configuration-files/shipment-types",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.ShipmentType)),
            async lazy() {
              const { ShipmentTypesPage } = await import("@/routes/shipment-type/page");
              return { Component: ShipmentTypesPage };
            },
          },
          {
            path: "/shipment-management/configuration-files/service-types",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.ServiceType)),
            async lazy() {
              const { ServiceTypesPage } = await import("@/routes/service-type/page");
              return { Component: ServiceTypesPage };
            },
          },
          {
            path: "/shipment-management/configuration-files/hazardous-materials",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.HazardousMaterial),
            ),
            async lazy() {
              const { HazardousMaterialsPage } = await import("@/routes/hazardous-material/page");
              return { Component: HazardousMaterialsPage };
            },
          },
          {
            path: "/shipment-management/configuration-files/commodities",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.Commodity)),
            async lazy() {
              const { CommoditiesPage } = await import("@/routes/commodity/page");
              return { Component: CommoditiesPage };
            },
          },
          {
            path: "/edi/overview",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.EDI)),
            async lazy() {
              const { EDIOverviewPage } = await import("@/routes/edi/page");
              return { Component: EDIOverviewPage };
            },
          },
          {
            path: "/edi/partners",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.EDI)),
            async lazy() {
              const { EDIPartnersPage } = await import("@/routes/edi/page");
              return { Component: EDIPartnersPage };
            },
          },
          {
            path: "/edi/communication-profiles",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.EDI)),
            async lazy() {
              const { EDICommunicationProfilesPage } = await import("@/routes/edi/page");
              return { Component: EDICommunicationProfilesPage };
            },
          },
          {
            path: "/edi/mapping-profiles",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.EDI)),
            async lazy() {
              const { EDIMappingProfilesPage } = await import("@/routes/edi/page");
              return { Component: EDIMappingProfilesPage };
            },
          },
          {
            path: "/edi/designer",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.EDI)),
            async lazy() {
              const { EDIDesignerPage } = await import("@/routes/edi/page");
              return { Component: EDIDesignerPage };
            },
          },
          {
            path: "/edi/transfers/inbound",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.EDI)),
            async lazy() {
              const { EDIInboundTransfersPage } = await import("@/routes/edi/page");
              return { Component: EDIInboundTransfersPage };
            },
          },
          {
            path: "/edi/transfers/outbound",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.EDI)),
            async lazy() {
              const { EDIOutboundTransfersPage } = await import("@/routes/edi/page");
              return { Component: EDIOutboundTransfersPage };
            },
          },
          {
            path: "/edi/messages",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.EDI)),
            async lazy() {
              const { EDIMessagesPage } = await import("@/routes/edi/page");
              return { Component: EDIMessagesPage };
            },
          },
          {
            path: "/edi/inbound-files",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.EDI)),
            async lazy() {
              const { EDIInboundFilesPage } = await import("@/routes/edi/page");
              return { Component: EDIInboundFilesPage };
            },
          },
          {
            path: "/edi/test-cases",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.EDI)),
            async lazy() {
              const { EDITestCasesPage } = await import("@/routes/edi/page");
              return { Component: EDITestCasesPage };
            },
          },
          {
            path: "/billing/queue",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.BillingQueue),
              createPrefetchLoader(lazyPrefetch(() => import("@/routes/billing-queue/page"))),
            ),
            async lazy() {
              const { BillingQueuePage } = await import("@/routes/billing-queue/page");
              return { Component: BillingQueuePage };
            },
          },
          {
            path: "/payroll/workspace",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.DriverSettlement),
            ),
            async lazy() {
              const { SettlementWorkspacePage } =
                await import("@/routes/settlement-workspace/page");
              return { Component: SettlementWorkspacePage };
            },
          },
          {
            path: "/payroll/settlements",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.DriverSettlement),
            ),
            async lazy() {
              const { DriverSettlementsPage } = await import("@/routes/driver-settlement/page");
              return { Component: DriverSettlementsPage };
            },
          },
          {
            path: "/payroll/disputes",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.SettlementDispute),
            ),
            async lazy() {
              const { SettlementDisputesPage } = await import("@/routes/settlement-dispute/page");
              return { Component: SettlementDisputesPage };
            },
          },
          {
            path: "/payroll/expenses",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.DriverExpense),
            ),
            async lazy() {
              const { DriverExpensesPage } = await import("@/routes/driver-expense/page");
              return { Component: DriverExpensesPage };
            },
          },
          {
            path: "/payroll/settlement-batches",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.DriverSettlement),
            ),
            async lazy() {
              const { SettlementBatchesPage } = await import("@/routes/settlement-batch/page");
              return { Component: SettlementBatchesPage };
            },
          },
          {
            path: "/payroll/pay-events",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.DriverSettlement),
            ),
            async lazy() {
              const { DriverPayEventsPage } = await import("@/routes/driver-pay-event/page");
              return { Component: DriverPayEventsPage };
            },
          },
          {
            path: "/payroll/pay-profiles",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.DriverPayProfile),
            ),
            async lazy() {
              const { PayProfilesPage } = await import("@/routes/pay-profile/page");
              return { Component: PayProfilesPage };
            },
          },
          {
            path: "/payroll/deductions",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.RecurringDeduction),
            ),
            async lazy() {
              const { RecurringDeductionsPage } = await import("@/routes/recurring-deduction/page");
              return { Component: RecurringDeductionsPage };
            },
          },
          {
            path: "/payroll/earnings",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.RecurringEarning),
            ),
            async lazy() {
              const { RecurringEarningsPage } = await import("@/routes/recurring-earning/page");
              return { Component: RecurringEarningsPage };
            },
          },
          {
            path: "/payroll/pay-codes",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.PayCode),
            ),
            async lazy() {
              const { PayCodesPage } = await import("@/routes/pay-code/page");
              return { Component: PayCodesPage };
            },
          },
          {
            path: "/payroll/advances",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.PayAdvance),
            ),
            async lazy() {
              const { PayAdvancesPage } = await import("@/routes/pay-advance/page");
              return { Component: PayAdvancesPage };
            },
          },
          {
            path: "/payroll/escrow-accounts",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.EscrowAccount),
            ),
            async lazy() {
              const { EscrowAccountsPage } = await import("@/routes/escrow-account/page");
              return { Component: EscrowAccountsPage };
            },
          },
          {
            path: "/carrier-settlements/workspace",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.Brokerage),
              createPermissionLoader(Resource.CarrierSettlement),
            ),
            async lazy() {
              const { CarrierSettlementWorkspacePage } =
                await import("@/routes/carrier-settlement-workspace/page");
              return { Component: CarrierSettlementWorkspacePage };
            },
          },
          {
            path: "/carrier-settlements/settlements",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.Brokerage),
              createPermissionLoader(Resource.CarrierSettlement),
            ),
            async lazy() {
              const { CarrierSettlementsPage } = await import("@/routes/carrier-settlement/page");
              return { Component: CarrierSettlementsPage };
            },
          },
          {
            path: "/carrier-settlements/batches",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.Brokerage),
              createPermissionLoader(Resource.CarrierSettlement),
            ),
            async lazy() {
              const { CarrierSettlementBatchesPage } =
                await import("@/routes/carrier-settlement-batch/page");
              return { Component: CarrierSettlementBatchesPage };
            },
          },
          {
            path: "/carrier-settlements/cost-events",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.Brokerage),
              createPermissionLoader(Resource.CarrierSettlement),
            ),
            async lazy() {
              const { CarrierCostEventsPage } = await import("@/routes/carrier-cost-event/page");
              return { Component: CarrierCostEventsPage };
            },
          },
          {
            path: "/carrier-settlements/invoice-matching",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.Brokerage),
              createPermissionLoader(Resource.CarrierInvoiceMatch),
            ),
            async lazy() {
              const { CarrierInvoiceMatchingPage } =
                await import("@/routes/carrier-invoice-matching/page");
              return { Component: CarrierInvoiceMatchingPage };
            },
          },
          {
            path: "/billing/invoices",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.Invoice)),
            async lazy() {
              const { InvoicesPage } = await import("@/routes/invoice/page");
              return { Component: InvoicesPage };
            },
          },
          {
            path: "/billing/pending-approvals",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.Invoice)),
            async lazy() {
              const { InvoiceApprovalPage } = await import("@/routes/invoice-approval/page");
              return { Component: InvoiceApprovalPage };
            },
          },
          {
            path: "/billing/reconciliation-exceptions",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.Invoice)),
            async lazy() {
              const { InvoiceReconciliationPage } =
                await import("@/routes/invoice-reconciliation/page");
              return { Component: InvoiceReconciliationPage };
            },
          },
          {
            path: "/billing/adjustment-batches",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.Invoice)),
            async lazy() {
              const { InvoiceAdjustmentBatchPage } =
                await import("@/routes/invoice-adjustment-batch/page");
              return { Component: InvoiceAdjustmentBatchPage };
            },
          },
          {
            path: "/billing/configuration-files/charge-types",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.ChargeType)),
            async lazy() {
              const { PlaceholderPage } = await import("@/routes/placeholder-page");
              return { Component: PlaceholderPage };
            },
          },
          {
            path: "/billing/rate-agreements",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.RateAgreement)),
            async lazy() {
              const { RateAgreementPage } = await import("@/routes/rate-agreement/page");
              return { Component: RateAgreementPage };
            },
          },
          {
            path: "/billing/configuration-files/rate-zones",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.RateZone)),
            async lazy() {
              const { RateZonePage } = await import("@/routes/rate-zone/page");
              return { Component: RateZonePage };
            },
          },
          {
            path: "/billing/configuration-files/rate-matrices",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.RateMatrix)),
            async lazy() {
              const { RateMatrixPage } = await import("@/routes/rate-matrix/page");
              return { Component: RateMatrixPage };
            },
          },
          {
            path: "/detention/configuration-files/detention-policies",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.DetentionPolicy),
            ),
            async lazy() {
              const { DetentionPolicyPage } = await import("@/routes/detention-policy/page");
              return { Component: DetentionPolicyPage };
            },
          },
          {
            path: "/detention/intelligence",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.DetentionPolicy),
            ),
            async lazy() {
              const { DetentionIntelligencePage } =
                await import("@/routes/detention-intelligence/page");
              return { Component: DetentionIntelligencePage };
            },
          },
          {
            path: "/detention/desk",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.DetentionPolicy),
            ),
            async lazy() {
              const { DetentionDeskPage } = await import("@/routes/detention-desk/page");
              return { Component: DetentionDeskPage };
            },
          },
          {
            path: "/billing/configuration-files/accessorial-charges",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.AccessorialCharge),
            ),
            async lazy() {
              const { AccessorialChargesPage } = await import("@/routes/accessorial-charge/page");
              return { Component: AccessorialChargesPage };
            },
          },
          {
            path: "/billing/configuration-files/customers",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.Customer)),
            async lazy() {
              const { CustomersPage } = await import("@/routes/customer/page");
              return { Component: CustomersPage };
            },
          },
          {
            path: "/billing/configuration-files/document-types",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.DocumentType)),
            async lazy() {
              const { DocumentTypesPage } = await import("@/routes/document-type/page");
              return { Component: DocumentTypesPage };
            },
          },
          {
            path: "/billing/configuration-files/document-packet-rules",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.DocumentType)),
            async lazy() {
              const { DocumentPacketRulesPage } =
                await import("@/routes/document-packet-rule/page");
              return { Component: DocumentPacketRulesPage };
            },
          },

          {
            path: "/accounting",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.AccountsReceivable),
            ),
            async lazy() {
              const { AccountingDashboardPage } =
                await import("@/routes/accounting-dashboard/page");
              return { Component: AccountingDashboardPage };
            },
          },
          {
            path: "/accounting/manual-journals",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.ManualJournal)),
            async lazy() {
              const { ManualJournalsPage } = await import("@/routes/manual-journal/page");
              return { Component: ManualJournalsPage };
            },
          },
          {
            path: "/accounting/journal-reversals",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.JournalReversal),
            ),
            async lazy() {
              const { JournalReversalsPage } = await import("@/routes/journal-reversal/page");
              return { Component: JournalReversalsPage };
            },
          },
          {
            path: "/accounting/journal-entries/:id",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.JournalEntry)),
            async lazy() {
              const { JournalEntryDetailPage } = await import("@/routes/journal-entry/page");
              return { Component: JournalEntryDetailPage };
            },
          },
          {
            path: "/accounting/journal-entries/source/:type/:sourceId",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.JournalEntry)),
            async lazy() {
              const { SourceDrillDownPage } =
                await import("@/routes/journal-entry/_components/source-drill-down-page");
              return { Component: SourceDrillDownPage };
            },
          },
          {
            path: "/accounting/reports/trial-balance",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.AccountingReport),
            ),
            async lazy() {
              const { TrialBalancePage } = await import("@/routes/trial-balance/page");
              return { Component: TrialBalancePage };
            },
          },
          {
            path: "/accounting/reports/income-statement",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.AccountingReport),
            ),
            async lazy() {
              const { IncomeStatementPage } = await import("@/routes/income-statement/page");
              return { Component: IncomeStatementPage };
            },
          },
          {
            path: "/accounting/reports/balance-sheet",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.AccountingReport),
            ),
            async lazy() {
              const { BalanceSheetPage } = await import("@/routes/balance-sheet/page");
              return { Component: BalanceSheetPage };
            },
          },
          {
            path: "/accounting/ar/aging",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.AccountsReceivable),
            ),
            async lazy() {
              const { ARAgingPage } = await import("@/routes/ar-aging/page");
              return { Component: ARAgingPage };
            },
          },
          {
            path: "/accounting/ar/customer-ledger",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.AccountsReceivable),
            ),
            async lazy() {
              const { CustomerLedgerPage } = await import("@/routes/customer-ledger/page");
              return { Component: CustomerLedgerPage };
            },
          },
          {
            path: "/accounting/ar/open-items",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.AccountsReceivable),
            ),
            async lazy() {
              const { AROpenItemsPage } = await import("@/routes/ar-open-items/page");
              return { Component: AROpenItemsPage };
            },
          },
          {
            path: "/accounting/ar/payments",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.CustomerPayment),
            ),
            async lazy() {
              const { CustomerPaymentsPage } = await import("@/routes/customer-payments/page");
              return { Component: CustomerPaymentsPage };
            },
          },
          {
            path: "/accounting/ar/customer-statement/:customerId",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.AccountsReceivable),
            ),
            async lazy() {
              const { CustomerStatementPage } = await import("@/routes/customer-statement/page");
              return { Component: CustomerStatementPage };
            },
          },
          {
            path: "/accounting/reconciliation/bank-receipts",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.BankReceipt)),
            async lazy() {
              const { BankReceiptPage } = await import("@/routes/bank-receipt/page");
              return { Component: BankReceiptPage };
            },
          },
          {
            path: "/accounting/reconciliation/work-queue",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.BankReceiptWorkItem),
            ),
            async lazy() {
              const { BankReceiptQueuePage: BankReceiptWorkQueuePage } =
                await import("@/routes/bank-receipt-queue/page");
              return { Component: BankReceiptWorkQueuePage };
            },
          },
          {
            path: "/accounting/reconciliation/summary",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.BankReceipt)),
            async lazy() {
              const { ReconciliationSummaryPage } =
                await import("@/routes/reconciliation-summary/page");
              return { Component: ReconciliationSummaryPage };
            },
          },
          {
            path: "/accounting/reconciliation/import-batches",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.BankReceipt)),
            async lazy() {
              const { BankReceiptBatchPage } = await import("@/routes/bank-receipt-batch/page");
              return { Component: BankReceiptBatchPage };
            },
          },
          {
            path: "/accounting/reconciliation/import-batches/:batchId",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.BankReceipt)),
            async lazy() {
              const { BankReceiptBatchDetailPage } =
                await import("@/routes/bank-receipt-batch/detail-page");
              return { Component: BankReceiptBatchDetailPage };
            },
          },
          {
            path: "/accounting/configuration-files/account-types",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.AccountType)),
            async lazy() {
              const { AccountTypesPage } = await import("@/routes/account-type/page");
              return { Component: AccountTypesPage };
            },
          },
          {
            path: "/billing/configuration-files/formula-templates",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.FormulaTemplate),
            ),
            async lazy() {
              const { FormulaTemplatesPage } = await import("@/routes/formula-template/page");
              return { Component: FormulaTemplatesPage };
            },
          },
          {
            path: "/billing/configuration-files/formula-templates/new",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.FormulaTemplate, Operation.Create),
            ),
            async lazy() {
              const { FormulaStudioCreatePage } =
                await import("@/routes/formula-template/new/page");
              return { Component: FormulaStudioCreatePage };
            },
          },
          {
            path: "/billing/configuration-files/formula-templates/:id/edit",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.FormulaTemplate),
            ),
            async lazy() {
              const { FormulaStudioEditPage } = await import("@/routes/formula-template/[id]/page");
              return { Component: FormulaStudioEditPage };
            },
          },
          {
            path: "/fuel/configuration-files/surcharge",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.FuelSurchargeProgram),
            ),
            async lazy() {
              const { FuelManagementPage } = await import("@/routes/fuel-management/page");
              return { Component: FuelManagementPage };
            },
          },
          {
            path: "/equipment/tractors",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.Tractor),
            ),
            async lazy() {
              const { TractorsPage } = await import("@/routes/tractor/page");
              return { Component: TractorsPage };
            },
          },
          {
            path: "/equipment/trailers",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.Trailer),
            ),
            async lazy() {
              const { TrailersPage } = await import("@/routes/trailer/page");
              return { Component: TrailersPage };
            },
          },
          {
            path: "/fuel/purchases",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.FuelPurchase),
            ),
            async lazy() {
              const { FuelPurchasesPage } = await import("@/routes/fuel-purchase/page");
              return { Component: FuelPurchasesPage };
            },
          },
          {
            path: "/fuel/jurisdiction-mileage",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.IFTAJurisdictionMileage),
            ),
            async lazy() {
              const { JurisdictionMileagePage } =
                await import("@/routes/ifta-jurisdiction-mileage/page");
              return { Component: JurisdictionMileagePage };
            },
          },
          {
            path: "/fuel/ifta-returns",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.IFTAReturn),
              createPrefetchLoader(lazyPrefetch(() => import("@/routes/ifta-return/page"))),
            ),
            async lazy() {
              const { IftaReturnsPage } = await import("@/routes/ifta-return/page");
              return { Component: IftaReturnsPage };
            },
          },
          {
            path: "/fuel/configuration-files/fuel-cards",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.FuelCard),
            ),
            async lazy() {
              const { FuelCardsPage } = await import("@/routes/fuel-card/page");
              return { Component: FuelCardsPage };
            },
          },
          {
            path: "/fuel/unassigned-cards",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.FuelCard),
            ),
            async lazy() {
              const { UnassignedFuelCardsPage } = await import("@/routes/fuel-card/page");
              return { Component: UnassignedFuelCardsPage };
            },
          },
          {
            path: "/fuel/configuration-files/ifta-tax-rates",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.IFTATaxRate),
            ),
            async lazy() {
              const { IftaTaxRatesPage } = await import("@/routes/ifta-tax-rate/page");
              return { Component: IftaTaxRatesPage };
            },
          },
          {
            path: "/equipment/configuration-files/equipment-types",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.EquipmentType)),
            async lazy() {
              const { EquipmentTypesPage } = await import("@/routes/equipment-type/page");
              return { Component: EquipmentTypesPage };
            },
          },

          {
            path: "/equipment/configuration-files/equipment-manufacturers",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.EquipmentManufacturer),
            ),
            async lazy() {
              const { EquipmentManufacturersPage } =
                await import("@/routes/equipment-manufacturer/page");
              return { Component: EquipmentManufacturersPage };
            },
          },
          {
            path: "/dispatch/configuration-files/location-categories",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.LocationCategory),
            ),
            async lazy() {
              const { LocationCategoriesPage } = await import("@/routes/location-category/page");
              return { Component: LocationCategoriesPage };
            },
          },
          {
            path: "/dispatch/configuration-files/fleet-codes",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.FleetCode, Operation.Read),
            ),
            async lazy() {
              const { FleetCodesPage } = await import("@/routes/fleet-code/page");
              return { Component: FleetCodesPage };
            },
          },
          {
            path: "/reports",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.Report, Operation.Read),
              createPrefetchLoader(lazyPrefetch(() => import("@/routes/reports/page"))),
            ),
            async lazy() {
              const { ReportsPage } = await import("@/routes/reports/page");
              return { Component: ReportsPage };
            },
          },
          {
            path: "/reports/runs",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.Report, Operation.Read),
            ),
            async lazy() {
              const { ReportRunsPage } = await import("@/routes/reports/runs/page");
              return { Component: ReportRunsPage };
            },
          },
          {
            path: "/reports/explore",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.Report, Operation.Read),
            ),
            async lazy() {
              const { ReportExplorePage } = await import("@/routes/reports/explore/page");
              return { Component: ReportExplorePage };
            },
          },
          {
            path: "/reports/explore/:definitionId",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.Report, Operation.Read),
            ),
            async lazy() {
              const { ReportExplorePage } = await import("@/routes/reports/explore/page");
              return { Component: ReportExplorePage };
            },
          },
          {
            path: "/reports/dashboards/:dashboardId",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.Report, Operation.Read),
            ),
            async lazy() {
              const { ReportDashboardPage } = await import("@/routes/reports/dashboards/page");
              return { Component: ReportDashboardPage };
            },
          },
          {
            path: "/reports/builder",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.Report, Operation.Create),
            ),
            async lazy() {
              const { ReportBuilderPage } = await import("@/routes/reports/builder/page");
              return { Component: ReportBuilderPage };
            },
          },
          {
            path: "/reports/builder/:definitionId",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.Report, Operation.Read),
            ),
            async lazy() {
              const { ReportBuilderPage } = await import("@/routes/reports/builder/page");
              return { Component: ReportBuilderPage };
            },
          },
          {
            path: "/dispatch/locations",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.Location)),
            async lazy() {
              const { LocationsPage } = await import("@/routes/location/page");
              return { Component: LocationsPage };
            },
          },
          {
            path: "/dispatch/console",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.ShipmentMove),
              createPrefetchLoader(lazyPrefetch(() => import("@/routes/dispatch-console/page"))),
            ),
            async lazy() {
              const { DispatchConsolePage } = await import("@/routes/dispatch-console/page");
              return { Component: DispatchConsolePage };
            },
          },
          {
            path: "/hr/workers",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.Worker),
              createPrefetchLoader(lazyPrefetch(() => import("@/routes/worker/page"))),
            ),
            async lazy() {
              const { WorkersPage } = await import("@/routes/worker/page");
              return { Component: WorkersPage };
            },
          },
          {
            path: "/hr/checklist-templates",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.WorkerChecklistTemplate),
            ),
            async lazy() {
              const { WorkerChecklistTemplatesPage } =
                await import("@/routes/worker-checklist-template/page");
              return { Component: WorkerChecklistTemplatesPage };
            },
          },
          {
            path: "/hr/review-templates",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.PerformanceReviewTemplate),
            ),
            async lazy() {
              const { ReviewTemplatesPage } = await import("@/routes/review-template/page");
              return { Component: ReviewTemplatesPage };
            },
          },
          {
            path: "/hr/training-courses",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.TrainingCourse),
            ),
            async lazy() {
              const { TrainingCoursesPage } = await import("@/routes/training-course/page");
              return { Component: TrainingCoursesPage };
            },
          },
          {
            path: "/hr/credential-types",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.WorkerCredentialType),
            ),
            async lazy() {
              const { WorkerCredentialTypesPage } =
                await import("@/routes/worker-credential-type/page");
              return { Component: WorkerCredentialTypesPage };
            },
          },
          {
            path: "/hr/holidays",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.OrgHoliday),
            ),
            async lazy() {
              const { HolidayCalendarPage } = await import("@/routes/holiday/page");
              return { Component: HolidayCalendarPage };
            },
          },
          {
            path: "/hr/leave-settings",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.WorkerLeave),
            ),
            async lazy() {
              const { LeaveControlPage } = await import("@/routes/leave-control/page");
              return { Component: LeaveControlPage };
            },
          },
          {
            path: "/hr/my-team",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.Worker),
              createPrefetchLoader(lazyPrefetch(() => import("@/routes/my-team/page"))),
            ),
            async lazy() {
              const { MyTeamPage } = await import("@/routes/my-team/page");
              return { Component: MyTeamPage };
            },
          },
          {
            path: "/hr/benefits",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.BenefitPlan),
              createPrefetchLoader(lazyPrefetch(() => import("@/routes/benefits/page"))),
            ),
            async lazy() {
              const { BenefitsPage } = await import("@/routes/benefits/page");
              return { Component: BenefitsPage };
            },
          },
          {
            path: "/hr/policies",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.WorkerPolicy),
            ),
            async lazy() {
              const { PoliciesPage } = await import("@/routes/policies/page");
              return { Component: PoliciesPage };
            },
          },
          {
            path: "/hr/time-attendance",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.Timesheet),
              createPrefetchLoader(lazyPrefetch(() => import("@/routes/time-attendance/page"))),
            ),
            async lazy() {
              const { TimeAttendancePage } = await import("@/routes/time-attendance/page");
              return { Component: TimeAttendancePage };
            },
          },
          {
            path: "/hr/scheduling",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.WorkerSchedule),
              createPrefetchLoader(lazyPrefetch(() => import("@/routes/scheduling/page"))),
            ),
            async lazy() {
              const { SchedulingPage } = await import("@/routes/scheduling/page");
              return { Component: SchedulingPage };
            },
          },
          {
            path: "/hr/org-structure",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.JobPosition),
              createPrefetchLoader(lazyPrefetch(() => import("@/routes/org-structure/page"))),
            ),
            async lazy() {
              const { OrgStructurePage } = await import("@/routes/org-structure/page");
              return { Component: OrgStructurePage };
            },
          },
          {
            path: "/hr/fleet-safety",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.WorkerSafetyEvent),
              createPrefetchLoader(lazyPrefetch(() => import("@/routes/fleet-safety/page"))),
            ),
            async lazy() {
              const { FleetSafetyPage } = await import("@/routes/fleet-safety/page");
              return { Component: FleetSafetyPage };
            },
          },
          {
            path: "/hr/osha-log",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.WorkerInjury),
              createPrefetchLoader(lazyPrefetch(() => import("@/routes/osha/page"))),
            ),
            async lazy() {
              const { OshaLogPage } = await import("@/routes/osha/page");
              return { Component: OshaLogPage };
            },
          },
          {
            path: "/hr/random-testing",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.DOTRandomPool),
              createPrefetchLoader(lazyPrefetch(() => import("@/routes/random-testing/page"))),
            ),
            async lazy() {
              const { RandomTestingPage } = await import("@/routes/random-testing/page");
              return { Component: RandomTestingPage };
            },
          },
          {
            path: "/hr/pto-policies",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.AssetOperations),
              createPermissionLoader(Resource.PTOPolicy),
            ),
            async lazy() {
              const { PTOPoliciesPage } = await import("@/routes/pto-policy/page");
              return { Component: PTOPoliciesPage };
            },
          },
          {
            // Moved to Human Resource Management. Links already sent — stored
            // notifications, bookmarks — keep working and carry their query on.
            path: "/dispatch/workers",
            loader: ({ request }) => redirect(`/hr/workers${new URL(request.url).search}`),
          },
          {
            path: "/dispatch/configuration-files/checklist-templates",
            loader: ({ request }) =>
              redirect(`/hr/checklist-templates${new URL(request.url).search}`),
          },
          {
            path: "/dispatch/configuration-files/review-templates",
            loader: ({ request }) => redirect(`/hr/review-templates${new URL(request.url).search}`),
          },
          {
            path: "/equipment/fuel-purchases",
            loader: ({ request }) => redirect(`/fuel/purchases${new URL(request.url).search}`),
          },
          {
            path: "/equipment/jurisdiction-mileage",
            loader: ({ request }) =>
              redirect(`/fuel/jurisdiction-mileage${new URL(request.url).search}`),
          },
          {
            path: "/equipment/ifta/returns",
            loader: ({ request }) => redirect(`/fuel/ifta-returns${new URL(request.url).search}`),
          },
          {
            path: "/equipment/configuration-files/fuel-cards",
            loader: ({ request }) =>
              redirect(`/fuel/configuration-files/fuel-cards${new URL(request.url).search}`),
          },
          {
            path: "/equipment/configuration-files/ifta-tax-rates",
            loader: ({ request }) =>
              redirect(`/fuel/configuration-files/ifta-tax-rates${new URL(request.url).search}`),
          },
          {
            path: "/billing/fuel-management",
            loader: ({ request }) =>
              redirect(`/fuel/configuration-files/surcharge${new URL(request.url).search}`),
          },
          {
            path: "/dispatch/configuration-files/training-courses",
            loader: ({ request }) => redirect(`/hr/training-courses${new URL(request.url).search}`),
          },
          {
            path: "/dispatch/configuration-files/credential-types",
            loader: ({ request }) => redirect(`/hr/credential-types${new URL(request.url).search}`),
          },
          {
            path: "/dispatch/configuration-files/holidays",
            loader: ({ request }) => redirect(`/hr/holidays${new URL(request.url).search}`),
          },
          {
            path: "/dispatch/configuration-files/pto-policies",
            loader: ({ request }) => redirect(`/hr/pto-policies${new URL(request.url).search}`),
          },
          {
            path: "/dispatch/carriers",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.Brokerage),
              createPermissionLoader(Resource.Carrier),
            ),
            async lazy() {
              const { CarriersPage } = await import("@/routes/carrier/page");
              return { Component: CarriersPage };
            },
          },
          {
            path: "/dispatch/routing-guides",
            loader: combineLoaders(
              protectedLoader,
              createCapabilityLoader(OrganizationCapability.Brokerage),
              createPermissionLoader(Resource.RoutingGuide),
            ),
            async lazy() {
              const { RoutingGuidesPage } = await import("@/routes/routing-guide/page");
              return { Component: RoutingGuidesPage };
            },
          },
          {
            path: "/accounting/configuration-files/fiscal-years",
            loader: combineLoaders(protectedLoader, createPermissionLoader(Resource.FiscalYear)),
            async lazy() {
              const { FiscalYearsPage } = await import("@/routes/fiscal-year/page");
              return { Component: FiscalYearsPage };
            },
          },

          {
            path: "admin",
            Component: AdminLayout,
            HydrateFallback: LoadingSkeleton,
            loader: protectedLoader,
            children: [
              {
                path: "billing-controls",
                loader: createPermissionLoader(Resource.BillingControl),
                async lazy() {
                  const { BillingControlPage } = await import("@/routes/billing-control/page");
                  return { Component: BillingControlPage };
                },
              },
              {
                path: "organization-settings",
                loader: combineLoaders(
                  createPermissionLoader(Resource.Organization, Operation.Read),
                  createPrefetchLoader(
                    lazyPrefetch(() => import("@/routes/admin/organization-settings/page")),
                  ),
                ),
                async lazy() {
                  const { OrganizationSettingsPage } =
                    await import("@/routes/admin/organization-settings/page");
                  return { Component: OrganizationSettingsPage };
                },
              },
              {
                path: "accounting-control",
                loader: createPermissionLoader(Resource.AccountingControl),
                async lazy() {
                  const { AccountingControlPage } =
                    await import("@/routes/accounting-control/page");
                  return { Component: AccountingControlPage };
                },
              },
              {
                path: "cost-control",
                loader: createPermissionLoader(Resource.CostingControl),
                async lazy() {
                  const { CostControlPage } = await import("@/routes/cost-control/page");
                  return { Component: CostControlPage };
                },
              },
              {
                path: "invoice-adjustment-controls",
                loader: createPermissionLoader(Resource.InvoiceAdjustmentControl, Operation.Read),
                async lazy() {
                  const { InvoiceAdjustmentControlPage } =
                    await import("@/routes/invoice-adjustment-control/page");
                  return { Component: InvoiceAdjustmentControlPage };
                },
              },
              {
                path: "table-change-alerts",
                loader: createPermissionLoader(Resource.TableChangeAlert, Operation.Read),
                async lazy() {
                  const { TableChangeAlertPage } = await import("@/routes/table-change-alert/page");
                  return { Component: TableChangeAlertPage };
                },
              },
              {
                path: "data-entry-controls",
                loader: createPermissionLoader(Resource.DataEntryControl, Operation.Read),
                async lazy() {
                  const { DataEntryControlPage } = await import("@/routes/data-entry-control/page");
                  return { Component: DataEntryControlPage };
                },
              },
              {
                path: "dispatch-controls",
                loader: createPermissionLoader(Resource.DispatchControl, Operation.Read),
                async lazy() {
                  const { DispatchControlPage } = await import("@/routes/dispatch-control/page");
                  return { Component: DispatchControlPage };
                },
              },
              {
                path: "settlement-control",
                loader: combineLoaders(
                  createCapabilityLoader(OrganizationCapability.AssetOperations),
                  createPermissionLoader(Resource.SettlementControl, Operation.Read),
                ),
                async lazy() {
                  const { SettlementControlPage } =
                    await import("@/routes/settlement-control/page");
                  return { Component: SettlementControlPage };
                },
              },
              {
                path: "carrier-settlement-control",
                loader: combineLoaders(
                  createCapabilityLoader(OrganizationCapability.Brokerage),
                  createPermissionLoader(Resource.CarrierSettlementControl, Operation.Read),
                ),
                async lazy() {
                  const { CarrierSettlementControlPage } =
                    await import("@/routes/carrier-settlement-control/page");
                  return { Component: CarrierSettlementControlPage };
                },
              },
              {
                path: "dash-control",
                loader: combineLoaders(
                  createCapabilityLoader(OrganizationCapability.AssetOperations),
                  createPermissionLoader(Resource.DashControl, Operation.Read),
                ),
                async lazy() {
                  const { DashControlPage } = await import("@/routes/dash-control/page");
                  return { Component: DashControlPage };
                },
              },
              {
                path: "agent-control",
                loader: createPermissionLoader(Resource.AgentControl, Operation.Read),
                async lazy() {
                  const { AgentControlPage } = await import("@/routes/agent-control/page");
                  return { Component: AgentControlPage };
                },
              },
              {
                path: "distance-controls",
                loader: createPermissionLoader(Resource.DistanceControl),
                async lazy() {
                  const { DistanceControlsPage } =
                    await import("@/routes/admin/distance-controls/page");
                  return { Component: DistanceControlsPage };
                },
              },
              {
                path: "distance-overrides",
                loader: createPermissionLoader(Resource.DistanceOverride),
                async lazy() {
                  const { DistanceOverridesPage } = await import("@/routes/distance-override/page");
                  return { Component: DistanceOverridesPage };
                },
              },
              {
                path: "distance-profiles",
                loader: createPermissionLoader(Resource.DistanceProfile),
                async lazy() {
                  const { DistanceProfilesPage } =
                    await import("@/routes/admin/distance-profiles/page");
                  return { Component: DistanceProfilesPage };
                },
              },
              {
                path: "stored-mileages",
                loader: createPermissionLoader(Resource.StoredMileage),
                async lazy() {
                  const { StoredMileagesPage } =
                    await import("@/routes/admin/stored-mileages/page");
                  return { Component: StoredMileagesPage };
                },
              },
              {
                path: "shipment-controls",
                loader: createPermissionLoader(Resource.ShipmentControl, Operation.Read),
                async lazy() {
                  const { ShipmentControlPage } = await import("@/routes/shipment-control/page");
                  return { Component: ShipmentControlPage };
                },
              },
              {
                path: "document-intelligence",
                loader: createPermissionLoader(Resource.DocumentControl, Operation.Read),
                async lazy() {
                  const { DocumentIntelligencePage } =
                    await import("@/routes/admin/document-intelligence/page");
                  return { Component: DocumentIntelligencePage };
                },
              },
              {
                path: "document-parsing-rules",
                loader: createPermissionLoader(Resource.DocumentParsingRule, Operation.Read),
                async lazy() {
                  const { DocumentParsingRulesPage } =
                    await import("@/routes/admin/document-parsing-rules/page");
                  return { Component: DocumentParsingRulesPage };
                },
              },
              {
                path: "sequence-configs",
                loader: createPermissionLoader(Resource.SequenceConfig, Operation.Read),
                async lazy() {
                  const { SequenceConfigPage } =
                    await import("@/routes/admin/sequence-config/page");
                  return { Component: SequenceConfigPage };
                },
              },
              {
                path: "hold-reasons",
                loader: combineLoaders(
                  protectedLoader,
                  createPermissionLoader(Resource.HoldReason),
                ),
                async lazy() {
                  const { HoldReasonsPage } = await import("@/routes/hold-reason/page");
                  return { Component: HoldReasonsPage };
                },
              },
              {
                path: "jurisdiction-rules",
                loader: combineLoaders(
                  protectedLoader,
                  createPermissionLoader(Resource.JurisdictionRule),
                ),
                async lazy() {
                  const { JurisdictionRulesPage } = await import("@/routes/jurisdiction-rule/page");
                  return { Component: JurisdictionRulesPage };
                },
              },
              {
                path: "jurisdiction-rule-overrides",
                loader: combineLoaders(
                  protectedLoader,
                  createPermissionLoader(Resource.JurisdictionRuleOverride),
                ),
                async lazy() {
                  const { JurisdictionRuleOverridesPage } =
                    await import("@/routes/jurisdiction-rule-override/page");
                  return { Component: JurisdictionRuleOverridesPage };
                },
              },
              {
                path: "service-failure-reason-codes",
                loader: combineLoaders(
                  protectedLoader,
                  createPermissionLoader(Resource.ServiceFailureReasonCode),
                ),
                async lazy() {
                  const { ServiceFailureReasonCodesPage } =
                    await import("@/routes/service-failure-reason-code/page");
                  return { Component: ServiceFailureReasonCodesPage };
                },
              },
              {
                path: "hazmat-segregation-rules",
                loader: createPermissionLoader(Resource.HazmatSegregationRule),
                async lazy() {
                  const { HazmatSegregationRulesPage } =
                    await import("@/routes/hazmat-segregation-rule/page");
                  return { Component: HazmatSegregationRulesPage };
                },
              },
              {
                path: "home-layouts",
                loader: createPermissionLoader(Resource.HomeLayoutPreset, Operation.Read),
                async lazy() {
                  const { HomeLayoutsPage } = await import("@/routes/admin/home-layouts/page");
                  return { Component: HomeLayoutsPage };
                },
              },
              {
                path: "home-layouts/new",
                loader: createPermissionLoader(Resource.HomeLayoutPreset, Operation.Create),
                async lazy() {
                  const { NewHomeLayoutPage } =
                    await import("@/routes/admin/home-layouts/new/page");
                  return { Component: NewHomeLayoutPage };
                },
              },
              {
                path: "home-layouts/:id",
                loader: createPermissionLoader(Resource.HomeLayoutPreset, Operation.Update),
                async lazy() {
                  const { EditHomeLayoutPage } =
                    await import("@/routes/admin/home-layouts/[id]/page");
                  return { Component: EditHomeLayoutPage };
                },
              },
              {
                path: "roles",
                loader: createPermissionLoader(Resource.Role, Operation.Read),
                async lazy() {
                  const { RolesPage } = await import("@/routes/admin/roles/page");
                  return { Component: RolesPage };
                },
              },
              {
                path: "roles/new",
                loader: createPermissionLoader(Resource.Role, Operation.Create),
                async lazy() {
                  const { RoleCreatePage } = await import("@/routes/admin/roles/new/page");
                  return { Component: RoleCreatePage };
                },
              },
              {
                path: "roles/:id/edit",
                loader: createPermissionLoader(Resource.Role, Operation.Update),
                async lazy() {
                  const { RoleEditPage } = await import("@/routes/admin/roles/[id]/edit/page");
                  return { Component: RoleEditPage };
                },
              },
              {
                path: "users",
                loader: createPermissionLoader(Resource.User, Operation.Read),
                async lazy() {
                  const { UsersPage } = await import("@/routes/admin/users/page");
                  return { Component: UsersPage };
                },
              },
              {
                path: "audit-logs",
                loader: createPermissionLoader(Resource.AuditLog, Operation.Read),
                async lazy() {
                  const { AuditLogsPage } = await import("@/routes/admin/audit-logs/page");
                  return { Component: AuditLogsPage };
                },
              },
              {
                path: "database-sessions",
                loader: createPermissionLoader(Resource.DatabaseSession, Operation.Read),
                async lazy() {
                  const { DatabaseSessionsPage } =
                    await import("@/routes/admin/database-sessions/page");
                  return { Component: DatabaseSessionsPage };
                },
              },
              {
                path: "graphql-explorer",
                loader: createPermissionLoader(Resource.Organization, Operation.Read),
                async lazy() {
                  const { GraphQLExplorerPage } =
                    await import("@/routes/admin/graphql-explorer/page");
                  return { Component: GraphQLExplorerPage };
                },
              },
              {
                path: "integrations",
                loader: createPermissionLoader(Resource.Integration, Operation.Read),
                async lazy() {
                  const { IntegrationsPage } = await import("@/routes/admin/integrations/page");
                  return { Component: IntegrationsPage };
                },
              },
              {
                path: "api-keys",
                loader: createPermissionLoader(Resource.APIKey, Operation.Read),
                async lazy() {
                  const { APIKeysPage } = await import("@/routes/admin/api-keys/page");
                  return { Component: APIKeysPage };
                },
              },
              {
                path: "document-operations",
                loader: createPermissionLoader(Resource.DocumentOperation, Operation.Read),
                async lazy() {
                  const { DocumentOperationsPage } =
                    await import("@/routes/admin/document-operations/page");
                  return { Component: DocumentOperationsPage };
                },
              },
              {
                path: "custom-fields",
                loader: createPermissionLoader(Resource.CustomFieldDefinition, Operation.Read),
                async lazy() {
                  const { CustomFieldDefinitionsPage } =
                    await import("@/routes/admin/custom-fields/page");
                  return { Component: CustomFieldDefinitionsPage };
                },
              },
            ],
          },
        ],
      },
      {
        // Public carrier-facing pages: no auth, no app chrome. An external
        // carrier lands here from an emailed offer link, logged in or not.
        children: [
          {
            path: "/tender-offer/:token",
            async lazy() {
              const { TenderOfferPublicPage } = await import("@/routes/tender-offer-public/page");
              return { Component: TenderOfferPublicPage };
            },
          },
          {
            path: "/tender-offer/:token/accept",
            async lazy() {
              const { TenderOfferPublicPage } = await import("@/routes/tender-offer-public/page");
              return { Component: TenderOfferPublicPage };
            },
          },
          {
            path: "/tender-offer/:token/decline",
            async lazy() {
              const { TenderOfferPublicPage } = await import("@/routes/tender-offer-public/page");
              return { Component: TenderOfferPublicPage };
            },
          },
          {
            path: "/rate-confirmation/:token",
            async lazy() {
              const { RateConfirmationPublicPage } =
                await import("@/routes/rate-confirmation-public/page");
              return { Component: RateConfirmationPublicPage };
            },
          },
        ],
      },
      {
        loader: guestLoader,
        children: [
          {
            path: "/login",
            async lazy() {
              const { AuthPage } = await import("@/routes/auth/page");
              return { Component: AuthPage };
            },
          },
          {
            path: "/login/:orgSlug",
            async lazy() {
              const { AuthPage } = await import("@/routes/auth/page");
              return { Component: AuthPage };
            },
          },
          {
            // Opened from the emailed reset link. Guest-only, like the sign-in page:
            // somebody already signed in has no use for it.
            path: "/auth/reset",
            async lazy() {
              const { ResetPasswordPage } = await import("@/routes/auth/reset-page");
              return { Component: ResetPasswordPage };
            },
          },
        ],
      },
    ],
  },
];

export const router = createBrowserRouter(routes);
