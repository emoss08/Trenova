import { Resource } from "@trenova/shared/types/permission";

/**
 * The bank receipt card's sections built from records other than the receipt,
 * by the resource each reads: the payment it was matched to, and its entry in
 * the reconciliation queue. The server leaves out what the reader may not read;
 * the card and its skeleton leave out the same. Its own module, so the skeleton
 * reads it without the card's code.
 */
export const BANK_RECEIPT_SECTIONS = {
  payment: Resource.CustomerPayment,
  invoices: Resource.Invoice,
  workItem: Resource.BankReceiptWorkItem,
} as const;
