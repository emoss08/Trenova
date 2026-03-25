import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const dataEntryControl = createQueryKeys("dataEntryControl", {
  get: () => ({
    queryKey: ["get"],
    queryFn: async () => apiService.dataEntryControlService.get(),
  }),
});
