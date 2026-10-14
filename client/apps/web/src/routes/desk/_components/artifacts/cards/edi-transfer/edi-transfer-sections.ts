import { Resource } from "@trenova/shared/types/permission";

/**
 * The EDI transfer card's parts built from records other than the tender, by
 * the resource each reads: the shipment it offers or built. Someone who may
 * read the tender but not shipments sees the card without that row. Its own
 * module, so the skeleton reads it without the card's code.
 */
export const EDI_TRANSFER_SECTIONS = {
  shipment: Resource.Shipment,
} as const;
