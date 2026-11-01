import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const documentControl = createQueryKeys("documentControl", {
  get: () => ({
    queryKey: ["get"],
    queryFn: async ({ signal }: { signal: AbortSignal }) =>
      apiService.documentControlService.get({ signal }),
  }),
  extractionStatus: () => ({
    queryKey: ["extraction-status"],
    queryFn: async ({ signal }: { signal: AbortSignal }) =>
      apiService.documentService.getExtractionStatus({ signal }),
  }),
});
