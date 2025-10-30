import * as z from "zod";
import {
    optionalStringSchema,
    timestampSchema,
    versionSchema,
} from "./helpers";

export const VariableContext = z.enum([
  "Invoice",
  "Customer",
  "Shipment",
  "Organization",
  "System",
]);

export const VariableValueType = z.enum([
  "String",
  "Number",
  "Date",
  "Boolean",
  "Currency",
]);

export const variableFormatSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  // * Core Fields
  name: z
    .string()
    .min(1, { error: "Name is required" })
    .max(100, { error: "Name must be less than 100 characters" }),
  description: optionalStringSchema,
  valueType: VariableValueType,
  formatSql: z.string().min(1, { error: "Format SQL is required" }),
  isActive: z.boolean(),
  isSystem: z.boolean(),
});

export const variableSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  // * Core Fields
  key: z
    .string()
    .min(2, { error: "Key must be at least 2 characters" })
    .max(100, { error: "Key must be less than 100 characters" }),
  displayName: z
    .string()
    .min(1, { error: "Display name is required" })
    .max(255, { error: "Display name must be less than 255 characters" }),
  description: optionalStringSchema,
  category: z
    .string()
    .min(1, { error: "Category is required" })
    .max(100, { error: "Category must be less than 100 characters" }),
  query: z.string().min(1, { error: "Query is required" }),
  appliesTo: VariableContext,
  requiredParams: z.array(z.string()),
  defaultValue: optionalStringSchema,
  formatId: optionalStringSchema,
  valueType: VariableValueType,
  isActive: z.boolean(),
  isSystem: z.boolean(),
  isValidated: z.boolean(),
  tags: z.array(z.string()),

  // * Relationships
  format: variableFormatSchema.nullish(),
});

export const variableTestRequestSchema = z.object({
  variable: variableSchema,
  testParams: z.record(z.string(), z.any()),
});

export const variableFormatTestRequestSchema = z.object({
  format: variableFormatSchema,
  testValue: z.string().min(1, { error: "Test value is required" }),
});

export type VariableSchema = z.infer<typeof variableSchema>;
export type VariableFormatSchema = z.infer<typeof variableFormatSchema>;
export type VariableTestRequestSchema = z.infer<
  typeof variableTestRequestSchema
>;
export type VariableFormatTestRequestSchema = z.infer<
  typeof variableFormatTestRequestSchema
>;
