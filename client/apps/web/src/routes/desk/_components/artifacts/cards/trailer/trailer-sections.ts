import { Resource } from "@trenova/shared/types/permission";

/**
 * The trailer card's sections built from records other than the trailer, by
 * the resource each reads. Someone who may read the trailer but not one of
 * these sees the card without that section. Its own module, so the skeleton
 * reads it without the card's code.
 */
export const TRAILER_SECTIONS = {
  shipments: Resource.Shipment,
} as const;
