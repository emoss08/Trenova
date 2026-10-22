// formula-template.ts types formula templates: one Tariff source file each
// (docs/engineering/formula-language.md), with the inputs and pinned library imports the server
// derives from it through `tariff.check`, and the versioning, review, scheduling and Studio
// endpoints around it. The Tariff values, the bill and the diagnostics are in ./tariff.

import { translate } from "@trenova/shared/i18n/runtime";
import { linkedRecordSchema } from "@trenova/shared/lib/linked-record";
import { z } from "zod";
import {
  tariffBillSchema,
  tariffDiagnosticsSchema,
  tariffEvalErrorSchema,
  tariffInputDeclarationsSchema,
  tariffLookupSchema,
  tariffRangeSchema,
  tariffRefusalSchema,
  tariffTestResultSchema,
  tariffTraceEventSchema,
  type TariffInputValues,
} from "./tariff";
import { userSchema } from "./user";

export const formulaTemplateStatusSchema = z.enum(["Active", "Inactive", "Draft", "InReview"]);
export type FormulaTemplateStatus = z.infer<typeof formulaTemplateStatusSchema>;

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

/**
 * An organisation library a saved version imports, pinned to the library version it was checked
 * against (approval pins each to the version approved at that moment).
 */
export const formulaImportSchema = z.object({
  /** The module path the source imports: `org.<name>`. */
  path: z.string(),
  libraryId: z.string(),
  versionNumber: z.number().int(),
});
export type FormulaImport = z.output<typeof formulaImportSchema>;

export const formulaImportsSchema = z
  .array(formulaImportSchema)
  .nullish()
  .transform((value) => value ?? []);
const importsSchema = formulaImportsSchema;

/**
 * A formula template as the API sends it and the Studio edits it. `source` is the whole tariff;
 * `inputs`, `imports` and `testCount` are what the server's `tariff.check` derived from it when
 * it was saved, so a contract form can read the declarations without the module.
 */
export const formulaTemplateSchema = z.object({
  id: z.string().optional(),
  organizationId: z.string().optional(),
  businessUnitId: z.string().optional(),
  name: z
    .string()
    .trim()
    .min(1, { error: () => translate("Name is required") })
    .max(100, { error: () => translate("Name must be 100 characters or fewer") }),
  description: z.string().default(""),
  source: z.string().refine((value) => value.trim().length > 0, {
    error: () => translate("The tariff's source is required"),
  }),
  status: formulaTemplateStatusSchema.default("Draft"),
  inputs: tariffInputDeclarationsSchema,
  imports: importsSchema,
  testCount: z.number().int().default(0),
  submittedById: z.string().nullish(),
  submittedAt: z.number().nullish(),
  approvedById: z.string().nullish(),
  approvedAt: z.number().nullish(),
  reviewComment: z.string().nullish(),
  version: z.number().optional(),
  sourceTemplateId: z.string().nullish(),
  sourceVersionNumber: z.number().nullish(),
  currentVersionNumber: z.number().optional(),
  billing: billingVersionSchema.nullish(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),
  usageCount: z.number().optional(),
});
export type FormulaTemplate = z.output<typeof formulaTemplateSchema>;
export type FormulaTemplateFormValues = z.input<typeof formulaTemplateSchema>;
export const listFormulaTemplateResponseSchema = z.array(formulaTemplateSchema);
export type ListFormulaTemplateResponse = z.infer<typeof listFormulaTemplateResponseSchema>;

/** What a save writes: the person's content. Status, inputs and imports are the server's. */
export type FormulaTemplateWrite = Pick<FormulaTemplate, "name" | "description" | "source"> & {
  version?: number;
};

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
  source: z.string(),
  status: formulaTemplateStatusSchema,
  inputs: tariffInputDeclarationsSchema,
  imports: importsSchema,
  testCount: z.number().int().default(0),
  effectiveFrom: z.number().nullish(),
  changeMessage: z.string().optional(),
  changeSummary: z.record(z.string(), fieldChangeSchema).nullish(),
  tags: z
    .array(versionTagSchema)
    .nullish()
    .transform((v) => v ?? []),
  createdById: z.string(),
  createdAt: z.number(),

  createdBy: linkedRecordSchema("user", userSchema).nullish(),
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

/**
 * What the Studio's preview asks `POST /formula-templates/test`: the source as it is, the
 * sample input values, and a real shipment to rate instead of the sample record.
 */
export type FormulaPreviewRequest = {
  source: string;
  inputs?: TariffInputValues;
  shipmentId?: string;
};

/**
 * What the preview answers: every finding, the bill (or why the tariff refused, or the error
 * that stopped it), the trace and the table lookups, and the file's tests as the server ran them.
 */
export const formulaPreviewResponseSchema = z.object({
  diagnostics: tariffDiagnosticsSchema,
  bill: tariffBillSchema.nullish().transform((value) => value ?? null),
  refusal: tariffRefusalSchema.nullish().transform((value) => value ?? null),
  error: tariffEvalErrorSchema.nullish().transform((value) => value ?? null),
  trace: z
    .array(tariffTraceEventSchema)
    .nullish()
    .transform((value) => value ?? []),
  lookups: z
    .array(tariffLookupSchema)
    .nullish()
    .transform((value) => value ?? []),
  tests: z
    .array(tariffTestResultSchema)
    .nullish()
    .transform((value) => value ?? []),
});
export type FormulaPreviewResponse = z.output<typeof formulaPreviewResponseSchema>;

/** "Save as test": a test block from a real shipment's facts and the bill it gets. */
export type ScaffoldTestRequest = {
  source: string;
  shipmentId: string;
  inputs?: TariffInputValues;
  name: string;
};
export const scaffoldTestResponseSchema = z.object({ test: z.string() });
export type ScaffoldTestResponse = z.output<typeof scaffoldTestResponseSchema>;

export const backtestRequestSchema = z
  .object({
    source: z.string().optional(),
    versionNumber: z.number().int().optional(),
    limit: z.number().int().min(1).max(500).optional(),
  })
  .superRefine((value, ctx) => {
    const hasSource = !!value.source;
    const hasVersion = value.versionNumber != null;

    if (hasSource === hasVersion) {
      ctx.addIssue({
        code: "custom",
        message: translate("Provide exactly one of a source or a version number"),
        path: ["source"],
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
  /** The candidate refused to rate this shipment, in the author's words. */
  candidateRefusal: z.string().optional(),
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
  refusedCount: z.number().default(0),
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

export const importTemplatesResponseSchema = z.object({
  created: listFormulaTemplateResponseSchema,
  renamed: z.record(z.string(), z.string()).nullish(),
});
export type ImportTemplatesResponse = z.output<typeof importTemplatesResponseSchema>;

/**
 * What review compares: the source and the pinned imports of the approved version a submission
 * would replace, against the submission's, with the field changes between them.
 */
export const reviewDiffResponseSchema = z.object({
  hasApprovedBase: z.boolean(),
  baseVersion: z.number(),
  currentVersion: z.number(),
  baseSource: z.string(),
  currentSource: z.string(),
  baseImports: importsSchema,
  currentImports: importsSchema,
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

/** A template the platform ships, as Tariff source with its own tests. */
export const standardTemplateSchema = z.object({
  name: z.string(),
  description: z
    .string()
    .nullish()
    .transform((v) => v ?? ""),
  source: z.string(),
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
  actor: linkedRecordSchema("user", userSchema).nullish(),
  requestedReviewerId: z.string().nullish(),
  requestedReviewer: linkedRecordSchema("user", userSchema).nullish(),
  reviewerNote: z
    .string()
    .nullish()
    .transform((v) => v ?? ""),
});
export type FormulaTemplateReview = z.output<typeof formulaTemplateReviewSchema>;
export const listFormulaTemplateReviewsResponseSchema = z.array(formulaTemplateReviewSchema);

/**
 * The server's verdict on a submission: whether it can be submitted and approved, each check
 * (a clean `tariff.check`, every test passing, the dependents gate), and the test totals.
 */
export const readinessResponseSchema = z.object({
  canSubmit: z.boolean(),
  canApprove: z.boolean(),
  checks: z.array(readinessCheckSchema),
  testTotal: z.number(),
  testPassed: z.number(),
  testFailing: z
    .array(z.string())
    .nullish()
    .transform((v) => v ?? []),
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

/** Someone who may approve: holds the resource's approve permission, and is not the author. */
export const formulaReviewerSchema = z.object({
  id: z.string(),
  name: z.string(),
  username: z.string().nullish(),
  profilePicUrl: z.string().nullish(),
});
export type FormulaReviewer = z.output<typeof formulaReviewerSchema>;
export const listFormulaReviewersResponseSchema = z.array(formulaReviewerSchema);

/** What the Studio sends Nova: the file as it is in the editor, saved or not. */
export type FormulaDraftContent = {
  templateId?: string;
  name: string;
  description: string;
  source: string;
};

/**
 * Nova's reading of a rating: one sentence per statement it ran, each pinned to the source range
 * the trace recorded, and the bill it is explaining.
 */
export const formulaExplainResponseSchema = z.object({
  items: z
    .array(z.object({ range: tariffRangeSchema, says: z.string() }))
    .nullish()
    .transform((v) => v ?? []),
  bill: tariffBillSchema.nullish().transform((value) => value ?? null),
  generated: z.boolean().default(false),
});
export type FormulaExplainResponse = z.output<typeof formulaExplainResponseSchema>;

/** Tests Nova suggests, as `test` blocks to add to the source, each with why it matters. */
export const suggestedTestsResponseSchema = z.object({
  tests: z
    .array(z.object({ name: z.string(), source: z.string(), why: z.string() }))
    .nullish()
    .transform((v) => v ?? []),
  generated: z.boolean().default(false),
});
export type SuggestedTestsResponse = z.output<typeof suggestedTestsResponseSchema>;

export const reviewNoteResponseSchema = z.object({
  note: z.string(),
  generated: z.boolean().default(false),
  backtest: backtestSummarySchema.nullish(),
});
export type ReviewNoteResponse = z.output<typeof reviewNoteResponseSchema>;
