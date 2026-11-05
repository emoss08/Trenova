import {
  recurringShipmentService,
  type MatchRecurringShipmentsInput,
} from "@/services/recurring-shipment";
import type { RecurringShipment } from "@/types/recurring-shipment";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const recurringShipment = createQueryKeys("recurringShipment", {
  get: (id: RecurringShipment["id"], expandDetails?: boolean) => ({
    queryKey: ["get", id, expandDetails],
    queryFn: async () => recurringShipmentService.getById(id as string, expandDetails),
  }),
  match: (input: MatchRecurringShipmentsInput) => ({
    queryKey: ["match", input],
    queryFn: async () => recurringShipmentService.match(input),
  }),
  listRuns: (id: RecurringShipment["id"], params?: { limit?: number; offset?: number }) => ({
    queryKey: ["runs", id, params],
    queryFn: async () => recurringShipmentService.listRuns(id as string, params),
  }),
});
