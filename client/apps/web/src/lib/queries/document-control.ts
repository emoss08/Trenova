import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const documentControl = createQueryKeys("documentControl", {
  get: () => ({
    queryKey: ["get"],
    queryFn: async () => apiService.documentControlService.get(),
  }),
});
