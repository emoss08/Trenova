import { AppErrorLayout, AppLayout, OnboardingLayout } from "@/routes/app-layout";
import { AdminLayout } from "@/routes/admin-layout";
import { DeskLoadingScreen } from "@/routes/desk/desk-loading-screen";
import { DeskShellLayout } from "@/routes/desk/shell-layout";
import { RootLayout } from "@/routes/root-layout";
import { editionRoutes } from "@/lib/edition-routes";
import { edition } from "@/lib/edition";
import { requireManagedInstance } from "@/lib/managed-instance";
import {
  onboardingExitRedirect,
  loadOnboardingSnapshot,
  resolveOnboardingRedirect,
} from "@/lib/onboarding-gate";
import { PlanFeature } from "@/lib/plan-feature";
import { requirePlanFeature } from "@/lib/plan-feature-gate";
import { lazyPage } from "@/lib/route-catalogs";
import { anyOfGates, requireCapability, requirePermission } from "@/lib/route-permission";
import { prefetchPage } from "@/lib/route-prefetch";
import { legacySettingsRedirect } from "@/lib/settings-path";
import { UrlStateAdapter } from "@/lib/url-state-adapter";
import { RouteErrorBoundary } from "@trenova/shared/components/error-boundary";
import LoadingSkeleton from "@trenova/shared/components/loading-skeleton";
import {
  gates,
  redirectKeepingSearch,
  redirectTo,
  requireGuest,
  requireSession,
  type RouteGate,
} from "@trenova/shared/lib/router/gates";
import {
  parseSearch,
  stringifySearch,
  validateSearchRecord,
} from "@trenova/shared/lib/router/search";
import { OrganizationCapability } from "@trenova/shared/types/organization-capability";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { createRootRoute, createRoute, createRouter, redirect } from "@tanstack/react-router";

// A cloud organization that has not finished the welcome wizard is sent to it from every
// signed-in page; the wizard's own path is exempt, so this cannot loop.
const onboardingRedirect: RouteGate = async ({ location }) => {
  const target = await resolveOnboardingRedirect(location.pathname);
  if (target) {
    throw redirect({ href: target });
  }
};

/**
 * The gate every signed-in layout runs: the session, the onboarding redirect, then the
 * edition's own gates, the first to throw deciding. Each layout gets its own, because a gate
 * remembers the path it last let through (see `gates`).
 */
function protectedGate(): RouteGate {
  return gates(requireSession("/login"), onboardingRedirect, ...edition.protectedGates);
}

// The wizard is only for an organization that still owes it. Anybody else who opens
// the address — a completed organization, a self-hosted install — goes home.
const onboardingGate: RouteGate = async () => {
  const target = onboardingExitRedirect(await loadOnboardingSnapshot());
  if (target) {
    throw redirect({ href: target });
  }
};

// Settings used to open over the page from `/?settings=<section>`, and emails already sent
// still link there.
const legacySettingsGate: RouteGate = ({ location }) => {
  const target = legacySettingsRedirect(location.searchStr);
  if (target) {
    throw redirect({ href: target });
  }
};

const guestGate = requireGuest("/");

// A tender's record link opens the inbound page; one we sent is carried to the outbound page.
const ediSentTransferGate: RouteGate = async ({ location, abortController }) => {
  const url = new URL(`${location.pathname}${location.searchStr}`, window.location.origin);
  if (!url.searchParams.has("panelEntityId")) {
    return;
  }
  const { sentTransferPath } = await import("@/lib/edi/transfer-page");
  const sent = await sentTransferPath(url, abortController.signal);
  if (sent) {
    throw redirect({ href: sent });
  }
};

// A layout entered for the first time (the app's first load, or arriving from sign-in) draws
// its loading screen at once, as the page used to before the router had answered. A layout
// that stays while its page changes is already loaded, so a change of page keeps the page
// standing with the progress bar until the next one is ready.
const SHOWN_AT_ONCE = { pendingMs: 0, pendingMinMs: 0 } as const;

const rootRoute = createRootRoute({
  component: RootLayout,
  validateSearch: validateSearchRecord,
  pendingComponent: LoadingSkeleton,
  pendingMs: 0,
});

// A page that fails to load or render keeps the shell standing, so the way to
// anywhere else is still on screen.
const appRoute = createRoute({
  getParentRoute: () => rootRoute,
  id: "app",
  pendingComponent: LoadingSkeleton,
  ...SHOWN_AT_ONCE,
  beforeLoad: protectedGate(),
  component: AppLayout,
  errorComponent: AppErrorLayout,
});

const adminRoute = createRoute({
  getParentRoute: () => appRoute,
  path: "admin",
  component: AdminLayout,
});

// The welcome wizard a new cloud organization walks through once. It is a
// full-screen step of its own rather than a page in the app: nothing in the
// sidebar is useful until the organization has a profile. It still runs the
// session's gates, so a password change or role activation comes first.
const onboardingRoute = createRoute({
  getParentRoute: () => rootRoute,
  id: "onboarding",
  pendingComponent: LoadingSkeleton,
  ...SHOWN_AT_ONCE,
  beforeLoad: protectedGate(),
  component: OnboardingLayout,
});

// The Desk runs outside the app chrome.
//
// It is a room rather than a page: a conversation on one side and
// the work it produces on the other, both of which want the whole
// window. Mounted under AppLayout it paid for a sidebar, a
// breadcrumb and a page header before a word of conversation —
// roughly 190px of furniture on a surface whose entire job is to
// give the work room. It keeps the protected loader, so it is no
// less guarded than anything inside; it simply wears none of the
// frame. The way back is in its own chrome.
const deskShellRoute = createRoute({
  getParentRoute: () => rootRoute,
  id: "deskShell",
  beforeLoad: protectedGate(),
  component: DeskShellLayout,
  pendingComponent: DeskLoadingScreen,
  ...SHOWN_AT_ONCE,
});

const deskRoute = createRoute({
  getParentRoute: () => deskShellRoute,
  path: "/desk",
  beforeLoad: gates(requirePermission(Resource.Assistant, Operation.Read)),
  loader: prefetchPage(() => import("@/routes/desk/page")),
  component: lazyPage(() => import("@/routes/desk/page"), "DeskPage"),
});

// Public carrier-facing pages: no auth, no app chrome. An external
// carrier lands here from an emailed offer link, logged in or not.
const publicRoute = createRoute({
  getParentRoute: () => rootRoute,
  id: "public",
});

const guestRoute = createRoute({
  getParentRoute: () => rootRoute,
  id: "guest",
  pendingComponent: LoadingSkeleton,
  ...SHOWN_AT_ONCE,
  beforeLoad: gates(guestGate),
});

export const routeTree = rootRoute.addChildren([
  appRoute.addChildren([
    createRoute({
      getParentRoute: () => appRoute,
      path: "/",
      beforeLoad: gates(legacySettingsGate),
      loader: prefetchPage(() => import("@/routes/home/page")),
      component: lazyPage(() => import("@/routes/home/page"), "Home"),
    }),
    // Settings is a room: the module rail stays, and its sections take the panel's
    // place and its fold.
    createRoute({
      getParentRoute: () => appRoute,
      path: "/settings",
      loader: prefetchPage(() => import("@/routes/settings/page")),
      staticData: { frame: "room" },
      component: lazyPage(() => import("@/routes/settings/page"), "SettingsPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/intake",
      beforeLoad: gates(requirePermission(Resource.CaptureBatch, Operation.Read)),
      loader: prefetchPage(() => import("@/routes/intake/page")),
      component: lazyPage(() => import("@/routes/intake/page"), "IntakePage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/capture/pair",
      component: lazyPage(() => import("@/routes/capture/pair/page"), "CapturePairPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/capture/devices",
      loader: prefetchPage(() => import("@/routes/capture/devices/page")),
      component: lazyPage(() => import("@/routes/capture/devices/page"), "CaptureDevicesPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/organization/data-retention",
      beforeLoad: gates(requirePermission(Resource.Organization)),
      loader: prefetchPage(() => import("@/routes/organization/data-retention/page")),
      component: lazyPage(
        () => import("@/routes/organization/data-retention/page"),
        "DataRetentionPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/organization/email-profiles",
      beforeLoad: gates(
        requirePlanFeature(PlanFeature.Administration),
        requirePermission(Resource.EmailProfile),
      ),
      component: lazyPage(
        () => import("@/routes/organization/email-profiles/page"),
        "EmailProfilesPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/organization/email-logs",
      beforeLoad: gates(
        requirePlanFeature(PlanFeature.Administration),
        requirePermission(Resource.EmailLog),
      ),
      component: lazyPage(() => import("@/routes/organization/email-logs/page"), "EmailLogsPage"),
    }),
    // The assistant is a floating panel on every page now; an old
    // link opens it on the home screen.
    createRoute({
      getParentRoute: () => appRoute,
      path: "/assistant",
      beforeLoad: gates(redirectTo("/?assistant=open")),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/insights",
      beforeLoad: gates(requirePermission(Resource.Insight, Operation.Read)),
      component: lazyPage(() => import("@/routes/insights/page"), "InsightsPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/shipment-management/shipments",
      beforeLoad: gates(requirePermission(Resource.Shipment)),
      loader: prefetchPage(() => import("@/routes/shipment/page")),
      component: lazyPage(() => import("@/routes/shipment/page"), "ShipmentsPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/shipment-management/recurring-shipments",
      beforeLoad: gates(requirePermission(Resource.RecurringShipment)),
      component: lazyPage(
        () => import("@/routes/recurring-shipment/page"),
        "RecurringShipmentsPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/shipment-management/orders",
      beforeLoad: gates(requirePermission(Resource.Order)),
      component: lazyPage(() => import("@/routes/order/page"), "OrdersPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/shipment-management/service-failures",
      beforeLoad: gates(requirePermission(Resource.ServiceFailure)),
      component: lazyPage(() => import("@/routes/service-failure/page"), "ServiceFailuresPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/shipment-management/shipments/import",
      beforeLoad: gates(
        requirePermission(Resource.Shipment, Operation.Create),
        requirePlanFeature(PlanFeature.DocumentIntelligence),
      ),
      component: lazyPage(() => import("@/routes/shipment/import-page"), "ShipmentImportPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/shipment-management/configuration-files/shipment-types",
      beforeLoad: gates(requirePermission(Resource.ShipmentType)),
      component: lazyPage(() => import("@/routes/shipment-type/page"), "ShipmentTypesPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/shipment-management/configuration-files/service-types",
      beforeLoad: gates(requirePermission(Resource.ServiceType)),
      component: lazyPage(() => import("@/routes/service-type/page"), "ServiceTypesPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/shipment-management/configuration-files/hazardous-materials",
      beforeLoad: gates(requirePermission(Resource.HazardousMaterial)),
      component: lazyPage(
        () => import("@/routes/hazardous-material/page"),
        "HazardousMaterialsPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/shipment-management/configuration-files/commodities",
      beforeLoad: gates(requirePermission(Resource.Commodity)),
      component: lazyPage(() => import("@/routes/commodity/page"), "CommoditiesPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/edi/overview",
      beforeLoad: gates(
        requirePlanFeature(PlanFeature.Administration),
        requirePermission(Resource.EDI),
      ),
      component: lazyPage(() => import("@/routes/edi/page"), "EDIOverviewPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/edi/partners",
      beforeLoad: gates(
        requirePlanFeature(PlanFeature.Administration),
        requirePermission(Resource.EDI),
      ),
      component: lazyPage(() => import("@/routes/edi/page"), "EDIPartnersPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/edi/communication-profiles",
      beforeLoad: gates(
        requirePlanFeature(PlanFeature.Administration),
        requirePermission(Resource.EDI),
      ),
      component: lazyPage(() => import("@/routes/edi/page"), "EDICommunicationProfilesPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/edi/mapping-profiles",
      beforeLoad: gates(
        requirePlanFeature(PlanFeature.Administration),
        requirePermission(Resource.EDI),
      ),
      component: lazyPage(() => import("@/routes/edi/page"), "EDIMappingProfilesPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/edi/lading",
      beforeLoad: gates(
        requirePlanFeature(PlanFeature.Administration),
        requirePermission(Resource.EDI),
      ),
      component: lazyPage(() => import("@/routes/edi/lading/page"), "LadingProjectsPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/edi/lading/$partnerId",
      beforeLoad: gates(
        requirePlanFeature(PlanFeature.Administration),
        requirePermission(Resource.EDI),
      ),
      component: lazyPage(() => import("@/routes/edi/lading/[partnerId]/page"), "LadingStudioPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/edi/transfers/inbound",
      beforeLoad: gates(
        requirePlanFeature(PlanFeature.Administration),
        requirePermission(Resource.EDI),
        ediSentTransferGate,
      ),
      component: lazyPage(() => import("@/routes/edi/page"), "EDIInboundTransfersPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/edi/transfers/outbound",
      beforeLoad: gates(
        requirePlanFeature(PlanFeature.Administration),
        requirePermission(Resource.EDI),
      ),
      component: lazyPage(() => import("@/routes/edi/page"), "EDIOutboundTransfersPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/edi/messages",
      beforeLoad: gates(
        requirePlanFeature(PlanFeature.Administration),
        requirePermission(Resource.EDI),
      ),
      component: lazyPage(() => import("@/routes/edi/page"), "EDIMessagesPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/edi/inbound-files",
      beforeLoad: gates(
        requirePlanFeature(PlanFeature.Administration),
        requirePermission(Resource.EDI),
      ),
      component: lazyPage(() => import("@/routes/edi/page"), "EDIInboundFilesPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/edi/flows",
      beforeLoad: gates(
        requirePlanFeature(PlanFeature.Administration),
        requirePermission(Resource.EDI),
      ),
      component: lazyPage(() => import("@/routes/edi/page"), "EDIFlowInstancesPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/billing/queue",
      beforeLoad: gates(requirePermission(Resource.BillingQueue)),
      loader: prefetchPage(() => import("@/routes/billing-queue/page")),
      component: lazyPage(() => import("@/routes/billing-queue/page"), "BillingQueuePage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/billing/factoring",
      beforeLoad: gates(
        requirePlanFeature(PlanFeature.Factoring),
        requirePermission(Resource.Invoice),
      ),
      component: lazyPage(() => import("@/routes/factoring/page"), "FactoringPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/billing/factoring/submissions",
      beforeLoad: gates(redirectTo("/billing/factoring?view=schedules")),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/billing/factoring/remittances",
      beforeLoad: gates(redirectTo("/billing/factoring?view=remittances")),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/billing/factoring/remittances/$id",
      beforeLoad: gates(requirePermission(Resource.FactoringRemittance)),
      loader: prefetchPage(() => import("@/routes/factoring-remittance/page")),
      component: lazyPage(
        () => import("@/routes/factoring-remittance/page"),
        "FactoringRemittancePage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/billing/factoring/submissions/$id",
      beforeLoad: gates(requirePermission(Resource.FactoringSubmission)),
      component: lazyPage(
        () => import("@/routes/factoring-submission/page"),
        "FactoringSubmissionPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/payroll/workspace",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.DriverSettlement),
      ),
      component: lazyPage(
        () => import("@/routes/settlement-workspace/page"),
        "SettlementWorkspacePage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/payroll/settlements",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.DriverSettlement),
      ),
      component: lazyPage(() => import("@/routes/driver-settlement/page"), "DriverSettlementsPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/payroll/disputes",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.SettlementDispute),
      ),
      component: lazyPage(
        () => import("@/routes/settlement-dispute/page"),
        "SettlementDisputesPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/payroll/expenses",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.DriverExpense),
      ),
      component: lazyPage(() => import("@/routes/driver-expense/page"), "DriverExpensesPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/payroll/settlement-batches",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.DriverSettlement),
      ),
      component: lazyPage(() => import("@/routes/settlement-batch/page"), "SettlementBatchesPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/payroll/pay-events",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.DriverSettlement),
      ),
      component: lazyPage(() => import("@/routes/driver-pay-event/page"), "DriverPayEventsPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/payroll/pay-profiles",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.DriverPayProfile),
      ),
      component: lazyPage(() => import("@/routes/pay-profile/page"), "PayProfilesPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/payroll/deductions",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.RecurringDeduction),
      ),
      component: lazyPage(
        () => import("@/routes/recurring-deduction/page"),
        "RecurringDeductionsPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/payroll/earnings",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.RecurringEarning),
      ),
      component: lazyPage(() => import("@/routes/recurring-earning/page"), "RecurringEarningsPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/payroll/pay-codes",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.PayCode),
      ),
      component: lazyPage(() => import("@/routes/pay-code/page"), "PayCodesPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/payroll/advances",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.PayAdvance),
      ),
      component: lazyPage(() => import("@/routes/pay-advance/page"), "PayAdvancesPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/payroll/escrow-accounts",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.EscrowAccount),
      ),
      component: lazyPage(() => import("@/routes/escrow-account/page"), "EscrowAccountsPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/carrier-settlements/workspace",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.Brokerage),
        requirePermission(Resource.CarrierSettlement),
      ),
      component: lazyPage(
        () => import("@/routes/carrier-settlement-workspace/page"),
        "CarrierSettlementWorkspacePage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/carrier-settlements/settlements",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.Brokerage),
        requirePermission(Resource.CarrierSettlement),
      ),
      component: lazyPage(
        () => import("@/routes/carrier-settlement/page"),
        "CarrierSettlementsPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/carrier-settlements/batches",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.Brokerage),
        requirePermission(Resource.CarrierSettlement),
      ),
      component: lazyPage(
        () => import("@/routes/carrier-settlement-batch/page"),
        "CarrierSettlementBatchesPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/carrier-settlements/cost-events",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.Brokerage),
        requirePermission(Resource.CarrierSettlement),
      ),
      component: lazyPage(
        () => import("@/routes/carrier-cost-event/page"),
        "CarrierCostEventsPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/carrier-settlements/invoice-matching",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.Brokerage),
        requirePermission(Resource.CarrierInvoiceMatch),
      ),
      component: lazyPage(
        () => import("@/routes/carrier-invoice-matching/page"),
        "CarrierInvoiceMatchingPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/billing/invoices",
      beforeLoad: gates(requirePermission(Resource.Invoice)),
      component: lazyPage(() => import("@/routes/invoice/page"), "InvoicesPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/billing/invoice-register",
      beforeLoad: gates(requirePermission(Resource.Invoice)),
      component: lazyPage(() => import("@/routes/invoice-register/page"), "InvoiceRegisterPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/billing/pending-approvals",
      beforeLoad: gates(requirePermission(Resource.Invoice)),
      component: lazyPage(() => import("@/routes/invoice-approval/page"), "InvoiceApprovalPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/billing/reconciliation-exceptions",
      beforeLoad: gates(requirePermission(Resource.Invoice)),
      component: lazyPage(
        () => import("@/routes/invoice-reconciliation/page"),
        "InvoiceReconciliationPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/billing/adjustment-batches",
      beforeLoad: gates(requirePermission(Resource.Invoice)),
      component: lazyPage(
        () => import("@/routes/invoice-adjustment-batch/page"),
        "InvoiceAdjustmentBatchPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/billing/configuration-files/charge-types",
      beforeLoad: gates(requirePermission(Resource.ChargeType)),
      component: lazyPage(() => import("@/routes/placeholder-page"), "PlaceholderPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/billing/rate-agreements",
      beforeLoad: gates(requirePermission(Resource.RateAgreement)),
      component: lazyPage(() => import("@/routes/rate-agreement/page"), "RateAgreementPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/billing/configuration-files/rate-zones",
      beforeLoad: gates(requirePermission(Resource.RateZone)),
      component: lazyPage(() => import("@/routes/rate-zone/page"), "RateZonePage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/billing/configuration-files/rate-matrices",
      beforeLoad: gates(requirePermission(Resource.RateMatrix)),
      component: lazyPage(() => import("@/routes/rate-matrix/page"), "RateMatrixPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/detention/configuration-files/detention-policies",
      beforeLoad: gates(requirePermission(Resource.DetentionPolicy)),
      component: lazyPage(() => import("@/routes/detention-policy/page"), "DetentionPolicyPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/detention/intelligence",
      beforeLoad: gates(requirePermission(Resource.Detention)),
      loader: prefetchPage(() => import("@/routes/detention-intelligence/page")),
      component: lazyPage(
        () => import("@/routes/detention-intelligence/page"),
        "DetentionIntelligencePage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/detention/desk",
      beforeLoad: gates(requirePermission(Resource.Detention)),
      component: lazyPage(() => import("@/routes/detention-desk/page"), "DetentionDeskPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/billing/configuration-files/accessorial-charges",
      beforeLoad: gates(requirePermission(Resource.AccessorialCharge)),
      component: lazyPage(
        () => import("@/routes/accessorial-charge/page"),
        "AccessorialChargesPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/billing/configuration-files/customers",
      beforeLoad: gates(requirePermission(Resource.Customer)),
      component: lazyPage(() => import("@/routes/customer/page"), "CustomersPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/billing/configuration-files/factoring-companies",
      beforeLoad: gates(requirePermission(Resource.FactoringCompany)),
      component: lazyPage(
        () => import("@/routes/factoring-company/page"),
        "FactoringCompaniesPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/billing/configuration-files/factoring-companies/new",
      beforeLoad: gates(requirePermission(Resource.FactoringCompany, Operation.Create)),
      component: lazyPage(
        () => import("@/routes/factoring-company/new/page"),
        "FactoringCompanyCreatePage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/billing/configuration-files/factoring-companies/agreements/$id",
      beforeLoad: gates(requirePermission(Resource.FactoringAgreement)),
      component: lazyPage(
        () => import("@/routes/factoring-company/record-redirect-page"),
        "FactoringAgreementRedirectPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/billing/configuration-files/factoring-companies/notices/$id",
      beforeLoad: gates(requirePermission(Resource.FactoringNoa)),
      component: lazyPage(
        () => import("@/routes/factoring-company/record-redirect-page"),
        "FactoringNoticeRedirectPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/billing/configuration-files/factoring-companies/$id",
      beforeLoad: gates(requirePermission(Resource.FactoringCompany)),
      loader: prefetchPage(() => import("@/routes/factoring-company/[id]/page")),
      component: lazyPage(
        () => import("@/routes/factoring-company/[id]/page"),
        "FactoringCompanyPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/billing/configuration-files/document-types",
      beforeLoad: gates(requirePermission(Resource.DocumentType)),
      component: lazyPage(() => import("@/routes/document-type/page"), "DocumentTypesPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/billing/configuration-files/document-packet-rules",
      beforeLoad: gates(requirePermission(Resource.DocumentType)),
      component: lazyPage(
        () => import("@/routes/document-packet-rule/page"),
        "DocumentPacketRulesPage",
      ),
    }),
    // Case checklists moved into the Desk's settings; an old link
    // opens them there.
    createRoute({
      getParentRoute: () => appRoute,
      path: "/billing/configuration-files/case-checklists",
      beforeLoad: gates(redirectTo("/desk?settings=checklists")),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/accounting",
      beforeLoad: gates(requirePermission(Resource.AccountsReceivable)),
      component: lazyPage(
        () => import("@/routes/accounting-dashboard/page"),
        "AccountingDashboardPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/accounting/manual-journals",
      beforeLoad: gates(requirePermission(Resource.ManualJournal)),
      component: lazyPage(() => import("@/routes/manual-journal/page"), "ManualJournalsPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/accounting/journal-reversals",
      beforeLoad: gates(requirePermission(Resource.JournalReversal)),
      component: lazyPage(() => import("@/routes/journal-reversal/page"), "JournalReversalsPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/accounting/journals-to-post",
      beforeLoad: gates(requirePermission(Resource.JournalEntry)),
      component: lazyPage(() => import("@/routes/journal-review/page"), "JournalReviewPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/accounting/journal-entries/$id",
      beforeLoad: gates(requirePermission(Resource.JournalEntry)),
      component: lazyPage(() => import("@/routes/journal-entry/page"), "JournalEntryDetailPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/accounting/journal-entries/source/$type/$sourceId",
      beforeLoad: gates(requirePermission(Resource.JournalEntry)),
      component: lazyPage(
        () => import("@/routes/journal-entry/_components/source-drill-down-page"),
        "SourceDrillDownPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/accounting/reports/trial-balance",
      beforeLoad: gates(requirePermission(Resource.AccountingReport)),
      component: lazyPage(() => import("@/routes/trial-balance/page"), "TrialBalancePage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/accounting/reports/income-statement",
      beforeLoad: gates(requirePermission(Resource.AccountingReport)),
      component: lazyPage(() => import("@/routes/income-statement/page"), "IncomeStatementPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/accounting/reports/balance-sheet",
      beforeLoad: gates(requirePermission(Resource.AccountingReport)),
      component: lazyPage(() => import("@/routes/balance-sheet/page"), "BalanceSheetPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/accounting/ar/aging",
      beforeLoad: gates(requirePermission(Resource.AccountsReceivable)),
      component: lazyPage(() => import("@/routes/ar-aging/page"), "ARAgingPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/accounting/ar/customer-ledger",
      beforeLoad: gates(requirePermission(Resource.AccountsReceivable)),
      component: lazyPage(() => import("@/routes/customer-ledger/page"), "CustomerLedgerPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/accounting/ar/collections",
      beforeLoad: gates(requirePermission(Resource.AccountsReceivable)),
      component: lazyPage(() => import("@/routes/collections/page"), "CollectionsPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/accounting/ar/open-items",
      beforeLoad: gates(requirePermission(Resource.AccountsReceivable)),
      component: lazyPage(() => import("@/routes/ar-open-items/page"), "AROpenItemsPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/accounting/ar/late-charges",
      beforeLoad: gates(requirePermission(Resource.AccountsReceivable)),
      loader: prefetchPage(() => import("@/routes/late-charges/page")),
      component: lazyPage(() => import("@/routes/late-charges/page"), "LateChargesPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/accounting/ar/payments",
      beforeLoad: gates(requirePermission(Resource.CustomerPayment)),
      component: lazyPage(() => import("@/routes/customer-payments/page"), "CustomerPaymentsPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/accounting/ar/customer-statement/$customerId",
      beforeLoad: gates(requirePermission(Resource.AccountsReceivable)),
      component: lazyPage(
        () => import("@/routes/customer-statement/page"),
        "CustomerStatementPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/accounting/reconciliation/bank-receipts",
      beforeLoad: gates(requirePermission(Resource.BankReceipt)),
      component: lazyPage(() => import("@/routes/bank-receipt/page"), "BankReceiptPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/accounting/reconciliation/work-queue",
      beforeLoad: gates(requirePermission(Resource.BankReceiptWorkItem)),
      component: lazyPage(() => import("@/routes/bank-receipt-queue/page"), "BankReceiptQueuePage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/accounting/reconciliation/summary",
      beforeLoad: gates(requirePermission(Resource.BankReceipt)),
      component: lazyPage(
        () => import("@/routes/reconciliation-summary/page"),
        "ReconciliationSummaryPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/accounting/reconciliation/import-batches",
      beforeLoad: gates(requirePermission(Resource.BankReceipt)),
      component: lazyPage(() => import("@/routes/bank-receipt-batch/page"), "BankReceiptBatchPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/accounting/reconciliation/import-batches/$batchId",
      beforeLoad: gates(requirePermission(Resource.BankReceipt)),
      component: lazyPage(
        () => import("@/routes/bank-receipt-batch/detail-page"),
        "BankReceiptBatchDetailPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/accounting/sync",
      beforeLoad: gates(requirePermission(Resource.AccountingSync)),
      component: lazyPage(
        () => import("@/routes/accounting-sync/page"),
        "AccountingSyncLedgerPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/accounting/sync/inbound",
      beforeLoad: gates(requirePermission(Resource.AccountingSync)),
      component: lazyPage(
        () => import("@/routes/accounting-inbound/page"),
        "AccountingInboundPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/accounting/sync/drift",
      beforeLoad: gates(requirePermission(Resource.AccountingSync)),
      component: lazyPage(() => import("@/routes/accounting-drift/page"), "AccountingDriftPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/accounting/sync/mappings",
      beforeLoad: gates(requirePermission(Resource.AccountingIntegration)),
      component: lazyPage(
        () => import("@/routes/accounting-mapping/page"),
        "AccountingMappingsPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/accounting/configuration-files/account-types",
      beforeLoad: gates(requirePermission(Resource.AccountType)),
      component: lazyPage(() => import("@/routes/account-type/page"), "AccountTypesPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/billing/configuration-files/formula-templates",
      beforeLoad: gates(requirePermission(Resource.FormulaTemplate)),
      component: lazyPage(() => import("@/routes/formula-template/page"), "FormulaTemplatesPage"),
    }),
    // The studio is a room: the module rail stays, and its rate trace takes
    // the panel's place and always stands, the panel folded in its favour.
    createRoute({
      getParentRoute: () => appRoute,
      path: "/billing/configuration-files/formula-templates/new",
      beforeLoad: gates(requirePermission(Resource.FormulaTemplate, Operation.Create)),
      loader: prefetchPage(() => import("@/routes/formula-template/new/page")),
      staticData: { frame: "room", roomRail: "stands" },
      component: lazyPage(
        () => import("@/routes/formula-template/new/page"),
        "FormulaStudioCreatePage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/billing/configuration-files/formula-templates/$id/edit",
      beforeLoad: gates(requirePermission(Resource.FormulaTemplate)),
      loader: prefetchPage(() => import("@/routes/formula-template/[id]/page")),
      staticData: { frame: "room", roomRail: "stands" },
      component: lazyPage(
        () => import("@/routes/formula-template/[id]/page"),
        "FormulaStudioEditPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/billing/configuration-files/formula-libraries",
      beforeLoad: gates(requirePermission(Resource.FormulaLibrary)),
      component: lazyPage(() => import("@/routes/formula-library/page"), "FormulaLibrariesPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/billing/configuration-files/formula-libraries/new",
      beforeLoad: gates(requirePermission(Resource.FormulaLibrary, Operation.Create)),
      loader: prefetchPage(() => import("@/routes/formula-library/new/page")),
      staticData: { frame: "room", roomRail: "stands" },
      component: lazyPage(
        () => import("@/routes/formula-library/new/page"),
        "FormulaLibraryCreatePage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/billing/configuration-files/formula-libraries/$id/edit",
      beforeLoad: gates(requirePermission(Resource.FormulaLibrary)),
      loader: prefetchPage(() => import("@/routes/formula-library/[id]/page")),
      staticData: { frame: "room", roomRail: "stands" },
      component: lazyPage(
        () => import("@/routes/formula-library/[id]/page"),
        "FormulaLibraryEditPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/fuel/configuration-files/surcharge",
      beforeLoad: gates(requirePermission(Resource.FuelSurchargeProgram)),
      component: lazyPage(() => import("@/routes/fuel-management/page"), "FuelManagementPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/equipment/tractors",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.Tractor),
      ),
      component: lazyPage(() => import("@/routes/tractor/page"), "TractorsPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/equipment/trailers",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.Trailer),
      ),
      component: lazyPage(() => import("@/routes/trailer/page"), "TrailersPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/fuel/purchases",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.FuelPurchase),
      ),
      component: lazyPage(() => import("@/routes/fuel-purchase/page"), "FuelPurchasesPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/fuel/jurisdiction-mileage",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.IFTAJurisdictionMileage),
      ),
      component: lazyPage(
        () => import("@/routes/ifta-jurisdiction-mileage/page"),
        "JurisdictionMileagePage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/fuel/ifta-returns",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.IFTAReturn),
      ),
      loader: prefetchPage(() => import("@/routes/ifta-return/page")),
      component: lazyPage(() => import("@/routes/ifta-return/page"), "IftaReturnsPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/fuel/configuration-files/fuel-cards",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.FuelCard),
      ),
      component: lazyPage(() => import("@/routes/fuel-card/page"), "FuelCardsPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/fuel/feed-runs",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.FuelPurchaseImport),
      ),
      component: lazyPage(() => import("@/routes/fuel-feed-run/page"), "FuelFeedRunsPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/fuel/unassigned-cards",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.FuelCard),
      ),
      component: lazyPage(() => import("@/routes/fuel-card/page"), "UnassignedFuelCardsPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/fuel/configuration-files/ifta-tax-rates",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.IFTATaxRate),
      ),
      component: lazyPage(() => import("@/routes/ifta-tax-rate/page"), "IftaTaxRatesPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/equipment/configuration-files/equipment-types",
      beforeLoad: gates(requirePermission(Resource.EquipmentType)),
      component: lazyPage(() => import("@/routes/equipment-type/page"), "EquipmentTypesPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/equipment/configuration-files/equipment-manufacturers",
      beforeLoad: gates(requirePermission(Resource.EquipmentManufacturer)),
      component: lazyPage(
        () => import("@/routes/equipment-manufacturer/page"),
        "EquipmentManufacturersPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/dispatch/configuration-files/location-categories",
      beforeLoad: gates(requirePermission(Resource.LocationCategory)),
      component: lazyPage(
        () => import("@/routes/location-category/page"),
        "LocationCategoriesPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/dispatch/configuration-files/fleet-codes",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.FleetCode, Operation.Read),
      ),
      component: lazyPage(() => import("@/routes/fleet-code/page"), "FleetCodesPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/reports",
      beforeLoad: gates(requirePermission(Resource.Report, Operation.Read)),
      loader: prefetchPage(() => import("@/routes/reports/page")),
      component: lazyPage(() => import("@/routes/reports/page"), "ReportsPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/reports/runs",
      beforeLoad: gates(requirePermission(Resource.Report, Operation.Read)),
      component: lazyPage(() => import("@/routes/reports/runs/page"), "ReportRunsPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/reports/explore",
      beforeLoad: gates(requirePermission(Resource.Report, Operation.Read)),
      component: lazyPage(() => import("@/routes/reports/explore/page"), "ReportExplorePage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/reports/explore/$definitionId",
      beforeLoad: gates(requirePermission(Resource.Report, Operation.Read)),
      component: lazyPage(() => import("@/routes/reports/explore/page"), "ReportExplorePage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/reports/dashboards/$dashboardId",
      beforeLoad: gates(requirePermission(Resource.Report, Operation.Read)),
      component: lazyPage(() => import("@/routes/reports/dashboards/page"), "ReportDashboardPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/reports/builder",
      beforeLoad: gates(requirePermission(Resource.Report, Operation.Create)),
      component: lazyPage(() => import("@/routes/reports/builder/page"), "ReportBuilderPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/reports/builder/$definitionId",
      beforeLoad: gates(requirePermission(Resource.Report, Operation.Read)),
      component: lazyPage(() => import("@/routes/reports/builder/page"), "ReportBuilderPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/dispatch/locations",
      beforeLoad: gates(requirePermission(Resource.Location)),
      component: lazyPage(() => import("@/routes/location/page"), "LocationsPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/dispatch/console",
      beforeLoad: gates(requirePermission(Resource.ShipmentMove)),
      loader: prefetchPage(() => import("@/routes/dispatch-console/page")),
      component: lazyPage(() => import("@/routes/dispatch-console/page"), "DispatchConsolePage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/dispatch/outlook",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePlanFeature(PlanFeature.DispatchIntelligence),
        requirePermission(Resource.ShipmentMove),
      ),
      loader: prefetchPage(() => import("@/routes/freight-outlook/page")),
      component: lazyPage(() => import("@/routes/freight-outlook/page"), "FreightOutlookPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/hr/workers",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.Worker),
      ),
      loader: prefetchPage(() => import("@/routes/worker/page")),
      component: lazyPage(() => import("@/routes/worker/page"), "WorkersPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/hr/checklist-templates",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.WorkerChecklistTemplate),
      ),
      component: lazyPage(
        () => import("@/routes/worker-checklist-template/page"),
        "WorkerChecklistTemplatesPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/hr/review-templates",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.PerformanceReviewTemplate),
      ),
      component: lazyPage(() => import("@/routes/review-template/page"), "ReviewTemplatesPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/hr/training-courses",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.TrainingCourse),
      ),
      component: lazyPage(() => import("@/routes/training-course/page"), "TrainingCoursesPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/hr/credential-types",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.WorkerCredentialType),
      ),
      component: lazyPage(
        () => import("@/routes/worker-credential-type/page"),
        "WorkerCredentialTypesPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/hr/holidays",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.OrgHoliday),
      ),
      component: lazyPage(() => import("@/routes/holiday/page"), "HolidayCalendarPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/hr/leave-settings",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.WorkerLeave),
      ),
      component: lazyPage(() => import("@/routes/leave-control/page"), "LeaveControlPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/hr/my-team",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.Worker),
      ),
      loader: prefetchPage(() => import("@/routes/my-team/page")),
      component: lazyPage(() => import("@/routes/my-team/page"), "MyTeamPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/hr/benefits",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.BenefitPlan),
      ),
      loader: prefetchPage(() => import("@/routes/benefits/page")),
      component: lazyPage(() => import("@/routes/benefits/page"), "BenefitsPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/hr/policies",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.WorkerPolicy),
      ),
      component: lazyPage(() => import("@/routes/policies/page"), "PoliciesPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/hr/time-attendance",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.Timesheet),
      ),
      loader: prefetchPage(() => import("@/routes/time-attendance/page")),
      component: lazyPage(() => import("@/routes/time-attendance/page"), "TimeAttendancePage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/hr/scheduling",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.WorkerSchedule),
      ),
      loader: prefetchPage(() => import("@/routes/scheduling/page")),
      component: lazyPage(() => import("@/routes/scheduling/page"), "SchedulingPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/hr/org-structure",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.JobPosition),
      ),
      loader: prefetchPage(() => import("@/routes/org-structure/page")),
      component: lazyPage(() => import("@/routes/org-structure/page"), "OrgStructurePage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/hr/fleet-safety",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.WorkerSafetyEvent),
      ),
      loader: prefetchPage(() => import("@/routes/fleet-safety/page")),
      component: lazyPage(() => import("@/routes/fleet-safety/page"), "FleetSafetyPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/hr/osha-log",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.WorkerInjury),
      ),
      loader: prefetchPage(() => import("@/routes/osha/page")),
      component: lazyPage(() => import("@/routes/osha/page"), "OshaLogPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/hr/random-testing",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.DOTRandomPool),
      ),
      loader: prefetchPage(() => import("@/routes/random-testing/page")),
      component: lazyPage(() => import("@/routes/random-testing/page"), "RandomTestingPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/hr/pto-policies",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.AssetOperations),
        requirePermission(Resource.PTOPolicy),
      ),
      component: lazyPage(() => import("@/routes/pto-policy/page"), "PTOPoliciesPage"),
    }),
    // Moved to Human Resource Management. Links already sent — stored
    // notifications, bookmarks — keep working and carry their query on.
    createRoute({
      getParentRoute: () => appRoute,
      path: "/dispatch/workers",
      beforeLoad: gates(redirectKeepingSearch("/hr/workers")),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/dispatch/configuration-files/checklist-templates",
      beforeLoad: gates(redirectKeepingSearch("/hr/checklist-templates")),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/dispatch/configuration-files/review-templates",
      beforeLoad: gates(redirectKeepingSearch("/hr/review-templates")),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/equipment/fuel-purchases",
      beforeLoad: gates(redirectKeepingSearch("/fuel/purchases")),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/equipment/jurisdiction-mileage",
      beforeLoad: gates(redirectKeepingSearch("/fuel/jurisdiction-mileage")),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/equipment/ifta/returns",
      beforeLoad: gates(redirectKeepingSearch("/fuel/ifta-returns")),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/equipment/configuration-files/fuel-cards",
      beforeLoad: gates(redirectKeepingSearch("/fuel/configuration-files/fuel-cards")),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/equipment/configuration-files/ifta-tax-rates",
      beforeLoad: gates(redirectKeepingSearch("/fuel/configuration-files/ifta-tax-rates")),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/billing/fuel-management",
      beforeLoad: gates(redirectKeepingSearch("/fuel/configuration-files/surcharge")),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/dispatch/configuration-files/training-courses",
      beforeLoad: gates(redirectKeepingSearch("/hr/training-courses")),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/dispatch/configuration-files/credential-types",
      beforeLoad: gates(redirectKeepingSearch("/hr/credential-types")),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/dispatch/configuration-files/holidays",
      beforeLoad: gates(redirectKeepingSearch("/hr/holidays")),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/dispatch/configuration-files/pto-policies",
      beforeLoad: gates(redirectKeepingSearch("/hr/pto-policies")),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/dispatch/carriers",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.Brokerage),
        requirePermission(Resource.Carrier),
      ),
      component: lazyPage(() => import("@/routes/carrier/page"), "CarriersPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/dispatch/carrier-monitoring",
      beforeLoad: gates(
        requirePlanFeature(PlanFeature.CarrierBrokerage),
        requirePermission(Resource.CarrierIntelligence),
      ),
      component: lazyPage(
        () => import("@/routes/carrier-monitoring/page"),
        "CarrierMonitoringPage",
      ),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/dispatch/carrier-sourcing",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.Brokerage),
        requirePlanFeature(PlanFeature.CarrierBrokerage),
        requirePermission(Resource.CarrierSourcing),
      ),
      component: lazyPage(() => import("@/routes/carrier-sourcing/page"), "CarrierSourcingPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/dispatch/routing-guides",
      beforeLoad: gates(
        requireCapability(OrganizationCapability.Brokerage),
        requirePermission(Resource.RoutingGuide),
      ),
      component: lazyPage(() => import("@/routes/routing-guide/page"), "RoutingGuidesPage"),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: "/accounting/configuration-files/fiscal-years",
      beforeLoad: gates(requirePermission(Resource.FiscalYear)),
      component: lazyPage(() => import("@/routes/fiscal-year/page"), "FiscalYearsPage"),
    }),
    adminRoute.addChildren([
      // Settings at /admin draws only its frame; the section index fills it.
      createRoute({
        getParentRoute: () => adminRoute,
        path: "/",
        component: lazyPage(() => import("@/routes/section-index/page"), "SectionIndexPage"),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "billing-controls",
        beforeLoad: gates(requirePermission(Resource.BillingControl)),
        component: lazyPage(() => import("@/routes/billing-control/page"), "BillingControlPage"),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "organization-settings",
        beforeLoad: gates(requirePermission(Resource.Organization, Operation.Read)),
        loader: prefetchPage(() => import("@/routes/admin/organization-settings/page")),
        component: lazyPage(
          () => import("@/routes/admin/organization-settings/page"),
          "OrganizationSettingsPage",
        ),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "accounting-control",
        beforeLoad: gates(requirePermission(Resource.AccountingControl)),
        component: lazyPage(
          () => import("@/routes/accounting-control/page"),
          "AccountingControlPage",
        ),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "cost-control",
        beforeLoad: gates(requirePermission(Resource.CostingControl)),
        component: lazyPage(() => import("@/routes/cost-control/page"), "CostControlPage"),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "invoice-adjustment-controls",
        beforeLoad: gates(requirePermission(Resource.InvoiceAdjustmentControl, Operation.Read)),
        component: lazyPage(
          () => import("@/routes/invoice-adjustment-control/page"),
          "InvoiceAdjustmentControlPage",
        ),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "table-change-alerts",
        beforeLoad: gates(requirePermission(Resource.TableChangeAlert, Operation.Read)),
        component: lazyPage(
          () => import("@/routes/table-change-alert/page"),
          "TableChangeAlertPage",
        ),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "data-entry-controls",
        beforeLoad: gates(requirePermission(Resource.DataEntryControl, Operation.Read)),
        component: lazyPage(
          () => import("@/routes/data-entry-control/page"),
          "DataEntryControlPage",
        ),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "dispatch-controls",
        beforeLoad: gates(requirePermission(Resource.DispatchControl, Operation.Read)),
        component: lazyPage(() => import("@/routes/dispatch-control/page"), "DispatchControlPage"),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "settlement-control",
        beforeLoad: gates(
          requireCapability(OrganizationCapability.AssetOperations),
          requirePermission(Resource.SettlementControl, Operation.Read),
        ),
        component: lazyPage(
          () => import("@/routes/settlement-control/page"),
          "SettlementControlPage",
        ),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "carrier-settlement-control",
        beforeLoad: gates(
          requireCapability(OrganizationCapability.Brokerage),
          requirePermission(Resource.CarrierSettlementControl, Operation.Read),
        ),
        component: lazyPage(
          () => import("@/routes/carrier-settlement-control/page"),
          "CarrierSettlementControlPage",
        ),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "dash-control",
        beforeLoad: gates(
          requireCapability(OrganizationCapability.AssetOperations),
          requirePermission(Resource.DashControl, Operation.Read),
        ),
        component: lazyPage(() => import("@/routes/dash-control/page"), "DashControlPage"),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "agent-control",
        beforeLoad: gates(
          requirePlanFeature(PlanFeature.AgentAutomation),
          requirePermission(Resource.AgentControl, Operation.Read),
        ),
        component: lazyPage(() => import("@/routes/agent-control/page"), "AgentControlPage"),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "distance-controls",
        beforeLoad: gates(requirePermission(Resource.DistanceControl)),
        component: lazyPage(
          () => import("@/routes/admin/distance-controls/page"),
          "DistanceControlsPage",
        ),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "distance-overrides",
        beforeLoad: gates(requirePermission(Resource.DistanceOverride)),
        component: lazyPage(
          () => import("@/routes/distance-override/page"),
          "DistanceOverridesPage",
        ),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "distance-profiles",
        beforeLoad: gates(requirePermission(Resource.DistanceProfile)),
        component: lazyPage(
          () => import("@/routes/admin/distance-profiles/page"),
          "DistanceProfilesPage",
        ),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "stored-mileages",
        beforeLoad: gates(requirePermission(Resource.StoredMileage)),
        component: lazyPage(
          () => import("@/routes/admin/stored-mileages/page"),
          "StoredMileagesPage",
        ),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "shipment-controls",
        beforeLoad: gates(requirePermission(Resource.ShipmentControl, Operation.Read)),
        component: lazyPage(() => import("@/routes/shipment-control/page"), "ShipmentControlPage"),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "capture",
        beforeLoad: gates(
          requirePlanFeature(PlanFeature.DocumentIntelligence),
          requirePermission(Resource.CaptureProfile, Operation.Read),
        ),
        component: lazyPage(() => import("@/routes/admin/capture/page"), "CaptureAdminPage"),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "document-intelligence",
        beforeLoad: gates(
          requirePlanFeature(PlanFeature.DocumentIntelligence),
          requirePermission(Resource.DocumentControl, Operation.Read),
        ),
        component: lazyPage(
          () => import("@/routes/admin/document-intelligence/page"),
          "DocumentIntelligencePage",
        ),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "sequence-configs",
        beforeLoad: gates(requirePermission(Resource.SequenceConfig, Operation.Read)),
        component: lazyPage(
          () => import("@/routes/admin/sequence-config/page"),
          "SequenceConfigPage",
        ),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "hold-reasons",
        beforeLoad: gates(requirePermission(Resource.HoldReason)),
        component: lazyPage(() => import("@/routes/hold-reason/page"), "HoldReasonsPage"),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "jurisdiction-rules",
        beforeLoad: gates(requirePermission(Resource.JurisdictionRule)),
        component: lazyPage(
          () => import("@/routes/jurisdiction-rule/page"),
          "JurisdictionRulesPage",
        ),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "jurisdiction-rule-overrides",
        beforeLoad: gates(requirePermission(Resource.JurisdictionRuleOverride)),
        component: lazyPage(
          () => import("@/routes/jurisdiction-rule-override/page"),
          "JurisdictionRuleOverridesPage",
        ),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "service-failure-reason-codes",
        beforeLoad: gates(requirePermission(Resource.ServiceFailureReasonCode)),
        component: lazyPage(
          () => import("@/routes/service-failure-reason-code/page"),
          "ServiceFailureReasonCodesPage",
        ),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "hazmat-segregation-rules",
        beforeLoad: gates(requirePermission(Resource.HazmatSegregationRule)),
        component: lazyPage(
          () => import("@/routes/hazmat-segregation-rule/page"),
          "HazmatSegregationRulesPage",
        ),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "home-layouts",
        beforeLoad: gates(requirePermission(Resource.HomeLayoutPreset, Operation.Read)),
        component: lazyPage(() => import("@/routes/admin/home-layouts/page"), "HomeLayoutsPage"),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "home-layouts/new",
        beforeLoad: gates(requirePermission(Resource.HomeLayoutPreset, Operation.Create)),
        component: lazyPage(
          () => import("@/routes/admin/home-layouts/new/page"),
          "NewHomeLayoutPage",
        ),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "home-layouts/$id",
        beforeLoad: gates(requirePermission(Resource.HomeLayoutPreset, Operation.Update)),
        component: lazyPage(
          () => import("@/routes/admin/home-layouts/[id]/page"),
          "EditHomeLayoutPage",
        ),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "roles",
        beforeLoad: gates(requirePermission(Resource.Role, Operation.Read)),
        component: lazyPage(() => import("@/routes/admin/roles/page"), "RolesPage"),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "roles/new",
        beforeLoad: gates(requirePermission(Resource.Role, Operation.Create)),
        component: lazyPage(() => import("@/routes/admin/roles/new/page"), "RoleCreatePage"),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "roles/$id/edit",
        beforeLoad: gates(requirePermission(Resource.Role, Operation.Update)),
        component: lazyPage(() => import("@/routes/admin/roles/[id]/edit/page"), "RoleEditPage"),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "users",
        beforeLoad: gates(requirePermission(Resource.User, Operation.Read)),
        component: lazyPage(() => import("@/routes/admin/users/page"), "UsersPage"),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "audit-logs",
        beforeLoad: gates(requirePermission(Resource.AuditLog, Operation.Read)),
        loader: prefetchPage(() => import("@/routes/admin/audit-logs/page")),
        component: lazyPage(() => import("@/routes/admin/audit-logs/page"), "AuditLogsPage"),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "integrations",
        beforeLoad: gates(
          requirePlanFeature(PlanFeature.Administration),
          requirePermission(Resource.Integration, Operation.Read),
        ),
        component: lazyPage(() => import("@/routes/admin/integrations/page"), "IntegrationsPage"),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "integrations/quickbooks/callback",
        beforeLoad: gates(requirePermission(Resource.AccountingIntegration, Operation.Manage)),
        component: lazyPage(
          () => import("@/routes/admin/integrations/quickbooks-callback/page"),
          "QuickBooksCallbackPage",
        ),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "integrations/xero/callback",
        beforeLoad: gates(requirePermission(Resource.AccountingIntegration, Operation.Manage)),
        component: lazyPage(
          () => import("@/routes/admin/integrations/xero-callback/page"),
          "XeroCallbackPage",
        ),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "integrations/business-central/callback",
        beforeLoad: gates(requirePermission(Resource.AccountingIntegration, Operation.Manage)),
        component: lazyPage(
          () => import("@/routes/admin/integrations/business-central-callback/page"),
          "BusinessCentralCallbackPage",
        ),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "integrations/netsuite/callback",
        beforeLoad: gates(requirePermission(Resource.AccountingIntegration, Operation.Manage)),
        component: lazyPage(
          () => import("@/routes/admin/integrations/netsuite-callback/page"),
          "NetSuiteCallbackPage",
        ),
      }),
      // Agents are configured from Agent Control; the old standalone
      // page forwards so a saved link still lands somewhere useful.
      createRoute({
        getParentRoute: () => adminRoute,
        path: "agents",
        beforeLoad: gates(redirectTo("/admin/agent-control")),
      }),
      // Providers are a tab of AI Control now; saved links keep working.
      createRoute({
        getParentRoute: () => adminRoute,
        path: "ai-providers",
        beforeLoad: gates(redirectTo("/admin/agent-control?tab=providers")),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "inbound-mailboxes",
        beforeLoad: gates(requirePermission(Resource.InboundMailbox, Operation.Read)),
        loader: prefetchPage(() => import("@/routes/admin/inbound-mailboxes/page")),
        component: lazyPage(
          () => import("@/routes/admin/inbound-mailboxes/page"),
          "InboundMailboxesPage",
        ),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "api-keys",
        beforeLoad: gates(
          requirePlanFeature(PlanFeature.APIKeys),
          requirePermission(Resource.APIKey, Operation.Read),
        ),
        component: lazyPage(() => import("@/routes/admin/api-keys/page"), "APIKeysPage"),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "support-access",
        beforeLoad: gates(requirePermission(Resource.SupportAccess, Operation.Read)),
        loader: prefetchPage(() => import("@/routes/admin/support-access/page")),
        component: lazyPage(
          () => import("@/routes/admin/support-access/page"),
          "SupportAccessPage",
        ),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "plan",
        beforeLoad: gates(
          requireManagedInstance,
          requirePermission(Resource.Organization, Operation.Update),
        ),
        loader: prefetchPage(() => import("@/routes/admin/plan/page")),
        component: lazyPage(() => import("@/routes/admin/plan/page"), "PlanPage"),
      }),
      createRoute({
        getParentRoute: () => adminRoute,
        path: "custom-fields",
        beforeLoad: gates(requirePermission(Resource.CustomFieldDefinition, Operation.Read)),
        component: lazyPage(
          () => import("@/routes/admin/custom-fields/page"),
          "CustomFieldDefinitionsPage",
        ),
      }),
      ...editionRoutes(adminRoute, "admin"),
    ]),
    ...editionRoutes(appRoute, "protected"),
    createRoute({
      getParentRoute: () => appRoute,
      path: "$",
      component: lazyPage(() => import("@/routes/section-index/page"), "SectionIndexPage"),
    }),
  ]),
  onboardingRoute.addChildren([
    createRoute({
      getParentRoute: () => onboardingRoute,
      path: "/onboarding",
      beforeLoad: gates(onboardingGate),
      component: lazyPage(() => import("@/routes/onboarding/page"), "OnboardingPage"),
    }),
  ]),
  deskShellRoute.addChildren([
    deskRoute.addChildren([
      createRoute({
        getParentRoute: () => deskRoute,
        path: "/",
        component: lazyPage(() => import("@/routes/desk/home-page"), "DeskHomePage"),
      }),
      createRoute({
        getParentRoute: () => deskRoute,
        path: "t/$threadId",
        component: lazyPage(
          () => import("@/routes/desk/conversation-page"),
          "DeskConversationPage",
        ),
      }),
      // A link to one artifact, named by its slug: the conversation
      // opens with it beside it.
      createRoute({
        getParentRoute: () => deskRoute,
        path: "c/$threadId/a/$slug",
        component: lazyPage(
          () => import("@/routes/desk/artifact-link-page"),
          "DeskArtifactLinkPage",
        ),
      }),
      // What an agent laid out, on the canvas, with the conversation
      // that made it in the margin.
      createRoute({
        getParentRoute: () => deskRoute,
        path: "v/$viewThreadId/$slug",
        component: lazyPage(() => import("@/routes/desk/view-page"), "DeskViewPage"),
      }),
      createRoute({
        getParentRoute: () => deskRoute,
        path: "agents/$agentId",
        component: lazyPage(() => import("@/routes/desk/agent-page"), "DeskAgentCapabilitiesPage"),
      }),
      createRoute({
        getParentRoute: () => deskRoute,
        path: "handled/$id",
        beforeLoad: gates(requirePermission(Resource.AgentProposal, Operation.Read)),
        component: lazyPage(() => import("@/routes/desk/handled-page"), "DeskHandledPage"),
      }),
      createRoute({
        getParentRoute: () => deskRoute,
        path: "r/$kind/$id",
        component: lazyPage(() => import("@/routes/desk/case-page"), "DeskCasePage"),
      }),
      createRoute({
        getParentRoute: () => deskRoute,
        path: "triage",
        beforeLoad: gates(requirePermission(Resource.InboundMessage, Operation.Read)),
        loader: prefetchPage(() => import("@/routes/inbox/page")),
        component: lazyPage(() => import("@/routes/desk/triage-page"), "DeskTriagePage"),
      }),
      // Its Decisions lens is for whoever may read proposals; the
      // rest of the page is the Watchtower's.
      createRoute({
        getParentRoute: () => deskRoute,
        path: "needs",
        beforeLoad: gates(
          anyOfGates(
            requirePermission(Resource.Watchtower, Operation.Read),
            requirePermission(Resource.AgentProposal, Operation.Read),
          ),
        ),
        component: lazyPage(() => import("@/routes/desk/needs-page"), "DeskNeedsPage"),
      }),
      createRoute({
        getParentRoute: () => deskRoute,
        path: "watchtower",
        beforeLoad: gates(redirectTo("/desk/needs?lens=watchtower")),
      }),
      createRoute({
        getParentRoute: () => deskRoute,
        path: "scheduled",
        loader: prefetchPage(() => import("@/routes/desk/scheduled-page")),
        component: lazyPage(() => import("@/routes/desk/scheduled-page"), "DeskScheduledRoutePage"),
      }),
      createRoute({
        getParentRoute: () => deskRoute,
        path: "memory",
        loader: prefetchPage(() => import("@/routes/desk/memory-page")),
        component: lazyPage(() => import("@/routes/desk/memory-page"), "DeskMemoryRoutePage"),
      }),
    ]),
  ]),
  publicRoute.addChildren([
    createRoute({
      getParentRoute: () => publicRoute,
      path: "/tender-offer/$token",
      component: lazyPage(
        () => import("@/routes/tender-offer-public/page"),
        "TenderOfferPublicPage",
      ),
    }),
    createRoute({
      getParentRoute: () => publicRoute,
      path: "/tender-offer/$token/accept",
      component: lazyPage(
        () => import("@/routes/tender-offer-public/page"),
        "TenderOfferPublicPage",
      ),
    }),
    createRoute({
      getParentRoute: () => publicRoute,
      path: "/tender-offer/$token/decline",
      component: lazyPage(
        () => import("@/routes/tender-offer-public/page"),
        "TenderOfferPublicPage",
      ),
    }),
    createRoute({
      getParentRoute: () => publicRoute,
      path: "/rate-confirmation/$token",
      component: lazyPage(
        () => import("@/routes/rate-confirmation-public/page"),
        "RateConfirmationPublicPage",
      ),
    }),
    createRoute({
      getParentRoute: () => publicRoute,
      path: "/carrier-tracking/$token",
      component: lazyPage(
        () => import("@/routes/carrier-tracking-public/page"),
        "CarrierTrackingPublicPage",
      ),
    }),
    createRoute({
      getParentRoute: () => publicRoute,
      path: "/shipment-tracking/$token",
      component: lazyPage(
        () => import("@/routes/shipment-tracking-public/page"),
        "ShipmentTrackingPublicPage",
      ),
    }),
    ...editionRoutes(publicRoute, "public"),
  ]),
  guestRoute.addChildren([
    createRoute({
      getParentRoute: () => guestRoute,
      path: "/login",
      component: lazyPage(() => import("@/routes/auth/page"), "AuthPage"),
    }),
    createRoute({
      getParentRoute: () => guestRoute,
      path: "/login/$orgSlug",
      component: lazyPage(() => import("@/routes/auth/page"), "AuthPage"),
    }),
    // Opened from the emailed reset link. Guest-only, like the sign-in page:
    // somebody already signed in has no use for it.
    createRoute({
      getParentRoute: () => guestRoute,
      path: "/auth/reset",
      component: lazyPage(() => import("@/routes/auth/reset-page"), "ResetPasswordPage"),
    }),
    ...editionRoutes(guestRoute, "guest"),
  ]),
]);

export const router = createRouter({
  routeTree,
  parseSearch,
  stringifySearch,
  // A route without an error component of its own draws the failure where its page would
  // have been; inside the shell or the Desk the frame stays standing around it.
  defaultErrorComponent: RouteErrorBoundary,
  // Route loaders only warm TanStack Query, which decides freshness itself.
  defaultPreloadStaleTime: 0,
  InnerWrap: UrlStateAdapter,
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}
