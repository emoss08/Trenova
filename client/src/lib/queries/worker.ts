import {
  UpcomingWorkerPtoDocument,
  WorkerPtoChartDataDocument,
} from "@/graphql/generated/graphql";
import { requestGraphQL } from "@/lib/graphql";
import type { GenericLimitOffsetResponse } from "@/types/server";
import type {
  ListUpcomingPTORequest,
  PTOChartDataRequest,
  PTOChartDataPoint,
  WorkerPTO,
} from "@/types/worker";
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
  const data = await requestGraphQL({
    document: UpcomingWorkerPtoDocument,
    operationName: "UpcomingWorkerPto",
    variables: {
      first: req.filter.limit,
      offset: req.filter.offset,
      status: req.status,
      type: req.type,
      startDate: req.startDate,
      endDate: req.endDate,
      workerId: req.workerId,
      fleetCodeId: req.fleetCodeId,
      timezone: req.timezone,
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
      const data = await requestGraphQL({
        document: WorkerPtoChartDataDocument,
        operationName: "WorkerPtoChartData",
        variables: req,
      });

      return data.workerPTOChartData as PTOChartDataPoint[];
    },
  }),
});
