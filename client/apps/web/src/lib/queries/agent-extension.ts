import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const agentExtension = createQueryKeys("agentExtension", {
  catalog: () => ({
    queryKey: ["catalog"],
    queryFn: () => apiService.agentExtensionService.getCatalog(),
  }),
  config: (type: string) => ({
    queryKey: ["config", type],
    queryFn: () => apiService.agentExtensionService.getConfig(type),
  }),
});
