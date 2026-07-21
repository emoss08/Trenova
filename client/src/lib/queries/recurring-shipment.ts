import { apiService } from "@/services/api";
import type { MatchRecurringShipmentsInput } from "@/services/recurring-shipment";
import type { RecurringShipment } from "@/types/recurring-shipment";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const recurringShipment = createQueryKeys("recurringShipment", {
  get: (id: RecurringShipment["id"], expandDetails?: boolean) => ({
    queryKey: ["get", id, expandDetails],
    queryFn: async () => apiService.recurringShipmentService.getById(id as string, expandDetails),
  }),
  match: (input: MatchRecurringShipmentsInput) => ({
    queryKey: ["match", input],
    queryFn: async () => apiService.recurringShipmentService.match(input),
  }),
  listRuns: (id: RecurringShipment["id"], params?: { limit?: number; offset?: number }) => ({
    queryKey: ["runs", id, params],
    queryFn: async () => apiService.recurringShipmentService.listRuns(id as string, params),
  }),
});
