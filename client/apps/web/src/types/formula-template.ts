import { z } from "zod";
import { decimalStringSchema } from "./helpers";
import { userSchema } from "./user";

export const VariableValueType = z.enum([
  "Number",
  "String",
  "Boolean",
  "Date",
  "Array",
  "Object",
  "Any",
]);
export type VariableValueType = z.infer<typeof VariableValueType>;

export const variableDefinitionSchema = z.object({
  name: z.string().min(1, "Name is required"),
  type: VariableValueType,
  description: z.string().default(""),
  required: z.boolean().default(false),
  defaultValue: z.any().optional(),
  source: z.string().optional(),
});
export type VariableDefinition = z.output<typeof variableDefinitionSchema>;
export type VariableDefinitionInput = z.input<typeof variableDefinitionSchema>;

export const breakdownDefinitionSchema = z.object({
  name: z
    .string()
    .min(1, "Name is required")
    .regex(
      /^[a-zA-Z][a-zA-Z0-9_]*$/,
      "Must start with a letter and contain only letters, numbers, and underscores",
    ),
  label: z.string().min(1, "Label is required"),
  expression: z.string().min(1, "Expression is required"),
});
export type BreakdownDefinition = z.output<typeof breakdownDefinitionSchema>;
export type BreakdownDefinitionInput = z.input<typeof breakdownDefinitionSchema>;

export const MAX_BREAKDOWN_DEFINITIONS = 20;

export const breakdownDefinitionsSchema = z
  .array(breakdownDefinitionSchema)
  .max(
    MAX_BREAKDOWN_DEFINITIONS,
    `A maximum of ${MAX_BREAKDOWN_DEFINITIONS} breakdown items is allowed`,
  )
  .default([]);

export const formulaTemplateStatusSchema = z.enum(["Active", "Inactive", "Draft", "InReview"]);
export type FormulaTemplateStatus = z.infer<typeof formulaTemplateStatusSchema>;

export const formulaTemplateTypeSchema = z.enum(["FreightCharge", "AccessorialCharge"]);
export type FormulaTemplateType = z.infer<typeof formulaTemplateTypeSchema>;

export const formulaTemplateSchema = z
  .object({
    id: z.string().optional(),
    organizationId: z.string().optional(),
    businessUnitId: z.string().optional(),
    name: z.string().min(1, "Name is required").max(100),
    description: z.string().default(""),
    type: formulaTemplateTypeSchema,
    expression: z.string().min(1, "Expression is required"),
    status: formulaTemplateStatusSchema.default("Draft"),
    schemaId: z.string().default("shipment"),
    variableDefinitions: z.array(variableDefinitionSchema).default([]),
    breakdownDefinitions: breakdownDefinitionsSchema,
    minCharge: decimalStringSchema,
    maxCharge: decimalStringSchema,
    submittedById: z.string().nullish(),
    submittedAt: z.number().nullish(),
    approvedById: z.string().nullish(),
    approvedAt: z.number().nullish(),
    reviewComment: z.string().nullish(),
    metadata: z.record(z.any(), z.any()).nullish(),
    version: z.number().optional(),
    sourceTemplateId: z.string().nullish(),
    sourceVersionNumber: z.number().nullish(),
    currentVersionNumber: z.number().optional(),
    createdAt: z.number().optional(),
    updatedAt: z.number().optional(),
  })
  .superRefine((value, ctx) => {
    if (value.minCharge != null && value.minCharge < 0) {
      ctx.addIssue({
        code: "custom",
        message: "Minimum charge cannot be negative",
        path: ["minCharge"],
      });
    }

    if (value.maxCharge != null && value.maxCharge < 0) {
      ctx.addIssue({
        code: "custom",
        message: "Maximum charge cannot be negative",
        path: ["maxCharge"],
      });
    }

    if (value.minCharge != null && value.maxCharge != null && value.minCharge > value.maxCharge) {
      ctx.addIssue({
        code: "custom",
        message: "Minimum charge cannot exceed maximum charge",
        path: ["minCharge"],
      });
    }
  });
export type FormulaTemplate = z.output<typeof formulaTemplateSchema>;
export type FormulaTemplateFormValues = z.input<typeof formulaTemplateSchema>;
export const listFormulaTemplateResponseSchema = z.array(formulaTemplateSchema);
export type ListFormulaTemplateResponse = z.infer<typeof listFormulaTemplateResponseSchema>;

export const fieldChangeSchema = z.object({
  from: z.any(),
  to: z.any(),
  type: z.enum(["created", "updated", "deleted"]),
  fieldType: z.string(),
  path: z.string(),
});
export type FieldChange = z.infer<typeof fieldChangeSchema>;

export const versionTagSchema = z.enum(["Stable", "Production", "Draft", "Testing", "Deprecated"]);
export type VersionTag = z.infer<typeof versionTagSchema>;

export const VERSION_TAG_OPTIONS: {
  value: VersionTag;
  label: string;
  color: string;
  description: string;
}[] = [
  {
    value: "Stable",
    label: "Stable",
    color: "bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300",
    description: "Tested and ready for use",
  },
  {
    value: "Production",
    label: "Production",
    color: "bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300",
    description: "Currently in production",
  },
  {
    value: "Draft",
    label: "Draft",
    color: "bg-gray-100 text-gray-700 dark:bg-gray-900/40 dark:text-gray-300",
    description: "Work in progress",
  },
  {
    value: "Testing",
    label: "Testing",
    color: "bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300",
    description: "Under testing",
  },
  {
    value: "Deprecated",
    label: "Deprecated",
    color: "bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300",
    description: "No longer recommended",
  },
];

export const formulaTemplateVersionSchema = z.object({
  id: z.string(),
  templateId: z.string(),
  organizationId: z.string(),
  businessUnitId: z.string(),
  versionNumber: z.number(),
  name: z.string(),
  description: z.string().optional(),
  type: formulaTemplateTypeSchema,
  expression: z.string(),
  status: formulaTemplateStatusSchema,
  schemaId: z.string(),
  variableDefinitions: z.array(variableDefinitionSchema).default([]),
  breakdownDefinitions: breakdownDefinitionsSchema,
  minCharge: decimalStringSchema,
  maxCharge: decimalStringSchema,
  effectiveFrom: z.number().nullish(),
  metadata: z.record(z.any(), z.any()).nullish(),
  changeMessage: z.string().optional(),
  changeSummary: z.record(z.string(), fieldChangeSchema).nullish(),
  tags: z
    .array(versionTagSchema)
    .nullish()
    .transform((v) => v ?? []),
  createdById: z.string(),
  createdAt: z.number(),

  createdBy: userSchema.nullish(),
});
export type FormulaTemplateVersion = z.infer<typeof formulaTemplateVersionSchema>;

export const versionDiffSchema = z.object({
  fromVersion: z.number(),
  toVersion: z.number(),
  changes: z.record(z.string(), fieldChangeSchema),
  changeCount: z.number(),
});
export type VersionDiff = z.infer<typeof versionDiffSchema>;

export const forkLineageSchema: z.ZodType<ForkLineage> = z.lazy(() =>
  z.object({
    templateId: z.string(),
    templateName: z.string(),
    sourceTemplateId: z.string().nullish(),
    sourceVersion: z.number().nullish(),
    forkedTemplates: z.array(forkLineageSchema).optional(),
  }),
);
export type ForkLineage = {
  templateId: string;
  templateName: string;
  sourceTemplateId?: string | null;
  sourceVersion?: number | null;
  forkedTemplates?: ForkLineage[];
};

export const testExpressionRequestSchema = z.object({
  expression: z.string(),
  schemaId: z.string(),
  variables: z.record(z.any(), z.any()).default({}),
  shipmentId: z.string().optional(),
  breakdowns: z.array(breakdownDefinitionSchema).optional(),
});
export type TestExpressionRequest = z.infer<typeof testExpressionRequestSchema>;

export const testBreakdownItemSchema = z.object({
  name: z.string(),
  label: z.string().optional().default(""),
  amount: z.coerce.number(),
  error: z.string().optional(),
});
export type TestBreakdownItem = z.output<typeof testBreakdownItemSchema>;

export const testExpressionResponseSchema = z.object({
  valid: z.boolean(),
  result: z.any().optional(),
  error: z.string().optional(),
  message: z.string(),
  resolvedVariables: z.record(z.string(), z.any()).nullish(),
  breakdown: z.array(testBreakdownItemSchema).nullish(),
});
export type TestExpressionResponse = z.infer<typeof testExpressionResponseSchema>;

export const backtestRequestSchema = z
  .object({
    expression: z.string().optional(),
    versionNumber: z.number().int().optional(),
    limit: z.number().int().min(1).max(500).optional(),
  })
  .superRefine((value, ctx) => {
    const hasExpression = !!value.expression;
    const hasVersion = value.versionNumber != null;

    if (hasExpression === hasVersion) {
      ctx.addIssue({
        code: "custom",
        message: "Provide exactly one of expression or version number",
        path: ["expression"],
      });
    }
  });
export type BacktestRequest = z.infer<typeof backtestRequestSchema>;

export const backtestResultSchema = z.object({
  shipmentId: z.string(),
  proNumber: z
    .string()
    .nullish()
    .transform((v) => v ?? ""),
  currentAmount: z.coerce.number(),
  candidateAmount: z.coerce.number(),
  delta: z.coerce.number(),
  deltaPct: z.coerce.number(),
  currentError: z.string().optional(),
  candidateError: z.string().optional(),
  guardrailApplied: z.boolean().default(false),
});
export type BacktestResult = z.output<typeof backtestResultSchema>;

export const backtestSummarySchema = z.object({
  shipmentCount: z.number(),
  evaluatedCount: z.number(),
  changedCount: z.number(),
  increasedCount: z.number(),
  decreasedCount: z.number(),
  errorCount: z.number(),
  currentTotal: z.coerce.number(),
  candidateTotal: z.coerce.number(),
  totalDelta: z.coerce.number(),
  totalDeltaPct: z.coerce.number(),
  maxIncrease: z.coerce.number(),
  maxDecrease: z.coerce.number(),
});
export type BacktestSummary = z.output<typeof backtestSummarySchema>;

export const backtestResponseSchema = z.object({
  results: z
    .array(backtestResultSchema)
    .nullish()
    .transform((v) => v ?? []),
  summary: backtestSummarySchema,
});
export type BacktestResponse = z.output<typeof backtestResponseSchema>;

export const updateEffectiveDateRequestSchema = z.object({
  effectiveFrom: z.number().nullable(),
});
export type UpdateEffectiveDateRequest = z.infer<typeof updateEffectiveDateRequestSchema>;

export type SchemaVariableType = "Number" | "String" | "Boolean" | "Integer";

export type SchemaVariable = {
  name: string;
  type: SchemaVariableType;
  description: string;
  category: string;
  nullable?: boolean;
};

export const VARIABLE_CATEGORIES = [
  { id: "shipment", label: "Shipment Fields" },
  { id: "customer", label: "Customer" },
  { id: "equipment", label: "Equipment" },
  { id: "computed", label: "Computed Rollups" },
] as const;

export const SHIPMENT_VARIABLES: SchemaVariable[] = [
  // Shipment Fields
  { name: "proNumber", type: "String", description: "PRO tracking number", category: "shipment" },
  { name: "status", type: "String", description: "Current shipment status", category: "shipment" },
  {
    name: "weight",
    type: "Number",
    description: "Shipment weight",
    category: "shipment",
    nullable: true,
  },
  {
    name: "pieces",
    type: "Integer",
    description: "Number of pieces",
    category: "shipment",
    nullable: true,
  },
  {
    name: "temperatureMin",
    type: "Number",
    description: "Minimum temperature requirement",
    category: "shipment",
    nullable: true,
  },
  {
    name: "temperatureMax",
    type: "Number",
    description: "Maximum temperature requirement",
    category: "shipment",
    nullable: true,
  },
  { name: "ratingUnit", type: "Integer", description: "Rating unit value", category: "shipment" },

  // Customer
  { name: "customer.name", type: "String", description: "Customer name", category: "customer" },
  { name: "customer.code", type: "String", description: "Customer code", category: "customer" },

  // Equipment
  {
    name: "tractorType.name",
    type: "String",
    description: "Tractor type name",
    category: "equipment",
  },
  {
    name: "tractorType.code",
    type: "String",
    description: "Tractor type code",
    category: "equipment",
  },
  {
    name: "tractorType.costPerMile",
    type: "Number",
    description: "Tractor cost per mile",
    category: "equipment",
    nullable: true,
  },
  {
    name: "trailerType.name",
    type: "String",
    description: "Trailer type name",
    category: "equipment",
  },
  {
    name: "trailerType.code",
    type: "String",
    description: "Trailer type code",
    category: "equipment",
  },
  {
    name: "trailerType.costPerMile",
    type: "Number",
    description: "Trailer cost per mile",
    category: "equipment",
    nullable: true,
  },

  // Computed Rollups
  {
    name: "totalDistance",
    type: "Number",
    description: "Total shipment distance in miles",
    category: "computed",
  },
  {
    name: "totalStops",
    type: "Integer",
    description: "Number of stops on the shipment",
    category: "computed",
  },
  {
    name: "totalWeight",
    type: "Number",
    description: "Total weight in pounds",
    category: "computed",
  },
  {
    name: "totalPieces",
    type: "Integer",
    description: "Total number of pieces",
    category: "computed",
  },
  {
    name: "totalLinearFeet",
    type: "Number",
    description: "Total linear feet",
    category: "computed",
  },
  {
    name: "hasHazmat",
    type: "Boolean",
    description: "Whether shipment contains hazmat",
    category: "computed",
  },
  {
    name: "requiresTemperatureControl",
    type: "Boolean",
    description: "Temperature controlled shipment",
    category: "computed",
  },
  {
    name: "temperatureDifferential",
    type: "Number",
    description: "Temperature range differential",
    category: "computed",
  },
  {
    name: "baseRate",
    type: "Number",
    description: "Base rate per unit for freight charge calculation",
    category: "computed",
  },
  {
    name: "freightChargeAmount",
    type: "Number",
    description: "Calculated freight charge amount",
    category: "computed",
  },
  {
    name: "otherChargeAmount",
    type: "Number",
    description: "Sum of other charges",
    category: "computed",
  },
  {
    name: "currentTotalCharge",
    type: "Number",
    description: "Current total charge",
    category: "computed",
  },
];

export const AVAILABLE_VARIABLES = SHIPMENT_VARIABLES;

export const AVAILABLE_FUNCTIONS = [
  { name: "abs", signature: "abs(x)", description: "Absolute value" },
  {
    name: "min",
    signature: "min(a, b, ...)",
    description: "Minimum of values",
  },
  {
    name: "max",
    signature: "max(a, b, ...)",
    description: "Maximum of values",
  },
  {
    name: "pow",
    signature: "pow(base, exp)",
    description: "Power function",
  },
  {
    name: "round",
    signature: "round(x, decimals?)",
    description: "Round to decimal places",
  },
  { name: "ceil", signature: "ceil(x)", description: "Round up" },
  { name: "floor", signature: "floor(x)", description: "Round down" },
  { name: "sqrt", signature: "sqrt(x)", description: "Square root" },
  {
    name: "sum",
    signature: "sum(a, b, ...)",
    description: "Sum of values",
  },
  {
    name: "avg",
    signature: "avg(a, b, ...)",
    description: "Average of values",
  },
  {
    name: "coalesce",
    signature: "coalesce(a, b, ...)",
    description: "First non-null value",
  },
  {
    name: "clamp",
    signature: "clamp(value, min, max)",
    description: "Clamp value to range",
  },
] as const;

export const bulkUpdateStatusRequestSchema = z.object({
  templateIds: z.array(z.string()),
  status: formulaTemplateStatusSchema,
});

export type BulkUpdateStatusRequest = z.infer<typeof bulkUpdateStatusRequestSchema>;

export const bulkDuplicateFormulaTemplateRequestSchema = z.object({
  templateIds: z.array(z.string()).min(1, { error: "Template Ids are required" }),
});

export type BulkDuplicateFormulaTemplateRequest = z.infer<
  typeof bulkDuplicateFormulaTemplateRequestSchema
>;

export const createVersionRequestSchema = z.object({
  changeMessage: z.string().optional(),
});
export type CreateVersionRequest = z.infer<typeof createVersionRequestSchema>;

export const rollbackRequestSchema = z.object({
  targetVersion: z.number(),
  changeMessage: z.string().optional(),
});
export type RollbackRequest = z.infer<typeof rollbackRequestSchema>;

export const forkRequestSchema = z.object({
  newName: z.string().min(1, "Name is required"),
  sourceVersion: z.number().optional(),
  changeMessage: z.string().optional(),
});
export type ForkRequest = z.infer<typeof forkRequestSchema>;

export const templateUsageCountSchema = z.object({
  type: z.string(),
  count: z.number(),
});
export type TemplateUsageCount = z.infer<typeof templateUsageCountSchema>;

export const templateUsageResponseSchema = z.object({
  inUse: z.boolean(),
  usages: z.array(templateUsageCountSchema),
});
export type TemplateUsageResponse = z.infer<typeof templateUsageResponseSchema>;
