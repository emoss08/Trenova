import { http } from "@/lib/http-client";
import { Resource } from "@/types/audit-entry";
import {
    DocumentStatus,
    DocumentType,
    ResourceFolder,
    VersionInfo,
} from "@/types/document";
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

export const DocumentService = {
  /**
   * List documents with optional filtering
   */
  async listDocuments(
    filter: DocumentsFilter = {},
  ): Promise<LimitOffsetResponse<Document>> {
    const params = new URLSearchParams();

    // Add all filters to params
    Object.entries(filter).forEach(([key, value]) => {
      if (value !== undefined) {
        if (Array.isArray(value)) {
          value.forEach((v) => params.append(key, v.toString()));
        } else {
          params.append(key, value.toString());
        }
      }
    });

    const response = await http.get<LimitOffsetResponse<Document>>(
      "/documents",
      {
        params: Object.fromEntries(params.entries()),
      },
    );
    return response.data;
  },

  /**
   * Get document by ID
   */
  async getDocumentById(id: string): Promise<Document> {
    const response = await http.get<Document>(`/documents/${id}`);
    return response.data;
  },

  /**
   * Get document resource folders (grouped by resource type)
   */
  async getResourceFolders(): Promise<ResourceFolder[]> {
    // In the real implementation, you'll need to create a backend endpoint for this
    // For now, we'll use the available endpoints to simulate the structure
    const resourceTypes = [
      { type: Resource.Shipment, name: "Shipments" },
      { type: Resource.Worker, name: "Drivers" },
      { type: Resource.Equipment, name: "Equipment" },
      { type: Resource.Customer, name: "Customers" },
    ];

    const folders: ResourceFolder[] = [];

    // For each resource type, count the documents
    for (const resource of resourceTypes) {
      const response = await this.listDocuments({
        resourceType: resource.type,
        limit: 1, // We only need count, not data
      });

      folders.push({
        resourceType: resource.type,
        resourceId: resource.type.toLowerCase() + "s",
        resourceName: resource.name,
        documentCount: response.count,
      });
    }

    return folders;
  },

  /**
   * Get subfolders for a specific resource type (e.g., all shipments)
   */
  async getSubfolders(resourceType: Resource): Promise<ResourceFolder[]> {
    // This is a simplified approach. In a real implementation, you might need a dedicated endpoint
    // or a more sophisticated query to get resource entities by type
    const response = await http.get<any[]>(`/${resourceType.toLowerCase()}s`, {
      params: {
        limit: "10", // Limit to top 10 entities to avoid overloading
        fields: "id,name", // Only get the necessary fields
      },
    });

    return response.data.map((entity) => ({
      resourceType: resourceType,
      resourceId: entity.id,
      resourceName: entity.name || `${resourceType} #${entity.id}`,
      documentCount: 0, // You might need another call to get document count per entity
    }));
  },

  /**
   * Upload a document
   */
  async uploadDocument(
    file: File,
    options: DocumentUploadOptions,
  ): Promise<Document> {
    const formData = new FormData();
    formData.append("file", file);

    // Add all options to form data
    Object.entries(options).forEach(([key, value]) => {
      if (value !== undefined) {
        if (Array.isArray(value)) {
          formData.append(key, JSON.stringify(value));
        } else {
          formData.append(key, value.toString());
        }
      }
    });

    const response = await http.post<Document>("/documents", formData, {
      headers: {
        "Content-Type": "multipart/form-data",
      },
    });

    return response.data;
  },

  /**
   * Upload multiple documents in a single request
   */
  async bulkUploadDocuments(
    files: File[],
    resourceType: Resource,
    resourceId: string,
  ): Promise<{
    successful: Document[];
    failed: { fileName: string; error: string }[];
  }> {
    const formData = new FormData();

    // Append all files with the same key name
    files.forEach((file) => {
      formData.append("files", file);
    });

    formData.append("resourceType", resourceType);
    formData.append("resourceId", resourceId);

    // If we have metadata for each file
    const metadata = files.map((file) => ({
      fileName: file.name,
      documentType: DocumentType.Other, // Default, can be customized per file
    }));

    formData.append("metadata", JSON.stringify(metadata));

    const response = await http.post<{
      successful: Document[];
      failed: { fileName: string; error: string }[];
    }>("/documents/bulk", formData, {
      headers: {
        "Content-Type": "multipart/form-data",
      },
    });

    return response.data;
  },

  /**
   * Get document content URL
   */
  getDocumentContentUrl(documentId: string): string {
    return `/documents/${documentId}/content`;
  },

  /**
   * Get document download URL (pre-signed)
   */
  async getDocumentDownloadUrl(
    documentId: string,
    expiryMinutes?: number,
  ): Promise<{
    url: string;
    expiryMinutes: number;
    fileName: string;
  }> {
    const params = expiryMinutes ? { expiry: expiryMinutes } : {};
    const response = await http.get(`/documents/${documentId}/download`, {
      params: Object.fromEntries(Object.entries(params)),
    });
    return response.data as {
      url: string;
      expiryMinutes: number;
      fileName: string;
    };
  },

  /**
   * Approve document
   */
  async approveDocument(documentId: string): Promise<Document> {
    const response = await http.put<Document>(
      `/documents/${documentId}/approve`,
    );
    return response.data;
  },

  /**
   * Reject document
   */
  async rejectDocument(documentId: string, reason: string): Promise<Document> {
    const response = await http.put<Document>(
      `/documents/${documentId}/reject`,
      { reason },
    );
    return response.data;
  },

  /**
   * Archive document
   */
  async archiveDocument(documentId: string): Promise<Document> {
    const response = await http.put<Document>(
      `/documents/${documentId}/archive`,
    );
    return response.data;
  },

  /**
   * Delete document
   */
  async deleteDocument(documentId: string): Promise<void> {
    await http.delete(`/documents/${documentId}`);
  },

  /**
   * Get document versions
   */
  async getDocumentVersions(documentId: string): Promise<VersionInfo[]> {
    const response = await http.get<VersionInfo[]>(
      `/documents/${documentId}/versions`,
    );
    return response.data;
  },

  /**
   * Restore document version
   */
  async restoreDocumentVersion(
    documentId: string,
    versionId: string,
  ): Promise<Document> {
    const response = await http.post<Document>(
      `/documents/${documentId}/versions/${versionId}/restore`,
    );
    return response.data;
  },
};

export default DocumentService;
