import type {
  UpcomingWorkerPtoQuery,
  UpcomingWorkerPtoQueryVariables,
  WorkerPtoChartDataQuery,
  WorkerPtoChartDataQueryVariables,
} from "@trenova/graphql/generated/graphql";
import {
  UpcomingWorkerPtoDocument,
  WorkerPtoChartDataDocument,
} from "@trenova/graphql/generated/documents/worker/page";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { GenericLimitOffsetResponse } from "@trenova/shared/types/server";
import type {
  ListUpcomingPTORequest,
  PTOChartDataRequest,
  PTOChartDataPoint,
  WorkerPTO,
} from "@trenova/shared/types/worker";
import { createQueryKeys } from "@lukemorales/query-key-factory";
import {
  fetchDeskWorker,
  fetchWorkerCredentials,
  fetchWorkerHos,
  fetchWorkerStanding,
} from "@/lib/graphql/worker-card";
import { fetchWorkerCredentialTypeChoices } from "@/lib/graphql/worker-credential";
import { fetchWorkerOverview } from "@/lib/graphql/worker-overview";
import { fetchWorkerRecord } from "@/lib/graphql/worker-record";

type WorkerPTOConnection = {
  edges?: Array<{ node: unknown }>;
  pageInfo?: {
    hasNextPage?: boolean;
    endCursor?: string | null;
  };
  totalCount?: number | null;
  totalCountExact?: boolean | null;
};

function workerPTOConnectionToLimitOffset(
  connection: WorkerPTOConnection,
): GenericLimitOffsetResponse<WorkerPTO> {
  const results = (connection.edges ?? []).map((edge) => edge.node as WorkerPTO);

  return {
    results,
    count: connection.totalCount ?? results.length,
    totalCountExact: connection.totalCount == null || connection.totalCountExact !== false,
    next: connection.pageInfo?.hasNextPage ? (connection.pageInfo.endCursor ?? null) : null,
    prev: null,
  };
}

export async function fetchUpcomingWorkerPTO(
  req: ListUpcomingPTORequest,
  options?: { signal?: AbortSignal },
): Promise<GenericLimitOffsetResponse<WorkerPTO>> {
  const data = await requestGraphQL<UpcomingWorkerPtoQuery, UpcomingWorkerPtoQueryVariables>({
    document: UpcomingWorkerPtoDocument,
    operationName: "UpcomingWorkerPto",
    variables: {
      input: {
        first: req.filter.limit,
        after: req.filter.after,
        status: req.status,
        type: req.type,
        startDate: req.startDate,
        endDate: req.endDate,
        workerId: req.workerId,
        fleetCodeId: req.fleetCodeId,
        timezone: req.timezone,
      },
    },
    signal: options?.signal,
  });

  return workerPTOConnectionToLimitOffset(data.upcomingWorkerPTO as WorkerPTOConnection);
}

export const worker = createQueryKeys("worker", {
  record: (workerId: string) => ({
    queryKey: ["record", workerId],
    queryFn: ({ signal }) => fetchWorkerRecord(workerId, { signal }),
  }),
  overview: (workerId: string) => ({
    queryKey: ["overview", workerId],
    queryFn: ({ signal }) => fetchWorkerOverview(workerId, { signal }),
  }),
  card: (workerId: string) => ({
    queryKey: ["card", workerId],
    queryFn: ({ signal }) => fetchDeskWorker(workerId, { signal }),
  }),
  hos: (workerId: string, violations: number) => ({
    queryKey: ["hos", workerId, violations],
    queryFn: ({ signal }) => fetchWorkerHos(workerId, violations, { signal }),
  }),
  standing: (workerId: string) => ({
    queryKey: ["standing", workerId],
    queryFn: ({ signal }) => fetchWorkerStanding(workerId, { signal }),
  }),
  credentials: (workerId: string) => ({
    queryKey: ["credentials", workerId],
    queryFn: ({ signal }) => fetchWorkerCredentials(workerId, { signal }),
  }),
  credentialTypeChoices: () => ({
    queryKey: ["credential-type-choices"],
    queryFn: ({ signal }) => fetchWorkerCredentialTypeChoices({ signal }),
  }),
  listUpcomingPTO: (req: ListUpcomingPTORequest) => ({
    queryKey: ["list-upcoming-pto", req],
    queryFn: ({ signal }) => fetchUpcomingWorkerPTO(req, { signal }),
  }),
  ptoChartData: (req: PTOChartDataRequest) => ({
    queryKey: ["pto-chart-data", req],
    queryFn: async ({ signal }) => {
      const data = await requestGraphQL<WorkerPtoChartDataQuery, WorkerPtoChartDataQueryVariables>({
        document: WorkerPtoChartDataDocument,
        operationName: "WorkerPtoChartData",
        variables: { input: req },
        signal,
      });

      return data.workerPTOChartData as PTOChartDataPoint[];
    },
  }),
});

/**
 * The worker page's two reads as its components observe them. The route loader is what keeps
 * them fresh on arrival (it fetches anything older than a few seconds), every write refreshes what
 * it changed (`useWorkerInvalidation`), realtime events invalidate them, and the header refreshes
 * on request; so a component that mounts later (a tab opened or shown again, a pane) reads what
 * is there instead of fetching it again. Prefetch with the plain factory entries.
 */
export function workerRecordQuery(workerId: string) {
  return { ...worker.record(workerId), refetchOnMount: false } as const;
}

/** The overview as the page's components observe it; see `workerRecordQuery`. */
export function workerOverviewQuery(workerId: string) {
  return { ...worker.overview(workerId), refetchOnMount: false } as const;
}
