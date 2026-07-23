import {
  UpcomingWorkerPtoDocument,
  WorkerPtoChartDataDocument,
  type UpcomingWorkerPtoQuery,
  type UpcomingWorkerPtoQueryVariables,
  type WorkerPtoChartDataQuery,
  type WorkerPtoChartDataQueryVariables,
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

type WorkerPTOConnection = {
  edges?: Array<{ node: unknown }>;
  pageInfo?: {
    hasNextPage?: boolean;
    endCursor?: string | null;
  };
  totalCount?: number | null;
};

function workerPTOConnectionToLimitOffset(
  connection: WorkerPTOConnection,
): GenericLimitOffsetResponse<WorkerPTO> {
  const results = (connection.edges ?? []).map(
    (edge) => edge.node as WorkerPTO,
  );

  return {
    results,
    count: connection.totalCount ?? results.length,
    next: connection.pageInfo?.hasNextPage
      ? (connection.pageInfo.endCursor ?? null)
      : null,
    prev: null,
  };
}

export async function fetchUpcomingWorkerPTO(
  req: ListUpcomingPTORequest,
): Promise<GenericLimitOffsetResponse<WorkerPTO>> {
  const data = await requestGraphQL<
    UpcomingWorkerPtoQuery,
    UpcomingWorkerPtoQueryVariables
  >({
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
  });

  return workerPTOConnectionToLimitOffset(
    data.upcomingWorkerPTO as WorkerPTOConnection,
  );
}

export const worker = createQueryKeys("worker", {
  listUpcomingPTO: (req: ListUpcomingPTORequest) => ({
    queryKey: ["list-upcoming-pto", req],
    queryFn: () => fetchUpcomingWorkerPTO(req),
  }),
  ptoChartData: (req: PTOChartDataRequest) => ({
    queryKey: ["pto-chart-data", req],
    queryFn: async () => {
      const data = await requestGraphQL<
        WorkerPtoChartDataQuery,
        WorkerPtoChartDataQueryVariables
      >({
        document: WorkerPtoChartDataDocument,
        operationName: "WorkerPtoChartData",
        variables: { input: req },
      });

      return data.workerPTOChartData as PTOChartDataPoint[];
    },
  }),
});
