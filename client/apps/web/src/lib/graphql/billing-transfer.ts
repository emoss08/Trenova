import {
  BillingTransferCandidateIdsDocument,
  BillingTransferCandidatesDocument,
  BulkTransferShipmentsToBillingDocument,
  type BillingTransferCandidateIdsQuery,
  type BillingTransferCandidatesQuery,
  type BulkTransferShipmentsToBillingMutation,
  type ShipmentBillingTransferFailureCode,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type BillingTransferCandidateStatus = "Completed" | "ReadyToInvoice";

export type BillingTransferCandidateFilters = {
  query: string;
  status: BillingTransferCandidateStatus | null;
};

export type BillingTransferCandidateConnection = BillingTransferCandidatesQuery["shipments"];
export type BillingTransferCandidate = BillingTransferCandidateConnection["edges"][number]["node"];
export type BillingTransferCandidateIds =
  BillingTransferCandidateIdsQuery["shipmentBillingTransferCandidateIds"];
export type BulkBillingTransferResponse =
  BulkTransferShipmentsToBillingMutation["bulkTransferShipmentsToBilling"];
export type BulkBillingTransferResult = BulkBillingTransferResponse["results"][number];
export type BillingTransferFailureCode = ShipmentBillingTransferFailureCode;

type RequestOptions = { signal?: AbortSignal };

function searchValue(query: string): string | undefined {
  const trimmed = query.trim();
  return trimmed === "" ? undefined : trimmed;
}

export async function listBillingTransferCandidatesGraphQL(
  req: BillingTransferCandidateFilters & { first: number; after?: string | null },
  options?: RequestOptions,
): Promise<BillingTransferCandidateConnection> {
  const data = await requestGraphQL({
    document: BillingTransferCandidatesDocument,
    operationName: "BillingTransferCandidates",
    variables: {
      input: {
        first: req.first,
        after: req.after ?? undefined,
        query: searchValue(req.query),
        status: req.status ?? undefined,
        billingTransferEligible: true,
        includeCustomer: true,
        expandShipmentDetails: false,
      },
      includeTotalCount: !req.after,
    },
    signal: options?.signal,
  });
  return data.shipments;
}

export async function listBillingTransferCandidateIdsGraphQL(
  req: BillingTransferCandidateFilters,
  options?: RequestOptions,
): Promise<BillingTransferCandidateIds> {
  const data = await requestGraphQL({
    document: BillingTransferCandidateIdsDocument,
    operationName: "BillingTransferCandidateIds",
    variables: {
      input: {
        query: searchValue(req.query),
        status: req.status ?? undefined,
      },
    },
    signal: options?.signal,
  });
  return data.shipmentBillingTransferCandidateIds;
}

export async function bulkTransferShipmentsToBillingGraphQL(
  shipmentIds: string[],
): Promise<BulkBillingTransferResponse> {
  const data = await requestGraphQL({
    document: BulkTransferShipmentsToBillingDocument,
    operationName: "BulkTransferShipmentsToBilling",
    variables: {
      input: {
        shipmentIds,
        billType: "Invoice",
        markCompletedReadyToInvoice: true,
      },
    },
  });
  return data.bulkTransferShipmentsToBilling;
}
