import { accountingControlService } from "@/services/accounting-control";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const accountingControl = createQueryKeys("accountingControl", {
  get: () => ({
    queryKey: ["get"],
    queryFn: async () => accountingControlService.get(),
  }),
});
