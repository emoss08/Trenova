import { api } from "@trenova/shared/lib/api";
import { API_BASE_URL } from "@trenova/shared/lib/constants";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  type CreateDocumentUploadSessionParams,
  type BulkUploadDocumentParams,
  type BulkUploadDocumentResponse,
  type DeliveryFacts,
  type Document,
  type DocumentSignatureStatus,
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
import {
  documentExtractionStatusSchema,
  type DocumentExtractionStatus,
} from "@/types/document-control";
import { z } from "zod";
import { extractionFrom, type Extraction } from "@/lib/document-reading/extraction-payload";
import {
  ApproveDocumentDocument,
  AttachDocumentToShipmentDocument,
  BulkDeleteDocumentsDocument,
  DeleteDocumentDocument,
  DocumentContentDocument,
  DocumentDocument,
  DocumentExtractionStatusDocument,
  DocumentImportAssistantHistoryDocument,
  DocumentPacketSummaryDocument,
  DocumentReadingDocument,
  DocumentShipmentDraftDocument,
  DocumentsByResourceDocument,
  DocumentVersionsDocument,
  OpenDocumentImportAssistantThreadDocument,
  ReextractDocumentDocument,
  RejectDocumentDocument,
  RestoreDocumentVersionDocument,
  SetDocumentDeliveryFactsDocument,
  SetDocumentSignatureDocument,
  type DocumentDeliveryFactsInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { translate } from "@trenova/shared/i18n/runtime";

export type DocumentContentAction = "download" | "view" | "preview";

export type SetDocumentSignatureParams = {
  status: DocumentSignatureStatus;
  signedAt: number | null;
  version: number;
};

export function documentContentUrl(documentId: string, action: DocumentContentAction): string {
  return `${API_BASE_URL}${documentContentEndpoint(documentId, action)}`;
}

/** A document's content as an endpoint for the app's request helpers (`api.download`). */
export function documentContentEndpoint(documentId: string, action: DocumentContentAction): string {
  return `/documents/${encodeURIComponent(documentId)}/${action}/`;
}

/** One page of a PDF or an image, numbered from 1, drawn by the server as an image to read. */
export function documentPageImageEndpoint(documentId: string, page: number): string {
  return `/documents/${encodeURIComponent(documentId)}/pages/${page}/image/`;
}

function asRecord(value: unknown): Record<string, unknown> {
  return value && typeof value === "object" && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : {};
}

/** The editable facts a person records; who recorded them and when are the server's. */
export function toDeliveryFactsInput(facts: Partial<DeliveryFacts>): DocumentDeliveryFactsInput {
  return {
    signedBy: facts.signedBy ?? null,
    pieces: facts.pieces ?? null,
    osd: facts.osd ?? null,
    osdNote: facts.osdNote ?? null,
    seal: facts.seal ?? null,
    arrivedAt: facts.arrivedAt ?? null,
    departedAt: facts.departedAt ?? null,
    amount: facts.amount ?? null,
    reference: facts.reference ?? null,
    chargeId: facts.chargeId ?? null,
    chargeLabel: facts.chargeLabel ?? null,
  };
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

  public async getExtractionStatus(options?: {
    signal?: AbortSignal;
  }): Promise<DocumentExtractionStatus> {
    const data = await requestGraphQL({
      document: DocumentExtractionStatusDocument,
      operationName: "DocumentExtractionStatus",
      signal: options?.signal,
    });
    return safeParse(
      documentExtractionStatusSchema,
      data.documentExtractionStatus,
      "Document Extraction Status",
    );
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
    params?: { includeDocumentType?: boolean },
    options: { signal?: AbortSignal } = {},
  ): Promise<Document[]> {
    const data = await requestGraphQL({
      document: DocumentsByResourceDocument,
      operationName: "DocumentsByResource",
      variables: {
        input: {
          resourceType,
          resourceId,
          query: query?.trim() || undefined,
          includeDocumentType: params?.includeDocumentType,
        },
      },
      signal: options.signal,
    });
    return safeParse(z.array(documentSchema), data.documentsByResource, "Document");
  }

  public async getContent(
    documentId: string,
    options?: { signal?: AbortSignal },
  ): Promise<DocumentContent> {
    const data = await requestGraphQL({
      document: DocumentContentDocument,
      operationName: "DocumentContent",
      variables: { documentId },
      signal: options?.signal,
    });
    const content = data.documentContent;
    return safeParse(
      documentContentSchema,
      {
        ...content,
        pages: (content.pages ?? []).map((page) => ({ ...page, metadata: page.metadata ?? {} })),
      },
      "Document Content",
    );
  }

  public async getShipmentDraft(
    documentId: string,
    options: { signal?: AbortSignal } = {},
  ): Promise<DocumentShipmentDraft> {
    const data = await requestGraphQL({
      document: DocumentShipmentDraftDocument,
      operationName: "DocumentShipmentDraft",
      variables: { documentId },
      signal: options.signal,
    });
    return safeParse(
      documentShipmentDraftSchema,
      data.documentShipmentDraft,
      "Document Shipment Draft",
    );
  }

  public async reextract(documentId: string): Promise<void> {
    await requestGraphQL({
      document: ReextractDocumentDocument,
      operationName: "ReextractDocument",
      variables: { documentId },
    });
  }

  public async getVersions(
    documentId: string,
    options: { signal?: AbortSignal } = {},
  ): Promise<Document[]> {
    const data = await requestGraphQL({
      document: DocumentVersionsDocument,
      operationName: "DocumentVersions",
      variables: { documentId },
      signal: options.signal,
    });
    return safeParse(z.array(documentSchema), data.documentVersions, "Document Versions");
  }

  public async restoreVersion(documentId: string): Promise<Document> {
    const data = await requestGraphQL({
      document: RestoreDocumentVersionDocument,
      operationName: "RestoreDocumentVersion",
      variables: { documentId },
    });
    return safeParse(documentSchema, data.restoreDocumentVersion, "Document");
  }

  public async approve(documentId: string): Promise<Document> {
    const data = await requestGraphQL({
      document: ApproveDocumentDocument,
      operationName: "ApproveDocument",
      variables: { documentId },
    });
    return safeParse(documentSchema, data.approveDocument, "Document");
  }

  public async reject(documentId: string, reason: string): Promise<Document> {
    const data = await requestGraphQL({
      document: RejectDocumentDocument,
      operationName: "RejectDocument",
      variables: { input: { documentId, reason } },
    });
    return safeParse(documentSchema, data.rejectDocument, "Document");
  }

  public async setSignature(
    documentId: string,
    params: SetDocumentSignatureParams,
  ): Promise<Document> {
    const data = await requestGraphQL({
      document: SetDocumentSignatureDocument,
      operationName: "SetDocumentSignature",
      variables: {
        input: {
          documentId,
          status: params.status,
          signedAt: params.signedAt,
          version: params.version,
        },
      },
    });
    return safeParse(documentSchema, data.setDocumentSignature, "Document");
  }

  /** The document as it was read: each value with the page and box it was found in. */
  public async getReading(
    documentId: string,
    options: { signal?: AbortSignal } = {},
  ): Promise<Extraction> {
    const data = await requestGraphQL({
      document: DocumentReadingDocument,
      operationName: "DocumentReading",
      variables: { documentId },
      signal: options.signal,
    });
    return extractionFrom(asRecord(data.documentReading));
  }

  /** Records what a delivery document says; a receipt naming an unbilled charge raises a check. */
  public async setDeliveryFacts(
    documentId: string,
    facts: Partial<DeliveryFacts>,
    version: number,
  ): Promise<Document> {
    const data = await requestGraphQL({
      document: SetDocumentDeliveryFactsDocument,
      operationName: "SetDocumentDeliveryFacts",
      variables: { input: { documentId, facts: toDeliveryFactsInput(facts), version } },
    });
    return safeParse(documentSchema, data.setDocumentDeliveryFacts, "Document");
  }

  public async attachToShipment(documentId: string, shipmentId: string): Promise<Document> {
    const data = await requestGraphQL({
      document: AttachDocumentToShipmentDocument,
      operationName: "AttachDocumentToShipment",
      variables: { input: { documentId, shipmentId } },
    });
    return safeParse(documentSchema, data.attachDocumentToShipment, "Document");
  }

  public async getPacketSummary(
    resourceType: string,
    resourceId: string,
    options: { signal?: AbortSignal } = {},
  ): Promise<DocumentPacketSummary> {
    const data = await requestGraphQL({
      document: DocumentPacketSummaryDocument,
      operationName: "DocumentPacketSummary",
      variables: { resourceType, resourceId },
      signal: options.signal,
    });
    return safeParse(
      documentPacketSummarySchema,
      data.documentPacketSummary,
      "Document Packet Summary",
    );
  }

  public async getById(
    documentId: string,
    options: { signal?: AbortSignal } = {},
  ): Promise<Document> {
    const data = await requestGraphQL({
      document: DocumentDocument,
      operationName: "Document",
      variables: { id: documentId },
      signal: options.signal,
    });
    if (!data.document) {
      throw new Error(translate("Document not found"));
    }
    return safeParse(documentSchema, data.document, "Document");
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
    await requestGraphQL({
      document: DeleteDocumentDocument,
      operationName: "DeleteDocument",
      variables: { id: documentId },
    });
  }

  public async bulkDelete(
    documentIds: string[],
  ): Promise<{ deletedCount: number; errorCount: number }> {
    const data = await requestGraphQL({
      document: BulkDeleteDocumentsDocument,
      operationName: "BulkDeleteDocuments",
      variables: { ids: documentIds },
    });
    return data.bulkDeleteDocuments;
  }

  /**
   * Opens, or returns, the person's conversation with the import assistant
   * about this document. Its turns go through the assistant's own routes.
   */
  public async openImportAssistantThread(documentId: string): Promise<PageThread> {
    const data = await requestGraphQL({
      document: OpenDocumentImportAssistantThreadDocument,
      operationName: "OpenDocumentImportAssistantThread",
      variables: { documentId },
    });
    return safeParse(
      pageThreadSchema,
      data.openDocumentImportAssistantThread,
      "Import Assistant Conversation",
    );
  }

  /**
   * A conversation from before the import assistant moved onto the shared
   * runtime, kept readable for one release. Active ones were carried over;
   * this shows the finished ones.
   */
  public async getImportAssistantHistory(
    documentId: string,
    options: { signal?: AbortSignal } = {},
  ): Promise<ImportAssistantChatHistoryResponse> {
    const data = await requestGraphQL({
      document: DocumentImportAssistantHistoryDocument,
      operationName: "DocumentImportAssistantHistory",
      variables: { documentId },
      signal: options.signal,
    });
    const history = data.documentImportAssistantHistory;
    return safeParse(
      importAssistantChatHistoryResponseSchema,
      {
        ...history,
        status: history.status || undefined,
        messages: (history.messages ?? []).map((message) => ({
          ...message,
          suggestions: (message.suggestions ?? []).map((suggestion) => ({
            ...suggestion,
            type: suggestion.type || undefined,
          })),
        })),
      },
      "Import Assistant Chat History",
    );
  }
}
