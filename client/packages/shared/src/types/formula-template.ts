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

export const formulaRoundingModeSchema = z.enum(["HalfUp", "HalfEven", "Up", "Down", "None"]);
export type FormulaRoundingMode = z.infer<typeof formulaRoundingModeSchema>;

export const DEFAULT_ROUNDING_MODE: FormulaRoundingMode = "HalfUp";
export const DEFAULT_ROUNDING_PRECISION = 2;
export const MAX_ROUNDING_PRECISION = 4;

const roundingPrecisionSchema = z
  .number()
  .int("Rounding precision must be a whole number")
  .min(0, "Rounding precision cannot be negative")
  .max(MAX_ROUNDING_PRECISION, `Rounding precision cannot exceed ${MAX_ROUNDING_PRECISION}`);

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
    roundingMode: formulaRoundingModeSchema.default(DEFAULT_ROUNDING_MODE),
    roundingPrecision: roundingPrecisionSchema.default(DEFAULT_ROUNDING_PRECISION),
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
    usageCount: z.number().optional(),
    scenarioCount: z.number().optional(),
  })
  .superRefine((value, ctx) => {
    const seenVariables = new Map<string, number>();
    value.variableDefinitions.forEach((definition, index) => {
      const name = definition.name.trim();
      if (!name) return;
      const first = seenVariables.get(name);
      if (first !== undefined) {
        ctx.addIssue({
          code: "custom",
          message: `Duplicate variable name; first declared as variable ${first + 1}`,
          path: ["variableDefinitions", index, "name"],
        });
      } else {
        seenVariables.set(name, index);
      }
    });

    const seenBreakdowns = new Map<string, number>();
    value.breakdownDefinitions.forEach((definition, index) => {
      const name = definition.name.trim();
      if (!name) return;
      const first = seenBreakdowns.get(name);
      if (first !== undefined) {
        ctx.addIssue({
          code: "custom",
          message: `Duplicate breakdown name; first used by item ${first + 1}`,
          path: ["breakdownDefinitions", index, "name"],
        });
      } else {
        seenBreakdowns.set(name, index);
      }
    });

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
  roundingMode: formulaRoundingModeSchema.default(DEFAULT_ROUNDING_MODE),
  roundingPrecision: roundingPrecisionSchema.default(DEFAULT_ROUNDING_PRECISION),
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
  minCharge: z.string().optional(),
  maxCharge: z.string().optional(),
  roundingMode: formulaRoundingModeSchema.optional(),
  roundingPrecision: z.number().int().optional(),
});
export type TestExpressionRequest = z.infer<typeof testExpressionRequestSchema>;

export const guardrailResultSchema = z.object({
  applied: z.boolean(),
  bound: z.enum(["min", "max"]).optional(),
  rawAmount: z.coerce.number(),
  minCharge: z.coerce.number().nullish(),
  maxCharge: z.coerce.number().nullish(),
});
export type GuardrailResult = z.output<typeof guardrailResultSchema>;

export const roundingResultSchema = z.object({
  mode: z.string(),
  precision: z.number(),
  applied: z.boolean(),
  unroundedAmount: z.coerce.number(),
});
export type RoundingResult = z.output<typeof roundingResultSchema>;

export const testBreakdownItemSchema = z.object({
  name: z.string(),
  label: z.string().optional().default(""),
  amount: z.coerce.number(),
  error: z.string().optional(),
});
export type TestBreakdownItem = z.output<typeof testBreakdownItemSchema>;

export const formulaValueSourceSchema = z.enum([
  "field",
  "computed",
  "input",
  "override",
  "default",
  "sample",
]);
export type FormulaValueSource = z.infer<typeof formulaValueSourceSchema>;

export const lookupMatchSchema = z.object({
  matchedKey: z.string().optional(),
  adjusted: z.boolean().optional(),
  bandMin: z.coerce.number().nullish(),
  bandMax: z.coerce.number().nullish(),
});
export type LookupMatch = z.output<typeof lookupMatchSchema>;

export const lookupTraceSchema = z.object({
  scope: z.string(),
  table: z.string(),
  keys: z.array(z.any()),
  value: z.number(),
  match: lookupMatchSchema.nullish(),
  error: z.string().optional(),
});
export type LookupTrace = z.output<typeof lookupTraceSchema>;

export const formulaReceiptSchema = z.object({
  variables: z.array(
    z.object({
      name: z.string(),
      value: z.any(),
      source: formulaValueSourceSchema.catch("sample"),
    }),
  ),
  lookups: z.array(lookupTraceSchema).nullish(),
  rawAmount: z.coerce.number(),
  versionNumber: z.number().optional(),
  effectiveFrom: z.number().nullish(),
  durationMicros: z.number().optional(),
});
export type FormulaReceipt = z.output<typeof formulaReceiptSchema>;

export const expressionWarningSchema = z.object({
  scope: z.string(),
  field: z.string(),
  type: z.string().optional(),
  message: z.string(),
  suggestion: z.string(),
});
export type ExpressionWarning = z.output<typeof expressionWarningSchema>;

export const testExpressionResponseSchema = z.object({
  valid: z.boolean(),
  result: z.any().optional(),
  error: z.string().optional(),
  message: z.string(),
  resolvedVariables: z.record(z.string(), z.any()).nullish(),
  breakdown: z.array(testBreakdownItemSchema).nullish(),
  guardrail: guardrailResultSchema.nullish(),
  rounding: roundingResultSchema.nullish(),
  warnings: z.array(expressionWarningSchema).nullish(),
  receipt: formulaReceiptSchema.nullish(),
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
  currentErrorCount: z.number().default(0),
  candidateErrorCount: z.number().default(0),
  guardrailCount: z.number().default(0),
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
  { id: "origin", label: "Origin" },
  { id: "destination", label: "Destination" },
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

  // Origin
  {
    name: "origin.city",
    type: "String",
    description: "City of the first pickup stop",
    category: "origin",
  },
  {
    name: "origin.state",
    type: "String",
    description: "State abbreviation of the first pickup stop",
    category: "origin",
  },
  {
    name: "origin.zip",
    type: "String",
    description: "Postal code of the first pickup stop",
    category: "origin",
  },

  // Destination
  {
    name: "destination.city",
    type: "String",
    description: "City of the final delivery stop",
    category: "destination",
  },
  {
    name: "destination.state",
    type: "String",
    description: "State abbreviation of the final delivery stop",
    category: "destination",
  },
  {
    name: "destination.zip",
    type: "String",
    description: "Postal code of the final delivery stop",
    category: "destination",
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
    name: "totalHours",
    type: "Number",
    description: "Hours from first stop arrival to last stop departure",
    category: "computed",
  },
  {
    name: "pickupDayOfWeek",
    type: "Integer",
    description: "Pickup day of week in UTC (0 = Sunday through 6 = Saturday)",
    category: "computed",
  },
  {
    name: "pickupHour",
    type: "Integer",
    description: "Pickup hour of day in UTC (0-23)",
    category: "computed",
  },
  {
    name: "pickupMonth",
    type: "Integer",
    description: "Pickup month in UTC (1-12)",
    category: "computed",
  },
  {
    name: "isWeekendPickup",
    type: "Boolean",
    description: "Whether the pickup falls on a Saturday or Sunday (UTC)",
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

export const AVAILABLE_FUNCTIONS = [
  { name: "abs", signature: "abs(x)", description: "Absolute value" },
  {
    name: "min",
    signature: "min(...values)",
    description: "Minimum of values",
  },
  {
    name: "max",
    signature: "max(...values)",
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
    description: "Round half up to decimal places",
  },
  {
    name: "roundUp",
    signature: "roundUp(x, decimals?)",
    description: "Round up at decimal places",
  },
  {
    name: "roundDown",
    signature: "roundDown(x, decimals?)",
    description: "Round down at decimal places",
  },
  {
    name: "roundHalfEven",
    signature: "roundHalfEven(x, decimals?)",
    description: "Banker's rounding at decimal places",
  },
  {
    name: "roundTo",
    signature: "roundTo(x, increment)",
    description: "Round to the nearest multiple of increment",
  },
  { name: "ceil", signature: "ceil(x)", description: "Round up to a whole number" },
  { name: "floor", signature: "floor(x)", description: "Round down to a whole number" },
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
  {
    name: "lookup",
    signature: "lookup(table, key)",
    description: "Value for key in a single-axis rate matrix",
  },
  {
    name: "lookupOr",
    signature: "lookupOr(table, key, fallback)",
    description:
      "Like lookup, with a fallback when the key has no entry (a missing table still errors)",
  },
  {
    name: "lookup2",
    signature: "lookup2(table, rowKey, colKey)",
    description: "Value at a row/column intersection of a two-axis rate matrix",
  },
  {
    name: "lookup2Or",
    signature: "lookup2Or(table, rowKey, colKey, fallback)",
    description:
      "Like lookup2, with a fallback when the intersection has no cell (a missing table still errors)",
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

export const formulaSchemaVariableSchema = z.object({
  name: z.string(),
  type: z.string(),
  description: z.string().default(""),
  category: z.string().default(""),
  nullable: z.boolean().default(false),
  computed: z.boolean().default(false),
  enum: z.array(z.string()).nullish(),
});
export type FormulaSchemaVariable = z.output<typeof formulaSchemaVariableSchema>;

export const formulaSchemaFunctionSchema = z.object({
  name: z.string(),
  signature: z.string(),
  description: z.string().default(""),
  example: z.string().default(""),
  category: z.string().default(""),
  operator: z.boolean().default(false),
});
export type FormulaSchemaFunction = z.output<typeof formulaSchemaFunctionSchema>;

export const formulaSchemaResponseSchema = z.object({
  schemaId: z.string(),
  variables: z.array(formulaSchemaVariableSchema),
  functions: z.array(formulaSchemaFunctionSchema),
});
export type FormulaSchemaResponse = z.output<typeof formulaSchemaResponseSchema>;

export const importTemplatesResponseSchema = z.object({
  created: listFormulaTemplateResponseSchema,
  renamed: z.record(z.string(), z.string()).nullish(),
});
export type ImportTemplatesResponse = z.output<typeof importTemplatesResponseSchema>;

export const formulaTestCaseSchema = z.object({
  id: z.string(),
  templateId: z.string(),
  organizationId: z.string(),
  businessUnitId: z.string(),
  name: z.string(),
  description: z
    .string()
    .nullish()
    .transform((v) => v ?? ""),
  variables: z
    .record(z.string(), z.any())
    .nullish()
    .transform((v) => v ?? {}),
  expectedAmount: z.coerce.number(),
  tolerance: z.coerce.number(),
  version: z.number().optional(),
  createdById: z.string(),
  createdAt: z.number(),
  updatedAt: z.number(),
});
export type FormulaTestCase = z.output<typeof formulaTestCaseSchema>;

export const formulaTestCaseInputSchema = z.object({
  name: z.string().min(1, "Name is required").max(100),
  description: z.string().default(""),
  variables: z.record(z.string(), z.any()).default({}),
  expectedAmount: z.coerce.number().min(0, "Expected amount cannot be negative"),
  tolerance: z.coerce.number().min(0, "Tolerance cannot be negative").default(0.01),
});
export type FormulaTestCaseInput = z.input<typeof formulaTestCaseInputSchema>;
export type FormulaTestCaseValues = z.output<typeof formulaTestCaseInputSchema>;

export const testCaseResultSchema = z.object({
  testCaseId: z.string(),
  name: z.string(),
  passed: z.boolean(),
  expectedAmount: z.coerce.number(),
  actualAmount: z.coerce.number(),
  difference: z.coerce.number(),
  tolerance: z.coerce.number(),
  error: z.string().optional(),
});
export type TestCaseResult = z.output<typeof testCaseResultSchema>;

export const runTestCasesResponseSchema = z.object({
  results: z
    .array(testCaseResultSchema)
    .nullish()
    .transform((v) => v ?? []),
  total: z.number(),
  passed: z.number(),
  failed: z.number(),
});
export type RunTestCasesResponse = z.output<typeof runTestCasesResponseSchema>;

export type TestCaseCandidate = {
  expression: string;
  variableDefinitions?: VariableDefinitionInput[];
  breakdownDefinitions?: BreakdownDefinitionInput[];
  minCharge?: number | string | null;
  maxCharge?: number | string | null;
  roundingMode?: FormulaRoundingMode;
  roundingPrecision?: number;
};

export const reviewDiffResponseSchema = z.object({
  hasApprovedBase: z.boolean(),
  baseVersion: z.number(),
  currentVersion: z.number(),
  baseExpression: z.string(),
  currentExpression: z.string(),
  changes: z.record(z.string(), fieldChangeSchema),
  changeCount: z.number(),
});
export type ReviewDiffResponse = z.output<typeof reviewDiffResponseSchema>;

export const readinessCheckSchema = z.object({
  key: z.string(),
  label: z.string(),
  status: z.enum(["pass", "warn", "fail"]),
  detail: z.string().optional(),
});
export type ReadinessCheck = z.output<typeof readinessCheckSchema>;

export const standardTemplateSchema = z.object({
  name: z.string(),
  description: z
    .string()
    .nullish()
    .transform((v) => v ?? ""),
  type: formulaTemplateTypeSchema,
  expression: z.string(),
  schemaId: z.string(),
  variableDefinitions: z
    .array(variableDefinitionSchema)
    .nullish()
    .transform((v) => v ?? []),
});
export type StandardTemplate = z.output<typeof standardTemplateSchema>;
export const listStandardsResponseSchema = z.array(standardTemplateSchema);

export const readinessResponseSchema = z.object({
  canSubmit: z.boolean(),
  canApprove: z.boolean(),
  checks: z.array(readinessCheckSchema),
  scenarioTotal: z.number(),
  scenarioPassed: z.number(),
  scenarioFailing: z.array(z.string()).nullish(),
});
export type ReadinessResponse = z.output<typeof readinessResponseSchema>;

export const installStandardsResponseSchema = z.object({
  installed: listFormulaTemplateResponseSchema,
  skipped: z
    .array(z.string())
    .nullish()
    .transform((v) => v ?? []),
});
export type InstallStandardsResponse = z.output<typeof installStandardsResponseSchema>;

export const generateFormulaRequestSchema = z.object({
  instruction: z.string().min(1, "An instruction is required").max(4000),
  schemaId: z.string().optional(),
  templateType: formulaTemplateTypeSchema.optional(),
});
export type GenerateFormulaRequest = z.infer<typeof generateFormulaRequestSchema>;

export const proposedScenarioSchema = z.object({
  name: z.string(),
  description: z
    .string()
    .nullish()
    .transform((v) => v ?? ""),
  variables: z
    .record(z.string(), z.any())
    .nullish()
    .transform((v) => v ?? {}),
  expectedAmount: z.number().nullish(),
  valid: z.boolean(),
  error: z.string().nullish(),
});
export type ProposedScenario = z.output<typeof proposedScenarioSchema>;

export const generateFormulaResponseSchema = z.object({
  expression: z.string(),
  variableDefinitions: z
    .array(variableDefinitionSchema)
    .nullish()
    .transform((v) => v ?? []),
  explanation: z.string(),
  validation: testExpressionResponseSchema.nullish(),
  scenarios: z
    .array(proposedScenarioSchema)
    .nullish()
    .transform((v) => v ?? []),
  modelIdentifier: z.string().optional(),
});
export type GenerateFormulaResponse = z.output<typeof generateFormulaResponseSchema>;

export const explainFormulaResponseSchema = z.object({
  explanation: z.string(),
  modelIdentifier: z.string().optional(),
});
export type ExplainFormulaResponse = z.output<typeof explainFormulaResponseSchema>;
