import { fetchDeskServiceFailure } from "@/lib/graphql/service-failure-card";
import { serviceFailureService } from "@/services/service-failure";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const serviceFailure = createQueryKeys("serviceFailure", {
  /** One failure as the Desk's card reads it. */
  card: (id: string) => ({
    queryKey: ["card", id],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      fetchDeskServiceFailure(id, { signal }),
  }),
  get: (id: string) => ({
    queryKey: ["get", id],
    queryFn: async () => serviceFailureService.getById(id),
  }),
  listByShipment: (shipmentId: string) => ({
    queryKey: ["list-by-shipment", shipmentId],
    queryFn: async () => serviceFailureService.listByShipment(shipmentId, "limit=100"),
  }),
  edi214Readiness: (id: string, trigger?: "Reviewed" | "Resolved") => ({
    queryKey: ["edi-214-readiness", id, trigger],
    queryFn: async () => serviceFailureService.edi214Readiness(id, trigger),
  }),
  edi214Status: (id: string) => ({
    queryKey: ["edi-214-status", id],
    queryFn: async () => serviceFailureService.edi214Status(id),
  }),
});
