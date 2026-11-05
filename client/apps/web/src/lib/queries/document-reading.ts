import { documentService } from "@/services/document";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const documentReading = createQueryKeys("documentReading", {
  get: (documentId: string) => ({
    queryKey: [documentId],
    queryFn: async ({ signal }) => documentService.getReading(documentId, { signal }),
  }),
});
