import { defineLabels } from "@trenova/shared/i18n/labels";

/** The kinds document intelligence classifies a file as, in the reader's language. */
export const DOCUMENT_KIND_LABELS = defineLabels({
  RateConfirmation: "Rate confirmation",
  BillOfLading: "Bill of lading",
  ProofOfDelivery: "Proof of delivery",
  Invoice: "Invoice",
  LumperReceipt: "Lumper receipt",
  ScaleTicket: "Scale ticket",
  DeliveryReceipt: "Delivery receipt",
});

/** A kind's label, or nothing for a kind not in the list. */
export function documentKindName(kind: string): string {
  return Object.hasOwn(DOCUMENT_KIND_LABELS, kind)
    ? DOCUMENT_KIND_LABELS[kind as keyof typeof DOCUMENT_KIND_LABELS]
    : "";
}
