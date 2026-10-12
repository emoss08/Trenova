import { z } from "zod";
import { nullableStringSchema, optionalStringSchema, tenantInfoSchema } from "./helpers";
import { translate } from "@trenova/shared/i18n/runtime";

export const documentClassificationSchema = z.enum([
  "Public",
  "Private",
  "Sensitive",
  "Regulatory",
]);

export type DocumentClassification = z.infer<typeof documentClassificationSchema>;

export const documentCategorySchema = z.enum([
  "Shipment",
  "Worker",
  "Regulatory",
  "Profile",
  "Branding",
  "Invoice",
  "Contract",
  "Other",
]);

export type DocumentCategory = z.infer<typeof documentCategorySchema>;

export const documentTypeSchema = z.object({
  ...tenantInfoSchema.shape,
  code: z
    .string()
    .min(1, { error: () => translate("Code is required") })
    .max(10),
  name: z
    .string()
    .min(1, { error: () => translate("Name is required") })
    .max(100),
  description: optionalStringSchema,
  color: nullableStringSchema,
  documentClassification: documentClassificationSchema,
  documentCategory: documentCategorySchema,
  isSystem: z.boolean().optional(),
});

export type DocumentType = z.infer<typeof documentTypeSchema>;
