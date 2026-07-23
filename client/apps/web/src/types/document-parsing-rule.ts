import { z } from "zod";
import { optionalStringSchema, timestampSchema, versionSchema } from "./helpers";

const stringArraySchema = z.array(z.string()).optional().default([]);
const stringSchemaWithDefault = z.string().optional().default("");
const booleanSchemaWithDefault = z.boolean().optional().default(false);
const numberSchemaWithDefault = z.number().optional().default(0);

export const documentKindSchema = z.enum([
  "RateConfirmation",
  "BillOfLading",
  "ProofOfDelivery",
  "Invoice",
]);
export type DocumentKind = z.infer<typeof documentKindSchema>;

export const ruleVersionStatusSchema = z.enum(["Draft", "Published", "Archived"]);
export type RuleVersionStatus = z.infer<typeof ruleVersionStatusSchema>;

export const parserModeSchema = z.enum(["merge_with_base", "override_base"]);
export type ParserMode = z.infer<typeof parserModeSchema>;

export const reviewStatusSchema = z.enum(["", "Ready", "NeedsReview", "Unavailable"]);
export type ReviewStatus = z.infer<typeof reviewStatusSchema>;

export const fixtureFieldAssertionOperatorSchema = z.enum([
  "exists",
  "not_empty",
  "equals",
  "matches_regex",
  "one_of",
]);
export type FixtureFieldAssertionOperator = z.infer<typeof fixtureFieldAssertionOperatorSchema>;

export const matchConfigSchema = z.object({
  providerFingerprints: stringArraySchema,
  fileNameContains: stringArraySchema,
  requiresAll: stringArraySchema,
  requiresAny: stringArraySchema,
  sectionAnchors: stringArraySchema,
});
export type MatchConfig = z.infer<typeof matchConfigSchema>;

export const sectionRuleSchema = z.object({
  name: z.string().min(1),
  startAnchors: stringArraySchema,
  endAnchors: stringArraySchema,
  captureBlankLine: booleanSchemaWithDefault,
  allowMultiple: booleanSchemaWithDefault,
});
export type SectionRule = z.infer<typeof sectionRuleSchema>;

export const fieldRuleSchema = z.object({
  key: z.string().min(1),
  label: z.string().min(1),
  sectionNames: stringArraySchema,
  aliases: stringArraySchema,
  patterns: stringArraySchema,
  normalizer: stringSchemaWithDefault,
  required: booleanSchemaWithDefault,
  confidence: z.number().min(0).max(1).optional().default(0),
});
export type FieldRule = z.infer<typeof fieldRuleSchema>;

export const stopFieldRuleSchema = z.object({
  fieldKey: z.enum([
    "name",
    "addressLine1",
    "addressLine2",
    "city",
    "state",
    "postalCode",
    "date",
    "timeWindow",
  ]),
  aliases: stringArraySchema,
  patterns: stringArraySchema,
  normalizer: stringSchemaWithDefault,
  confidence: z.number().min(0).max(1).optional().default(0),
  required: booleanSchemaWithDefault,
});
export type StopFieldRule = z.infer<typeof stopFieldRuleSchema>;

export const stopRuleSchema = z.object({
  role: z.enum(["pickup", "delivery", "stop"]),
  required: booleanSchemaWithDefault,
  sectionNames: stringArraySchema,
  startAnchors: stringArraySchema,
  endAnchors: stringArraySchema,
  allowMultiple: booleanSchemaWithDefault,
  sequenceStart: z.number().int().optional().default(0),
  extractors: z.array(stopFieldRuleSchema).min(1),
  appointmentPatterns: stringArraySchema,
});
export type StopRule = z.infer<typeof stopRuleSchema>;

export const ruleDocumentSchema = z.object({
  sections: z.array(sectionRuleSchema).optional().default([]),
  fields: z.array(fieldRuleSchema).optional().default([]),
  stops: z.array(stopRuleSchema).optional().default([]),
});
export type RuleDocument = z.infer<typeof ruleDocumentSchema>;

export const ruleSetSchema = z.object({
  id: optionalStringSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  name: z.string().min(1).max(255),
  description: stringSchemaWithDefault,
  documentKind: documentKindSchema,
  priority: z.number().int().min(0),
  publishedVersionId: z.string().nullable().optional(),
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
});
export type RuleSet = z.infer<typeof ruleSetSchema>;
export type RuleSetFormValues = z.input<typeof ruleSetSchema>;

export const ruleVersionSchema = z.object({
  id: optionalStringSchema,
  ruleSetId: z.string(),
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  versionNumber: z.number().int().min(1),
  status: ruleVersionStatusSchema.optional().default("Draft"),
  label: stringSchemaWithDefault,
  parserMode: parserModeSchema,
  matchConfig: matchConfigSchema.optional().default({
    providerFingerprints: [],
    fileNameContains: [],
    requiresAll: [],
    requiresAny: [],
    sectionAnchors: [],
  }),
  ruleDocument: ruleDocumentSchema.optional().default({
    sections: [],
    fields: [],
    stops: [],
  }),
  validationSummary: z.record(z.string(), z.unknown()).optional().default({}),
  publishedAt: z.number().nullable().optional(),
  publishedById: z.string().nullable().optional(),
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
});
export type RuleVersion = z.infer<typeof ruleVersionSchema>;
export type RuleVersionFormValues = z.input<typeof ruleVersionSchema>;

export const pageSnapshotSchema = z.object({
  pageNumber: z.number().int().min(1),
  text: z.string(),
});
export type PageSnapshot = z.infer<typeof pageSnapshotSchema>;

export const fixtureFieldAssertionSchema = z.object({
  operator: fixtureFieldAssertionOperatorSchema,
  value: stringSchemaWithDefault,
  values: z.array(z.string()).optional().default([]),
  pattern: stringSchemaWithDefault,
});
export type FixtureFieldAssertion = z.infer<typeof fixtureFieldAssertionSchema>;

export const fixtureAssertionsSchema = z.object({
  expectedFields: z.record(z.string(), z.string()).optional().default({}),
  fieldAssertions: z
    .record(z.string(), z.array(fixtureFieldAssertionSchema).optional().default([]))
    .optional()
    .default({}),
  requiredStopRoles: stringArraySchema,
  minimumStopCount: z.number().int().min(0).optional().default(0),
  reviewStatus: reviewStatusSchema.optional().default(""),
});
export type FixtureAssertions = z.infer<typeof fixtureAssertionsSchema>;

export const fixtureSchema = z.object({
  id: optionalStringSchema,
  ruleSetId: z.string(),
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  name: z.string().min(1).max(255),
  description: stringSchemaWithDefault,
  fileName: stringSchemaWithDefault,
  providerFingerprint: stringSchemaWithDefault,
  textSnapshot: z.string().min(1),
  pageSnapshots: z.array(pageSnapshotSchema).optional().default([]),
  assertions: fixtureAssertionsSchema.optional().default({
    expectedFields: {},
    fieldAssertions: {},
    requiredStopRoles: [],
    minimumStopCount: 0,
    reviewStatus: "",
  }),
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
});
export type Fixture = z.infer<typeof fixtureSchema>;
export type FixtureFormValues = z.input<typeof fixtureSchema>;

export const documentParsingFieldSchema = z.object({
  key: z.string(),
  label: z.string(),
  value: z.string(),
  confidence: numberSchemaWithDefault,
  pageNumber: numberSchemaWithDefault,
  reviewRequired: booleanSchemaWithDefault,
  evidenceExcerpt: stringSchemaWithDefault,
  source: stringSchemaWithDefault,
  alternativeValues: z.array(z.string()).optional(),
});
export type DocumentParsingField = z.infer<typeof documentParsingFieldSchema>;

export const documentParsingStopSchema = z.object({
  sequence: z.number(),
  role: z.string(),
  name: z.string(),
  addressLine1: z.string(),
  addressLine2: z.string(),
  city: z.string(),
  state: z.string(),
  postalCode: z.string(),
  date: z.string(),
  timeWindow: z.string(),
  appointmentRequired: booleanSchemaWithDefault,
  pageNumber: numberSchemaWithDefault,
  evidenceExcerpt: stringSchemaWithDefault,
  confidence: numberSchemaWithDefault,
  reviewRequired: booleanSchemaWithDefault,
  source: stringSchemaWithDefault,
});
export type DocumentParsingStop = z.infer<typeof documentParsingStopSchema>;

export const documentParsingConflictSchema = z.object({
  key: z.string(),
  label: z.string(),
  values: z.array(z.string()),
  pageNumbers: z.array(z.number()),
  evidenceExcerpt: z.string(),
  source: z.string(),
});
export type DocumentParsingConflict = z.infer<typeof documentParsingConflictSchema>;

export const documentParsingRuleMetadataSchema = z.object({
  ruleSetId: z.string(),
  ruleSetName: z.string(),
  ruleVersionId: z.string(),
  versionNumber: z.number(),
  parserMode: z.string(),
  providerMatched: z.string(),
  matchSpecificity: z.number(),
});
export type DocumentParsingRuleMetadata = z.infer<typeof documentParsingRuleMetadataSchema>;

export const documentParsingAnalysisSchema = z.object({
  fields: z.record(z.string(), documentParsingFieldSchema).optional(),
  stops: z.array(documentParsingStopSchema).optional(),
  conflicts: z.array(documentParsingConflictSchema).optional(),
  missingFields: z.array(z.string()).optional(),
  signals: z.array(z.string()).optional(),
  reviewStatus: z.string().optional(),
  overallConfidence: z.number().optional(),
  metadata: documentParsingRuleMetadataSchema.nullable().optional(),
});
export type DocumentParsingAnalysis = z.infer<typeof documentParsingAnalysisSchema>;

export const simulationRequestSchema = z.object({
  fileName: z.string(),
  text: z.string(),
  pages: z.array(pageSnapshotSchema),
  providerFingerprint: z.string(),
  baseline: documentParsingAnalysisSchema.nullable().optional(),
});
export type SimulationRequest = z.infer<typeof simulationRequestSchema>;

export const simulationDiffSchema = z.object({
  addedFields: z.array(z.string()).optional(),
  changedFields: z.array(z.string()).optional(),
  addedStopRoles: z.array(z.string()).optional(),
  changedStopRoles: z.array(z.string()).optional(),
});
export type SimulationDiff = z.infer<typeof simulationDiffSchema>;

export const simulationResultSchema = z.object({
  matched: z.boolean(),
  validationPassed: z.boolean(),
  validationErrors: z.array(z.string()).optional(),
  metadata: documentParsingRuleMetadataSchema.nullable().optional(),
  baseline: documentParsingAnalysisSchema.nullable().optional(),
  candidate: documentParsingAnalysisSchema.nullable().optional(),
  diff: simulationDiffSchema,
});
export type SimulationResult = z.infer<typeof simulationResultSchema>;
