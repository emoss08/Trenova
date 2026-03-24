import { z } from "zod";

export const documentStatusSchema = z.enum([
  "Draft",
  "Active",
  "Archived",
  "Expired",
  "Pending",
  "Rejected",
  "PendingApproval",
]);

export type DocumentStatus = z.infer<typeof documentStatusSchema>;

export const documentSchema = z.object({
  id: z.string(),
  organizationId: z.string(),
  businessUnitId: z.string(),
  fileName: z.string(),
  originalName: z.string(),
  fileSize: z.number(),
  fileType: z.string(),
  storagePath: z.string(),
  status: documentStatusSchema,
  description: z.string().nullable().optional(),
  resourceId: z.string(),
  resourceType: z.string(),
  expirationDate: z.number().nullable().optional(),
  tags: z.array(z.string()).nullable().optional(),
  isPublic: z.boolean(),
  uploadedById: z.string(),
  approvedById: z.string().nullable().optional(),
  approvedAt: z.number().nullable().optional(),
  previewStoragePath: z.string().nullable().optional(),
  documentTypeId: z.string().nullable().optional(),
  version: z.number(),
  createdAt: z.number(),
  updatedAt: z.number(),
});

export type Document = z.infer<typeof documentSchema>;

export const uploadDocumentResponseSchema = documentSchema;

export const bulkUploadDocumentResponseSchema = z.object({
  documents: z.array(documentSchema),
  errorCount: z.number(),
  successCount: z.number(),
});

export type BulkUploadDocumentResponse = z.infer<
  typeof bulkUploadDocumentResponseSchema
>;

export const downloadUrlResponseSchema = z.object({
  url: z.string(),
});

export type DownloadUrlResponse = z.infer<typeof downloadUrlResponseSchema>;

export interface UploadDocumentParams {
  file: File;
  resourceId: string;
  resourceType: string;
  description?: string;
  tags?: string[];
  documentTypeId?: string;
}

export interface BulkUploadDocumentParams {
  files: File[];
  resourceId: string;
  resourceType: string;
}
