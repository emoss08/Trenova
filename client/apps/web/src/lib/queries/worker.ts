import {
  UpcomingWorkerPtoDocument,
  WorkerPtoChartDataDocument,
  WorkerPtoTableDocument,
  type UpcomingWorkerPtoQuery,
  type UpcomingWorkerPtoQueryVariables,
  type WorkerPtoChartDataQuery,
  type WorkerPtoChartDataQueryVariables,
  type WorkerPtoTableQuery,
  type WorkerPtoTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
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

export const WORKER_PTO_HISTORY_PAGE_SIZE = 100;

export async function fetchWorkerPTOHistory(
  workerId: string,
  options?: { signal?: AbortSignal },
): Promise<WorkerPTO[]> {
  const data = await requestGraphQL<WorkerPtoTableQuery, WorkerPtoTableQueryVariables>({
    document: WorkerPtoTableDocument,
    operationName: "WorkerPtoTable",
    variables: {
      input: {
        first: WORKER_PTO_HISTORY_PAGE_SIZE,
        workerId,
        includeWorker: false,
        sort: [{ field: "startDate", direction: "desc" }],
      },
    },
    signal: options?.signal,
  });

  return (data.workerPTOEntries.edges ?? []).map((edge) => edge.node as WorkerPTO);
}

export const worker = createQueryKeys("worker", {
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
  ptoHistory: (workerId: string) => ({
    queryKey: ["pto-history", workerId],
    queryFn: ({ signal }) => fetchWorkerPTOHistory(workerId, { signal }),
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
