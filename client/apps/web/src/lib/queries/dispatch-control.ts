import { dispatchControlService } from "@/services/dispatch-control";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const dispatchControl = createQueryKeys("dispatchControl", {
  get: () => ({
    queryKey: ["get"],
    queryFn: async () => dispatchControlService.get(),
  }),
});
