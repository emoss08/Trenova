import { dataEntryControlService } from "@/services/data-entry-control";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const dataEntryControl = createQueryKeys("dataEntryControl", {
  get: () => ({
    queryKey: ["get"],
    queryFn: async () => dataEntryControlService.get(),
  }),
});
