import {
  BillingTransferCandidateIdsDocument,
  BillingTransferCandidatesDocument,
  BillingTransferRunDocument,
  BillingTransferRunItemsDocument,
  BulkTransferShipmentsToBillingDocument,
  CancelBillingTransferRunDocument,
  MyActiveBillingTransferRunDocument,
  RetryBillingTransferRunDocument,
  StartBillingTransferRunDocument,
  type BillingTransferCandidateIdsQuery,
  type BillingTransferCandidatesQuery,
  type BillingTransferRunItemsQuery,
  type BillingTransferRunQuery,
  type BillingTransferRunScope,
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

export type BillingTransferRun = BillingTransferRunQuery["billingTransferRun"];
export type BillingTransferRunItemConnection =
  BillingTransferRunItemsQuery["billingTransferRunItems"];
export type BillingTransferRunItem = BillingTransferRunItemConnection["edges"][number]["node"];
export type BillingTransferRunStatus = BillingTransferRun["status"];
export type BillingTransferItemStatus = BillingTransferRunItem["status"];

export type StartBillingTransferRunVariables = {
  scope: BillingTransferRunScope;
  shipmentIds?: string[];
  query?: string;
  status?: BillingTransferCandidateStatus | null;
};

export async function startBillingTransferRunGraphQL(
  req: StartBillingTransferRunVariables,
  options?: RequestOptions,
): Promise<BillingTransferRun> {
  const data = await requestGraphQL({
    document: StartBillingTransferRunDocument,
    operationName: "StartBillingTransferRun",
    variables: {
      input: {
        scope: req.scope,
        shipmentIds: req.shipmentIds,
        query: searchValue(req.query ?? ""),
        status: req.status ?? undefined,
        billType: "Invoice",
        markCompletedReadyToInvoice: true,
      },
    },
    signal: options?.signal,
  });
  return data.startBillingTransferRun;
}

export async function getBillingTransferRunGraphQL(
  id: string,
  options?: RequestOptions,
): Promise<BillingTransferRun> {
  const data = await requestGraphQL({
    document: BillingTransferRunDocument,
    operationName: "BillingTransferRun",
    variables: { id },
    signal: options?.signal,
  });
  return data.billingTransferRun;
}

export async function getMyActiveBillingTransferRunGraphQL(
  options?: RequestOptions,
): Promise<BillingTransferRun | null> {
  const data = await requestGraphQL({
    document: MyActiveBillingTransferRunDocument,
    operationName: "MyActiveBillingTransferRun",
    variables: {},
    signal: options?.signal,
  });
  return data.myActiveBillingTransferRun;
}

export async function cancelBillingTransferRunGraphQL(
  id: string,
  options?: RequestOptions,
): Promise<BillingTransferRun> {
  const data = await requestGraphQL({
    document: CancelBillingTransferRunDocument,
    operationName: "CancelBillingTransferRun",
    variables: { id },
    signal: options?.signal,
  });
  return data.cancelBillingTransferRun;
}

export async function retryBillingTransferRunGraphQL(
  id: string,
  options?: RequestOptions,
): Promise<BillingTransferRun> {
  const data = await requestGraphQL({
    document: RetryBillingTransferRunDocument,
    operationName: "RetryBillingTransferRun",
    variables: { id },
    signal: options?.signal,
  });
  return data.retryBillingTransferRun;
}

export async function listBillingTransferRunItemsGraphQL(
  req: {
    runId: string;
    first: number;
    after?: string | null;
    statuses?: BillingTransferItemStatus[];
  },
  options?: RequestOptions,
): Promise<BillingTransferRunItemConnection> {
  const data = await requestGraphQL({
    document: BillingTransferRunItemsDocument,
    operationName: "BillingTransferRunItems",
    variables: {
      runId: req.runId,
      input: { first: req.first, after: req.after ?? undefined },
      filter: req.statuses?.length ? { statuses: req.statuses } : undefined,
      includeTotalCount: !req.after,
    },
    signal: options?.signal,
  });
  return data.billingTransferRunItems;
}
