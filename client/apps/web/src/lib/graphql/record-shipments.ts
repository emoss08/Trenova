import {
  DeskRecordShipmentsDocument,
  type DeskRecordShipmentsQuery,
  type ShipmentsInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type RecordShipment = DeskRecordShipmentsQuery["shipments"]["edges"][number]["node"];

/**
 * The record a card lists shipments for: a customer's freight, the loads a
 * tractor pulled, a trailer carried or a worker drove on any of their moves, or
 * the loads with a stop at a location.
 */
export type ShipmentScope =
  | { kind: "customer"; id: string }
  | { kind: "tractor"; id: string }
  | { kind: "trailer"; id: string }
  | { kind: "worker"; id: string }
  | { kind: "location"; id: string };

/** One page of a record's shipments, newest first. `total` is read with the first page only. */
export type RecordShipmentsPage = {
  rows: RecordShipment[];
  total: number | null;
  endCursor: string | null;
  hasNextPage: boolean;
};

function scopeInput(
  scope: ShipmentScope,
): Pick<
  ShipmentsInput,
  "fieldFilters" | "tractorIds" | "trailerIds" | "workerIds" | "stopLocationIds"
> {
  switch (scope.kind) {
    case "customer":
      return { fieldFilters: [{ field: "customerId", operator: "eq", value: scope.id }] };
    case "tractor":
      return { tractorIds: [scope.id] };
    case "trailer":
      return { trailerIds: [scope.id] };
    case "worker":
      return { workerIds: [scope.id] };
    case "location":
      return { stopLocationIds: [scope.id] };
  }
}

/** A page of the shipments a record lists, newest first, each with its stops for the lane. */
export async function fetchRecordShipments(
  scope: ShipmentScope,
  page: { first: number; after: string | null },
  options?: { signal?: AbortSignal },
): Promise<RecordShipmentsPage> {
  const firstPage = page.after === null;
  const data = await requestGraphQL({
    document: DeskRecordShipmentsDocument,
    operationName: "DeskRecordShipments",
    variables: {
      input: {
        first: page.first,
        after: page.after,
        expandShipmentDetails: true,
        sort: [{ field: "createdAt", direction: "desc" }],
        ...scopeInput(scope),
      },
      includeTotalCount: firstPage,
    },
    signal: options?.signal,
  });
  const rows = (data.shipments.edges ?? []).map((edge) => edge.node);
  return {
    rows,
    total: firstPage ? (data.shipments.totalCount ?? rows.length) : null,
    endCursor: data.shipments.pageInfo.endCursor ?? null,
    hasNextPage: data.shipments.pageInfo.hasNextPage,
  };
}
