import { sequenceConfigService } from "@/services/sequence-config";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const sequenceConfig = createQueryKeys("sequenceConfig", {
  get: () => ({
    queryKey: ["get"],
    queryFn: async () => sequenceConfigService.get(),
  }),
});
