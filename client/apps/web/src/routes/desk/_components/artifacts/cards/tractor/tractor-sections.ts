import { Resource } from "@trenova/shared/types/permission";

/**
 * The tractor card's sections built from records other than the tractor, by
 * the resource each reads. Someone who may read the tractor but not one of
 * these sees the card without that section. Its own module, so the skeleton
 * reads it without the card's code.
 */
export const TRACTOR_SECTIONS = {
  shipments: Resource.Shipment,
} as const;
