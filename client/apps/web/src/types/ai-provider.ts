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

export const aiTaskSchema = z.enum([
  "DocumentClassification",
  "DocumentExtraction",
  "BillingDiagnosis",
  "FormulaAssistant",
  "ScopeClassification",
  "AssistantChat",
  "OperationalInsights",
  "General",
]);

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
  maxTokens: z.number().default(8192),
  /** `[]Task` with nullzero on the server: a provider with no tasks arrives as null. */
  tasks: z.preprocess((value) => value ?? [], z.array(aiTaskSchema)),
  priority: z.number().default(100),
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
  maxTokens: z.number().min(256).max(200000).default(8192),
  tasks: z.array(aiTaskSchema).default([]),
  priority: z.number().min(0).default(100),
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
});

export const aiTaskDescriptorSchema = z.object({
  task: aiTaskSchema,
  label: z.string(),
  description: z.string(),
  requiresTrust: z.boolean().default(false),
  volumeGuidance: z.string().optional().default(""),
});

export const aiProviderCatalogSchema = z.object({
  kinds: z.array(aiProviderKindDescriptorSchema),
  presets: z.array(aiProviderPresetSchema),
  tasks: z.array(aiTaskDescriptorSchema),
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
