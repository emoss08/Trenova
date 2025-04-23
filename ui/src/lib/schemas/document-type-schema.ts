import { DocumentCategory, DocumentClassification } from "@/types/billing";
import { z } from "zod";

export const documentTypeSchema = z.object({
  id: z.string().optional(),
  organizationId: z.string().optional(),
  businessUnitId: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),

  // * Core Fields
  code: z
    .string()
    .min(1, "Code must be at least 1 character")
    .max(10, "Code must be less than 10 characters"),
  name: z
    .string()
    .min(1, "Name must be at least 1 character")
    .max(100, "Name must be less than 100 characters"),
  description: z.string().optional(),
  color: z
    .string()
    .regex(
      /^#?([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/,
      "Color must be a valid hex color",
    )
    .optional(),
  documentClassification: z.nativeEnum(DocumentClassification, {
    message: "Document classification is required",
  }),
  documentCategory: z.nativeEnum(DocumentCategory, {
    message: "Document category is required",
  }),
});

export type DocumentTypeSchema = z.infer<typeof documentTypeSchema>;
