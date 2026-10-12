import { Resource } from "@trenova/shared/types/permission";

/**
 * The customer card's sections built from records other than the customer, by
 * the resource each reads. Someone who may read the customer but not one of
 * these sees the card without that section. Its own module, so the skeleton
 * reads it without the card's code.
 */
export const CUSTOMER_SECTIONS = {
  receivables: Resource.AccountsReceivable,
  shipments: Resource.Shipment,
  agreements: Resource.RateAgreement,
} as const;
