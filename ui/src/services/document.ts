/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { http } from "@/lib/http-client";
import { DocumentTypeSchema } from "@/lib/schemas/document-type-schema";
import { Resource } from "@/types/audit-entry";
import { Document, DocumentStatus, DocumentType } from "@/types/document";
import { LimitOffsetResponse } from "@/types/server";

export interface DocumentUploadOptions {
  resourceType: Resource;
  resourceId: string;
  documentType: DocumentType;
  description?: string;
  tags?: string[];
  isPublic?: boolean;
  requireApproval?: boolean;
  expirationDate?: number;
}

export interface DocumentsFilter {
  resourceType?: Resource;
  resourceId?: string;
  documentType?: DocumentType;
  statuses?: DocumentStatus[];
  tags?: string[];
  query?: string;
  limit?: number;
  offset?: number;
  sortBy?: string;
  sortDir?: "ASC" | "DESC";
  expirationDateStart?: number;
  expirationDateEnd?: number;
  createdAtStart?: number;
  createdAtEnd?: number;
}

export type DocumentCountByResource = {
  resourceType: Resource;
  count: number;
  totalSize: number;
  lastModified: number;
};

export type ResourceSubFolder = {
  folderName: string;
  count: number;
  totalSize: number;
  lastModified: number;
  resourceId: string;
};

export class DocumentAPI {
  async getDocumentCountByResource() {
    const response = await http.get<DocumentCountByResource[]>(
      "/documents/count-by-resource",
    );
    return response.data;
  }

  async getResourceSubFolders(resourceType: Resource) {
    const response = await http.get<ResourceSubFolder[]>(
      `/documents/${resourceType}/sub-folders/`,
    );
    return response.data;
  }

  async getByResourceID(
    resourceType: Resource,
    resourceId: string,
    limit?: number,
    offset?: number,
  ) {
    const response = await http.get<LimitOffsetResponse<Document>>(
      `/documents/${resourceType}/${resourceId}/`,
      {
        params: { limit: limit?.toString(), offset: offset?.toString() },
      },
    );
    return response.data;
  }

  async getDocumentTypes() {
    const response =
      await http.get<LimitOffsetResponse<DocumentTypeSchema>>(
        "/document-types",
      );
    return response.data;
  }

  async delete(docID: string) {
    await http.delete(`/documents/${docID}`);
  }
}
