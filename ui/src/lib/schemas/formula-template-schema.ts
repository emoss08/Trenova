/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { string, z } from "zod";
import {
    decimalStringSchema,
    nullableStringSchema,
    optionalStringSchema,
    timestampSchema,
    versionSchema,
} from "./helpers";

export const ContextType = z.enum(["BUILT_IN", "CUSTOM"]);

const ValueType = z.enum([
  "NUMBER",
  "STRING",
  "BOOLEAN",
  "DATE",
  "ARRAY",
  "OBJECT",
  "ANY",
]);

export const FormulaAction = z.enum([
  "CREATE",
  "READ",
  "UPDATE",
  "DELETE",
  "TEST",
  "APPROVE",
]);

export const AdjustmentType = z.enum(["DISCOUNT", "SURCHARGE", "MULTIPLIER"]);

export const Category = z.enum([
  "BaseRate",
  "DistanceBased",
  "WeightBased",
  "DimensionalWeight",
  "FuelSurcharge",
  "Accessorial",
  "TimeBasedRate",
  "ZoneBased",
  "Custom",
]);

export const BuiltInContextName = z.enum([
  "equipmentType",
  "hazmat",
  "temperature",
  "route",
  "time",
  "shipment",
  "customer",
]);

export const ParameterOption = z.object({
  value: z.any(),
  label: z.string(),
  description: z.string(),
});

export const TemplateParameter = z.object({
  name: string(),
  type: ValueType,
  description: z.string(),
  defaultValue: z.any(),
  required: z.boolean(),
  minValue: z.number().optional(),
  maxValue: z.number().optional(),
  options: z.array(ParameterOption).optional(),
});

export const TemplateExample = z.object({
  name: string(),
  description: string(),
  parameters: z.record(string(), z.any()),
  shipmentData: z.record(string(), z.any()).nullish(),
  expectedRate: z.number(),
});

export const TemplateRequirement = z.object({
  type: z.enum(["variable", "field", "function"]),
  name: string(),
  description: string(),
});

export const templateVariable = z.object({
  name: z.string(),
  type: ValueType,
  description: z.string(),
  required: z.boolean(),
  defaultValue: z.any().optional(),
  source: z.string(),
});

export const formulaTemplateSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  name: string().min(3, { error: "Name must be at least 3 characters" }),
  description: nullableStringSchema,
  category: Category,
  expression: string().min(1, { error: "Expression is required" }), // * Expression evaluator on the server will validate this.
  variables: z.array(templateVariable).nullish(),
  parameters: z.array(TemplateParameter).nullish(),
  tags: z.array(string()).nullish(),
  examples: z.array(TemplateExample).nullish(),
  requirements: z.array(TemplateRequirement),
  minRate: decimalStringSchema,
  maxRate: decimalStringSchema,
  outputUnit: z.string().optional(),
  isActive: z.boolean(),
  isDefault: z.boolean(),
});

export type FormulaTemplateSchema = z.infer<typeof formulaTemplateSchema>;
