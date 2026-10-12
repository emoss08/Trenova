import type { TranslateFn } from "@trenova/shared/i18n/use-t";

/** What a built-in checklist step is called, wherever it is shown. */
export function builtinStepLabel(key: string, t: TranslateFn): string {
  switch (key) {
    case "delivered":
      return t("Delivered");
    case "pod":
      return t("Proof of delivery received");
    case "paperwork":
      return t("Paperwork in");
    case "rateConfirmation":
      return t("Rate confirmation matched");
    case "carrierRateConfirmed":
      return t("Carrier confirmed the rate");
    case "accessorials":
      return t("Accessorials approved");
    case "customerNotified":
      return t("Customer notified");
    case "billingHolds":
      return t("Nothing holding the bill");
    case "posted":
      return t("Posted");
    case "sent":
      return t("Sent to the customer");
    case "dispute":
      return t("Clear of dispute");
    case "paid":
      return t("Paid");
    default:
      return key;
  }
}

/**
 * What a built-in step checks, in a line, and for a step whose setting is
 * kept elsewhere, where it is set.
 */
export function builtinStepRule(key: string, t: TranslateFn): string {
  switch (key) {
    case "delivered":
      return t("The shipment is completed. Nothing is billed before it.");
    case "pod":
      return t("Follows the customer's billing profile: required when it asks for a POD.");
    case "paperwork":
      return t("Follows the customer's billing profile and its required document types.");
    case "rateConfirmation":
      return t("Follows rate validation in billing control.");
    case "carrierRateConfirmed":
      return t("Every carrier on the shipment confirmed its rate confirmation.");
    case "accessorials":
      return t("No detention charge waits on approval or is disputed.");
    case "customerNotified":
      return t("The customer was emailed after delivery.");
    case "billingHolds":
      return t("No credit hold or unresolved service failure. Billing refuses these anyway.");
    case "posted":
      return t("The invoice is posted. Nothing is closed before it.");
    case "sent":
      return t("The invoice was sent to the customer.");
    case "dispute":
      return t("No dispute is open on the invoice.");
    case "paid":
      return t("The invoice is paid in full.");
    default:
      return "";
  }
}
