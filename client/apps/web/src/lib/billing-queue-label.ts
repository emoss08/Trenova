import type { BillingQueueItem } from "@trenova/shared/types/billing-queue";

/** What a person calls a queue item: its billing number, else its shipment's PRO. */
export function billingQueueItemLabel(item: BillingQueueItem): string {
  return item.number || item.shipment?.proNumber || item.shipmentId.slice(0, 12);
}
