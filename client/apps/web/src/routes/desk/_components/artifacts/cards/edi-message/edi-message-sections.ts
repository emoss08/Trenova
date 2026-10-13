import { Resource } from "@trenova/shared/types/permission";

/**
 * The EDI message card's parts built from records other than the message, by
 * the resource each reads: the load a message carries and the invoice it
 * bills. Someone who may read the message but not the load or invoice sees the
 * card without that row. Its own module, so the skeleton reads it without the
 * card's code.
 */
export const EDI_MESSAGE_SECTIONS = {
  shipment: Resource.Shipment,
  invoice: Resource.Invoice,
} as const;
