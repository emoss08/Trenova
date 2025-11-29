import * as z from "zod";
import { documentTypeSchema } from "./document-type-schema";
import {
  nullableStringSchema,
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";
import { userSchema } from "./user-schema";

export const pageSizeSchema = z.enum(["Letter", "A4", "Legal"]);
export const orientationSchema = z.enum(["Portrait", "Landscape"]);
export const templateStatusSchema = z.enum(["Draft", "Active", "Archived"]);
export const generationStatusSchema = z.enum([
  "Pending",
  "Processing",
  "Completed",
  "Failed",
]);
export const deliveryMethodSchema = z.enum([
  "None",
  "Email",
  "Download",
  "Portal",
]);

export const documentTemplateSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  code: z.string().min(1, { error: "Code is required" }),
  name: z.string().min(1, { error: "Name is required" }),
  description: optionalStringSchema,
  documentTypeId: nullableStringSchema,
  htmlContent: z.string().min(1, { error: "HTML content is required" }),
  cssContent: optionalStringSchema,
  headerHtml: optionalStringSchema,
  footerHtml: optionalStringSchema,
  pageSize: pageSizeSchema,
  orientation: orientationSchema,
  marginTop: z.number().min(0, { error: "Margin top must be non-negative" }),
  marginBottom: z
    .number()
    .min(0, { error: "Margin bottom must be non-negative" }),
  marginLeft: z.number().min(0, { error: "Margin left must be non-negative" }),
  marginRight: z
    .number()
    .min(0, { error: "Margin right must be non-negative" }),
  status: templateStatusSchema,
  isDefault: z.boolean(),
  isSystem: z.boolean(),

  // Relations
  documentType: documentTypeSchema.nullish(),
  createdBy: userSchema.nullish(),
  updatedBy: userSchema.nullish(),
});

export type DocumentTemplateSchema = z.infer<typeof documentTemplateSchema>;
export type PageSizeSchema = z.infer<typeof pageSizeSchema>;
export type OrientationSchema = z.infer<typeof orientationSchema>;
export type TemplateStatusSchema = z.infer<typeof templateStatusSchema>;
export type GenerationStatusSchema = z.infer<typeof generationStatusSchema>;
export type DeliveryMethodSchema = z.infer<typeof deliveryMethodSchema>;
