import { shipmentControlService } from "@/services/shipment-control";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const shipmentControl = createQueryKeys("shipmentControl", {
  get: () => ({
    queryKey: ["get"],
    queryFn: async () => shipmentControlService.get(),
  }),
});
