import { apiService } from "@/services/api";
import type { MatchRecurringShipmentsInput } from "@/services/recurring-shipment";
import type { RecurringShipment } from "@/types/recurring-shipment";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const recurringShipment = createQueryKeys("recurringShipment", {
  match: (input: MatchRecurringShipmentsInput) => ({
    queryKey: ["match", input],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.recurringShipmentService.match(input, { signal }),
  }),
  listRuns: (id: RecurringShipment["id"], limit: number) => ({
    queryKey: ["runs", id, limit],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.recurringShipmentService.listRuns(id as string, limit, { signal }),
  }),
});
