import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const dispatchControl = createQueryKeys("dispatchControl", {
  get: () => ({
    queryKey: ["get"],
    queryFn: async () => apiService.dispatchControlService.get(),
  }),
});
