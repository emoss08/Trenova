import { Resource } from "@trenova/shared/types/permission";

/**
 * The invoice card's sections built from records other than the invoice, by
 * the resource each reads. Someone who may read the invoice but not one of
 * these sees the card without that section. Its own module, so the skeleton
 * reads it without the card's code.
 */
export const INVOICE_SECTIONS = {
  shipments: Resource.Shipment,
  books: Resource.AccountingSync,
  activity: Resource.AuditLog,
} as const;
