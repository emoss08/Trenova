import { z } from "zod";
import {
  nullableTextSchema,
  optionalStringSchema,
  tenantInfoSchema,
} from "@trenova/shared/types/helpers";
import { translate } from "@trenova/shared/i18n/runtime";

export const serviceFailureReasonCategorySchema = z.enum([
  "Carrier",
  "Customer",
  "Facility",
  "Weather",
  "Equipment",
  "Documentation",
  "Driver",
  "Shipper",
  "Consignee",
  "Appointment",
  "Other",
]);

export type ServiceFailureReasonCategory = z.infer<typeof serviceFailureReasonCategorySchema>;

export const serviceFailureReasonCodeAppliesToSchema = z.enum([
  "Pickup",
  "Delivery",
  "Both",
  "All",
]);

export type ServiceFailureReasonCodeAppliesTo = z.infer<
  typeof serviceFailureReasonCodeAppliesToSchema
>;

export const serviceFailureReasonCodeSchema = z.object({
  ...tenantInfoSchema.shape,
  code: z
    .string()
    .min(1, { error: () => translate("Code is required") })
    .max(64, { error: () => translate("Code must be less than 64 characters") }),
  label: z
    .string()
    .min(1, { error: () => translate("Label is required") })
    .max(120, { error: () => translate("Label must be less than 120 characters") }),
  description: nullableTextSchema,
  category: serviceFailureReasonCategorySchema.default("Carrier"),
  appliesTo: serviceFailureReasonCodeAppliesToSchema.default("Both"),
  defaultStatusCode: optionalStringSchema.default(""),
  defaultReasonCode: optionalStringSchema.default(""),
  defaultExceptionCode: optionalStringSchema.default(""),
  defaultNote: nullableTextSchema,
  active: z.boolean().default(true),
  sortOrder: z.number().int().min(0).default(100),
  externalMap: z.record(z.string(), z.unknown()).nullish(),
  archivedAt: z.number().int().nullable().optional(),
  archivedById: z.string().nullable().optional(),
  activatedAt: z.number().int().nullable().optional(),
  activatedById: z.string().nullable().optional(),
});

export type ServiceFailureReasonCode = z.infer<typeof serviceFailureReasonCodeSchema>;
