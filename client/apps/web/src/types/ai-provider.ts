import { z } from "zod";

/**
 * A provider is identified by the wire protocol it speaks rather than the vendor
 * operating it, which is why one "OpenAI-compatible" entry covers vLLM, SGLang,
 * OpenRouter, Groq and the rest. Ollama is separate because its compatible
 * endpoint ignores JSON schemas while its native one honours them.
 */
export const aiProviderKindSchema = z.enum([
  "AnthropicMessages",
  "OpenAIResponses",
  "OpenAIChat",
  "Ollama",
]);

export const structuredOutputModeSchema = z.enum(["JSONSchema", "JSONMode", "Prompted"]);

/**
 * How hard a model is asked to think before it answers. Off sends no
 * reasoning parameter, which models without reasoning reject outright.
 */
export const reasoningEffortSchema = z.enum(["Off", "Low", "Medium", "High"]);

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
    message: "Embedding dimensions must be 768, 1024 or 1536",
  });

export const aiProviderTestOutcomeSchema = z.object({
  success: z.boolean(),
  message: z.string(),
  modelIdentifier: z.string().optional().default(""),
  schemaHonoured: z.boolean().default(false),
  latencyMs: z.number().default(0),
  detail: z.string().optional().default(""),
  testedAt: z.number(),
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
  extraBody: extraBodySchema,
  inputCostPerMillion: pricePerMillionSchema,
  outputCostPerMillion: pricePerMillionSchema,
  maxTokens: z.number().default(8192),
  /** `[]Task` with nullzero on the server: a provider with no tasks arrives as null. */
  tasks: z.preprocess((value) => value ?? [], z.array(aiTaskSchema)),
  priority: z.number().default(100),
  embeddingDimensions: z.number().nullable().optional().default(null),
  embeddingInputStyle: embeddingInputStyleSchema.default("None"),
  trusted: z.boolean().default(false),
  enabled: z.boolean().default(false),
  lastTest: aiProviderTestOutcomeSchema.nullable().optional().default(null),
  version: z.number().default(0),
  createdAt: z.number(),
  updatedAt: z.number(),
});

export const saveAIProviderRequestSchema = z.object({
  name: z.string().trim().min(1, "Name is required"),
  description: z.string().optional().default(""),
  kind: aiProviderKindSchema,
  baseUrl: z.string().optional().default(""),
  model: z.string().trim().min(1, "Model is required"),
  /**
   * Omitted entirely when unchanged, so editing a provider never round-trips the
   * secret. An empty string clears the stored credential.
   */
  apiKey: z.string().nullable().optional(),
  allowPrivateNetwork: z.boolean().default(false),
  structuredOutputMode: structuredOutputModeSchema,
  reasoningEffort: reasoningEffortSchema.default("Off"),
  extraBody: extraBodySchema,
  inputCostPerMillion: z.number().min(0).nullable().default(null),
  outputCostPerMillion: z.number().min(0).nullable().default(null),
  maxTokens: z.number().min(256).max(200000).default(8192),
  tasks: z.array(aiTaskSchema).default([]),
  priority: z.number().min(0).default(100),
  embeddingDimensions: embeddingDimensionsSchema.nullable().default(null),
  embeddingInputStyle: embeddingInputStyleSchema.default("None"),
  trusted: z.boolean().default(false),
  enabled: z.boolean().default(false),
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
});

export type AIProvider = z.infer<typeof aiProviderSchema>;
export type AIProviderTestOutcome = z.infer<typeof aiProviderTestOutcomeSchema>;
export type AIProviderKind = z.infer<typeof aiProviderKindSchema>;
export type AITask = z.infer<typeof aiTaskSchema>;
export type StructuredOutputMode = z.infer<typeof structuredOutputModeSchema>;
export type ReasoningEffort = z.infer<typeof reasoningEffortSchema>;
export type SaveAIProviderRequest = z.infer<typeof saveAIProviderRequestSchema>;
export type TestAIProviderResult = z.infer<typeof testAIProviderResultSchema>;
export type AIProviderCatalog = z.infer<typeof aiProviderCatalogSchema>;
export type AIProviderPreset = z.infer<typeof aiProviderPresetSchema>;
export type AIProviderKindDescriptor = z.infer<typeof aiProviderKindDescriptorSchema>;
export type AITaskDescriptor = z.infer<typeof aiTaskDescriptorSchema>;
export type EmbeddingInputStyle = z.infer<typeof embeddingInputStyleSchema>;
export type EmbeddingInputStyleDescriptor = z.infer<typeof embeddingInputStyleDescriptorSchema>;
