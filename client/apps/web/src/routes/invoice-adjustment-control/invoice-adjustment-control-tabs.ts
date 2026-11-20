import type { SettingsTabSpec } from "@/components/settings/settings-tab";
import type { InvoiceAdjustmentControl } from "@/types/invoice-adjustment-control";
import { defineLabels } from "@trenova/shared/i18n/labels";

export type InvoiceAdjustmentControlTab = "eligibility" | "approval" | "credits";

export type InvoiceAdjustmentControlField = keyof InvoiceAdjustmentControl & string;

export type InvoiceAdjustmentControlTabSpec = SettingsTabSpec<InvoiceAdjustmentControlField> & {
  value: InvoiceAdjustmentControlTab;
};

const TAB_LABELS = defineLabels({
  eligibility: "Eligibility",
  approval: "Approval",
  credits: "Credits & rebills",
});

/** The form fields each tab holds, so a refused save marks and opens the tab at fault. */
const TAB_FIELDS: Record<InvoiceAdjustmentControlTab, readonly InvoiceAdjustmentControlField[]> = {
  eligibility: [
    "partiallyPaidInvoiceAdjustmentPolicy",
    "paidInvoiceAdjustmentPolicy",
    "disputedInvoiceAdjustmentPolicy",
    "adjustmentAccountingDatePolicy",
    "closedPeriodAdjustmentPolicy",
  ],
  approval: [
    "adjustmentReasonRequirement",
    "standardAdjustmentApprovalPolicy",
    "standardAdjustmentApprovalThreshold",
    "writeOffApprovalPolicy",
    "writeOffApprovalThreshold",
    "requireSeparateApprover",
  ],
  credits: [
    "customerCreditBalancePolicy",
    "overCreditPolicy",
    "replacementInvoiceReviewPolicy",
    "rerateVarianceTolerancePercent",
    "supersededInvoiceVisibilityPolicy",
  ],
};

const TAB_ORDER: readonly InvoiceAdjustmentControlTab[] = ["eligibility", "approval", "credits"];

/**
 * The tabs of invoice adjustment control, in order: which invoices can be adjusted and which
 * period an adjustment books in, who signs off on adjustments and write-offs, and how credits
 * and replacement invoices land. Every organization that bills uses all three, so none is
 * hidden.
 */
export function invoiceAdjustmentControlTabs(): InvoiceAdjustmentControlTabSpec[] {
  return TAB_ORDER.map((tab) => ({
    value: tab,
    label: TAB_LABELS[tab],
    fields: TAB_FIELDS[tab],
  }));
}

/** The tabs the form and its loading state both draw. */
export function useInvoiceAdjustmentControlTabs(): InvoiceAdjustmentControlTabSpec[] {
  return invoiceAdjustmentControlTabs();
}
