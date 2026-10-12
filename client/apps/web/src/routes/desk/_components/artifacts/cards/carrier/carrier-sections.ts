import { Resource } from "@trenova/shared/types/permission";

/**
 * The carrier card's sections built from records other than the carrier, by
 * the resource each reads. Someone who may read the carrier but not one of
 * these sees the card without that section. Its own module, so the skeleton
 * reads it without the card's code.
 */
export const CARRIER_SECTIONS = {
  eligibility: Resource.ShipmentMove,
  intelligence: Resource.CarrierIntelligence,
  shipments: Resource.Shipment,
  settlements: Resource.CarrierSettlement,
} as const;
