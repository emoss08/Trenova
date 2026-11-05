import { billingControlService } from "@/services/billing-control";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const billingControl = createQueryKeys("billingControl", {
  get: () => ({
    queryKey: ["get"],
    queryFn: async () => billingControlService.get(),
  }),
});
