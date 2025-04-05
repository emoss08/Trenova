import { DocumentCategory, DocumentClassification } from "@/types/billing";
import { type InferType, object, string } from "yup";

export const documentTypeSchema = object({
  id: string().optional(),
  organizationId: string().nullable().optional(),
  businessUnitId: string().nullable().optional(),
  code: string()
    .min(1, "Code must be at least 1 character")
    .max(10, "Code must be less than 10 characters")
    .required("Code is required"),
  name: string()
    .min(1, "Name must be at least 1 character")
    .max(100, "Name must be less than 100 characters")
    .required("Name is required"),
  description: string().optional(),
  color: string()
    .matches(
      /^#?([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/,
      "Color must be a valid hex color",
    )
    .optional(),
  documentClassification: string()
    .required("Document classification is required")
    .oneOf(
      Object.values(DocumentClassification),
      "Document classification must be valid",
    ),
  documentCategory: string()
    .required("Document category is required")
    .oneOf(Object.values(DocumentCategory), "Document category must be valid"),
});

export type DocumentTypeSchema = InferType<typeof documentTypeSchema>;
