import { Resource } from "@trenova/shared/types/permission";

/**
 * The location card's sections built from records other than the location, by
 * the resource each reads. Someone who may read the location but not one of
 * these sees the card without that section. Its own module, so the skeleton
 * reads it without the card's code.
 */
export const LOCATION_SECTIONS = {
  detention: Resource.Detention,
  shipments: Resource.Shipment,
  failures: Resource.ServiceFailure,
} as const;
