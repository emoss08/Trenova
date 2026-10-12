import { Resource } from "@trenova/shared/types/permission";

/**
 * The order card's sections built from records other than the order, by the
 * resource each reads. Someone who may read the order but not one of these
 * sees the card without that section. Its own module, so the skeleton reads it
 * without the card's code.
 */
export const ORDER_SECTIONS = {
  shipments: Resource.Shipment,
  activity: Resource.AuditLog,
} as const;
