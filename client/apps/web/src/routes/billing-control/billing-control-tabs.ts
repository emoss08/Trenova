import type { SettingsTabSpec } from "@/components/settings/settings-tab";
import { useBillingQueueReviewOnPlan } from "@/lib/billing-queue-nova-plan";
import { PlanFeature } from "@/lib/plan-feature";
import type { BillingControl } from "@/types/billing-control";
import { usePlanFeature } from "@trenova/shared/hooks/use-plan-feature";
import { defineLabels } from "@trenova/shared/i18n/labels";

export type BillingControlTab = "invoices" | "automation" | "exceptions" | "rating" | "collections";

export type BillingControlField = keyof BillingControl & string;

export type BillingControlTabSpec = SettingsTabSpec<BillingControlField> & {
  value: BillingControlTab;
};

const TAB_LABELS = defineLabels({
  invoices: "Invoices",
  automation: "Automation",
  exceptions: "Exceptions",
  rating: "Rating",
  collections: "Collections",
});

/** The form fields each tab holds, so a refused save marks and opens the tab at fault. */
const TAB_FIELDS: Record<BillingControlTab, readonly BillingControlField[]> = {
  invoices: [
    "defaultPaymentTerm",
    "defaultInvoiceTerms",
    "showDueDateOnInvoice",
    "showBalanceDueOnInvoice",
    "defaultInvoiceFooter",
  ],
  automation: [
    "readyToBillAssignmentMode",
    "billingQueueTransferMode",
    "billingQueueTransferSchedule",
    "billingQueueTransferBatchSize",
    "invoiceDraftCreationMode",
    "autoInvoiceBatchSize",
    "notifyOnAutoInvoiceCreation",
    "invoicePostingMode",
    "queueAgentReviewEnabled",
    "queueAutoApproveMode",
    "queueAutoApproveMaxAmount",
  ],
  exceptions: [
    "shipmentBillingRequirementEnforcement",
    "rateValidationEnforcement",
    "rateVarianceAutoResolutionMode",
    "rateVarianceTolerancePercent",
    "billingExceptionDisposition",
    "notifyOnBillingExceptions",
  ],
  rating: [
    "unratedShipmentDisposition",
    "fallbackFormulaTemplateId",
    "requireRateOverrideReason",
    "enforceMarginFloor",
  ],
  collections: [
    "lateChargeAssessmentMode",
    "lateChargeMinimumAmount",
    "paymentRemindersEnabled",
    "paymentReminderDays",
    "monthlyStatementsEnabled",
    "monthlyStatementDay",
  ],
};

const TAB_ORDER: readonly BillingControlTab[] = [
  "invoices",
  "automation",
  "exceptions",
  "rating",
  "collections",
];

/**
 * The tabs of billing control, in order. Every organization bills, so every tab applies to
 * every organization; what a plan leaves out is a row inside a tab, not a tab.
 */
export function billingControlTabs(): BillingControlTabSpec[] {
  return TAB_ORDER.map((tab) => ({
    value: tab,
    label: TAB_LABELS[tab],
    fields: TAB_FIELDS[tab],
  }));
}

/** The tabs this organization sees, for the form and its loading state alike. */
export function useBillingControlTabs(): BillingControlTabSpec[] {
  return billingControlTabs();
}

/**
 * What the plan adds to the tabs' rows: Nova's review of queue items (and approval on its
 * word), and rating's formula templates and margin floors. The tabs and the loading state
 * both read it, so the outline has the form's rows.
 */
export type BillingControlPlan = { novaReviews: boolean; rating: boolean };

export function useBillingControlPlan(): BillingControlPlan {
  const novaReviews = useBillingQueueReviewOnPlan();
  const rating = usePlanFeature(PlanFeature.Rating).available;

  return { novaReviews, rating };
}
