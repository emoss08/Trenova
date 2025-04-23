import { z } from "zod";

export const documentUploadSchema = z.object({
  id: z.string().optional(),
  organizationId: z.string().optional(),
  businessUnitId: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),

  // * Core Fields
  resourceType: z.string().min(1, "Resource type is required"),
  resourceId: z.string().min(1, "Resource ID is required"),
  documentType: z.string().min(1, "Document type is required"),
  isPublic: z.boolean(),
  requireApproval: z.boolean(),
});

export type DocumentUploadSchema = z.infer<typeof documentUploadSchema>;
