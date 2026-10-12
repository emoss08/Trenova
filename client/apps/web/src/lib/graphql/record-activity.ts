import {
  DeskRecordActivityDocument,
  type DeskRecordActivityQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type ActivityEntry =
  DeskRecordActivityQuery["auditEntriesByResourceId"]["edges"][number]["node"];

/** One page of what was done to a record, newest first. `total` is read with the first page only. */
export type RecordActivityPage = {
  rows: ActivityEntry[];
  total: number | null;
  endCursor: string | null;
  hasNextPage: boolean;
};

/**
 * A page of what was done to one record, newest first. Reads are left out:
 * the card tells what changed, not who looked.
 */
export async function fetchRecordActivity(
  resourceId: string,
  page: { first: number; after: string | null },
  options?: { signal?: AbortSignal },
): Promise<RecordActivityPage> {
  const firstPage = page.after === null;
  const data = await requestGraphQL({
    document: DeskRecordActivityDocument,
    operationName: "DeskRecordActivity",
    variables: {
      resourceId,
      input: {
        first: page.first,
        after: page.after,
        fieldFilters: [{ field: "operation", operator: "ne", value: "read" }],
        sort: [{ field: "timestamp", direction: "desc" }],
      },
      includeTotalCount: firstPage,
    },
    signal: options?.signal,
  });
  const connection = data.auditEntriesByResourceId;
  const rows = connection.edges.map((edge) => edge.node);
  return {
    rows,
    total: firstPage ? (connection.totalCount ?? rows.length) : null,
    endCursor: connection.pageInfo.endCursor ?? null,
    hasNextPage: connection.pageInfo.hasNextPage,
  };
}
