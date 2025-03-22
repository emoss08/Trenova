import { http } from "@/lib/http-client";
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

/**
 * Get document count by resource
 */
export async function getDocumentCountByResource(): Promise<
  DocumentCountByResource[]
> {
  const response = await http.get<DocumentCountByResource[]>(
    "/documents/count-by-resource",
  );
  return response.data;
}

export async function getResourceSubFolders(
  resourceType: Resource,
): Promise<ResourceSubFolder[]> {
  const response = await http.get<ResourceSubFolder[]>(
    `/documents/${resourceType}/sub-folders/`,
  );
  return response.data;
}

export async function getDocumentsByResourceID(
  resourceType: Resource,
  resourceId: string,
): Promise<LimitOffsetResponse<Document>> {
  const response = await http.get<LimitOffsetResponse<Document>>(
    `/documents/${resourceType}/${resourceId}/`,
  );
  return response.data;
}
