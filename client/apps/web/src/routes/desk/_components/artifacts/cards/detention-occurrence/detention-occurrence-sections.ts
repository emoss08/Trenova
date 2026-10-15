import { Resource } from "@trenova/shared/types/permission";

/**
 * The detention card's sections built from records other than the charge, by
 * the resource each reads. Someone who may read the charge but not one of
 * these sees the card without that section. Its own module, so the skeleton
 * reads it without the card's code.
 */
export const DETENTION_OCCURRENCE_SECTIONS = {
  shipments: Resource.Shipment,
  location: Resource.Location,
  customer: Resource.Customer,
} as const;
