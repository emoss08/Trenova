import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const billingQueue = createQueryKeys("billingQueue", {
  stats: () => ({
    queryKey: ["stats"],
    queryFn: async () => apiService.billingQueueService.getStats(),
  }),
  get: (itemId: string) => ({
    queryKey: ["get", itemId],
    queryFn: async () =>
      apiService.billingQueueService.getById(itemId, {
        expandShipmentDetails: "true",
      }),
  }),
  neighbors: (itemId: string, params: Record<string, string> = {}) => ({
    queryKey: ["neighbors", itemId, params],
    queryFn: async () => apiService.billingQueueService.getNeighbors(itemId, params),
  }),
  activity: (itemId: string) => ({
    queryKey: ["activity", itemId],
    queryFn: async () => apiService.billingQueueService.getActivity(itemId),
  }),
  summaries: (itemIds: string[]) => ({
    queryKey: ["summaries", itemIds],
    queryFn: async () => apiService.billingQueueService.getSummaries(itemIds),
  }),
  approvalRun: (runId: string) => ({
    queryKey: ["approvalRun", runId],
    queryFn: async () => apiService.billingQueueService.getBulkApprove(runId),
  }),
});
