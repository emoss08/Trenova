import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  type BulkUploadDocumentParams,
  type BulkUploadDocumentResponse,
  type Document,
  type DownloadUrlResponse,
  type UploadDocumentParams,
  bulkUploadDocumentResponseSchema,
  documentSchema,
  downloadUrlResponseSchema,
} from "@/types/document";
import { z } from "zod";

export class DocumentService {
  public async upload(params: UploadDocumentParams): Promise<Document> {
    const formData = new FormData();
    formData.append("file", params.file);
    formData.append("resourceId", params.resourceId);
    formData.append("resourceType", params.resourceType);

    if (params.description) {
      formData.append("description", params.description);
    }

    if (params.tags && params.tags.length > 0) {
      params.tags.forEach((tag) => formData.append("tags", tag));
    }

    if (params.documentTypeId) {
      formData.append("documentTypeId", params.documentTypeId);
    }

    const response = await api.upload<Document>("/documents/upload/", formData);
    return safeParse(documentSchema, response, "Document");
  }

  public async bulkUpload(
    params: BulkUploadDocumentParams,
  ): Promise<BulkUploadDocumentResponse> {
    const formData = new FormData();
    formData.append("resourceId", params.resourceId);
    formData.append("resourceType", params.resourceType);

    params.files.forEach((file) => formData.append("files", file));

    const response = await api.upload<BulkUploadDocumentResponse>(
      "/documents/upload-bulk/",
      formData,
    );
    return safeParse(bulkUploadDocumentResponseSchema, response, "Bulk Upload Document");
  }

  public async getByResource(
    resourceType: string,
    resourceId: string,
  ): Promise<Document[]> {
    const response = await api.get<Document[]>(
      `/documents/resource/${resourceType}/${resourceId}/`,
    );
    return safeParse(z.array(documentSchema), response, "Document");
  }

  public async getById(documentId: string): Promise<Document> {
    const response = await api.get<Document>(`/documents/${documentId}/`);
    return safeParse(documentSchema, response, "Document");
  }

  public async getDownloadUrl(documentId: string): Promise<string> {
    const response = await api.get<DownloadUrlResponse>(
      `/documents/${documentId}/download/`,
    );
    const parsed = await safeParse(downloadUrlResponseSchema, response, "Download URL");
    return parsed.url;
  }

  public async getViewUrl(documentId: string): Promise<string> {
    const response = await api.get<DownloadUrlResponse>(
      `/documents/${documentId}/view/`,
    );
    const parsed = await safeParse(downloadUrlResponseSchema, response, "View URL");
    return parsed.url;
  }

  public async getPreviewUrl(documentId: string): Promise<string | null> {
    try {
      const response = await api.get<DownloadUrlResponse>(
        `/documents/${documentId}/preview/`,
      );
      const parsed = await safeParse(downloadUrlResponseSchema, response, "Preview URL");
      return parsed.url;
    } catch {
      return null;
    }
  }

  public async delete(documentId: string): Promise<void> {
    await api.delete(`/documents/${documentId}/`);
  }

  public async bulkDelete(
    documentIds: string[],
  ): Promise<{ deletedCount: number; errorCount: number }> {
    const response = await api.post<{
      deletedCount: number;
      errorCount: number;
    }>("/documents/bulk-delete/", { ids: documentIds });
    return response;
  }
}
