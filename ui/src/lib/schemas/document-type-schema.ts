import { DocumentCategory, DocumentClassification } from "@/types/billing";
import * as z from "zod/v4";

export const documentTypeSchema = z.object({
  id: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),

  // * Core Fields
  code: z
    .string({
      error: "Code must be at least 1 character",
    })
    .min(1, "Code must be at least 1 character")
    .max(10, "Code must be less than 10 characters"),
  name: z
    .string({
      error: "Name must be at least 1 character",
    })
    .min(1, {
      error: "Name must be at least 1 character",
    })
    .max(100, {
      error: "Name must be less than 100 characters",
    }),
  description: z.string().optional(),
  color: z
    .string()
    .regex(
      /^#?([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/,
      "Color must be a valid hex color",
    )
    .optional(),
  documentClassification: z.enum(DocumentClassification, {
    error: "Document classification is required",
  }),
  documentCategory: z.enum(DocumentCategory, {
    error: "Document category is required",
  }),
});

export type DocumentTypeSchema = z.infer<typeof documentTypeSchema>;
