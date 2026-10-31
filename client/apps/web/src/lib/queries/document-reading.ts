import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const documentReading = createQueryKeys("documentReading", {
  get: (documentId: string) => ({
    queryKey: [documentId],
    queryFn: async ({ signal }) => apiService.documentService.getReading(documentId, { signal }),
  }),
});
