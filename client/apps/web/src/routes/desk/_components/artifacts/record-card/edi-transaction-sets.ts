import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { ediTransactionSetSchema, type EDIMessage } from "@trenova/shared/types/edi";

type EDITransactionSet = EDIMessage["transactionSet"];

/** What a transaction set is, in the words a dispatcher uses for it. */
export function transactionName(set: EDITransactionSet, t: TranslateFn): string {
  switch (set) {
    case "204":
      return t("Load tender");
    case "210":
      return t("Freight invoice");
    case "214":
      return t("Shipment status");
    case "990":
      return t("Tender response");
    case "997":
      return t("Functional acknowledgment");
    case "999":
      return t("Implementation acknowledgment");
  }
}

/**
 * What a transaction set read off a partner's X12 is: one Trenova handles by
 * its name, any other by its number, since a partner may send sets Trenova
 * does not read.
 */
export function transactionSetName(code: string | null | undefined, t: TranslateFn): string {
  const set = ediTransactionSetSchema.safeParse(code);
  if (set.success) {
    return transactionName(set.data, t);
  }
  return code ? t("Transaction set {0}", code) : t("Unnamed transaction set");
}

/** What a functional group carries, by its GS01 functional identifier code. */
export function functionalGroupName(code: string | null | undefined, t: TranslateFn): string {
  switch (code) {
    case "SM":
      return t("Load tenders");
    case "IM":
      return t("Freight invoices");
    case "QM":
      return t("Shipment statuses");
    case "GF":
      return t("Tender responses");
    case "FA":
      return t("Acknowledgments");
    default:
      return code ? t("Group {0}", code) : t("Unnamed group");
  }
}
