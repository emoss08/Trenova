import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  type DocumentOperationsDiagnostics,
  documentOperationsDiagnosticsSchema,
} from "@/types/document-operations";

export class DocumentOperationsService {
  public async getDiagnostics(documentId: string): Promise<DocumentOperationsDiagnostics> {
    const response = await api.get<DocumentOperationsDiagnostics>(
      `/admin/document-operations/${documentId}/`,
    );
    return safeParse(
      documentOperationsDiagnosticsSchema,
      response,
      "Document Operations Diagnostics",
    );
  }

  public async reextract(documentId: string): Promise<void> {
    await api.post(`/admin/document-operations/${documentId}/reextract/`);
  }

  public async regeneratePreview(documentId: string): Promise<void> {
    await api.post(`/admin/document-operations/${documentId}/regenerate-preview/`);
  }

  public async resyncSearch(documentId: string): Promise<void> {
    await api.post(`/admin/document-operations/${documentId}/resync-search/`);
  }
}
