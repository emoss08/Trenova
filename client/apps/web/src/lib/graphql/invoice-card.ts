import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  DeskInvoiceCardDocument,
  type DeskInvoiceCardQuery,
  DeskInvoiceNeighborFragmentDoc,
  type DeskInvoiceNeighborFragment,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

type DeskInvoiceResponse = NonNullable<DeskInvoiceCardQuery["invoice"]>;

export type DeskInvoiceNeighbor = DeskInvoiceNeighborFragment;

export type DeskInvoice = Omit<DeskInvoiceResponse, "referenceInvoice" | "relatedInvoices"> & {
  /** The invoice a memo corrects, when it corrects one. */
  referenceInvoice: DeskInvoiceNeighbor | null;
  /** Other invoices that bill any shipment on this one, split-bill siblings first. */
  relatedInvoices: DeskInvoiceNeighbor[];
};
export type DeskInvoiceChargeGroup = DeskInvoice["chargeSummary"][number];
export type DeskInvoicePayment = DeskInvoice["paymentApplications"][number];
export type DeskInvoiceCredit = DeskInvoice["creditApplications"][number];
export type DeskInvoiceDispute = DeskInvoice["disputes"][number];

/**
 * One invoice as the Desk's card reads it: its charges summed by how each was
 * billed rather than line by line, what settled it, its disputes and where it
 * went. An invoice the server does not return is an error rather than an empty
 * card, so the card says it could not be read.
 */
export async function fetchDeskInvoice(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<DeskInvoice> {
  const data = await requestGraphQL({
    document: DeskInvoiceCardDocument,
    operationName: "DeskInvoiceCard",
    variables: { id },
    signal: options?.signal,
  });
  if (!data.invoice) {
    throw new Error("Invoice not found");
  }
  const { referenceInvoice, relatedInvoices, ...invoice } = data.invoice;
  return {
    ...invoice,
    referenceInvoice: getFragmentData(DeskInvoiceNeighborFragmentDoc, referenceInvoice),
    relatedInvoices: relatedInvoices.map((related) =>
      getFragmentData(DeskInvoiceNeighborFragmentDoc, related),
    ),
  };
}
