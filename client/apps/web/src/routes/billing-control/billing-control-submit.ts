import type { BillingControl } from "@/types/billing-control";

/**
 * The record as it is saved: a setting the page hides because another one makes it
 * meaningless is sent the way the server requires it then, so a save is never refused for a
 * field the person cannot see.
 */
export function prepareBillingControlForSubmit(values: BillingControl): BillingControl {
  const next = { ...values };

  if (next.invoiceDraftCreationMode === "ManualOnly") {
    next.autoInvoiceBatchSize = null;
    next.notifyOnAutoInvoiceCreation = false;
  }

  if (next.rateValidationEnforcement !== "RequireReview") {
    next.rateVarianceAutoResolutionMode = "Disabled";
  }

  if (next.rateValidationEnforcement === "Ignore") {
    next.rateVarianceTolerancePercent = 0;
  }

  return next;
}
