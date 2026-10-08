import { defineLabels, translateLabel } from "@trenova/shared/i18n/labels";
import type { StopType } from "@trenova/shared/types/shipment";

export const STOP_TYPE_LABELS: Record<StopType, string> = defineLabels({
  Pickup: "Pickup",
  Delivery: "Delivery",
  SplitPickup: "Split Pickup",
  SplitDelivery: "Split Delivery",
});

export function stopTypeLabel(type: string): string {
  return Object.hasOwn(STOP_TYPE_LABELS, type)
    ? STOP_TYPE_LABELS[type as StopType]
    : translateLabel(type);
}
