import { documentControlService } from "@/services/document-control";
import { documentService } from "@/services/document";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const documentControl = createQueryKeys("documentControl", {
  get: () => ({
    queryKey: ["get"],
    queryFn: async () => documentControlService.get(),
  }),
  extractionStatus: () => ({
    queryKey: ["extraction-status"],
    queryFn: async ({ signal }: { signal: AbortSignal }) =>
      documentService.getExtractionStatus({ signal }),
  }),
});
