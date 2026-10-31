type LabeledItem = {
  id: string;
  number?: string | null;
  shipmentId?: string | null;
  shipment?: { proNumber?: string | null } | null;
};

/** What a person calls a queue item: its billing number, else its shipment's PRO. */
export function billingQueueItemLabel(item: LabeledItem): string {
  return item.number || item.shipment?.proNumber || (item.shipmentId ?? item.id).slice(0, 12);
}
