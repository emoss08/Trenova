import type { FieldFilterInput, WorkerActivityQuery } from "@trenova/graphql/generated/graphql";
import { WorkerActivityDocument } from "@trenova/graphql/generated/documents/worker-activity/operations";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { Connection } from "./connection";

export type WorkerActivityEntry =
  WorkerActivityQuery["auditEntriesBySubject"]["edges"][number]["node"];

export const WORKER_ACTIVITY_KEY = "worker-activity";

export type WorkerActivityFilter = {
  /** Words matched against the entry's record reference and comment. */
  query: string;
  fieldFilters: FieldFilterInput[];
};

/**
 * A page of a worker's whole file, newest first: every entry whose subject is the worker, its
 * sub-records' included. Reads are left out; the trail tells what changed, not who looked. The
 * total is counted with the first page only.
 */
export async function fetchWorkerActivity(
  workerId: string,
  page: { first: number; after: string | null },
  filter: WorkerActivityFilter,
  options?: { signal?: AbortSignal },
): Promise<Connection<WorkerActivityEntry>> {
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
      includeTotalCount: page.after === null,
    },
    signal: options?.signal,
  });
  return data.auditEntriesBySubject;
}
