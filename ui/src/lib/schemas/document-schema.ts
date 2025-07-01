import * as z from "zod/v4";
import {
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

export const documentUploadSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  // * Core Fields
  resourceType: z.string().min(1, { error: "Resource type is required" }),
  resourceId: z.string().min(1, { error: "Resource ID is required" }),
  documentType: z.string().min(1, { error: "Document type is required" }),
  isPublic: z.boolean(),
  requireApproval: z.boolean(),
});

export type DocumentUploadSchema = z.infer<typeof documentUploadSchema>;
