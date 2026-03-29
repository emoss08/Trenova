import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  type CreateDocumentUploadSessionParams,
  type BulkUploadDocumentParams,
  type BulkUploadDocumentResponse,
  type Document,
  type DocumentContent,
  type DocumentPacketSummary,
  type DocumentShipmentDraft,
  type DocumentUploadPartTarget,
  type DocumentUploadSession,
  type DocumentUploadSessionState,
  type DownloadUrlResponse,
  type UploadDocumentParams,
  bulkUploadDocumentResponseSchema,
  documentContentSchema,
  documentPacketSummarySchema,
  documentSchema,
  documentShipmentDraftSchema,
  documentUploadPartTargetSchema,
  documentUploadSessionSchema,
  documentUploadSessionStateSchema,
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

    if (params.lineageId) {
      formData.append("lineageId", params.lineageId);
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
    if (params.lineageId) {
      formData.append("lineageId", params.lineageId);
    }

    params.files.forEach((file) => formData.append("files", file));

    const response = await api.upload<BulkUploadDocumentResponse>(
      "/documents/upload-bulk/",
      formData,
    );
    return safeParse(bulkUploadDocumentResponseSchema, response, "Bulk Upload Document");
  }

  public async createUploadSession(
    params: CreateDocumentUploadSessionParams,
  ): Promise<DocumentUploadSession> {
    const response = await api.post<DocumentUploadSession>("/documents/uploads/", params);
    return safeParse(documentUploadSessionSchema, response, "Document Upload Session");
  }

  public async listActiveUploadSessions(
    resourceType: string,
    resourceId: string,
  ): Promise<DocumentUploadSession[]> {
    const response = await api.get<DocumentUploadSession[]>(
      `/documents/uploads/active/?resourceType=${encodeURIComponent(resourceType)}&resourceId=${encodeURIComponent(resourceId)}`,
    );
    return safeParse(z.array(documentUploadSessionSchema), response, "Document Upload Sessions");
  }

  public async getUploadSession(
    sessionId: string,
  ): Promise<DocumentUploadSessionState> {
    const response = await api.get<DocumentUploadSessionState>(
      `/documents/uploads/${sessionId}/`,
    );
    return safeParse(
      documentUploadSessionStateSchema,
      response,
      "Document Upload Session State",
    );
  }

  public async getUploadPartTargets(
    sessionId: string,
    partNumbers: number[],
  ): Promise<DocumentUploadPartTarget[]> {
    const response = await api.post<{ parts: DocumentUploadPartTarget[] }>(
      `/documents/uploads/${sessionId}/parts/`,
      { partNumbers },
    );
    return (
      await safeParse(
      z.object({ parts: z.array(documentUploadPartTargetSchema) }),
      response,
      "Document Upload Part Targets",
      )
    ).parts;
  }

  public async completeUploadSession(sessionId: string): Promise<DocumentUploadSession> {
    const response = await api.post<DocumentUploadSession>(
      `/documents/uploads/${sessionId}/complete/`,
    );
    return safeParse(documentUploadSessionSchema, response, "Document Upload Session");
  }

  public async cancelUploadSession(sessionId: string): Promise<void> {
    await api.post(`/documents/uploads/${sessionId}/cancel/`);
  }

  public async getByResource(
    resourceType: string,
    resourceId: string,
    query?: string,
  ): Promise<Document[]> {
    const search = query?.trim()
      ? `?query=${encodeURIComponent(query.trim())}`
      : "";
    const response = await api.get<Document[]>(
      `/documents/resource/${resourceType}/${resourceId}/${search}`,
    );
    return safeParse(z.array(documentSchema), response, "Document");
  }

  public async getContent(documentId: string): Promise<DocumentContent> {
    const response = await api.get<DocumentContent>(`/documents/${documentId}/content/`);
    return safeParse(documentContentSchema, response, "Document Content");
  }

  public async getShipmentDraft(documentId: string): Promise<DocumentShipmentDraft> {
    const response = await api.get<DocumentShipmentDraft>(
      `/documents/${documentId}/shipment-draft/`,
    );
    return safeParse(documentShipmentDraftSchema, response, "Document Shipment Draft");
  }

  public async reextract(documentId: string): Promise<void> {
    await api.post(`/documents/${documentId}/shipment-draft/reextract/`);
  }

  public async getVersions(documentId: string): Promise<Document[]> {
    const response = await api.get<Document[]>(`/documents/${documentId}/versions/`);
    return safeParse(z.array(documentSchema), response, "Document Versions");
  }

  public async restoreVersion(documentId: string): Promise<Document> {
    const response = await api.post<Document>(`/documents/${documentId}/restore/`);
    return safeParse(documentSchema, response, "Document");
  }

  public async getPacketSummary(
    resourceType: string,
    resourceId: string,
  ): Promise<DocumentPacketSummary> {
    const response = await api.get<DocumentPacketSummary>(
      `/documents/resource/${resourceType}/${resourceId}/packet-summary/`,
    );
    return safeParse(documentPacketSummarySchema, response, "Document Packet Summary");
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
