import { z } from "zod";
import { translate } from "@trenova/shared/i18n/runtime";

/**
 * A provider is identified by the wire protocol it speaks rather than the vendor
 * operating it, which is why one "OpenAI-compatible" entry covers vLLM, SGLang,
 * OpenRouter, Groq and the rest.
 */
export const aiProviderKindSchema = z.enum(["AnthropicMessages", "OpenAIResponses", "OpenAIChat"]);

export const structuredOutputModeSchema = z.enum(["JSONSchema", "JSONMode", "Prompted"]);

/**
 * How hard a model is asked to think before it answers. Off sends no
 * reasoning parameter, which models without reasoning reject outright; a model
 * that reasons by default then reasons at its own default. None tells such a
 * model not to reason, and Minimal asks for the least it allows.
 */
export const reasoningEffortSchema = z.enum(["Off", "None", "Minimal", "Low", "Medium", "High"]);

/**
 * How a Claude model is asked to think. Auto reads the model id; Effort and
 * Budget say so for a model behind an id the server cannot read, such as a
 * gateway's alias. Only Anthropic Messages providers take anything but Auto.
 */
export const thinkingStyleSchema = z.enum(["Auto", "Effort", "Budget"]);

/**
 * Where an Anthropic Messages provider reaches Claude. Direct calls the base
 * URL with the API key, which every other kind always does.
 */
export const aiProviderPlatformSchema = z.enum([
  "Direct",
  "Bedrock",
  "VertexAI",
  "ClaudePlatformAWS",
  "ClaudePlatformGoogleCloud",
  "Foundry",
]);

/**
 * How a request to a platform proves who sent it. Server uses the cloud
 * credentials of the server Trenova runs on, where its operator allows it,
 * and stores nothing.
 */
export const aiProviderPlatformAuthSchema = z.enum([
  "ApiKey",
  "AccessKey",
  "ServiceAccount",
  "EntraID",
  "Server",
]);

/** A platform setting the editor asks for. */
export const aiProviderPlatformSettingSchema = z.enum([
  "region",
  "projectId",
  "workspaceId",
  "resource",
  "accessKeyId",
  "tenantId",
  "clientId",
]);

/**
 * What a platform needs besides its credential. None of it is secret; a
 * setting the platform does not take is empty.
 */
export const aiProviderPlatformSettingsSchema = z.object({
  auth: aiProviderPlatformAuthSchema,
  region: z
    .string()
    .nullish()
    .transform((value) => value ?? ""),
  projectId: z
    .string()
    .nullish()
    .transform((value) => value ?? ""),
  workspaceId: z
    .string()
    .nullish()
    .transform((value) => value ?? ""),
  accessKeyId: z
    .string()
    .nullish()
    .transform((value) => value ?? ""),
  resource: z
    .string()
    .nullish()
    .transform((value) => value ?? ""),
  tenantId: z
    .string()
    .nullish()
    .transform((value) => value ?? ""),
  clientId: z
    .string()
    .nullish()
    .transform((value) => value ?? ""),
});

/**
 * Vendor request fields the endpoint takes that the protocol does not
 * define. The server stores JSONB and sends null when there are none; an
 * absent object and an empty one mean the same thing here, so both become
 * null.
 */
export const extraBodySchema = z
  .record(z.string(), z.unknown())
  .nullish()
  .transform((value) => (value && Object.keys(value).length > 0 ? value : null));

/**
 * A price in USD per million tokens. The server stores a decimal and sends it
 * as a string; the form edits a number; either shape is accepted and both
 * become a number or null, since null is what "unknown" means here.
 */
export const pricePerMillionSchema = z
  .union([z.number(), z.string()])
  .nullish()
  .transform((value) => {
    if (value === null || value === undefined || value === "") {
      return null;
    }
    const parsed = typeof value === "number" ? value : Number(value);
    return Number.isFinite(parsed) ? parsed : null;
  });

export const aiTaskSchema = z.enum([
  "DocumentClassification",
  "DocumentExtraction",
  "BillingDiagnosis",
  "FormulaAssistant",
  "ScopeClassification",
  "AssistantChat",
  "OperationalInsights",
  "DailyBriefing",
  "QueryCompose",
  "InboundClassification",
  "AccountingMapping",
  "RemittanceExtraction",
  "EvaluationJudge",
  "Embedding",
  "General",
]);

/**
 * How an embedding endpoint is told a stored document from a search query.
 * Voyage takes an input_type field; nomic-embed-text wants a text prefix;
 * everything else is sent both the same way.
 */
export const embeddingInputStyleSchema = z.enum(["None", "VoyageInputType", "NomicPrefix"]);

/** The vector sizes the retrieval indexes are built for, mirrored from the server. */
export const EMBEDDING_DIMENSIONS = [768, 1024, 1536] as const;

export const embeddingDimensionsSchema = z
  .number()
  .int()
  .refine((value) => (EMBEDDING_DIMENSIONS as readonly number[]).includes(value), {
    error: () => translate("Embedding dimensions must be 768, 1024 or 1536"),
  });

export const capActionSchema = z.enum(["Next", "Stop"]);

export const PROVIDER_TIMEOUT_SECONDS = { min: 5, max: 600, initial: 60 } as const;
export const PROVIDER_MAX_CONCURRENT = { min: 1, max: 64, initial: 8 } as const;

export const aiProviderTestOutcomeSchema = z.object({
  success: z.boolean(),
  message: z.string(),
  modelIdentifier: z.string().optional().default(""),
  schemaHonoured: z.boolean().default(false),
  latencyMs: z.number().default(0),
  detail: z.string().optional().default(""),
  testedAt: z.number(),
});

/** The default advisor the server consults when a provider names none. */
export const DEFAULT_ADVISOR_MODEL = "claude-opus-5-5";

/** The platforms that serve Anthropic's advisor tool. */
export const ADVISOR_PLATFORMS = [
  "Direct",
  "ClaudePlatformAWS",
] as const satisfies readonly z.infer<typeof aiProviderPlatformSchema>[];

/**
 * The API refusing the advisor a provider declared, recorded on the first call
 * it refused. Calls go out without the advisor until the advisor, the model or
 * the platform changes.
 */
export const aiProviderAdvisorRefusalSchema = z.object({
  advisor: z.string(),
  model: z.string(),
  message: z.string(),
  refusedAt: z.number(),
});

export const aiProviderSchema = z.object({
  id: z.string(),
  businessUnitId: z.string(),
  organizationId: z.string(),
  name: z.string(),
  description: z.string().optional().default(""),
  kind: aiProviderKindSchema,
  baseUrl: z.string().optional().default(""),
  model: z.string(),
  /** The secret never leaves the server; only whether one is stored is reported. */
  hasApiKey: z.boolean().default(false),
  allowPrivateNetwork: z.boolean().default(false),
  structuredOutputMode: structuredOutputModeSchema,
  reasoningEffort: reasoningEffortSchema.default("Off"),
  thinkingStyle: thinkingStyleSchema.default("Auto"),
  platform: aiProviderPlatformSchema.default("Direct"),
  platformSettings: aiProviderPlatformSettingsSchema.nullable().optional().default(null),
  advisorEnabled: z.boolean().default(false),
  advisorModel: z
    .string()
    .nullish()
    .transform((value) => value ?? ""),
  advisorRefusal: aiProviderAdvisorRefusalSchema.nullable().optional().default(null),
  scripts: z.boolean().default(false),
  scriptWrites: z.boolean().default(false),
  extraBody: extraBodySchema,
  inputCostPerMillion: pricePerMillionSchema,
  outputCostPerMillion: pricePerMillionSchema,
  /** Null charges the protocol's usual share of the input price. */
  cacheReadCostPerMillion: pricePerMillionSchema,
  cacheWriteCostPerMillion: pricePerMillionSchema,
  maxTokens: z.number().default(8192),
  /** `[]Task` with nullzero on the server: a provider with no tasks arrives as null. */
  tasks: z.preprocess((value) => value ?? [], z.array(aiTaskSchema)),
  priority: z.number().default(100),
  embeddingDimensions: z.number().nullable().optional().default(null),
  embeddingInputStyle: embeddingInputStyleSchema.default("None"),
  trusted: z.boolean().default(false),
  enabled: z.boolean().default(false),
  lastTest: aiProviderTestOutcomeSchema.nullable().optional().default(null),
  timeoutSeconds: z.number().default(PROVIDER_TIMEOUT_SECONDS.initial),
  maxConcurrent: z.number().default(PROVIDER_MAX_CONCURRENT.initial),
  monthlyCapUsd: pricePerMillionSchema,
  onCap: capActionSchema.default("Next"),
  version: z.number().default(0),
  createdAt: z.number(),
  updatedAt: z.number(),
});

export const saveAIProviderRequestSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, { error: () => translate("Name is required") }),
  description: z.string().optional().default(""),
  kind: aiProviderKindSchema,
  baseUrl: z.string().optional().default(""),
  model: z
    .string()
    .trim()
    .min(1, { error: () => translate("Model is required") }),
  /**
   * Omitted entirely when unchanged, so editing a provider never round-trips the
   * secret. An empty string clears the stored credential.
   */
  apiKey: z.string().nullable().optional(),
  allowPrivateNetwork: z.boolean().default(false),
  structuredOutputMode: structuredOutputModeSchema,
  reasoningEffort: reasoningEffortSchema.default("Off"),
  thinkingStyle: thinkingStyleSchema.default("Auto"),
  platform: aiProviderPlatformSchema.default("Direct"),
  platformSettings: aiProviderPlatformSettingsSchema.nullable().default(null),
  advisorEnabled: z.boolean().default(false),
  advisorModel: z.string().trim().default(""),
  scripts: z.boolean().default(false),
  scriptWrites: z.boolean().default(false),
  extraBody: extraBodySchema,
  inputCostPerMillion: z.number().min(0).nullable().default(null),
  outputCostPerMillion: z.number().min(0).nullable().default(null),
  cacheReadCostPerMillion: z.number().min(0).nullable().default(null),
  cacheWriteCostPerMillion: z.number().min(0).nullable().default(null),
  maxTokens: z.number().min(256).max(200000).default(8192),
  tasks: z.array(aiTaskSchema).default([]),
  priority: z.number().min(0).default(100),
  embeddingDimensions: embeddingDimensionsSchema.nullable().default(null),
  embeddingInputStyle: embeddingInputStyleSchema.default("None"),
  trusted: z.boolean().default(false),
  enabled: z.boolean().default(false),
  timeoutSeconds: z
    .number()
    .int()
    .min(PROVIDER_TIMEOUT_SECONDS.min, {
      error: () => translate("At least {0} seconds", PROVIDER_TIMEOUT_SECONDS.min),
    })
    .max(PROVIDER_TIMEOUT_SECONDS.max, {
      error: () => translate("At most {0} seconds", PROVIDER_TIMEOUT_SECONDS.max),
    })
    .default(PROVIDER_TIMEOUT_SECONDS.initial),
  maxConcurrent: z
    .number()
    .int()
    .min(PROVIDER_MAX_CONCURRENT.min, {
      error: () => translate("At least {0}", PROVIDER_MAX_CONCURRENT.min),
    })
    .max(PROVIDER_MAX_CONCURRENT.max, {
      error: () => translate("At most {0}", PROVIDER_MAX_CONCURRENT.max),
    })
    .default(PROVIDER_MAX_CONCURRENT.initial),
  monthlyCapUsd: z
    .number()
    .positive({ error: () => translate("A cap must be more than $0") })
    .nullable()
    .default(null),
  onCap: capActionSchema.default("Next"),
  keepPreviousKey: z.boolean().default(false),
  version: z.number().default(0),
});

export const testAIProviderResultSchema = z.object({
  success: z.boolean(),
  message: z.string(),
  modelIdentifier: z.string().optional().default(""),
  schemaHonoured: z.boolean().default(false),
  latencyMs: z.number().default(0),
  detail: z.string().optional().default(""),
});

export const aiProviderKindDescriptorSchema = z.object({
  kind: aiProviderKindSchema,
  label: z.string(),
  description: z.string(),
  defaultBaseUrl: z.string().optional().default(""),
  requiresApiKey: z.boolean().default(false),
  requiresBaseUrl: z.boolean().default(false),
  defaultStructuredOutputMode: structuredOutputModeSchema,
  supportsStructuredEnforced: z.boolean().default(false),
  supportsEmbedding: z.boolean().default(false),
});

export const aiProviderPresetSchema = z.object({
  key: z.string(),
  label: z.string(),
  kind: aiProviderKindSchema,
  baseUrl: z.string().optional().default(""),
  structuredOutputMode: structuredOutputModeSchema,
  allowPrivateNetwork: z.boolean().default(false),
  requiresApiKey: z.boolean().default(false),
  selfHosted: z.boolean().default(false),
  exampleModel: z.string().optional().default(""),
  notes: z.string().optional().default(""),
  /** Vendor web domain, used to resolve a brand logo. Empty for generic servers. */
  domain: z.string().optional().default(""),
  /** Tasks the preset is made for; an embedding preset names only Embedding. */
  tasks: z.preprocess((value) => value ?? [], z.array(aiTaskSchema)),
  embeddingDimensions: z.number().optional().default(0),
  embeddingInputStyle: embeddingInputStyleSchema.optional(),
  /** Where an Anthropic Messages preset reaches Claude. */
  platform: aiProviderPlatformSchema.optional().default("Direct"),
});

export const aiProviderPlatformAuthDescriptorSchema = z.object({
  auth: aiProviderPlatformAuthSchema,
  requiresCredential: z.boolean(),
  settings: z.preprocess((value) => value ?? [], z.array(aiProviderPlatformSettingSchema)),
});

/**
 * The shape of a platform: the settings it needs, the ways it signs in (the
 * first being its default), and whether it lists its models.
 */
export const aiProviderPlatformDescriptorSchema = z.object({
  platform: aiProviderPlatformSchema,
  settings: z.preprocess((value) => value ?? [], z.array(aiProviderPlatformSettingSchema)),
  auths: z.preprocess((value) => value ?? [], z.array(aiProviderPlatformAuthDescriptorSchema)),
  listsModels: z.boolean().default(true),
});

export const aiTaskDescriptorSchema = z.object({
  task: aiTaskSchema,
  label: z.string(),
  description: z.string(),
  requiresTrust: z.boolean().default(false),
  volumeGuidance: z.string().optional().default(""),
});

export const embeddingInputStyleDescriptorSchema = z.object({
  style: embeddingInputStyleSchema,
  label: z.string(),
  description: z.string(),
});

export const aiProviderCatalogSchema = z.object({
  kinds: z.array(aiProviderKindDescriptorSchema),
  presets: z.array(aiProviderPresetSchema),
  tasks: z.array(aiTaskDescriptorSchema),
  embeddingDimensions: z.preprocess((value) => value ?? [], z.array(z.number())),
  embeddingInputStyles: z.preprocess(
    (value) => value ?? [],
    z.array(embeddingInputStyleDescriptorSchema),
  ),
  platforms: z.preprocess((value) => value ?? [], z.array(aiProviderPlatformDescriptorSchema)),
  /** Whether a platform provider may sign in with the server's own cloud identity. */
  serverCloudCredentials: z.boolean().default(false),
});

export type AIProvider = z.infer<typeof aiProviderSchema>;
export type AIProviderAdvisorRefusal = z.infer<typeof aiProviderAdvisorRefusalSchema>;
export type AIProviderTestOutcome = z.infer<typeof aiProviderTestOutcomeSchema>;
export type AIProviderKind = z.infer<typeof aiProviderKindSchema>;
export type AITask = z.infer<typeof aiTaskSchema>;
export type CapAction = z.infer<typeof capActionSchema>;
export type StructuredOutputMode = z.infer<typeof structuredOutputModeSchema>;
export type ReasoningEffort = z.infer<typeof reasoningEffortSchema>;
export type ThinkingStyle = z.infer<typeof thinkingStyleSchema>;
export type AIProviderPlatform = z.infer<typeof aiProviderPlatformSchema>;
export type AIProviderPlatformAuth = z.infer<typeof aiProviderPlatformAuthSchema>;
export type AIProviderPlatformSetting = z.infer<typeof aiProviderPlatformSettingSchema>;
export type AIProviderPlatformSettings = z.infer<typeof aiProviderPlatformSettingsSchema>;
export type AIProviderPlatformDescriptor = z.infer<typeof aiProviderPlatformDescriptorSchema>;
export type SaveAIProviderRequest = z.infer<typeof saveAIProviderRequestSchema>;
export type TestAIProviderResult = z.infer<typeof testAIProviderResultSchema>;
export type AIProviderCatalog = z.infer<typeof aiProviderCatalogSchema>;
export type AIProviderPreset = z.infer<typeof aiProviderPresetSchema>;
export type AIProviderKindDescriptor = z.infer<typeof aiProviderKindDescriptorSchema>;
export type AITaskDescriptor = z.infer<typeof aiTaskDescriptorSchema>;
export type EmbeddingInputStyle = z.infer<typeof embeddingInputStyleSchema>;
export type EmbeddingInputStyleDescriptor = z.infer<typeof embeddingInputStyleDescriptorSchema>;
