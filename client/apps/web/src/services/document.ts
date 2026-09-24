import { api } from "@trenova/shared/lib/api";
import { API_BASE_URL } from "@trenova/shared/lib/constants";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  type CreateDocumentUploadSessionParams,
  type BulkUploadDocumentParams,
  type BulkUploadDocumentResponse,
  type Document,
  type ImportAssistantChatHistoryResponse,
  importAssistantChatHistoryResponseSchema,
  type DocumentContent,
  type DocumentPacketSummary,
  type DocumentShipmentDraft,
  type DocumentUploadPartTarget,
  type DocumentUploadSession,
  type DocumentUploadSessionState,
  type UploadDocumentParams,
  bulkUploadDocumentResponseSchema,
  documentContentSchema,
  documentPacketSummarySchema,
  documentSchema,
  documentShipmentDraftSchema,
  documentUploadPartTargetSchema,
  documentUploadSessionSchema,
  documentUploadSessionStateSchema,
} from "@trenova/shared/types/document";
import { pageThreadSchema, type PageThread } from "@/types/assistant";
import { z } from "zod";

export type DocumentContentAction = "download" | "view" | "preview";

export function documentContentUrl(documentId: string, action: DocumentContentAction): string {
  return `${API_BASE_URL}/documents/${encodeURIComponent(documentId)}/${action}/`;
}

export class DocumentService {
  public async upload(params: UploadDocumentParams): Promise<Document> {
    const formData = new FormData();
    formData.append("file", params.file);
    formData.append("resourceId", params.resourceId);
    formData.append("resourceType", params.resourceType);
    if (params.processingProfile) {
      formData.append("processingProfile", params.processingProfile);
    }

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

  public async bulkUpload(params: BulkUploadDocumentParams): Promise<BulkUploadDocumentResponse> {
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

  public async getUploadSession(sessionId: string): Promise<DocumentUploadSessionState> {
    const response = await api.get<DocumentUploadSessionState>(`/documents/uploads/${sessionId}/`);
    return safeParse(documentUploadSessionStateSchema, response, "Document Upload Session State");
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
    params?: Record<string, string>,
  ): Promise<Document[]> {
    const searchParams = new URLSearchParams(params);
    if (query?.trim()) {
      searchParams.set("query", query.trim());
    }
    const qs = searchParams.toString();
    const endpoint = qs
      ? `/documents/resource/${resourceType}/${resourceId}/?${qs}`
      : `/documents/resource/${resourceType}/${resourceId}/`;
    const response = await api.get<Document[]>(endpoint);
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

  public async attachToShipment(documentId: string, shipmentId: string): Promise<Document> {
    const response = await api.post<Document>(`/documents/${documentId}/attach-to-shipment/`, {
      shipmentId,
    });
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
    return documentContentUrl(documentId, "download");
  }

  public async getViewUrl(documentId: string): Promise<string> {
    return documentContentUrl(documentId, "view");
  }

  public async getPreviewUrl(documentId: string): Promise<string | null> {
    return documentContentUrl(documentId, "preview");
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

  /**
   * Opens, or returns, the person's conversation with the import assistant
   * about this document. Its turns go through the assistant's own routes.
   */
  public async openImportAssistantThread(documentId: string): Promise<PageThread> {
    const response = await api.post<PageThread>(
      `/documents/${encodeURIComponent(documentId)}/import-assistant/thread/`,
      {},
    );
    return safeParse(pageThreadSchema, response, "Import Assistant Conversation");
  }

  /**
   * A conversation from before the import assistant moved onto the shared
   * runtime, kept readable for one release. Active ones were carried over;
   * this shows the finished ones.
   */
  public async getImportAssistantHistory(
    documentId: string,
  ): Promise<ImportAssistantChatHistoryResponse> {
    const response = await api.get<ImportAssistantChatHistoryResponse>(
      `/documents/${documentId}/import-assistant/history/`,
    );
    return safeParse(
      importAssistantChatHistoryResponseSchema,
      response,
      "Import Assistant Chat History",
    );
  }
}
