import {
  DeskRecordFailuresDocument,
  type DeskRecordFailuresQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type RecordFailure = DeskRecordFailuresQuery["serviceFailures"]["edges"][number]["node"];

/** Whose service failures a card lists: those at a location's stops, or on one shipment. */
export type FailureScope = { kind: "location" | "shipment"; id: string };

/** One page of a record's service failures. `total` is read with the first page only. */
export type RecordFailuresPage = {
  rows: RecordFailure[];
  total: number | null;
  endCursor: string | null;
  hasNextPage: boolean;
};

/** A page of the service failures in the scope, newest first. */
export async function fetchRecordFailures(
  scope: FailureScope,
  page: { first: number; after: string | null },
  options?: { signal?: AbortSignal },
): Promise<RecordFailuresPage> {
  const firstPage = page.after === null;
  const data = await requestGraphQL({
    document: DeskRecordFailuresDocument,
    operationName: "DeskRecordFailures",
    variables: {
      locationId: scope.kind === "location" ? scope.id : null,
      shipmentId: scope.kind === "shipment" ? scope.id : null,
      input: {
        first: page.first,
        after: page.after,
        sort: [{ field: "detectedAt", direction: "desc" }],
      },
      includeTotalCount: firstPage,
    },
    signal: options?.signal,
  });
  const rows = data.serviceFailures.edges.map((edge) => edge.node);
  return {
    rows,
    total: firstPage ? (data.serviceFailures.totalCount ?? rows.length) : null,
    endCursor: data.serviceFailures.pageInfo.endCursor ?? null,
    hasNextPage: data.serviceFailures.pageInfo.hasNextPage,
  };
}
