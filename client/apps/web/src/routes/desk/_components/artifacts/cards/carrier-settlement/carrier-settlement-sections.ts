import { Resource } from "@trenova/shared/types/permission";

/**
 * The carrier settlement card's sections built from records other than the
 * settlement, by the resource each reads. Someone who may read the settlement
 * but not one of these sees the card without that section. Its own module, so
 * the skeleton reads it without the card's code.
 */
export const CARRIER_SETTLEMENT_SECTIONS = {
  shipments: Resource.Shipment,
  carrier: Resource.Carrier,
  activity: Resource.AuditLog,
} as const;
