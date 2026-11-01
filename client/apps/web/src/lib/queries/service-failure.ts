import { fetchDeskServiceFailure } from "@/lib/graphql/service-failure-card";
import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const serviceFailure = createQueryKeys("serviceFailure", {
  /** One failure as the Desk's card reads it. */
  card: (id: string) => ({
    queryKey: ["card", id],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      fetchDeskServiceFailure(id, { signal }),
  }),
  listByShipment: (shipmentId: string) => ({
    queryKey: ["list-by-shipment", shipmentId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.serviceFailureService.listByShipment(shipmentId, { signal }),
  }),
  edi214Readiness: (id: string, trigger?: "Reviewed" | "Resolved") => ({
    queryKey: ["edi-214-readiness", id, trigger],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.serviceFailureService.edi214Readiness(id, trigger, { signal }),
  }),
  edi214Status: (id: string) => ({
    queryKey: ["edi-214-status", id],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.serviceFailureService.edi214Status(id, { signal }),
  }),
});
