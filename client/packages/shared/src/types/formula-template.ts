import { translate } from "@trenova/shared/i18n/runtime";
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
  name: z.string().min(1, { error: () => translate("Name is required") }),
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
    .min(1, { error: () => translate("Name is required") })
    .regex(/^[a-zA-Z][a-zA-Z0-9_]*$/, {
      error: () =>
        translate("Must start with a letter and contain only letters, numbers, and underscores"),
    }),
  label: z.string().min(1, { error: () => translate("Label is required") }),
  expression: z.string().min(1, { error: () => translate("Expression is required") }),
});
export type BreakdownDefinition = z.output<typeof breakdownDefinitionSchema>;
export type BreakdownDefinitionInput = z.input<typeof breakdownDefinitionSchema>;

export const MAX_BREAKDOWN_DEFINITIONS = 20;

export const breakdownDefinitionsSchema = z
  .array(breakdownDefinitionSchema)
  .max(MAX_BREAKDOWN_DEFINITIONS, {
    error: () =>
      translate("A maximum of {0} breakdown items is allowed", MAX_BREAKDOWN_DEFINITIONS),
  })
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
  .int({ error: () => translate("Rounding precision must be a whole number") })
  .min(0, { error: () => translate("Rounding precision cannot be negative") })
  .max(MAX_ROUNDING_PRECISION, {
    error: () => translate("Rounding precision cannot exceed {0}", MAX_ROUNDING_PRECISION),
  });

/** An approved snapshot whose start has not come yet: approved content waiting to bill. */
export const scheduledWindowSchema = z.object({
  versionNumber: z.number(),
  effectiveFrom: z.number(),
});
export type ScheduledWindow = z.output<typeof scheduledWindowSchema>;

const scheduledWindowsSchema = z
  .array(scheduledWindowSchema)
  .nullish()
  .transform((v) => v ?? []);

/** The approved snapshot that prices shipments today, whatever the row holds. */
export const billingVersionSchema = z.object({
  versionNumber: z.number(),
  effectiveFrom: z.number(),
  /** True while the window has not started and nothing bills until it does. */
  pending: z.boolean().optional(),
});
export type BillingVersion = z.output<typeof billingVersionSchema>;

/** The formula language this client writes; the server refuses one it cannot read. */
export const CURRENT_FORMULA_LANGUAGE_VERSION = 1;

export const formulaTemplateSchema = z
  .object({
    id: z.string().optional(),
    organizationId: z.string().optional(),
    businessUnitId: z.string().optional(),
    name: z
      .string()
      .min(1, { error: () => translate("Name is required") })
      .max(100),
    description: z.string().default(""),
    type: formulaTemplateTypeSchema,
    expression: z.string().min(1, { error: () => translate("Expression is required") }),
    status: formulaTemplateStatusSchema.default("Draft"),
    schemaId: z.string().default("shipment"),
    /** The formula language the expression is written in; every expression so far is version 1. */
    languageVersion: z.number().int().default(CURRENT_FORMULA_LANGUAGE_VERSION),
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
    billing: billingVersionSchema.nullish(),
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
          message: translate("Duplicate variable name; first declared as variable {0}", first + 1),
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
          message: translate("Duplicate breakdown name; first used by item {0}", first + 1),
          path: ["breakdownDefinitions", index, "name"],
        });
      } else {
        seenBreakdowns.set(name, index);
      }
    });

    if (value.minCharge != null && value.minCharge < 0) {
      ctx.addIssue({
        code: "custom",
        message: translate("Minimum charge cannot be negative"),
        path: ["minCharge"],
      });
    }

    if (value.maxCharge != null && value.maxCharge < 0) {
      ctx.addIssue({
        code: "custom",
        message: translate("Maximum charge cannot be negative"),
        path: ["maxCharge"],
      });
    }

    if (value.minCharge != null && value.maxCharge != null && value.minCharge > value.maxCharge) {
      ctx.addIssue({
        code: "custom",
        message: translate("Minimum charge cannot exceed maximum charge"),
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
    color:
      "bg-success-subtle text-success-foreground dark:bg-success-subtle/40 dark:text-success-foreground",
    description: "Tested and ready for use",
  },
  {
    value: "Production",
    label: "Production",
    color: "bg-info-subtle text-info-foreground dark:bg-info-subtle/40 dark:text-info-foreground",
    description: "Currently in production",
  },
  {
    value: "Draft",
    label: "Draft",
    color: "bg-sunken text-foreground dark:bg-sunken dark:text-foreground-subtle",
    description: "Work in progress",
  },
  {
    value: "Testing",
    label: "Testing",
    color:
      "bg-warning-subtle text-warning-foreground dark:bg-warning-subtle/40 dark:text-warning-foreground",
    description: "Under testing",
  },
  {
    value: "Deprecated",
    label: "Deprecated",
    color:
      "bg-danger-subtle text-danger-foreground dark:bg-danger-subtle/40 dark:text-danger-foreground",
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
  languageVersion: z.number().int().default(CURRENT_FORMULA_LANGUAGE_VERSION),
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
  /** Asks the engine to value every statement as well; the live preview leaves it off. */
  traceStatements: z.boolean().optional(),
});
export type TestExpressionRequest = z.infer<typeof testExpressionRequestSchema>;

/**
 * One statement of an expression as the engine evaluated it: each `let`, then
 * the final expression whose value is the raw amount. Offsets are UTF-16 code
 * units into the expression, the units CodeMirror counts in.
 */
export const formulaStatementSchema = z.object({
  index: z.number().int(),
  kind: z.enum(["let", "result"]),
  name: z.string().nullish(),
  code: z.string(),
  start: z.number().int(),
  end: z.number().int(),
  value: z.unknown().optional(),
  error: z.string().nullish(),
});
export type FormulaStatement = z.output<typeof formulaStatementSchema>;

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
  "provided",
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
  statements: z.array(formulaStatementSchema).nullish(),
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
        message: translate("Provide exactly one of expression or version number"),
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
  /** Billed: the amount on the shipment. Recomputed: its billed content rated again today. */
  currentSource: z.enum(["Billed", "Recomputed"]).default("Recomputed"),
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
  /** True when the run hit its time limit before rating every shipment. */
  truncated: z.boolean().default(false),
});
export type BacktestResponse = z.output<typeof backtestResponseSchema>;

export const updateEffectiveDateRequestSchema = z.object({
  effectiveFrom: z.number().nullable(),
});
export type UpdateEffectiveDateRequest = z.infer<typeof updateEffectiveDateRequestSchema>;

export const bulkUpdateStatusRequestSchema = z.object({
  templateIds: z.array(z.string()),
  status: formulaTemplateStatusSchema,
  /** Sent after the person has seen what uses the templates and still means to archive them. */
  confirmInUse: z.boolean().optional(),
});

export type BulkUpdateStatusRequest = z.infer<typeof bulkUpdateStatusRequestSchema>;

export const bulkDuplicateFormulaTemplateRequestSchema = z.object({
  templateIds: z.array(z.string()).min(1, { error: () => translate("Template Ids are required") }),
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
  newName: z.string().min(1, { error: () => translate("Name is required") }),
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

/** What a value is in the formula language, as the engine publishes it for each function's result. */
export const formulaValueTypeSchema = z.enum([
  "Number",
  "Integer",
  "String",
  "Boolean",
  "Date",
  "Duration",
  "Array",
  "Object",
  "Any",
]);
export type FormulaValueType = z.output<typeof formulaValueTypeSchema>;

/** A function's maxArgs when it takes any number of arguments. */
export const VARIADIC_ARGS = -1;

export const formulaSchemaFunctionSchema = z.object({
  name: z.string(),
  signature: z.string(),
  description: z.string().default(""),
  example: z.string().default(""),
  category: z.string().default(""),
  operator: z.boolean().default(false),
  minArgs: z.number().int().default(0),
  maxArgs: z.number().int().default(VARIADIC_ARGS),
  returns: formulaValueTypeSchema.catch("Any").default("Any"),
  /** Every argument must be a number, so text or a flag passed to it is an error. */
  numericArgs: z.boolean().default(false),
  /** The second argument is a predicate over each item, where `#` and `.field` read the item. */
  predicate: z.boolean().default(false),
});
export type FormulaSchemaFunction = z.output<typeof formulaSchemaFunctionSchema>;

/** Why the engine turns an expr builtin off; the editor words the reason. */
export const formulaDisabledReasonSchema = z.enum([
  "nondeterministic",
  "serialization",
  "textBuilding",
]);
export type FormulaDisabledReason = z.output<typeof formulaDisabledReasonSchema>;

export const formulaSchemaDisabledSchema = z.object({
  name: z.string(),
  reason: formulaDisabledReasonSchema,
  replacement: z.string().nullish(),
});
export type FormulaSchemaDisabled = z.output<typeof formulaSchemaDisabledSchema>;

export const formulaSchemaResponseSchema = z.object({
  schemaId: z.string(),
  variables: z.array(formulaSchemaVariableSchema),
  functions: z.array(formulaSchemaFunctionSchema),
  /** The expr builtins the engine keeps beyond the documented functions: callable, not listed in the reference. */
  builtins: z
    .array(formulaSchemaFunctionSchema)
    .nullish()
    .transform((v) => v ?? []),
  disabled: z
    .array(formulaSchemaDisabledSchema)
    .nullish()
    .transform((v) => v ?? []),
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
  /** The template's version number when the expected amount was recorded; 0 before any approval. */
  recordedOnVersion: z.number().int().nullish(),
  recordedById: z.string().nullish(),
  recordedBy: z
    .object({ id: z.string(), name: z.string(), username: z.string().nullish() })
    .nullish(),
  recordedAt: z.number().nullish(),
  createdAt: z.number(),
  updatedAt: z.number(),
});
export type FormulaTestCase = z.output<typeof formulaTestCaseSchema>;

export const formulaTestCaseInputSchema = z.object({
  name: z
    .string()
    .min(1, { error: () => translate("Name is required") })
    .max(100),
  description: z.string().default(""),
  variables: z.record(z.string(), z.any()).default({}),
  expectedAmount: z.coerce
    .number()
    .min(0, { error: () => translate("Expected amount cannot be negative") }),
  tolerance: z.coerce
    .number()
    .min(0, { error: () => translate("Tolerance cannot be negative") })
    .default(0.01),
  /** The editor content the expected amount was computed on, so a later break can be explained. */
  candidate: z.custom<TestCaseCandidate>().optional(),
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
  pendingSchedules: scheduledWindowsSchema,
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

export const formulaReviewDecisionSchema = z.enum([
  "Submitted",
  "Approved",
  "Rejected",
  "ChangesRequested",
  "Expired",
]);
export type FormulaReviewDecision = z.infer<typeof formulaReviewDecisionSchema>;

export const formulaTemplateReviewSchema = z.object({
  id: z.string(),
  templateId: z.string(),
  round: z.number().int(),
  decision: formulaReviewDecisionSchema,
  actorId: z.string().nullish(),
  comment: z
    .string()
    .nullish()
    .transform((v) => v ?? ""),
  baseVersionNumber: z.number().int().default(0),
  createdAt: z.number(),
  actor: userSchema.nullish(),
  requestedReviewerId: z.string().nullish(),
  requestedReviewer: userSchema.nullish(),
  reviewerNote: z
    .string()
    .nullish()
    .transform((v) => v ?? ""),
});
export type FormulaTemplateReview = z.output<typeof formulaTemplateReviewSchema>;
export const listFormulaTemplateReviewsResponseSchema = z.array(formulaTemplateReviewSchema);

export const readinessResponseSchema = z.object({
  canSubmit: z.boolean(),
  canApprove: z.boolean(),
  checks: z.array(readinessCheckSchema),
  scenarioTotal: z.number(),
  scenarioPassed: z.number(),
  scenarioFailing: z.array(z.string()).nullish(),
  pendingSchedules: scheduledWindowsSchema,
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

/** Someone who may approve a template: holds formula template approve, and is not the author asking. */
export const formulaReviewerSchema = z.object({
  id: z.string(),
  name: z.string(),
  username: z.string().nullish(),
  profilePicUrl: z.string().nullish(),
});
export type FormulaReviewer = z.output<typeof formulaReviewerSchema>;
export const listFormulaReviewersResponseSchema = z.array(formulaReviewerSchema);

/** What the studio sends Nova: the editor as it is, saved or not. */
export type FormulaDraftContent = {
  templateId?: string;
  name?: string;
  description?: string;
  type?: FormulaTemplateType;
  schemaId: string;
  expression: string;
  variableDefinitions: VariableDefinitionInput[];
  breakdownDefinitions?: BreakdownDefinitionInput[];
  minCharge?: string | null;
  maxCharge?: string | null;
  roundingMode?: FormulaRoundingMode;
  roundingPrecision?: number;
};

export const formulaExplainResponseSchema = z.object({
  statements: z
    .array(formulaStatementSchema)
    .nullish()
    .transform((v) => v ?? []),
  items: z
    .array(z.object({ statementIndex: z.number().int(), says: z.string() }))
    .nullish()
    .transform((v) => v ?? []),
  rawAmount: z.string().nullish(),
  billedAmount: z.string().nullish(),
  guardrail: guardrailResultSchema.nullish(),
});
export type FormulaExplainResponse = z.output<typeof formulaExplainResponseSchema>;

export const describeInputsResponseSchema = z.object({
  descriptions: z
    .record(z.string(), z.string())
    .nullish()
    .transform((v) => v ?? {}),
});
export type DescribeInputsResponse = z.output<typeof describeInputsResponseSchema>;

export const scenarioVerdictSchema = z.enum(["intended", "bug", "unclear"]);
export type ScenarioVerdict = z.infer<typeof scenarioVerdictSchema>;

export const scenarioTraceRowSchema = z.object({
  kind: z.enum(["input", "term", "formula", "policy", "billed"]),
  name: z.string(),
  before: z.string().nullish(),
  after: z.string().nullish(),
});
export type ScenarioTraceRow = z.output<typeof scenarioTraceRowSchema>;

export const scenarioExplanationSchema = z.object({
  verdict: scenarioVerdictSchema.catch("unclear"),
  why: z.string(),
  generated: z.boolean().default(false),
  expected: z.coerce.number(),
  actual: z.coerce.number().nullish(),
  delta: z.coerce.number().nullish(),
  baselineLabel: z.string(),
  recordedOnVersion: z.number().int().nullish(),
  recordedBy: z.object({ id: z.string(), name: z.string() }).nullish(),
  error: z.string().nullish(),
  rows: z
    .array(scenarioTraceRowSchema)
    .nullish()
    .transform((v) => v ?? []),
});
export type ScenarioExplanation = z.output<typeof scenarioExplanationSchema>;

export const reviewNoteResponseSchema = z.object({
  note: z.string(),
  generated: z.boolean().default(false),
  backtest: backtestSummarySchema.nullish(),
});
export type ReviewNoteResponse = z.output<typeof reviewNoteResponseSchema>;
