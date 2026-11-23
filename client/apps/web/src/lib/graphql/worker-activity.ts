import type { FieldFilterInput, WorkerActivityQuery } from "@trenova/graphql/generated/graphql";
import { WorkerActivityDocument } from "@trenova/graphql/generated/documents/worker-activity/operations";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type WorkerActivityEntry =
  WorkerActivityQuery["auditEntriesBySubject"]["edges"][number]["node"];

export const WORKER_ACTIVITY_KEY = "worker-activity";

export type WorkerActivityPage = {
  rows: WorkerActivityEntry[];
  /** Read with the first page only. */
  total: number | null;
  endCursor: string | null;
  hasNextPage: boolean;
};

export type WorkerActivityFilter = {
  /** Words matched against the entry's record reference and comment. */
  query: string;
  fieldFilters: FieldFilterInput[];
};

/**
 * A page of a worker's whole file, newest first: every entry whose subject is the worker, its
 * sub-records' included. Reads are left out; the trail tells what changed, not who looked.
 */
export async function fetchWorkerActivity(
  workerId: string,
  page: { first: number; after: string | null },
  filter: WorkerActivityFilter,
  options?: { signal?: AbortSignal },
): Promise<WorkerActivityPage> {
  const firstPage = page.after === null;
  const data = await requestGraphQL({
    document: WorkerActivityDocument,
    operationName: "WorkerActivity",
    variables: {
      subjectId: workerId,
      input: {
        first: page.first,
        after: page.after,
        query: filter.query.trim() || undefined,
        fieldFilters: [
          { field: "operation", operator: "ne", value: "read" },
          ...filter.fieldFilters,
        ],
        sort: [{ field: "timestamp", direction: "desc" }],
      },
      includeTotalCount: firstPage,
    },
    signal: options?.signal,
  });
  const connection = data.auditEntriesBySubject;
  return {
    rows: connection.edges.map((edge) => edge.node),
    total: firstPage ? (connection.totalCount ?? null) : null,
    endCursor: connection.pageInfo.endCursor ?? null,
    hasNextPage: connection.pageInfo.hasNextPage,
  };
}
