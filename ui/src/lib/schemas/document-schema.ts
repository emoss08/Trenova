import { boolean, InferType, object, string } from "yup";

export const documentUploadSchema = object({
  resourceType: string().required("Resource type is required"),
  resourceId: string().required("Resource ID is required"),
  documentType: string().required("Document type is required"),
  description: string().optional(),
  isPublic: boolean().required("Is public is required"),
  requireApproval: boolean().required("Require approval is required"),
});

export type DocumentUploadSchema = InferType<typeof documentUploadSchema>;
