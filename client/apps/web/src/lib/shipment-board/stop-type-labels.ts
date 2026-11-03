import { defineLabels } from "@trenova/shared/i18n/labels";

/** A stop's type as the board's views name it. */
export const STOP_TYPE_LABELS = defineLabels({
  Pickup: "Pickup",
  Delivery: "Delivery",
  SplitPickup: "Split pickup",
  SplitDelivery: "Split delivery",
});
