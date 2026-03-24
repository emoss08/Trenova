import { z } from "zod";
import { optionalStringSchema, tenantInfoSchema } from "./helpers";

export const fieldTypeSchema = z.enum([
  "text",
  "number",
  "date",
  "boolean",
  "select",
  "multiSelect",
]);
export type FieldType = z.infer<typeof fieldTypeSchema>;

export const selectOptionSchema = z.object({
  value: z.string().min(1, "Value is required"),
  label: z.string().min(1, "Label is required"),
  color: optionalStringSchema,
  description: optionalStringSchema,
});
export type SelectOption = z.infer<typeof selectOptionSchema>;

export const validationRulesSchema = z.object({
  minLength: z.number().int().optional(),
  maxLength: z.number().int().optional(),
  min: z.number().optional(),
  max: z.number().optional(),
  pattern: optionalStringSchema,
});
export type ValidationRules = z.infer<typeof validationRulesSchema>;

export const uiAttributesSchema = z.object({
  placeholder: optionalStringSchema,
  helpText: optionalStringSchema,
  width: optionalStringSchema,
});
export type UIAttributes = z.infer<typeof uiAttributesSchema>;

export const customFieldDefinitionSchema = z.object({
  ...tenantInfoSchema.shape,
  resourceType: z.string().min(1, "Resource type is required"),
  name: z
    .string()
    .min(1, "Name is required")
    .max(100)
    .regex(
      /^[a-z][a-z0-9_]*$/,
      "Must start with lowercase letter, only lowercase letters, numbers, underscores",
    ),
  label: z.string().min(1, "Label is required").max(150),
  description: optionalStringSchema,
  fieldType: fieldTypeSchema,
  isRequired: z.boolean().default(false),
  isActive: z.boolean().default(true),
  displayOrder: z.number().int().default(0),
  color: optionalStringSchema,
  options: z.array(selectOptionSchema).default([]),
  validationRules: validationRulesSchema.optional().nullable(),
  defaultValue: z.any().optional(),
  uiAttributes: uiAttributesSchema.optional().nullable(),
});

export type CustomFieldDefinition = z.infer<typeof customFieldDefinitionSchema>;

export const resourceTypesResponseSchema = z.object({
  resourceTypes: z.array(z.string()),
});
export type ResourceTypesResponse = z.infer<typeof resourceTypesResponseSchema>;

export const optionUsageStatsSchema = z.object({
  value: z.string(),
  label: z.string(),
  usageCount: z.number(),
});
export type OptionUsageStats = z.infer<typeof optionUsageStatsSchema>;

export const definitionUsageStatsSchema = z.object({
  definitionId: z.string(),
  totalValueCount: z.number(),
  resourceCount: z.number(),
  optionUsage: z.array(optionUsageStatsSchema).optional(),
});
export type DefinitionUsageStats = z.infer<typeof definitionUsageStatsSchema>;
